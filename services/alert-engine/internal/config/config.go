// Package config provides configuration for the alert-engine service.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the configuration for the alert-engine service.
type Config struct {
	// Service settings
	ServiceName string
	Environment string
	Version     string
	LogLevel    string

	// Kafka settings
	KafkaBrokers  []string
	AlertsTopic   string
	DLQTopic      string
	ConsumerGroup string

	// Server ports
	MetricsAddr string
	HealthAddr  string

	// Slack configuration
	SlackWebhookURL string
	SlackChannel    string
	SlackEnabled    bool

	// SendGrid email configuration
	SendGridAPIKey    string
	SendGridFromEmail string
	SendGridFromName  string
	EmailRecipients   []string
	EmailEnabled      bool

	// Webhook configuration
	WebhookURLs    []string
	WebhookEnabled bool
	WebhookHeaders map[string]string

	// Alert processing settings
	MaxRetries        int
	RetryDelaySeconds int
	BatchSize         int
	BatchTimeout      time.Duration

	// Grouping and suppression
	GroupingWindowSeconds    int
	SuppressionWindowSeconds int
	MaxAlertsPerGroup        int
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() *Config {
	return &Config{
		ServiceName: getEnv("SERVICE_NAME", "alert-engine"),
		Environment: getEnv("ENVIRONMENT", "development"),
		Version:     getEnv("VERSION", "1.0.0"),
		LogLevel:    getEnv("LOG_LEVEL", "debug"),

		KafkaBrokers:  strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		AlertsTopic:   getEnv("ALERTS_TOPIC", "alerts"),
		DLQTopic:      getEnv("DLQ_TOPIC", "alerts-dlq"),
		ConsumerGroup: getEnv("CONSUMER_GROUP", "alert-engine-group"),

		MetricsAddr: ":" + getEnv("METRICS_PORT", "9094"),
		HealthAddr:  ":" + getEnv("HEALTH_PORT", "8084"),

		// Slack
		SlackWebhookURL: getEnv("SLACK_WEBHOOK_URL", ""),
		SlackChannel:    getEnv("SLACK_CHANNEL", "#alerts"),
		SlackEnabled:    getEnvBool("SLACK_ENABLED", false),

		// SendGrid
		SendGridAPIKey:    getEnv("SENDGRID_API_KEY", ""),
		SendGridFromEmail: getEnv("SENDGRID_FROM_EMAIL", "alerts@example.com"),
		SendGridFromName:  getEnv("SENDGRID_FROM_NAME", "Alert Engine"),
		EmailRecipients:   strings.Split(getEnv("EMAIL_RECIPIENTS", ""), ","),
		EmailEnabled:      getEnvBool("EMAIL_ENABLED", false),

		// Webhook
		WebhookURLs:    strings.Split(getEnv("WEBHOOK_URLS", ""), ","),
		WebhookEnabled: getEnvBool("WEBHOOK_ENABLED", false),
		WebhookHeaders: parseHeaders(getEnv("WEBHOOK_HEADERS", "")),

		// Processing settings
		MaxRetries:        getEnvInt("MAX_RETRIES", 3),
		RetryDelaySeconds: getEnvInt("RETRY_DELAY_SECONDS", 5),
		BatchSize:         getEnvInt("BATCH_SIZE", 10),
		BatchTimeout:      time.Duration(getEnvInt("BATCH_TIMEOUT_SECONDS", 5)) * time.Second,

		// Grouping and suppression
		GroupingWindowSeconds:    getEnvInt("GROUPING_WINDOW_SECONDS", 60),
		SuppressionWindowSeconds: getEnvInt("SUPPRESSION_WINDOW_SECONDS", 300),
		MaxAlertsPerGroup:        getEnvInt("MAX_ALERTS_PER_GROUP", 10),
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

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func parseHeaders(headers string) map[string]string {
	result := make(map[string]string)
	if headers == "" {
		return result
	}
	pairs := strings.Split(headers, ";")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return result
}
