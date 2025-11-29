// Package main is the entry point for the orders service.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/metrics"
	"github.com/microservices-platform/pkg/shared/models"
	"github.com/microservices-platform/pkg/shared/tracing"
	"github.com/microservices-platform/services/orders/internal/config"
	"github.com/microservices-platform/services/orders/internal/core"
)

func main() {
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
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("starting orders service",
		zap.String("version", cfg.Version),
		zap.String("environment", cfg.Environment),
	)

	// Initialize tracing
	if cfg.TracingEnabled {
		tracingConfig := &tracing.Config{
			ServiceName:    cfg.ServiceName,
			ServiceVersion: cfg.Version,
			Environment:    cfg.Environment,
			Endpoint:       cfg.TracingEndpoint,
			SampleRate:     cfg.SampleRate,
			Enabled:        true,
		}
		tracer, err := tracing.NewTracer(tracingConfig)
		if err != nil {
			logger.Warn("failed to initialize tracing", zap.Error(err))
		} else {
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				tracer.Shutdown(ctx)
			}()
		}
	}

	// Initialize Prometheus metrics
	promMetrics := metrics.NewMetrics(cfg.ServiceName)

	// Initialize metrics publisher
	var publisher core.MetricsPublisher
	publisher, err = core.NewKafkaPublisher(
		cfg.KafkaBrokers,
		cfg.KafkaMetricsTopic,
		cfg.KafkaLogsTopic,
		logger,
		promMetrics,
	)
	if err != nil {
		logger.Warn("failed to create Kafka publisher, using mock",
			zap.Error(err),
		)
		publisher = core.NewMockPublisher(logger)
	}
	defer publisher.Close()

	// Initialize and start metrics generator
	metricsGenerator := core.NewMetricsGenerator(
		publisher,
		logger,
		promMetrics,
		cfg.MetricsInterval,
		cfg.LogsInterval,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := metricsGenerator.Start(ctx); err != nil {
		logger.Error("failed to start metrics generator", zap.Error(err))
	}

	// Create HTTP server with basic handlers
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"status":    "healthy",
				"service":   "orders",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
		})
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: true,
			Data:    map[string]interface{}{"ready": true},
		})
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	metricsServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.MetricsPort),
		Handler:      promMetrics.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Start servers
	go func() {
		logger.Info("starting HTTP server", zap.Int("port", cfg.HTTPPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	go func() {
		logger.Info("starting metrics server", zap.Int("port", cfg.MetricsPort))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server failed", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("received shutdown signal, gracefully shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := metricsGenerator.Stop(); err != nil {
		logger.Error("failed to stop metrics generator", zap.Error(err))
	}

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown failed", zap.Error(err))
	}

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("metrics server shutdown failed", zap.Error(err))
	}

	logger.Info("orders service stopped")
}
