// Package ports defines interfaces for the analyzer service.
package ports

import (
	"context"
	"time"

	"github.com/microservices-platform/pkg/shared/models"
)

// MetricsStore defines the interface for storing and retrieving metrics.
type MetricsStore interface {
	// AddMetric adds a metric to the sliding window.
	AddMetric(ctx context.Context, metric *models.ServiceMetric) error
	// GetMetricsInWindow retrieves metrics within the sliding window for a service and metric type.
	GetMetricsInWindow(ctx context.Context, serviceName models.ServiceName, metricType models.MetricType, windowSize time.Duration) ([]*models.ServiceMetric, error)
	// GetRollingAverage calculates the rolling average for a service and metric type.
	GetRollingAverage(ctx context.Context, serviceName models.ServiceName, metricType models.MetricType, windowSize time.Duration) (float64, error)
	// GetLatestMetric retrieves the most recent metric for a service and metric type.
	GetLatestMetric(ctx context.Context, serviceName models.ServiceName, metricType models.MetricType) (*models.ServiceMetric, error)
	// CleanupOldMetrics removes metrics older than the retention period.
	CleanupOldMetrics(ctx context.Context, retentionPeriod time.Duration) error
}

// RuleStore defines the interface for managing threshold rules.
type RuleStore interface {
	// GetRule retrieves a rule by ID.
	GetRule(ctx context.Context, id string) (*models.ThresholdRule, error)
	// GetAllRules retrieves all rules.
	GetAllRules(ctx context.Context) ([]*models.ThresholdRule, error)
	// GetRulesForService retrieves all rules for a specific service.
	GetRulesForService(ctx context.Context, serviceName models.ServiceName) ([]*models.ThresholdRule, error)
	// CreateRule creates a new rule.
	CreateRule(ctx context.Context, rule *models.ThresholdRule) error
	// UpdateRule updates an existing rule.
	UpdateRule(ctx context.Context, rule *models.ThresholdRule) error
	// DeleteRule deletes a rule.
	DeleteRule(ctx context.Context, id string) error
}

// AlertStore defines the interface for managing alerts and cooldowns.
type AlertStore interface {
	// RecordAlert records an alert for deduplication.
	RecordAlert(ctx context.Context, alert *models.Alert) error
	// IsAlertInCooldown checks if an alert is in cooldown period.
	IsAlertInCooldown(ctx context.Context, serviceName models.ServiceName, metricType models.MetricType, ruleID string) (bool, error)
	// SetAlertCooldown sets a cooldown for an alert.
	SetAlertCooldown(ctx context.Context, serviceName models.ServiceName, metricType models.MetricType, ruleID string, duration time.Duration) error
	// GetRecentAlerts retrieves recent alerts.
	GetRecentAlerts(ctx context.Context, limit int) ([]*models.Alert, error)
}

// AlertPublisher defines the interface for publishing alerts.
type AlertPublisher interface {
	// PublishAlert publishes an alert to the alert topic.
	PublishAlert(ctx context.Context, alert *models.Alert) error
	// Close closes the publisher.
	Close() error
}

// AnomalyDetector defines the interface for anomaly detection.
type AnomalyDetector interface {
	// DetectAnomalies analyzes metrics and returns detected anomalies.
	DetectAnomalies(ctx context.Context, serviceName models.ServiceName) ([]*models.Alert, error)
}

// MetricsConsumer defines the interface for consuming metrics from Kafka.
type MetricsConsumer interface {
	// Start starts consuming metrics.
	Start(ctx context.Context) error
	// Stop stops consuming metrics.
	Stop() error
}
