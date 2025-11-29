// Package config provides configuration management for the orders service.
package config

import (
	"time"

	"github.com/microservices-platform/pkg/shared/utils"
)

// Config holds all configuration for the orders service.
type Config struct {
	ServiceName string
	Version     string
	Environment string

	// Server configuration
	HTTPPort    int
	MetricsPort int

	// Kafka configuration
	KafkaBrokers      []string
	KafkaMetricsTopic string
	KafkaLogsTopic    string

	// Metrics generation
	MetricsInterval time.Duration
	LogsInterval    time.Duration

	// Logging
	LogLevel    string
	Development bool

	// Tracing
	TracingEnabled  bool
	TracingEndpoint string
	SampleRate      float64
}

// Load loads configuration from environment variables.
func Load() *Config {
	return &Config{
		ServiceName: "orders",
		Version:     utils.GetEnv("SERVICE_VERSION", "1.0.0"),
		Environment: utils.GetEnv("ENVIRONMENT", "development"),

		HTTPPort:    utils.GetEnvInt("HTTP_PORT", 8082),
		MetricsPort: utils.GetEnvInt("METRICS_PORT", 9092),

		KafkaBrokers:      utils.GetEnvStringSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
		KafkaMetricsTopic: utils.GetEnv("KAFKA_METRICS_TOPIC", "service-metrics"),
		KafkaLogsTopic:    utils.GetEnv("KAFKA_LOGS_TOPIC", "service-logs"),

		MetricsInterval: utils.GetEnvDuration("METRICS_INTERVAL", 5*time.Second),
		LogsInterval:    utils.GetEnvDuration("LOGS_INTERVAL", 3*time.Second),

		LogLevel:    utils.GetEnv("LOG_LEVEL", "info"),
		Development: utils.GetEnvBool("DEVELOPMENT", true),

		TracingEnabled:  utils.GetEnvBool("TRACING_ENABLED", false),
		TracingEndpoint: utils.GetEnv("TRACING_ENDPOINT", "localhost:4317"),
		SampleRate:      utils.GetEnvFloat64("TRACING_SAMPLE_RATE", 1.0),
	}
}
