// Package models provides shared data models and DTOs for the microservices platform.
// These models are used across all services for consistent data exchange.
package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ServiceName represents the name of a microservice in the platform.
type ServiceName string

const (
	ServiceAuth         ServiceName = "auth"
	ServiceOrders       ServiceName = "orders"
	ServicePayments     ServiceName = "payments"
	ServiceNotification ServiceName = "notification"
	ServiceAnalyzer     ServiceName = "analyzer"
	ServiceAlertEngine  ServiceName = "alert-engine"
	ServiceUIBackend    ServiceName = "ui-backend"
	// Aliases for backward compatibility
	ServiceNameAuth         = ServiceAuth
	ServiceNameOrders       = ServiceOrders
	ServiceNamePayments     = ServicePayments
	ServiceNameNotification = ServiceNotification
)

// MetricType represents the type of metric being reported.
type MetricType string

const (
	MetricTypeCPU         MetricType = "cpu"
	MetricTypeMemory      MetricType = "memory"
	MetricTypeLatency     MetricType = "latency"
	MetricTypeLatencyP95  MetricType = "latency_p95"
	MetricTypeLatencyP99  MetricType = "latency_p99"
	MetricTypeError       MetricType = "error"
	MetricTypeErrorRate   MetricType = "error_rate"
	MetricTypeStatus      MetricType = "status"
	MetricTypeRequestRate MetricType = "request_rate"
)

// LogLevel represents the severity level of a log entry.
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// ServiceStatus represents the health status of a service.
type ServiceStatus string

const (
	StatusHealthy   ServiceStatus = "healthy"
	StatusDegraded  ServiceStatus = "degraded"
	StatusUnhealthy ServiceStatus = "unhealthy"
	StatusUnknown   ServiceStatus = "unknown"
)

// AlertSeverity represents the severity level of an alert.
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertType represents the type of anomaly detected.
type AlertType string

const (
	AlertTypeThresholdViolation AlertType = "threshold_violation"
	AlertTypeErrorBurst         AlertType = "error_burst"
	AlertTypeLatencySpike       AlertType = "latency_spike"
	AlertTypeDeviationAnomaly   AlertType = "deviation_anomaly"
	AlertTypeMovingAvgAnomaly   AlertType = "moving_avg_anomaly"
)

// ServiceMetric represents a metric data point from a service.
// This is published to the "service-metrics" Kafka topic.
type ServiceMetric struct {
	ID          string      `json:"id" validate:"required,uuid"`
	ServiceName ServiceName `json:"service_name" validate:"required"`
	MetricType  MetricType  `json:"metric_type" validate:"required"`
	Value       float64     `json:"value" validate:"required"`
	Unit        string      `json:"unit" validate:"required"`
	Timestamp   time.Time   `json:"timestamp" validate:"required"`
	Labels      Labels      `json:"labels,omitempty"`
	TraceID     string      `json:"trace_id,omitempty"`
	SpanID      string      `json:"span_id,omitempty"`
	// Extended metrics for dashboard display
	CPUUsage     float64 `json:"cpu_usage,omitempty"`
	MemoryUsage  float64 `json:"memory_usage,omitempty"`
	LatencyP50   float64 `json:"latency_p50,omitempty"`
	LatencyP95   float64 `json:"latency_p95,omitempty"`
	LatencyP99   float64 `json:"latency_p99,omitempty"`
	ErrorRate    float64 `json:"error_rate,omitempty"`
	RequestCount float64 `json:"request_count,omitempty"`
}

// NewServiceMetric creates a new ServiceMetric with a generated ID.
func NewServiceMetric(serviceName ServiceName, metricType MetricType, value float64, unit string) *ServiceMetric {
	return &ServiceMetric{
		ID:          uuid.New().String(),
		ServiceName: serviceName,
		MetricType:  metricType,
		Value:       value,
		Unit:        unit,
		Timestamp:   time.Now().UTC(),
		Labels:      make(Labels),
	}
}

