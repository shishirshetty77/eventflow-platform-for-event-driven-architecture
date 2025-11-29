// Package config provides configuration management for the analyzer service.
package config

import (
	"time"

	"github.com/microservices-platform/pkg/shared/utils"
)

// Config holds all configuration for the analyzer service.
type Config struct {
	ServiceName string
	Version     string
	Environment string

	HTTPPort    int
	MetricsPort int

	// Kafka configuration
	KafkaBrokers       []string
	KafkaMetricsTopic  string
	KafkaLogsTopic     string
	KafkaAlertsTopic   string
	KafkaConsumerGroup string

	// Redis configuration
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Analysis configuration
	SlidingWindowSize time.Duration
	AnalysisInterval  time.Duration
	AlertCooldown     time.Duration

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
		ServiceName: "analyzer",
		Version:     utils.GetEnv("SERVICE_VERSION", "1.0.0"),
		Environment: utils.GetEnv("ENVIRONMENT", "development"),

		HTTPPort:    utils.GetEnvInt("HTTP_PORT", 8085),
		MetricsPort: utils.GetEnvInt("METRICS_PORT", 9095),

		KafkaBrokers:       utils.GetEnvStringSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
		KafkaMetricsTopic:  utils.GetEnv("KAFKA_METRICS_TOPIC", "service-metrics"),
		KafkaLogsTopic:     utils.GetEnv("KAFKA_LOGS_TOPIC", "service-logs"),
		KafkaAlertsTopic:   utils.GetEnv("KAFKA_ALERTS_TOPIC", "alerts"),
		KafkaConsumerGroup: utils.GetEnv("KAFKA_CONSUMER_GROUP", "analyzer-group"),

		RedisAddr:     utils.GetEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: utils.GetEnv("REDIS_PASSWORD", ""),
		RedisDB:       utils.GetEnvInt("REDIS_DB", 0),

		SlidingWindowSize: utils.GetEnvDuration("SLIDING_WINDOW_SIZE", 5*time.Minute),
		AnalysisInterval:  utils.GetEnvDuration("ANALYSIS_INTERVAL", 10*time.Second),
		AlertCooldown:     utils.GetEnvDuration("ALERT_COOLDOWN", 5*time.Minute),

		LogLevel:    utils.GetEnv("LOG_LEVEL", "info"),
		Development: utils.GetEnvBool("DEVELOPMENT", true),

		TracingEnabled:  utils.GetEnvBool("TRACING_ENABLED", false),
		TracingEndpoint: utils.GetEnv("TRACING_ENDPOINT", "localhost:4317"),
		SampleRate:      utils.GetEnvFloat64("TRACING_SAMPLE_RATE", 1.0),
	}
}
