// Package config provides configuration for the ui-backend service.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the configuration for the ui-backend service.
type Config struct {
	// Service settings
	ServiceName string
	Environment string
	Version     string
	LogLevel    string

	// Server settings
	HTTPAddr    string
	MetricsAddr string
	HealthAddr  string

	// Redis settings
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Kafka settings
	KafkaBrokers  []string
	MetricsTopic  string
	AlertsTopic   string
	ConsumerGroup string

	// JWT settings
	JWTSecret     string
	JWTExpiration time.Duration

	// Auth service URL
	AuthServiceURL string

	// CORS settings
	AllowedOrigins []string

	// WebSocket settings
	WSPingInterval time.Duration
	WSPongTimeout  time.Duration
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() *Config {
	return &Config{
		ServiceName: getEnv("SERVICE_NAME", "ui-backend"),
		Environment: getEnv("ENVIRONMENT", "development"),
		Version:     getEnv("VERSION", "1.0.0"),
		LogLevel:    getEnv("LOG_LEVEL", "debug"),

		HTTPAddr:    ":" + getEnv("HTTP_PORT", "8080"),
		MetricsAddr: ":" + getEnv("METRICS_PORT", "9095"),
		HealthAddr:  ":" + getEnv("HEALTH_PORT", "8085"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		KafkaBrokers:  strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		MetricsTopic:  getEnv("METRICS_TOPIC", "service-metrics"),
		AlertsTopic:   getEnv("ALERTS_TOPIC", "alerts"),
		ConsumerGroup: getEnv("CONSUMER_GROUP", "ui-backend-group"),

		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		JWTExpiration: time.Duration(getEnvInt("JWT_EXPIRATION_HOURS", 24)) * time.Hour,

		AuthServiceURL: getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),

		AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:3000"), ","),

		WSPingInterval: time.Duration(getEnvInt("WS_PING_INTERVAL_SECONDS", 30)) * time.Second,
		WSPongTimeout:  time.Duration(getEnvInt("WS_PONG_TIMEOUT_SECONDS", 60)) * time.Second,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
