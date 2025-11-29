// Package main is the entry point for the analyzer service.
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/metrics"
	"github.com/microservices-platform/pkg/shared/models"
	"github.com/microservices-platform/services/analyzer/internal/adapters"
	"github.com/microservices-platform/services/analyzer/internal/config"
	"github.com/microservices-platform/services/analyzer/internal/core"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize logger
	logger, err := logging.NewLogger(cfg.Environment, cfg.LogLevel)
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Info("starting analyzer service",
		zap.String("environment", cfg.Environment),
		zap.String("version", cfg.Version),
	)

	// Initialize metrics
	m := metrics.NewMetrics("analyzer", cfg.Environment, cfg.Version)

	// Initialize Redis store
	redisStore, err := adapters.NewRedisStore(
		cfg.RedisAddr,
		cfg.RedisPassword,
		cfg.RedisDB,
		time.Duration(cfg.SlidingWindowSeconds)*time.Second,
		logger,
	)
	if err != nil {
		logger.Fatal("failed to initialize Redis store", zap.Error(err))
	}
	defer redisStore.Close()

	logger.Info("connected to Redis", zap.String("addr", cfg.RedisAddr))

	// Initialize Kafka alert publisher
	var alertPublisher interface {
		PublishAlert(context.Context, *models.Alert) error
		Close() error
	}

	// Check if Kafka is available
	kafkaAvailable := len(cfg.KafkaBrokers) > 0 && cfg.KafkaBrokers[0] != ""
	if kafkaAvailable {
		kafkaPublisher, err := adapters.NewKafkaAlertPublisher(
			cfg.KafkaBrokers,
			cfg.AlertsTopic,
			logger,
			m,
		)
		if err != nil {
			logger.Warn("failed to initialize Kafka alert publisher, using mock",
				zap.Error(err),
			)
			alertPublisher = adapters.NewMockAlertPublisher(logger)
		} else {
			alertPublisher = kafkaPublisher
			logger.Info("Kafka alert publisher initialized",
				zap.Strings("brokers", cfg.KafkaBrokers),
				zap.String("topic", cfg.AlertsTopic),
			)
		}
	} else {
		logger.Info("Kafka not configured, using mock alert publisher")
		alertPublisher = adapters.NewMockAlertPublisher(logger)
	}
	defer alertPublisher.Close()

	// Initialize analyzer
	analyzerConfig := &core.AnalysisConfig{
		SlidingWindowSize:         time.Duration(cfg.SlidingWindowSeconds) * time.Second,
		RollingWindowSize:         time.Duration(cfg.RollingWindowSeconds) * time.Second,
		AnalysisInterval:          time.Duration(cfg.AnalysisIntervalSeconds) * time.Second,
		DefaultCPUThreshold:       cfg.DefaultCPUThreshold,
		DefaultMemoryThreshold:    cfg.DefaultMemoryThreshold,
		DefaultLatencyThreshold:   cfg.DefaultLatencyThreshold,
		DefaultErrorRateThreshold: cfg.DefaultErrorRateThreshold,
		DeviationMultiplier:       cfg.DeviationMultiplier,
		MinSamplesForDeviation:    cfg.MinSamplesForDeviation,
		DefaultCooldownPeriod:     time.Duration(cfg.DefaultCooldownSeconds) * time.Second,
	}

	analyzer := core.NewAnalyzer(
		analyzerConfig,
		redisStore,
		redisStore, // RedisStore implements both MetricsStore and RulesStore
		alertPublisher.(interface {
			PublishAlert(context.Context, *models.Alert) error
		}),
		logger,
	)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics consumer if Kafka is available
	if kafkaAvailable {
		consumer, err := adapters.NewKafkaMetricsConsumer(
			cfg.KafkaBrokers,
			cfg.MetricsTopic,
			cfg.LogsTopic,
			cfg.ConsumerGroup,
			redisStore,
			logger,
			m,
		)
		if err != nil {
			logger.Warn("failed to initialize Kafka metrics consumer",
				zap.Error(err),
			)
		} else {
			if err := consumer.Start(ctx); err != nil {
				logger.Error("failed to start metrics consumer", zap.Error(err))
			} else {
				defer consumer.Stop()
				logger.Info("Kafka metrics consumer started")
			}
		}
	}

	// Start analyzer
	if err := analyzer.Start(ctx); err != nil {
		logger.Fatal("failed to start analyzer", zap.Error(err))
	}
	defer analyzer.Stop()

	// Start metrics HTTP server
	metricsServer := &http.Server{
		Addr:    cfg.MetricsAddr,
		Handler: promhttp.Handler(),
	}

	go func() {
		logger.Info("starting metrics server", zap.String("addr", cfg.MetricsAddr))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server error", zap.Error(err))
		}
	}()

	// Start health check server
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"analyzer"}`))
	})
	healthMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Check Redis connectivity
		if err := redisStore.Ping(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not_ready","reason":"redis_unavailable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready","service":"analyzer"}`))
	})

	healthServer := &http.Server{
		Addr:    cfg.HealthAddr,
		Handler: healthMux,
	}

	go func() {
		logger.Info("starting health server", zap.String("addr", cfg.HealthAddr))
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("health server error", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("analyzer service started successfully")

	sig := <-sigCh
	logger.Info("received shutdown signal", zap.String("signal", sig.String()))

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown health server
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("health server shutdown error", zap.Error(err))
	}

	// Shutdown metrics server
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("metrics server shutdown error", zap.Error(err))
	}

	// Cancel context to stop analyzer and consumer
	cancel()

	logger.Info("analyzer service stopped")
}