// ServiceLog represents a structured log entry from a service.
// This is published to the "service-logs" Kafka topic.
type ServiceLog struct {
	ID          string      `json:"id" validate:"required,uuid"`
	ServiceName ServiceName `json:"service_name" validate:"required"`
	Level       LogLevel    `json:"level" validate:"required"`
	Message     string      `json:"message" validate:"required"`
	Timestamp   time.Time   `json:"timestamp" validate:"required"`
	Caller      string      `json:"caller,omitempty"`
	StackTrace  string      `json:"stack_trace,omitempty"`
	Fields      Fields      `json:"fields,omitempty"`
	TraceID     string      `json:"trace_id,omitempty"`
	SpanID      string      `json:"span_id,omitempty"`
	RequestID   string      `json:"request_id,omitempty"`
}

// NewServiceLog creates a new ServiceLog with a generated ID.
func NewServiceLog(serviceName ServiceName, level LogLevel, message string) *ServiceLog {
	return &ServiceLog{
		ID:          uuid.New().String(),
		ServiceName: serviceName,
		Level:       level,
		Message:     message,
		Timestamp:   time.Now().UTC(),
		Fields:      make(Fields),
	}
}

// Alert represents an alert event generated by the Analyzer.
type Alert struct {
	ID             string        `json:"id" validate:"required,uuid"`
	Type           AlertType     `json:"type" validate:"required"`
	Severity       AlertSeverity `json:"severity" validate:"required"`
	ServiceName    ServiceName   `json:"service_name" validate:"required"`
	MetricType     MetricType    `json:"metric_type,omitempty"`
	Title          string        `json:"title" validate:"required"`
	Description    string        `json:"description" validate:"required"`
	Message        string        `json:"message,omitempty"`
	Value          float64       `json:"value"`
	CurrentValue   float64       `json:"current_value,omitempty"`
	Threshold      float64       `json:"threshold,omitempty"`
	Timestamp      time.Time     `json:"timestamp" validate:"required"`
	ResolvedAt     *time.Time    `json:"resolved_at,omitempty"`
	Acknowledged   bool          `json:"acknowledged"`
	AcknowledgedBy string        `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time    `json:"acknowledged_at,omitempty"`
	Labels         Labels        `json:"labels,omitempty"`
	MetricID       string        `json:"metric_id,omitempty"`
	RuleID         string        `json:"rule_id,omitempty"`
	TraceID        string        `json:"trace_id,omitempty"`
}

// NewAlert creates a new Alert with a generated ID.
func NewAlert(alertType AlertType, severity AlertSeverity, serviceName ServiceName, title, description string) *Alert {
	return &Alert{
		ID:          uuid.New().String(),
		Type:        alertType,
		Severity:    severity,
		ServiceName: serviceName,
		Title:       title,
		Description: description,
		Timestamp:   time.Now().UTC(),
		Labels:      make(Labels),
	}
}

// ThresholdRule represents a configurable threshold rule for anomaly detection.
type ThresholdRule struct {
	ID              string        `json:"id" validate:"required,uuid"`
	Name            string        `json:"name" validate:"required,min=1,max=100"`
	Description     string        `json:"description,omitempty"`
	ServiceName     ServiceName   `json:"service_name" validate:"required"`
	MetricType      MetricType    `json:"metric_type" validate:"required"`
	Operator        string        `json:"operator" validate:"required,oneof=> < >= <= == !="`
	Threshold       float64       `json:"threshold" validate:"required"`
	Severity        AlertSeverity `json:"severity" validate:"required"`
	WindowSize      int           `json:"window_size" validate:"min=1,max=3600"`   // in seconds
	CooldownSec     int           `json:"cooldown_sec" validate:"min=0,max=86400"` // alert cooldown
	CooldownSeconds int           `json:"cooldown_seconds,omitempty"`              // alias
	Enabled         bool          `json:"enabled"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	NotifySlack     bool          `json:"notify_slack"`
	NotifyEmail     bool          `json:"notify_email"`
	NotifyWebhook   bool          `json:"notify_webhook"`
}

