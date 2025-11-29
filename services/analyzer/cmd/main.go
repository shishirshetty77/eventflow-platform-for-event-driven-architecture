// Package main is the entry point for the analyzer service.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/metrics"
	"github.com/microservices-platform/services/analyzer/internal/adapters"
	"github.com/microservices-platform/services/analyzer/internal/config"
	"github.com/microservices-platform/services/analyzer/internal/core"
	"github.com/microservices-platform/services/analyzer/internal/ports"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logConfig := &logging.Config{
		Level:       cfg.LogLevel,
		Development: cfg.Development,
		ServiceName: cfg.ServiceName,
		OutputPaths: []string{"stdout"},
	}
	logger, err := logging.NewLogger(logConfig)
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Info("starting analyzer service",
		zap.String("environment", cfg.Environment),
		zap.String("version", cfg.Version),
	)

	// Initialize metrics
	m := metrics.NewMetrics(cfg.ServiceName)

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}
	logger.Info("connected to Redis", zap.String("addr", cfg.RedisAddr))

	// Initialize Redis stores
	metricsStore := adapters.NewRedisMetricsStore(redisClient, logger)
	rulesStore := adapters.NewRedisRuleStore(redisClient, logger)

	// Initialize Kafka alert publisher
	var alertPublisher ports.AlertPublisher

	// Check if Kafka is available
	kafkaAvailable := len(cfg.KafkaBrokers) > 0 && cfg.KafkaBrokers[0] != ""
	if kafkaAvailable {
		kafkaPublisher, err := adapters.NewKafkaAlertPublisher(
			cfg.KafkaBrokers,
			cfg.KafkaAlertsTopic,
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
				zap.String("topic", cfg.KafkaAlertsTopic),
			)
		}
	} else {
		logger.Info("Kafka not configured, using mock alert publisher")
		alertPublisher = adapters.NewMockAlertPublisher(logger)
	}
	defer alertPublisher.Close()

	// Initialize analyzer
	analyzerConfig := &core.AnalysisConfig{
		SlidingWindowSize:         cfg.SlidingWindowSize,
		RollingWindowSize:         cfg.SlidingWindowSize,
		AnalysisInterval:          cfg.AnalysisInterval,
		DefaultCPUThreshold:       80.0,
		DefaultMemoryThreshold:    85.0,
		DefaultLatencyThreshold:   1000.0,
		DefaultErrorRateThreshold: 5.0,
		DeviationMultiplier:       2.0,
		MinSamplesForDeviation:    10,
		DefaultCooldownPeriod:     cfg.AlertCooldown,
	}

	analyzer := core.NewAnalyzer(
		analyzerConfig,
		metricsStore,
		rulesStore,
		alertPublisher,
		logger,
	)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics consumer if Kafka is available
	if kafkaAvailable {
		consumer, err := adapters.NewKafkaMetricsConsumer(
			cfg.KafkaBrokers,
			cfg.KafkaMetricsTopic,
			cfg.KafkaLogsTopic,
			cfg.KafkaConsumerGroup,
			metricsStore,
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
	metricsAddr := fmt.Sprintf(":%d", cfg.MetricsPort)
	metricsServer := &http.Server{
		Addr:    metricsAddr,
		Handler: promhttp.Handler(),
	}

	go func() {
		logger.Info("starting metrics server", zap.String("addr", metricsAddr))
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
		if err := redisClient.Ping(ctx).Err(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not_ready","reason":"redis_unavailable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready","service":"analyzer"}`))
	})

	healthAddr := fmt.Sprintf(":%d", cfg.HTTPPort)
	healthServer := &http.Server{
		Addr:    healthAddr,
		Handler: healthMux,
	}

	go func() {
		logger.Info("starting health server", zap.String("addr", healthAddr))
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
