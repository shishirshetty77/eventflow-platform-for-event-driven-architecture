// Package metrics provides Prometheus metrics utilities.
// It enables consistent metrics collection across all microservices.
package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for a service.
type Metrics struct {
	ServiceName string
	Registry    *prometheus.Registry

	// HTTP metrics
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	// Kafka metrics
	KafkaMessagesPublished *prometheus.CounterVec
	KafkaMessagesConsumed  *prometheus.CounterVec
	KafkaPublishDuration   *prometheus.HistogramVec
	KafkaPublishErrors     *prometheus.CounterVec

	// Business metrics
	OperationsTotal   *prometheus.CounterVec
	OperationDuration *prometheus.HistogramVec
	OperationErrors   *prometheus.CounterVec

	// System metrics
	CPUUsage    prometheus.Gauge
	MemoryUsage prometheus.Gauge
	Goroutines  prometheus.Gauge
	Uptime      prometheus.Gauge
}

// NewMetrics creates a new Metrics instance for the given service.
func NewMetrics(serviceName string) *Metrics {
	registry := prometheus.NewRegistry()
	factory := promauto.With(registry)

	m := &Metrics{
		ServiceName: serviceName,
		Registry:    registry,

		// HTTP metrics
		HTTPRequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_request_duration_seconds",
				Help: "HTTP request duration in seconds",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),
		HTTPRequestsInFlight: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently in flight",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
		),

		// Kafka metrics
		KafkaMessagesPublished: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafka_messages_published_total",
				Help: "Total number of Kafka messages published",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"topic"},
		),
		KafkaMessagesConsumed: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafka_messages_consumed_total",
				Help: "Total number of Kafka messages consumed",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"topic"},
		),
		KafkaPublishDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "kafka_publish_duration_seconds",
				Help: "Kafka message publish duration in seconds",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"topic"},
		),
		KafkaPublishErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafka_publish_errors_total",
				Help: "Total number of Kafka publish errors",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"topic"},
		),

		// Business metrics
		OperationsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "operations_total",
				Help: "Total number of business operations",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"operation", "status"},
		),
		OperationDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "operation_duration_seconds",
				Help: "Business operation duration in seconds",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
				Buckets: []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
			},
			[]string{"operation"},
		),
		OperationErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "operation_errors_total",
				Help: "Total number of operation errors",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"operation", "error_type"},
		),

		// System metrics
		CPUUsage: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_cpu_usage_percent",
				Help: "Current CPU usage percentage",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
		),
		MemoryUsage: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_memory_usage_percent",
				Help: "Current memory usage percentage",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
		),
		Goroutines: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_goroutines",
				Help: "Number of goroutines",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
		),
		Uptime: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_uptime_seconds",
				Help: "Service uptime in seconds",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
		),
	}

	// Register standard Go metrics
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	return m
}

// Handler returns an HTTP handler for Prometheus metrics endpoint.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// RecordHTTPRequest records HTTP request metrics.
func (m *Metrics) RecordHTTPRequest(method, path, status string, duration time.Duration) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// RecordKafkaPublish records Kafka publish metrics.
func (m *Metrics) RecordKafkaPublish(topic string, duration time.Duration, err error) {
	m.KafkaMessagesPublished.WithLabelValues(topic).Inc()
	m.KafkaPublishDuration.WithLabelValues(topic).Observe(duration.Seconds())
	if err != nil {
		m.KafkaPublishErrors.WithLabelValues(topic).Inc()
	}
}

// RecordKafkaConsume records Kafka consume metrics.
func (m *Metrics) RecordKafkaConsume(topic string) {
	m.KafkaMessagesConsumed.WithLabelValues(topic).Inc()
}

// RecordOperation records business operation metrics.
func (m *Metrics) RecordOperation(operation, status string, duration time.Duration) {
	m.OperationsTotal.WithLabelValues(operation, status).Inc()
	m.OperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordOperationError records operation error metrics.
func (m *Metrics) RecordOperationError(operation, errorType string) {
	m.OperationErrors.WithLabelValues(operation, errorType).Inc()
}

// UpdateSystemMetrics updates system metrics.
func (m *Metrics) UpdateSystemMetrics(cpu, memory float64, goroutines int, uptime time.Duration) {
	m.CPUUsage.Set(cpu)
	m.MemoryUsage.Set(memory)
	m.Goroutines.Set(float64(goroutines))
	m.Uptime.Set(uptime.Seconds())
}

// Timer is a helper for timing operations.
type Timer struct {
	start time.Time
}

// NewTimer creates a new Timer.
func NewTimer() *Timer {
	return &Timer{start: time.Now()}
}

// Elapsed returns the elapsed time since the timer was created.
func (t *Timer) Elapsed() time.Duration {
	return time.Since(t.start)
}

// ElapsedSeconds returns the elapsed time in seconds.
func (t *Timer) ElapsedSeconds() float64 {
	return time.Since(t.start).Seconds()
}
