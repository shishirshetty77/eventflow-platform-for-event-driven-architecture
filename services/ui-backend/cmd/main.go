package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/microservices-platform/pkg/shared/jwt"
	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/services/ui-backend/internal/config"
	"github.com/microservices-platform/services/ui-backend/internal/handlers"
	"github.com/microservices-platform/services/ui-backend/internal/store"
)

func main() {
	cfg := config.LoadConfig()

	logger, err := logging.NewLogger(cfg.Environment, cfg.LogLevel)
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Info("starting ui-backend service",
		zap.String("environment", cfg.Environment),
		zap.String("version", cfg.Version),
	)

	jwtService := jwt.NewTokenService(cfg.JWTSecret, cfg.JWTExpiration)

	redisStore, err := store.NewRedisStore(
		cfg.RedisAddr,
		cfg.RedisPassword,
		cfg.RedisDB,
		logger,
	)
	if err != nil {
		logger.Fatal("failed to initialize Redis store", zap.Error(err))
	}
	defer redisStore.Close()

	logger.Info("connected to Redis", zap.String("addr", cfg.RedisAddr))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wsHub := handlers.NewWSHub(logger)
	go wsHub.Run(ctx)

	handler := handlers.NewHandler(redisStore, jwtService, logger)
	wsHandler := handlers.NewWSHandler(wsHub, logger)

	kafkaAvailable := len(cfg.KafkaBrokers) > 0 && cfg.KafkaBrokers[0] != ""
	if kafkaAvailable {
		streamer := handlers.NewMetricsStreamer(
			cfg.KafkaBrokers,
			cfg.MetricsTopic,
			cfg.AlertsTopic,
			cfg.ConsumerGroup,
			wsHub,
			logger,
		)
		if err := streamer.Start(ctx); err != nil {
			logger.Warn("failed to start metrics streamer", zap.Error(err))
		} else {
			defer streamer.Stop()
			logger.Info("metrics streamer started")
		}
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Post("/api/auth/login", handler.Login)
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"ui-backend"}`))
	})

	r.Group(func(r chi.Router) {
		r.Use(handler.AuthMiddleware)

		r.Post("/api/auth/refresh", handler.RefreshToken)

		r.Get("/api/services", handler.GetServices)
		r.Get("/api/services/{service}/metrics", handler.GetServiceMetrics)

		r.Get("/api/metrics/latest", handler.GetLatestMetrics)

		r.Get("/api/alerts", handler.GetAlerts)
		r.Get("/api/alerts/{id}", handler.GetAlert)
		r.Post("/api/alerts/{id}/acknowledge", handler.AcknowledgeAlert)

		r.Get("/api/rules", handler.GetRules)
		r.Post("/api/rules", handler.CreateRule)
		r.Put("/api/rules/{id}", handler.UpdateRule)
		r.Delete("/api/rules/{id}", handler.DeleteRule)

		r.Get("/api/dashboard/stats", handler.GetDashboardStats)

		r.Get("/ws", wsHandler.ServeWS)
	})

	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: r,
	}

	go func() {
		logger.Info("starting HTTP server", zap.String("addr", cfg.HTTPAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()

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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("ui-backend service started successfully",
		zap.String("http_addr", cfg.HTTPAddr),
		zap.String("metrics_addr", cfg.MetricsAddr),
	)

	sig := <-sigCh
	logger.Info("received shutdown signal", zap.String("signal", sig.String()))

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("metrics server shutdown error", zap.Error(err))
	}

	cancel()

	logger.Info("ui-backend service stopped")
}
