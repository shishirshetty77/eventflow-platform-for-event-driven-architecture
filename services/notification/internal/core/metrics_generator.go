// Package core contains the business logic for the notification service.
package core

import (
	"context"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/microservices-platform/pkg/shared/kafka"
	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/metrics"
	"github.com/microservices-platform/pkg/shared/models"
)

// MetricsPublisher defines the interface for publishing metrics.
type MetricsPublisher interface {
	PublishMetric(ctx context.Context, metric *models.ServiceMetric) error
	PublishLog(ctx context.Context, log *models.ServiceLog) error
	Close() error
}

// MetricsGenerator generates realistic metrics and logs for the notification service.
type MetricsGenerator struct {
	serviceName     models.ServiceName
	publisher       MetricsPublisher
	logger          *logging.Logger
	metrics         *metrics.Metrics
	metricsInterval time.Duration
	logsInterval    time.Duration
	startTime       time.Time

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup

	// Simulated notification state
	emailsSent          int64
	smsSent             int64
	pushSent            int64
	notificationsFailed int64
}

// NewMetricsGenerator creates a new MetricsGenerator for the notification service.
func NewMetricsGenerator(
	publisher MetricsPublisher,
	logger *logging.Logger,
	m *metrics.Metrics,
	metricsInterval, logsInterval time.Duration,
) *MetricsGenerator {
	return &MetricsGenerator{
		serviceName:     models.ServiceNotification,
		publisher:       publisher,
		logger:          logger,
		metrics:         m,
		metricsInterval: metricsInterval,
		logsInterval:    logsInterval,
		startTime:       time.Now(),
	}
}

// Start starts the metrics generator.
func (g *MetricsGenerator) Start(ctx context.Context) error {
	g.mu.Lock()
	if g.running {
		g.mu.Unlock()
		return nil
	}
	g.running = true
	g.stopCh = make(chan struct{})
	g.mu.Unlock()

	g.logger.Info("starting notification metrics generator",
		zap.Duration("metrics_interval", g.metricsInterval),
		zap.Duration("logs_interval", g.logsInterval),
	)

	g.wg.Add(1)
	go g.generateMetrics(ctx)

	g.wg.Add(1)
	go g.generateLogs(ctx)

	return nil
}

// Stop stops the metrics generator.
func (g *MetricsGenerator) Stop() error {
	g.mu.Lock()
	if !g.running {
		g.mu.Unlock()
		return nil
	}
	g.running = false
	close(g.stopCh)
	g.mu.Unlock()

	g.wg.Wait()
	g.logger.Info("notification metrics generator stopped")
	return nil
}

func (g *MetricsGenerator) generateMetrics(ctx context.Context) {
	defer g.wg.Done()

	ticker := time.NewTicker(g.metricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-g.stopCh:
			return
		case <-ticker.C:
			g.publishMetrics(ctx)
		}
	}
}

func (g *MetricsGenerator) generateLogs(ctx context.Context) {
	defer g.wg.Done()

	ticker := time.NewTicker(g.logsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-g.stopCh:
			return
		case <-ticker.C:
			g.publishLogs(ctx)
		}
	}
}

