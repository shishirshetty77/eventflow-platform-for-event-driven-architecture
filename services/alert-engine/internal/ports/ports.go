// Package ports defines the interfaces for the alert-engine service.
package ports

import (
	"context"

	"github.com/microservices-platform/pkg/shared/models"
)

// AlertConsumer consumes alerts from a message queue.
type AlertConsumer interface {
	Start(ctx context.Context) error
	Stop() error
}

// AlertDispatcher dispatches alerts through various channels.
type AlertDispatcher interface {
	Dispatch(ctx context.Context, alert *models.Alert) error
	Name() string
	Enabled() bool
}

// AlertGrouper groups related alerts together.
type AlertGrouper interface {
	// AddAlert adds an alert to be grouped
	AddAlert(ctx context.Context, alert *models.Alert) error
	// GetGroups returns grouped alerts ready for dispatch
	GetGroups(ctx context.Context) ([]*AlertGroup, error)
	// MarkDispatched marks a group as dispatched
	MarkDispatched(ctx context.Context, groupID string) error
}

// AlertGroup represents a group of related alerts.
type AlertGroup struct {
	ID          string
	Alerts      []*models.Alert
	GroupKey    string
	FirstSeen   int64
	LastSeen    int64
	Count       int
	Severity    models.AlertSeverity
	ServiceName models.ServiceName
}

// DLQHandler handles dead letter queue operations.
type DLQHandler interface {
	SendToDLQ(ctx context.Context, alert *models.Alert, reason string, err error) error
	ProcessDLQ(ctx context.Context) error
}

// RetryPolicy defines retry behavior for failed dispatches.
type RetryPolicy interface {
	ShouldRetry(attempt int, err error) bool
	GetDelay(attempt int) int64
	MaxRetries() int
}

// SuppressionManager manages alert suppression.
type SuppressionManager interface {
	ShouldSuppress(ctx context.Context, alert *models.Alert) (bool, error)
	AddSuppression(ctx context.Context, alert *models.Alert, duration int64) error
	RemoveSuppression(ctx context.Context, key string) error
}

// DispatchResult represents the result of dispatching an alert.
type DispatchResult struct {
	Success        bool
	DispatcherName string
	Error          error
	Timestamp      int64
	RetryCount     int
}

// AlertProcessor processes alerts through grouping, suppression, and dispatch.
type AlertProcessor interface {
	Process(ctx context.Context, alert *models.Alert) error
	Start(ctx context.Context) error
	Stop() error
}
