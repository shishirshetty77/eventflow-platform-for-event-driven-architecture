// Package core contains the business logic for the auth service.
package core

import (
	"context"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/models"
	"github.com/microservices-platform/services/auth/internal/ports"
)

// MetricsGeneratorImpl generates realistic metrics and logs.
type MetricsGeneratorImpl struct {
	serviceName     models.ServiceName
	publisher       ports.MetricsPublisher
	logger          *logging.Logger
	metricsInterval time.Duration
	logsInterval    time.Duration
	startTime       time.Time

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup

	// Simulated state
	baseLatency    float64
	errorRate      float64
	operationCount int64
}

// NewMetricsGenerator creates a new MetricsGenerator.
func NewMetricsGenerator(
	serviceName models.ServiceName,
	publisher ports.MetricsPublisher,
	logger *logging.Logger,
	metricsInterval, logsInterval time.Duration,
) *MetricsGeneratorImpl {
	return &MetricsGeneratorImpl{
		serviceName:     serviceName,
		publisher:       publisher,
		logger:          logger,
		metricsInterval: metricsInterval,
		logsInterval:    logsInterval,
		startTime:       time.Now(),
		baseLatency:     50.0, // base latency in ms
		errorRate:       0.02, // 2% error rate
	}
}

// Start starts the metrics generator.
func (g *MetricsGeneratorImpl) Start(ctx context.Context) error {
	g.mu.Lock()
	if g.running {
		g.mu.Unlock()
		return nil
	}
	g.running = true
	g.stopCh = make(chan struct{})
	g.mu.Unlock()

	g.logger.Info("starting metrics generator",
		zap.String("service", string(g.serviceName)),
		zap.Duration("metrics_interval", g.metricsInterval),
		zap.Duration("logs_interval", g.logsInterval),
	)

	// Start metrics generation goroutine
	g.wg.Add(1)
	go g.generateMetrics(ctx)

	// Start logs generation goroutine
	g.wg.Add(1)
	go g.generateLogs(ctx)

	return nil
}

// Stop stops the metrics generator.
func (g *MetricsGeneratorImpl) Stop() error {
	g.mu.Lock()
	if !g.running {
		g.mu.Unlock()
		return nil
	}
	g.running = false
	close(g.stopCh)
	g.mu.Unlock()

	g.wg.Wait()
	g.logger.Info("metrics generator stopped")
	return nil
}