func (g *MetricsGenerator) publishMetrics(ctx context.Context) {
	// Simulate notification sending
	g.mu.Lock()
	notifType := rand.Intn(100)
	if notifType < 50 {
		g.emailsSent += int64(rand.Intn(20) + 1)
	} else if notifType < 80 {
		g.pushSent += int64(rand.Intn(30) + 1)
	} else {
		g.smsSent += int64(rand.Intn(5) + 1)
	}
	if rand.Float64() < 0.02 {
		g.notificationsFailed++
	}
	g.mu.Unlock()

	// CPU metric - notification service is typically lighter
	cpuMetric := g.generateCPUMetric()
	if err := g.publisher.PublishMetric(ctx, cpuMetric); err != nil {
		g.logger.Warn("failed to publish CPU metric", zap.Error(err))
	}

	// Memory metric
	memMetric := g.generateMemoryMetric()
	if err := g.publisher.PublishMetric(ctx, memMetric); err != nil {
		g.logger.Warn("failed to publish memory metric", zap.Error(err))
	}

	// Latency metric - external provider calls
	latencyMetric := g.generateLatencyMetric()
	if err := g.publisher.PublishMetric(ctx, latencyMetric); err != nil {
		g.logger.Warn("failed to publish latency metric", zap.Error(err))
	}

	// Error metric
	errorMetric := g.generateErrorMetric()
	if err := g.publisher.PublishMetric(ctx, errorMetric); err != nil {
		g.logger.Warn("failed to publish error metric", zap.Error(err))
	}

	// Status metric
	statusMetric := g.generateStatusMetric()
	if err := g.publisher.PublishMetric(ctx, statusMetric); err != nil {
		g.logger.Warn("failed to publish status metric", zap.Error(err))
	}
}

func (g *MetricsGenerator) generateCPUMetric() *models.ServiceMetric {
	// Notification service: lower CPU, mostly I/O bound
	baseCPU := 15.0 + rand.Float64()*25.0

	// Burst during batch notifications
	if rand.Float64() < 0.1 {
		baseCPU += 25.0 + rand.Float64()*20.0
	}

	if baseCPU > 90 {
		baseCPU = 90
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeCPU, baseCPU, "percent")
	metric.Labels["host"] = "notification-service-1"
	metric.Labels["environment"] = "development"
	return metric
}

func (g *MetricsGenerator) generateMemoryMetric() *models.ServiceMetric {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	memUsage := float64(memStats.Alloc) / 1024 / 1024
	memUsage += rand.Float64() * 60

	memPercent := (memUsage / 512) * 100
	if memPercent > 85 {
		memPercent = 80 + rand.Float64()*5
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeMemory, memPercent, "percent")
	metric.Labels["host"] = "notification-service-1"
	return metric
}

func (g *MetricsGenerator) generateLatencyMetric() *models.ServiceMetric {
	// External email/SMS providers have variable latency
	baseLatency := 100.0 + rand.NormFloat64()*40.0

	// SMTP or SMS provider latency spikes
	if rand.Float64() < 0.15 {
		baseLatency += 200 + rand.Float64()*300
	}

	if baseLatency < 30 {
		baseLatency = 30
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeLatency, baseLatency, "ms")
	metric.Labels["endpoint"] = g.randomEndpoint()
	metric.Labels["channel"] = g.randomChannel()
	return metric
}

func (g *MetricsGenerator) generateErrorMetric() *models.ServiceMetric {
	// Notifications can fail due to invalid addresses, provider issues
	errorRate := 1.5 + rand.NormFloat64()*1.0

	// Provider outages cause error spikes
	if rand.Float64() < 0.03 {
		errorRate += 15 + rand.Float64()*20
	}

	if errorRate < 0 {
		errorRate = 0
	}
	if errorRate > 100 {
		errorRate = 100
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeError, errorRate, "percent")
	metric.Labels["error_type"] = g.randomErrorType()
	return metric
}

func (g *MetricsGenerator) generateStatusMetric() *models.ServiceMetric {
	status := 1.0
	if rand.Float64() < 0.01 {
		status = 0.0
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeStatus, status, "boolean")
	return metric
}

func (g *MetricsGenerator) publishLogs(ctx context.Context) {
	logType := rand.Intn(100)

	var log *models.ServiceLog

	switch {
	case logType < 60:
		log = g.generateInfoLog()
	case logType < 80:
		log = g.generateDebugLog()
	case logType < 93:
		log = g.generateWarnLog()
	default:
		log = g.generateErrorLog()
	}

	if err := g.publisher.PublishLog(ctx, log); err != nil {
		g.logger.Warn("failed to publish log", zap.Error(err))
	}
}