// NewThresholdRule creates a new ThresholdRule with defaults.
func NewThresholdRule(name string, serviceName ServiceName, metricType MetricType, operator string, threshold float64) *ThresholdRule {
	now := time.Now().UTC()
	return &ThresholdRule{
		ID:          uuid.New().String(),
		Name:        name,
		ServiceName: serviceName,
		MetricType:  metricType,
		Operator:    operator,
		Threshold:   threshold,
		Severity:    AlertSeverityWarning,
		WindowSize:  60,
		CooldownSec: 300,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// ServiceHealthStatus represents the current health status of a service.
type ServiceHealthStatus struct {
	ServiceName   ServiceName   `json:"service_name"`
	Status        ServiceStatus `json:"status"`
	LastHeartbeat time.Time     `json:"last_heartbeat"`
	CPUUsage      float64       `json:"cpu_usage"`
	MemoryUsage   float64       `json:"memory_usage"`
	ErrorRate     float64       `json:"error_rate"`
	AvgLatency    float64       `json:"avg_latency"`
	Uptime        int64         `json:"uptime_seconds"`
	Version       string        `json:"version"`
}

// Labels is a map of key-value pairs for metric/log labeling.
type Labels map[string]string

// Fields is a map of additional fields for structured logging.
type Fields map[string]interface{}

// User represents a user in the system.
type User struct {
	ID        string    `json:"id" validate:"required,uuid"`
	Email     string    `json:"email" validate:"required,email"`
	Username  string    `json:"username,omitempty"`
	Password  string    `json:"-"` // Never serialized
	Name      string    `json:"name" validate:"required,min=1,max=100"`
	Role      string    `json:"role" validate:"required,oneof=admin operator viewer"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoginRequest represents a login request payload.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// LoginResponse represents a successful login response.
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	User      *User  `json:"user"`
}

// TokenClaims represents JWT token claims.
type TokenClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// PaginationRequest represents pagination parameters.
type PaginationRequest struct {
	Page     int `json:"page" validate:"min=1"`
	PageSize int `json:"page_size" validate:"min=1,max=100"`
}

// PaginatedResponse represents a paginated response wrapper.
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalCount int64       `json:"total_count"`
	TotalPages int         `json:"total_pages"`
}

// APIResponse represents a standard API response wrapper.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *APIMeta    `json:"meta,omitempty"`
}

// APIError represents an API error response.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// APIMeta represents API response metadata.
type APIMeta struct {
	RequestID string `json:"request_id"`
	Timestamp int64  `json:"timestamp"`
}

// WebSocketMessage represents a WebSocket message.
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WebSocketMessageType constants.
const (
	WSTypeMetric      = "metric"
	WSTypeLog         = "log"
	WSTypeAlert       = "alert"
	WSTypeStatus      = "status"
	WSTypeSubscribe   = "subscribe"
	WSTypeUnsubscribe = "unsubscribe"
	WSTypeError       = "error"
	WSTypePing        = "ping"
	WSTypePong        = "pong"
)

// JSON serialization helpers.

// ToJSON serializes the object to JSON bytes.
func (m *ServiceMetric) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON deserializes JSON bytes into ServiceMetric.
func (m *ServiceMetric) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}

// ToJSON serializes the object to JSON bytes.
func (l *ServiceLog) ToJSON() ([]byte, error) {
	return json.Marshal(l)
}

// FromJSON deserializes JSON bytes into ServiceLog.
func (l *ServiceLog) FromJSON(data []byte) error {
	return json.Unmarshal(data, l)
}

// ToJSON serializes the object to JSON bytes.
func (a *Alert) ToJSON() ([]byte, error) {
	return json.Marshal(a)
}

// FromJSON deserializes JSON bytes into Alert.
func (a *Alert) FromJSON(data []byte) error {
	return json.Unmarshal(data, a)
}

// ToJSON serializes the object to JSON bytes.
func (r *ThresholdRule) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// FromJSON deserializes JSON bytes into ThresholdRule.
func (r *ThresholdRule) FromJSON(data []byte) error {
	return json.Unmarshal(data, r)
}
