// Package core contains the business logic for the payments service.
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

// MetricsGenerator generates realistic metrics and logs for the payments service.
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

	// Simulated payment state
	paymentsProcessed int64
	paymentsSucceeded int64
	paymentsFailed    int64
	totalAmount       float64
}

// NewMetricsGenerator creates a new MetricsGenerator for the payments service.
func NewMetricsGenerator(
	publisher MetricsPublisher,
	logger *logging.Logger,
	m *metrics.Metrics,
	metricsInterval, logsInterval time.Duration,
) *MetricsGenerator {
	return &MetricsGenerator{
		serviceName:     models.ServicePayments,
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

	g.logger.Info("starting payments metrics generator",
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
	g.logger.Info("payments metrics generator stopped")
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
	// Simulate payment processing
	g.mu.Lock()
	g.paymentsProcessed += int64(rand.Intn(5) + 1)
	amount := rand.Float64()*500 + 10
	if rand.Float64() > 0.03 { // 97% success rate
		g.paymentsSucceeded++
		g.totalAmount += amount
	} else {
		g.paymentsFailed++
	}
	g.mu.Unlock()

	// CPU metric - payments has higher CPU due to encryption
	cpuMetric := g.generateCPUMetric()
	if err := g.publisher.PublishMetric(ctx, cpuMetric); err != nil {
		g.logger.Warn("failed to publish CPU metric", zap.Error(err))
	}

	// Memory metric
	memMetric := g.generateMemoryMetric()
	if err := g.publisher.PublishMetric(ctx, memMetric); err != nil {
		g.logger.Warn("failed to publish memory metric", zap.Error(err))
	}

	// Latency metric - payment gateway calls add latency
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
	// Payments service: higher CPU due to cryptographic operations
	baseCPU := 35.0 + rand.Float64()*40.0

	// Spike during batch processing
	if rand.Float64() < 0.1 {
		baseCPU += 20.0 + rand.Float64()*15.0
	}

	if baseCPU > 95 {
		baseCPU = 95
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeCPU, baseCPU, "percent")
	metric.Labels["host"] = "payments-service-1"
	metric.Labels["environment"] = "development"
	return metric
}

func (g *MetricsGenerator) generateMemoryMetric() *models.ServiceMetric {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	memUsage := float64(memStats.Alloc) / 1024 / 1024
	memUsage += rand.Float64() * 100 // Payment processing needs memory for transaction state

	memPercent := (memUsage / 1024) * 100
	if memPercent > 90 {
		memPercent = 85 + rand.Float64()*5
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeMemory, memPercent, "percent")
	metric.Labels["host"] = "payments-service-1"
	return metric
}

func (g *MetricsGenerator) generateLatencyMetric() *models.ServiceMetric {
	// Payment gateway calls are slower
	baseLatency := 150.0 + rand.NormFloat64()*50.0

	// External payment provider latency
	if rand.Float64() < 0.3 {
		baseLatency += 100 + rand.Float64()*200
	}

	// Occasional very slow responses from payment providers
	if rand.Float64() < 0.05 {
		baseLatency += 500 + rand.Float64()*500
	}

	if baseLatency < 50 {
		baseLatency = 50
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeLatency, baseLatency, "ms")
	metric.Labels["endpoint"] = g.randomEndpoint()
	metric.Labels["payment_provider"] = g.randomPaymentProvider()
	return metric
}

func (g *MetricsGenerator) generateErrorMetric() *models.ServiceMetric {
	// Payment errors are carefully monitored
	errorRate := 2.5 + rand.NormFloat64()*1.5

	// Gateway issues cause error spikes
	if rand.Float64() < 0.04 {
		errorRate += 10 + rand.Float64()*15
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
	if rand.Float64() < 0.015 {
		status = 0.0
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeStatus, status, "boolean")
	return metric
}

func (g *MetricsGenerator) publishLogs(ctx context.Context) {
	logType := rand.Intn(100)

	var log *models.ServiceLog

	switch {
	case logType < 50:
		log = g.generateInfoLog()
	case logType < 75:
		log = g.generateDebugLog()
	case logType < 90:
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
		"Payment processed successfully",
		"Credit card validated",
		"Payment authorization received",
		"Refund initiated",
		"Payment captured",
		"Transaction completed",
		"Payment method added to account",
		"Recurring payment scheduled",
		"Payment confirmation sent",
		"Settlement batch completed",
		"Fraud check passed",
		"Payment webhook received",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelInfo, messages[rand.Intn(len(messages))])
	log.Fields["transaction_id"] = "txn_" + randomHex(16)
	log.Fields["payment_id"] = "pay_" + randomHex(12)
	log.Fields["amount"] = rand.Float64()*500 + 10
	log.Fields["currency"] = "USD"
	log.Fields["provider"] = g.randomPaymentProvider()
	log.Caller = "internal/core/payment_service.go:178"
	return log
}

func (g *MetricsGenerator) generateDebugLog() *models.ServiceLog {
	messages := []string{
		"Validating card number checksum",
		"Encrypting payment data",
		"Connecting to payment gateway",
		"Parsing gateway response",
		"Updating transaction state",
		"Calculating transaction fees",
		"Checking fraud score",
		"Loading payment method from vault",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelDebug, messages[rand.Intn(len(messages))])
	log.Fields["trace_id"] = randomHex(32)
	log.Caller = "internal/core/payment_service.go:95"
	return log
}

func (g *MetricsGenerator) generateWarnLog() *models.ServiceLog {
	messages := []string{
		"Payment retry required",
		"Card expiring soon",
		"Payment gateway response slow",
		"Transaction amount above threshold",
		"Duplicate transaction detected",
		"Currency conversion rate unavailable",
		"Payment method verification pending",
		"Fraud score elevated",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelWarn, messages[rand.Intn(len(messages))])
	log.Fields["transaction_id"] = "txn_" + randomHex(16)
	log.Fields["warning_code"] = g.randomWarningCode()
	log.Caller = "internal/core/payment_service.go:234"
	return log
}

func (g *MetricsGenerator) generateErrorLog() *models.ServiceLog {
	messages := []string{
		"Payment declined: insufficient funds",
		"Card validation failed: invalid CVV",
		"Payment gateway timeout",
		"Transaction rollback required",
		"Fraud detection triggered - payment blocked",
		"Payment provider connection failed",
		"Invalid payment method",
		"Settlement failed: bank rejection",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelError, messages[rand.Intn(len(messages))])
	log.Fields["transaction_id"] = "txn_" + randomHex(16)
	log.Fields["error_code"] = g.randomErrorCode()
	log.Fields["stack_trace"] = "goroutine 1 [running]:\nmain.processPayment()\n\t/app/internal/core/payment_service.go:289 +0x4c1"
	log.Caller = "internal/core/payment_service.go:289"
	return log
}

func (g *MetricsGenerator) randomEndpoint() string {
	endpoints := []string{
		"/api/v1/payments",
		"/api/v1/payments/{id}",
		"/api/v1/payments/{id}/capture",
		"/api/v1/payments/{id}/refund",
		"/api/v1/payments/methods",
		"/api/v1/payments/webhook",
		"/health",
	}
	return endpoints[rand.Intn(len(endpoints))]
}

func (g *MetricsGenerator) randomPaymentProvider() string {
	providers := []string{"stripe", "paypal", "square", "adyen", "braintree"}
	return providers[rand.Intn(len(providers))]
}

func (g *MetricsGenerator) randomErrorType() string {
	types := []string{"declined", "validation", "gateway", "fraud", "timeout", "network", "internal"}
	return types[rand.Intn(len(types))]
}

func (g *MetricsGenerator) randomWarningCode() string {
	codes := []string{"PAY_WARN_001", "PAY_WARN_002", "PAY_WARN_003", "FRD_WARN_001", "GW_WARN_001"}
	return codes[rand.Intn(len(codes))]
}

func (g *MetricsGenerator) randomErrorCode() string {
	codes := []string{"PAY_ERR_001", "PAY_ERR_002", "PAY_ERR_003", "FRD_ERR_001", "GW_ERR_001", "NET_ERR_001"}
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