func (g *MetricsGenerator) generateInfoLog() *models.ServiceLog {
	messages := []string{
		"Email notification sent successfully",
		"Push notification delivered",
		"SMS sent to user",
		"Notification queued for delivery",
		"Template rendered successfully",
		"Batch notifications processed",
		"Notification preferences updated",
		"Email opened by recipient",
		"Push notification acknowledged",
		"Webhook notification delivered",
		"Notification scheduled for later",
		"Unsubscribe request processed",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelInfo, messages[rand.Intn(len(messages))])
	log.Fields["notification_id"] = "notif_" + randomHex(12)
	log.Fields["user_id"] = "user_" + randomHex(8)
	log.Fields["channel"] = g.randomChannel()
	log.Fields["template"] = g.randomTemplate()
	log.Caller = "internal/core/notification_service.go:145"
	return log
}

func (g *MetricsGenerator) generateDebugLog() *models.ServiceLog {
	messages := []string{
		"Rendering email template",
		"Connecting to SMTP server",
		"Loading notification preferences",
		"Validating email address format",
		"Fetching push notification token",
		"Building notification payload",
		"Checking rate limits for user",
		"Loading template variables",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelDebug, messages[rand.Intn(len(messages))])
	log.Fields["trace_id"] = randomHex(32)
	log.Caller = "internal/core/notification_service.go:89"
	return log
}

func (g *MetricsGenerator) generateWarnLog() *models.ServiceLog {
	messages := []string{
		"User has unsubscribed from email notifications",
		"Push notification token expired",
		"SMS rate limit approaching",
		"Email bounce detected",
		"Notification delivery delayed",
		"Invalid phone number format",
		"Email marked as spam by provider",
		"High notification queue depth",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelWarn, messages[rand.Intn(len(messages))])
	log.Fields["notification_id"] = "notif_" + randomHex(12)
	log.Fields["warning_code"] = g.randomWarningCode()
	log.Caller = "internal/core/notification_service.go:198"
	return log
}

func (g *MetricsGenerator) generateErrorLog() *models.ServiceLog {
	messages := []string{
		"Failed to send email: SMTP connection error",
		"Push notification failed: invalid token",
		"SMS delivery failed: carrier rejection",
		"Template rendering error",
		"Notification provider unavailable",
		"Failed to queue notification",
		"Email rejected by spam filter",
		"Invalid notification payload",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelError, messages[rand.Intn(len(messages))])
	log.Fields["notification_id"] = "notif_" + randomHex(12)
	log.Fields["error_code"] = g.randomErrorCode()
	log.Fields["stack_trace"] = "goroutine 1 [running]:\nmain.sendNotification()\n\t/app/internal/core/notification_service.go:256 +0x3a8"
	log.Caller = "internal/core/notification_service.go:256"
	return log
}

func (g *MetricsGenerator) randomEndpoint() string {
	endpoints := []string{
		"/api/v1/notifications",
		"/api/v1/notifications/email",
		"/api/v1/notifications/sms",
		"/api/v1/notifications/push",
		"/api/v1/notifications/batch",
		"/api/v1/notifications/preferences",
		"/health",
	}
	return endpoints[rand.Intn(len(endpoints))]
}

func (g *MetricsGenerator) randomChannel() string {
	channels := []string{"email", "sms", "push", "webhook", "slack"}
	return channels[rand.Intn(len(channels))]
}

func (g *MetricsGenerator) randomTemplate() string {
	templates := []string{
		"order_confirmation",
		"payment_receipt",
		"shipping_update",
		"password_reset",
		"welcome_email",
		"promotional",
		"alert_notification",
	}
	return templates[rand.Intn(len(templates))]
}

func (g *MetricsGenerator) randomErrorType() string {
	types := []string{"delivery", "validation", "provider", "template", "rate_limit", "timeout"}
	return types[rand.Intn(len(types))]
}

func (g *MetricsGenerator) randomWarningCode() string {
	codes := []string{"NOTIF_WARN_001", "NOTIF_WARN_002", "EMAIL_WARN_001", "SMS_WARN_001", "PUSH_WARN_001"}
	return codes[rand.Intn(len(codes))]
}