// generateMetrics periodically generates and publishes metrics.
func (g *MetricsGeneratorImpl) generateMetrics(ctx context.Context) {
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

// generateLogs periodically generates and publishes logs.
func (g *MetricsGeneratorImpl) generateLogs(ctx context.Context) {
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

// publishMetrics generates and publishes all metric types.
func (g *MetricsGeneratorImpl) publishMetrics(ctx context.Context) {
	g.mu.Lock()
	g.operationCount++
	g.mu.Unlock()

	// Generate CPU metric
	cpuMetric := g.generateCPUMetric()
	if err := g.publisher.PublishMetric(ctx, cpuMetric); err != nil {
		g.logger.Warn("failed to publish CPU metric", zap.Error(err))
	}

	// Generate Memory metric
	memMetric := g.generateMemoryMetric()
	if err := g.publisher.PublishMetric(ctx, memMetric); err != nil {
		g.logger.Warn("failed to publish memory metric", zap.Error(err))
	}

	// Generate Latency metric
	latencyMetric := g.generateLatencyMetric()
	if err := g.publisher.PublishMetric(ctx, latencyMetric); err != nil {
		g.logger.Warn("failed to publish latency metric", zap.Error(err))
	}

	// Generate Error metric
	errorMetric := g.generateErrorMetric()
	if err := g.publisher.PublishMetric(ctx, errorMetric); err != nil {
		g.logger.Warn("failed to publish error metric", zap.Error(err))
	}

	// Generate Status metric
	statusMetric := g.generateStatusMetric()
	if err := g.publisher.PublishMetric(ctx, statusMetric); err != nil {
		g.logger.Warn("failed to publish status metric", zap.Error(err))
	}
}

// generateCPUMetric generates a realistic CPU usage metric.
func (g *MetricsGeneratorImpl) generateCPUMetric() *models.ServiceMetric {
	// Simulate CPU usage between 10-80% with some spikes
	baseCPU := 25.0 + rand.Float64()*30.0

	// Add occasional spikes
	if rand.Float64() < 0.1 {
		baseCPU += 20.0 + rand.Float64()*25.0
	}

	// Clamp to reasonable range
	if baseCPU > 95 {
		baseCPU = 95
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeCPU, baseCPU, "percent")
	metric.Labels["host"] = "auth-service-1"
	metric.Labels["environment"] = "development"
	return metric
}

// generateMemoryMetric generates a realistic memory usage metric.
func (g *MetricsGeneratorImpl) generateMemoryMetric() *models.ServiceMetric {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Use actual memory stats but add some variation
	memUsage := float64(memStats.Alloc) / 1024 / 1024 // MB
	memUsage += rand.Float64() * 50                   // Add some variation

	// Convert to percentage (assuming 512MB allocated)
	memPercent := (memUsage / 512) * 100
	if memPercent > 95 {
		memPercent = 90 + rand.Float64()*5
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeMemory, memPercent, "percent")
	metric.Labels["host"] = "auth-service-1"
	metric.Labels["heap_mb"] = "allocated"
	return metric
}

// generateLatencyMetric generates a realistic latency metric.
func (g *MetricsGeneratorImpl) generateLatencyMetric() *models.ServiceMetric {
	// Base latency with normal distribution
	latency := g.baseLatency + rand.NormFloat64()*15

	// Add occasional latency spikes
	if rand.Float64() < 0.05 {
		latency += 100 + rand.Float64()*200
	}

	if latency < 5 {
		latency = 5
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeLatency, latency, "ms")
	metric.Labels["endpoint"] = g.randomEndpoint()
	metric.Labels["method"] = g.randomHTTPMethod()
	return metric
}

// generateErrorMetric generates an error rate metric.
func (g *MetricsGeneratorImpl) generateErrorMetric() *models.ServiceMetric {
	// Base error rate with some variation
	errorRate := g.errorRate * 100 // Convert to percentage
	errorRate += rand.NormFloat64() * 1.5

	// Occasional error bursts
	if rand.Float64() < 0.03 {
		errorRate += 5 + rand.Float64()*10
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

// generateStatusMetric generates a status metric (1 for healthy, 0 for unhealthy).
func (g *MetricsGeneratorImpl) generateStatusMetric() *models.ServiceMetric {
	status := 1.0

	// Very occasional unhealthy status
	if rand.Float64() < 0.01 {
		status = 0.0
	}

	metric := models.NewServiceMetric(g.serviceName, models.MetricTypeStatus, status, "boolean")
	metric.Labels["uptime_seconds"] = "active"
	return metric
}

// publishLogs generates and publishes various log types.
func (g *MetricsGeneratorImpl) publishLogs(ctx context.Context) {
	// Generate different types of logs
	logType := rand.Intn(100)

	var log *models.ServiceLog

	switch {
	case logType < 60: // 60% info logs
		log = g.generateInfoLog()
	case logType < 85: // 25% debug logs
		log = g.generateDebugLog()
	case logType < 95: // 10% warning logs
		log = g.generateWarnLog()
	default: // 5% error logs
		log = g.generateErrorLog()
	}

	if err := g.publisher.PublishLog(ctx, log); err != nil {
		g.logger.Warn("failed to publish log", zap.Error(err))
	}
}

// generateInfoLog generates a realistic info log.
func (g *MetricsGeneratorImpl) generateInfoLog() *models.ServiceLog {
	messages := []string{
		"User authentication successful",
		"Token generated for user session",
		"Login request processed",
		"User session validated",
		"Token refresh completed",
		"Password validation passed",
		"User profile retrieved",
		"Authentication middleware check passed",
		"Session extended for active user",
		"OAuth callback processed successfully",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelInfo, messages[rand.Intn(len(messages))])
	log.Fields["user_id"] = g.randomUserID()
	log.Fields["request_id"] = g.randomRequestID()
	log.Fields["duration_ms"] = rand.Intn(100) + 10
	log.Caller = "internal/core/service.go:125"
	return log
}

// generateDebugLog generates a realistic debug log.
func (g *MetricsGeneratorImpl) generateDebugLog() *models.ServiceLog {
	messages := []string{
		"Parsing JWT token claims",
		"Checking user permissions",
		"Loading user from cache",
		"Validating request headers",
		"Decoding authentication payload",
		"Checking rate limit for user",
		"Resolving user roles",
		"Preparing authentication context",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelDebug, messages[rand.Intn(len(messages))])
	log.Fields["trace_id"] = g.randomTraceID()
	log.Caller = "internal/core/service.go:89"
	return log
}

// generateWarnLog generates a realistic warning log.
func (g *MetricsGeneratorImpl) generateWarnLog() *models.ServiceLog {
	messages := []string{
		"Token expiring soon for user session",
		"Multiple failed login attempts detected",
		"Unusual login location detected",
		"Rate limit threshold approaching",
		"Session nearing timeout",
		"Deprecated authentication method used",
		"Weak password detected during login",
		"Suspicious authentication pattern observed",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelWarn, messages[rand.Intn(len(messages))])
	log.Fields["user_id"] = g.randomUserID()
	log.Fields["ip_address"] = g.randomIP()
	log.Fields["attempt_count"] = rand.Intn(5) + 1
	log.Caller = "internal/core/service.go:156"
	return log
}

// generateErrorLog generates a realistic error log.
func (g *MetricsGeneratorImpl) generateErrorLog() *models.ServiceLog {
	messages := []string{
		"Failed to validate token: signature mismatch",
		"Database connection error during user lookup",
		"Password hash verification failed",
		"Token generation failed: signing error",
		"User not found in database",
		"Session store connection timeout",
		"Failed to decode JWT claims",
		"Authentication service unavailable",
	}

	log := models.NewServiceLog(g.serviceName, models.LogLevelError, messages[rand.Intn(len(messages))])
	log.Fields["error_code"] = g.randomErrorCode()
	log.Fields["request_id"] = g.randomRequestID()
	log.Fields["stack_trace"] = "goroutine 1 [running]:\nmain.main()\n\t/app/cmd/main.go:45 +0x1a5"
	log.Caller = "internal/core/service.go:203"
	return log
}

// Helper functions for generating random values.

func (g *MetricsGeneratorImpl) randomEndpoint() string {
	endpoints := []string{"/api/v1/auth/login", "/api/v1/auth/register", "/api/v1/auth/refresh", "/api/v1/auth/validate", "/api/v1/auth/logout", "/health"}
	return endpoints[rand.Intn(len(endpoints))]
}

func (g *MetricsGeneratorImpl) randomHTTPMethod() string {
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	return methods[rand.Intn(len(methods))]
}

func (g *MetricsGeneratorImpl) randomErrorType() string {
	types := []string{"validation", "authentication", "authorization", "timeout", "internal", "connection"}
	return types[rand.Intn(len(types))]
}

func (g *MetricsGeneratorImpl) randomUserID() string {
	return "usr_" + randomHex(8)
}

func (g *MetricsGeneratorImpl) randomRequestID() string {
	return "req_" + randomHex(16)
}

func (g *MetricsGeneratorImpl) randomTraceID() string {
	return randomHex(32)
}

func (g *MetricsGeneratorImpl) randomErrorCode() string {
	codes := []string{"AUTH001", "AUTH002", "AUTH003", "DB001", "DB002", "NET001", "VAL001"}
	return codes[rand.Intn(len(codes))]
}

func (g *MetricsGeneratorImpl) randomIP() string {
	return "192.168." + string(rune(rand.Intn(255))) + "." + string(rune(rand.Intn(255)))
}

func randomHex(length int) string {
	const hexChars = "0123456789abcdef"
	result := make([]byte, length)
	for i := range result {
		result[i] = hexChars[rand.Intn(len(hexChars))]
	}
	return string(result)
}
