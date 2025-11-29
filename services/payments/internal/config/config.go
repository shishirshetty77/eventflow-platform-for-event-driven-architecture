// Package config provides configuration management for the payments service.
package config

import (
	"time"

	"github.com/microservices-platform/pkg/shared/utils"
)

// Config holds all configuration for the payments service.
type Config struct {
	ServiceName string
	Version     string
	Environment string

	HTTPPort    int
	MetricsPort int

	KafkaBrokers      []string
	KafkaMetricsTopic string
	KafkaLogsTopic    string

	MetricsInterval time.Duration
	LogsInterval    time.Duration

	LogLevel    string
	Development bool

	TracingEnabled  bool
	TracingEndpoint string
	SampleRate      float64
}

// Load loads configuration from environment variables.
func Load() *Config {
	return &Config{
		ServiceName: "payments",
		Version:     utils.GetEnv("SERVICE_VERSION", "1.0.0"),
		Environment: utils.GetEnv("ENVIRONMENT", "development"),

		HTTPPort:    utils.GetEnvInt("HTTP_PORT", 8083),
		MetricsPort: utils.GetEnvInt("METRICS_PORT", 9093),

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
