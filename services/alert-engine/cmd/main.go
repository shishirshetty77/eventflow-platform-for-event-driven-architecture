// Package main is the entry point for the alert-engine service.
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
	"github.com/microservices-platform/services/alert-engine/internal/config"
	"github.com/microservices-platform/services/alert-engine/internal/core"
	"github.com/microservices-platform/services/alert-engine/internal/dispatchers"
	"github.com/microservices-platform/services/alert-engine/internal/ports"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize logger
	logConfig := &logging.Config{
		Level:       cfg.LogLevel,
		Development: cfg.Environment == "development",
		ServiceName: cfg.ServiceName,
	}
	logger, err := logging.NewLogger(logConfig)
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Info("starting alert-engine service",
		zap.String("environment", cfg.Environment),
		zap.String("version", cfg.Version),
	)

	// Initialize metrics
	m := metrics.NewMetrics(cfg.ServiceName)

	// Initialize dispatchers
	dispatcherList := []ports.AlertDispatcher{
		dispatchers.NewSlackDispatcher(
			cfg.SlackWebhookURL,
			cfg.SlackChannel,
			logger,
			cfg.SlackEnabled,
		),
		dispatchers.NewEmailDispatcher(
			cfg.SendGridAPIKey,
			cfg.SendGridFromEmail,
			cfg.SendGridFromName,
			cfg.EmailRecipients,
			logger,
			cfg.EmailEnabled,
		),
		dispatchers.NewWebhookDispatcher(
			cfg.WebhookURLs,
			cfg.WebhookHeaders,
			logger,
			cfg.WebhookEnabled,
		),
	}

	// Log dispatcher status
	for _, d := range dispatcherList {
		logger.Info("dispatcher status",
			zap.String("name", d.Name()),
			zap.Bool("enabled", d.Enabled()),
		)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize processor
	processorConfig := &core.ProcessorConfig{
		MaxRetries:               cfg.MaxRetries,
		RetryDelaySeconds:        cfg.RetryDelaySeconds,
		GroupingWindowSeconds:    cfg.GroupingWindowSeconds,
		SuppressionWindowSeconds: cfg.SuppressionWindowSeconds,
		MaxAlertsPerGroup:        cfg.MaxAlertsPerGroup,
		BatchSize:                cfg.BatchSize,
		BatchTimeout:             cfg.BatchTimeout,
	}

	// Check if Kafka is available
	kafkaAvailable := len(cfg.KafkaBrokers) > 0 && cfg.KafkaBrokers[0] != ""

	var processor interface {
		Start(context.Context) error
		Stop() error
	}

	if kafkaAvailable {
		kafkaProcessor, err := core.NewAlertProcessor(
			processorConfig,
			cfg.KafkaBrokers,
			cfg.AlertsTopic,
			cfg.DLQTopic,
			cfg.ConsumerGroup,
			dispatcherList,
			logger,
			m,
		)
		if err != nil {
			logger.Warn("failed to initialize Kafka processor, using mock",
				zap.Error(err),
			)
			processor = core.NewMockAlertProcessor(processorConfig, dispatcherList, logger)
		} else {
			processor = kafkaProcessor
			logger.Info("Kafka alert processor initialized",
				zap.Strings("brokers", cfg.KafkaBrokers),
			)
		}
	} else {
		logger.Info("Kafka not configured, using mock alert processor")
		processor = core.NewMockAlertProcessor(processorConfig, dispatcherList, logger)
	}

	// Start processor
	if err := processor.Start(ctx); err != nil {
		logger.Fatal("failed to start processor", zap.Error(err))
	}
	defer processor.Stop()

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
		w.Write([]byte(`{"status":"healthy","service":"alert-engine"}`))
	})
	healthMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready","service":"alert-engine"}`))
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

	logger.Info("alert-engine service started successfully")

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

	// Cancel context to stop processor
	cancel()

	logger.Info("alert-engine service stopped")
}