func (g *MetricsGenerator) randomErrorCode() string {
	codes := []string{"NOTIF_ERR_001", "NOTIF_ERR_002", "EMAIL_ERR_001", "SMS_ERR_001", "PUSH_ERR_001", "TMPL_ERR_001"}
	return codes[rand.Intn(len(codes))]
}

func randomHex(length int) string {
	const hexChars = "0123456789abcdef"
	result := make([]byte, length)
	for i := range result {
		result[i] = hexChars[rand.Intn(len(hexChars))]
	}
	return string(result)
}

// KafkaPublisher publishes metrics and logs to Kafka.
type KafkaPublisher struct {
	metricsProducer *kafka.Producer
	logsProducer    *kafka.Producer
	logger          *logging.Logger
	metrics         *metrics.Metrics
}

// NewKafkaPublisher creates a new KafkaPublisher.
func NewKafkaPublisher(
	brokers []string,
	metricsTopic, logsTopic string,
	logger *logging.Logger,
	m *metrics.Metrics,
) (*KafkaPublisher, error) {
	metricsConfig := kafka.DefaultProducerConfig(brokers, metricsTopic)
	metricsProducer, err := kafka.NewProducer(metricsConfig, logger)
	if err != nil {
		return nil, err
	}

	logsConfig := kafka.DefaultProducerConfig(brokers, logsTopic)
	logsProducer, err := kafka.NewProducer(logsConfig, logger)
	if err != nil {
		metricsProducer.Close()
		return nil, err
	}

	return &KafkaPublisher{
		metricsProducer: metricsProducer,
		logsProducer:    logsProducer,
		logger:          logger,
		metrics:         m,
	}, nil
}

// PublishMetric publishes a metric to Kafka.
func (p *KafkaPublisher) PublishMetric(ctx context.Context, metric *models.ServiceMetric) error {
	timer := metrics.NewTimer()

	data, err := metric.ToJSON()
	if err != nil {
		return err
	}

	err = p.metricsProducer.Publish(ctx, []byte(metric.ID), data)
	if p.metrics != nil {
		p.metrics.RecordKafkaPublish(kafka.TopicServiceMetrics, timer.Elapsed(), err)
	}

	return err
}

// PublishLog publishes a log to Kafka.
func (p *KafkaPublisher) PublishLog(ctx context.Context, log *models.ServiceLog) error {
	timer := metrics.NewTimer()

	data, err := log.ToJSON()
	if err != nil {
		return err
	}

	err = p.logsProducer.Publish(ctx, []byte(log.ID), data)
	if p.metrics != nil {
		p.metrics.RecordKafkaPublish(kafka.TopicServiceLogs, timer.Elapsed(), err)
	}

	return err
}

// Close closes the Kafka producers.
func (p *KafkaPublisher) Close() error {
	var lastErr error
	if err := p.metricsProducer.Close(); err != nil {
		lastErr = err
	}
	if err := p.logsProducer.Close(); err != nil {
		lastErr = err
	}
	return lastErr
}

// MockPublisher is a mock implementation for testing without Kafka.
type MockPublisher struct {
	logger *logging.Logger
}

// NewMockPublisher creates a new MockPublisher.
func NewMockPublisher(logger *logging.Logger) *MockPublisher {
	return &MockPublisher{logger: logger}
}

// PublishMetric logs the metric.
func (p *MockPublisher) PublishMetric(ctx context.Context, metric *models.ServiceMetric) error {
	p.logger.Info("mock: metric published",
		zap.String("type", string(metric.MetricType)),
		zap.Float64("value", metric.Value),
	)
	return nil
}

// PublishLog logs the entry.
func (p *MockPublisher) PublishLog(ctx context.Context, log *models.ServiceLog) error {
	p.logger.Info("mock: log published",
		zap.String("level", string(log.Level)),
		zap.String("message", log.Message),
	)
	return nil
}

// Close is a no-op.
func (p *MockPublisher) Close() error {
	return nil
}
