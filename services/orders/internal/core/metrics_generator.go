// Package core contains the business logic for the orders service.
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

// MetricsGenerator generates realistic metrics and logs for the orders service.
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

	// Simulated state for orders service
	ordersCreated  int64
	ordersComplete int64
	ordersFailed   int64
}

// NewMetricsGenerator creates a new MetricsGenerator for the orders service.
func NewMetricsGenerator(
	publisher MetricsPublisher,
	logger *logging.Logger,
	m *metrics.Metrics,
	metricsInterval, logsInterval time.Duration,
) *MetricsGenerator {
	return &MetricsGenerator{
		serviceName:     models.ServiceOrders,
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

	g.logger.Info("starting orders metrics generator",
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
	g.logger.Info("orders metrics generator stopped")
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
	// Simulate order operations
	g.mu.Lock()
	g.ordersCreated += int64(rand.Intn(10) + 1)
	if rand.Float64() > 0.05 {
		g.ordersComplete += int64(rand.Intn(8) + 1)
	} else {
		g.ordersFailed += int64(rand.Intn(2) + 1)
	}
	g.mu.Unlock()

	// CPU metric - orders service typically has moderate CPU usage
	cpuMetric := g.generateCPUMetric()
	if err := g.publisher.PublishMetric(ctx, cpuMetric); err != nil {
		g.logger.Warn("failed to publish CPU metric", zap.Error(err))
	}

	// Memory metric
	memMetric := g.generateMemoryMetric()
	if err := g.publisher.PublishMetric(ctx, memMetric); err != nil {
		g.logger.Warn("failed to publish memory metric", zap.Error(err))
	}

	// Latency metric - orders processing latency
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
	// Orders service: moderate to high CPU during order processing
	baseCPU := 30.0 + rand.Float64()*35.0

	// Burst during high order volume
	if rand.Float64() < 0.15 {
		baseCPU += 15.0 + rand.Float64()*20.0
	}

	if baseCPU > 95 {
		baseCPU = 95
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeCPU, baseCPU, "percent")
	metric.Labels["host"] = "orders-service-1"
	metric.Labels["environment"] = "development"
	metric.Labels["orders_in_queue"] = string(rune(rand.Intn(100)))
	return metric
}

func (g *MetricsGenerator) generateMemoryMetric() *models.ServiceMetric {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	memUsage := float64(memStats.Alloc) / 1024 / 1024
	memUsage += rand.Float64() * 80 // Orders need more memory for order state

	memPercent := (memUsage / 1024) * 100
	if memPercent > 90 {
		memPercent = 85 + rand.Float64()*5
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeMemory, memPercent, "percent")
	metric.Labels["host"] = "orders-service-1"
	return metric
}

func (g *MetricsGenerator) generateLatencyMetric() *models.ServiceMetric {
	// Order processing typically takes longer than auth
	baseLatency := 80.0 + rand.NormFloat64()*25.0

	// Database queries and payment validation add latency
	if rand.Float64() < 0.2 {
		baseLatency += 50 + rand.Float64()*100
	}

	// Occasional slow queries
	if rand.Float64() < 0.05 {
		baseLatency += 200 + rand.Float64()*300
	}

	if baseLatency < 20 {
		baseLatency = 20
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeLatency, baseLatency, "ms")
	metric.Labels["endpoint"] = g.randomEndpoint()
	metric.Labels["method"] = g.randomHTTPMethod()
	return metric
}

func (g *MetricsGenerator) generateErrorMetric() *models.ServiceMetric {
	// Orders have slightly higher error rate due to validation, inventory checks
	errorRate := 3.0 + rand.NormFloat64()*2.0

	if rand.Float64() < 0.05 {
		errorRate += 8 + rand.Float64()*12
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
	if rand.Float64() < 0.02 {
		status = 0.0
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeStatus, status, "boolean")
	return metric
}

func (g *MetricsGenerator) publishLogs(ctx context.Context) {
	logType := rand.Intn(100)

	var log *models.ServiceLog

	switch {
	case logType < 55:
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
		"Order created successfully",
		"Order status updated to processing",
		"Order items validated",
		"Inventory check passed",
		"Order shipped to fulfillment",
		"Order completed",
		"Customer notified of order status",
		"Order payment confirmed",
		"Discount code applied to order",
		"Order queued for processing",
		"Shipping label generated",
		"Order tracking number assigned",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelInfo, messages[rand.Intn(len(messages))])
	log.Fields["order_id"] = "ord_" + randomHex(12)
	log.Fields["customer_id"] = "cust_" + randomHex(8)
	log.Fields["total_amount"] = rand.Float64()*500 + 10
	log.Fields["items_count"] = rand.Intn(10) + 1
	log.Caller = "internal/core/order_service.go:156"
	return log
}

func (g *MetricsGenerator) generateDebugLog() *models.ServiceLog {
	messages := []string{
		"Validating order items against inventory",
		"Calculating shipping costs",
		"Applying tax rules for region",
		"Checking customer loyalty status",
		"Loading order from cache",
		"Preparing order confirmation email",
		"Updating inventory reservations",
		"Processing bulk order batch",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelDebug, messages[rand.Intn(len(messages))])
	log.Fields["trace_id"] = randomHex(32)
	log.Caller = "internal/core/order_service.go:89"
	return log
}

func (g *MetricsGenerator) generateWarnLog() *models.ServiceLog {
	messages := []string{
		"Low inventory for requested item",
		"Payment processing delayed",
		"Order total exceeds typical amount",
		"Shipping address validation warning",
		"Customer has pending orders limit reached",
		"Discount code near expiration",
		"High order volume detected",
		"Database query taking longer than expected",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelWarn, messages[rand.Intn(len(messages))])
	log.Fields["order_id"] = "ord_" + randomHex(12)
	log.Fields["warning_code"] = g.randomWarningCode()
	log.Caller = "internal/core/order_service.go:203"
	return log
}

func (g *MetricsGenerator) generateErrorLog() *models.ServiceLog {
	messages := []string{
		"Failed to process order: inventory unavailable",
		"Payment validation failed for order",
		"Database connection timeout during order save",
		"Invalid shipping address format",
		"Order creation failed: duplicate order ID",
		"Failed to send order confirmation",
		"Inventory service unavailable",
		"Failed to calculate shipping: carrier API error",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelError, messages[rand.Intn(len(messages))])
	log.Fields["order_id"] = "ord_" + randomHex(12)
	log.Fields["error_code"] = g.randomErrorCode()
	log.Fields["stack_trace"] = "goroutine 1 [running]:\nmain.processOrder()\n\t/app/internal/core/order_service.go:245 +0x3b2"
	log.Caller = "internal/core/order_service.go:245"
	return log
}

func (g *MetricsGenerator) randomEndpoint() string {
	endpoints := []string{
		"/api/v1/orders",
		"/api/v1/orders/{id}",
		"/api/v1/orders/{id}/status",
		"/api/v1/orders/{id}/cancel",
		"/api/v1/orders/{id}/items",
		"/api/v1/orders/bulk",
		"/health",
	}
	return endpoints[rand.Intn(len(endpoints))]
}

func (g *MetricsGenerator) randomHTTPMethod() string {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	weights := []int{40, 30, 15, 10, 5}
	total := 0
	for _, w := range weights {
		total += w
	}
	r := rand.Intn(total)
	for i, w := range weights {
		r -= w
		if r < 0 {
			return methods[i]
		}
	}
	return methods[0]
}

func (g *MetricsGenerator) randomErrorType() string {
	types := []string{"validation", "inventory", "payment", "shipping", "database", "timeout", "internal"}
	return types[rand.Intn(len(types))]
}

func (g *MetricsGenerator) randomWarningCode() string {
	codes := []string{"ORD_WARN_001", "ORD_WARN_002", "ORD_WARN_003", "INV_WARN_001", "PAY_WARN_001"}
	return codes[rand.Intn(len(codes))]
}

func (g *MetricsGenerator) randomErrorCode() string {
	codes := []string{"ORD_ERR_001", "ORD_ERR_002", "ORD_ERR_003", "INV_ERR_001", "PAY_ERR_001", "DB_ERR_001"}
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

// PublishMetric logs the metric instead of publishing to Kafka.
func (p *MockPublisher) PublishMetric(ctx context.Context, metric *models.ServiceMetric) error {
	p.logger.Info("mock: metric published",
		zap.String("type", string(metric.MetricType)),
		zap.Float64("value", metric.Value),
	)
	return nil
}

// PublishLog logs the entry instead of publishing to Kafka.
func (p *MockPublisher) PublishLog(ctx context.Context, log *models.ServiceLog) error {
	p.logger.Info("mock: log published",
		zap.String("level", string(log.Level)),
		zap.String("message", log.Message),
	)
	return nil
}

// Close is a no-op for mock publisher.
func (p *MockPublisher) Close() error {
	return nil
}
