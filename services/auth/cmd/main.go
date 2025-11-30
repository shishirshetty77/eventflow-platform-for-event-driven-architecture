// Package main is the entry point for the auth service.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/microservices-platform/pkg/shared/jwt"
	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/metrics"
	"github.com/microservices-platform/pkg/shared/models"
	"github.com/microservices-platform/pkg/shared/tracing"
	"github.com/microservices-platform/services/auth/internal/adapters"
	"github.com/microservices-platform/services/auth/internal/config"
	"github.com/microservices-platform/services/auth/internal/core"
	"github.com/microservices-platform/services/auth/internal/ports"
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
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("starting auth service",
		zap.String("version", cfg.Version),
		zap.String("environment", cfg.Environment),
	)
	// Triggering CI pipeline - attempt 4

	// Initialize tracing
	var tracer *tracing.Tracer
	if cfg.TracingEnabled {
		tracingConfig := &tracing.Config{
			ServiceName:    cfg.ServiceName,
			ServiceVersion: cfg.Version,
			Environment:    cfg.Environment,
			Endpoint:       cfg.TracingEndpoint,
			SampleRate:     cfg.SampleRate,
			Enabled:        true,
		}
		tracer, err = tracing.NewTracer(tracingConfig)
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

	// Initialize user repository
	userRepo := adapters.NewInMemoryUserRepository()

	// Initialize JWT manager
	jwtConfig := &jwt.Config{
		SecretKey:   cfg.JWTSecretKey,
		Issuer:      cfg.JWTIssuer,
		TokenExpiry: cfg.JWTExpiry,
	}
	jwtManager := jwt.NewManager(jwtConfig)

	// Initialize auth service
	authService := core.NewAuthService(userRepo, jwtManager, logger)

	// Initialize metrics publisher
	var publisher ports.MetricsPublisher
	publisher, err = adapters.NewKafkaMetricsPublisher(
		cfg.KafkaBrokers,
		cfg.KafkaMetricsTopic,
		cfg.KafkaLogsTopic,
		logger,
		promMetrics,
	)
	if err != nil {
		logger.Warn("failed to create Kafka publisher, using mock publisher",
			zap.Error(err),
		)
		publisher = adapters.NewMockMetricsPublisher(logger)
	}
	defer publisher.Close()

	// Initialize and start metrics generator
	metricsGenerator := core.NewMetricsGenerator(
		models.ServiceAuth,
		publisher,
		logger,
		cfg.MetricsInterval,
		cfg.LogsInterval,
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics generator
	if err := metricsGenerator.Start(ctx); err != nil {
		logger.Error("failed to start metrics generator", zap.Error(err))
	}

	// Initialize HTTP handler
	httpHandler := adapters.NewHTTPHandler(authService, logger, promMetrics)

	// Create HTTP servers
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      httpHandler.Router(),
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

	// Start HTTP server
	go func() {
		logger.Info("starting HTTP server", zap.Int("port", cfg.HTTPPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// Start metrics server
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

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop metrics generator
	if err := metricsGenerator.Stop(); err != nil {
		logger.Error("failed to stop metrics generator", zap.Error(err))
	}

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown failed", zap.Error(err))
	}

	// Shutdown metrics server
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("metrics server shutdown failed", zap.Error(err))
	}

	logger.Info("auth service stopped")
}
