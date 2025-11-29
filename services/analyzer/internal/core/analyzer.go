// Package core provides the core analysis logic for the analyzer service.
package core

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/models"
	"github.com/microservices-platform/services/analyzer/internal/ports"
)

// AnalysisConfig holds configuration for the analyzer.
type AnalysisConfig struct {
	// Window sizes
	SlidingWindowSize time.Duration
	RollingWindowSize time.Duration
	AnalysisInterval  time.Duration

	// Default thresholds
	DefaultCPUThreshold       float64
	DefaultMemoryThreshold    float64
	DefaultLatencyThreshold   float64
	DefaultErrorRateThreshold float64

	// Deviation analysis
	DeviationMultiplier    float64 // Standard deviations from mean
	MinSamplesForDeviation int

	// Alert settings
	DefaultCooldownPeriod time.Duration
}

// DefaultAnalysisConfig returns the default configuration.
func DefaultAnalysisConfig() *AnalysisConfig {
	return &AnalysisConfig{
		SlidingWindowSize:         5 * time.Minute,
		RollingWindowSize:         15 * time.Minute,
		AnalysisInterval:          5 * time.Second,
		DefaultCPUThreshold:       80.0,
		DefaultMemoryThreshold:    80.0,
		DefaultLatencyThreshold:   1000.0, // 1 second
		DefaultErrorRateThreshold: 5.0,    // 5%
		DeviationMultiplier:       2.0,    // 2 standard deviations
		MinSamplesForDeviation:    10,
		DefaultCooldownPeriod:     5 * time.Minute,
	}
}

// Analyzer performs real-time anomaly detection on service metrics.
type Analyzer struct {
	config         *AnalysisConfig
	metricsStore   ports.MetricsStore
	rulesStore     ports.RulesStore
	alertPublisher ports.AlertPublisher
	logger         *logging.Logger

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewAnalyzer creates a new Analyzer.
func NewAnalyzer(
	config *AnalysisConfig,
	metricsStore ports.MetricsStore,
	rulesStore ports.RulesStore,
	alertPublisher ports.AlertPublisher,
	logger *logging.Logger,
) *Analyzer {
	if config == nil {
		config = DefaultAnalysisConfig()
	}

	return &Analyzer{
		config:         config,
		metricsStore:   metricsStore,
		rulesStore:     rulesStore,
		alertPublisher: alertPublisher,
		logger:         logger,
	}
}

// Start starts the analyzer.
func (a *Analyzer) Start(ctx context.Context) error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return nil
	}
	a.running = true
	a.stopCh = make(chan struct{})
	a.mu.Unlock()

	a.logger.Info("starting analyzer",
		zap.Duration("analysis_interval", a.config.AnalysisInterval),
	)

	a.wg.Add(1)
	go a.runAnalysisLoop(ctx)

	return nil
}

// Stop stops the analyzer.
func (a *Analyzer) Stop() error {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return nil
	}
	a.running = false
	close(a.stopCh)
	a.mu.Unlock()

	a.wg.Wait()
	a.logger.Info("analyzer stopped")
	return nil
}

func (a *Analyzer) runAnalysisLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(a.config.AnalysisInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.performAnalysis(ctx)
		}
	}
}

func (a *Analyzer) performAnalysis(ctx context.Context) {
	// Get all enabled rules
	rules, err := a.rulesStore.GetEnabledRules(ctx)
	if err != nil {
		a.logger.Error("failed to get enabled rules", zap.Error(err))
		return
	}

	// Track which services we need to analyze
	servicesToAnalyze := make(map[models.ServiceName]bool)
	for _, rule := range rules {
		servicesToAnalyze[rule.ServiceName] = true
	}

	// If no rules, analyze all known services from metrics
	if len(servicesToAnalyze) == 0 {
		servicesToAnalyze[models.ServiceNameAuth] = true
		servicesToAnalyze[models.ServiceNameOrders] = true
		servicesToAnalyze[models.ServiceNamePayments] = true
		servicesToAnalyze[models.ServiceNameNotification] = true
	}

	for service := range servicesToAnalyze {
		a.analyzeService(ctx, service, rules)
	}
}

func (a *Analyzer) analyzeService(ctx context.Context, service models.ServiceName, rules []*models.ThresholdRule) {
	// Get recent metrics
	metrics, err := a.metricsStore.GetMetricsWindow(ctx, service, a.config.SlidingWindowSize)
	if err != nil {
		a.logger.Warn("failed to get metrics window",
			zap.String("service", string(service)),
			zap.Error(err),
		)
		return
	}

	if len(metrics) == 0 {
		return
	}

	// Get the most recent metric
	latest := metrics[len(metrics)-1]

	// Check threshold rules
	for _, rule := range rules {
		if rule.ServiceName != service || !rule.Enabled {
			continue
		}
		a.checkThresholdRule(ctx, rule, latest, metrics)
	}

	// Perform deviation analysis
	a.checkDeviations(ctx, service, latest, metrics)
}

func (a *Analyzer) checkThresholdRule(ctx context.Context, rule *models.ThresholdRule, latest *models.ServiceMetric, history []*models.ServiceMetric) {
	var currentValue float64
	var exceeded bool

	switch rule.MetricType {
	case models.MetricTypeCPU:
		currentValue = latest.CPUUsage
		exceeded = currentValue > rule.Threshold
	case models.MetricTypeMemory:
		currentValue = latest.MemoryUsage
		exceeded = currentValue > rule.Threshold
	case models.MetricTypeLatencyP95:
		currentValue = latest.LatencyP95
		exceeded = currentValue > rule.Threshold
	case models.MetricTypeLatencyP99:
		currentValue = latest.LatencyP99
		exceeded = currentValue > rule.Threshold
	case models.MetricTypeErrorRate:
		currentValue = latest.ErrorRate
		exceeded = currentValue > rule.Threshold
	case models.MetricTypeRequestRate:
		currentValue = latest.RequestCount
		exceeded = currentValue > rule.Threshold
	default:
		return
	}

	if exceeded {
		a.generateAlert(ctx, rule.ServiceName, rule.MetricType, rule.Severity, currentValue, rule.Threshold, "threshold_violation")
	}
}

func (a *Analyzer) checkDeviations(ctx context.Context, service models.ServiceName, latest *models.ServiceMetric, history []*models.ServiceMetric) {
	if len(history) < a.config.MinSamplesForDeviation {
		return
	}

	// Check CPU deviation
	a.checkMetricDeviation(ctx, service, models.MetricTypeCPU, latest.CPUUsage, history, func(m *models.ServiceMetric) float64 { return m.CPUUsage })

	// Check memory deviation
	a.checkMetricDeviation(ctx, service, models.MetricTypeMemory, latest.MemoryUsage, history, func(m *models.ServiceMetric) float64 { return m.MemoryUsage })

	// Check latency P95 deviation
	a.checkMetricDeviation(ctx, service, models.MetricTypeLatencyP95, latest.LatencyP95, history, func(m *models.ServiceMetric) float64 { return m.LatencyP95 })

	// Check error rate deviation
	a.checkMetricDeviation(ctx, service, models.MetricTypeErrorRate, latest.ErrorRate, history, func(m *models.ServiceMetric) float64 { return m.ErrorRate })
}

func (a *Analyzer) checkMetricDeviation(
	ctx context.Context,
	service models.ServiceName,
	metricType models.MetricType,
	currentValue float64,
	history []*models.ServiceMetric,
	extractor func(*models.ServiceMetric) float64,
) {
	// Calculate mean and standard deviation
	var sum, sumSquares float64
	n := float64(len(history))

	for _, m := range history {
		v := extractor(m)
		sum += v
		sumSquares += v * v
	}

	mean := sum / n
	variance := (sumSquares / n) - (mean * mean)
	stdDev := math.Sqrt(variance)

	// Check if current value deviates significantly
	if stdDev > 0 {
		deviation := math.Abs(currentValue-mean) / stdDev
		if deviation > a.config.DeviationMultiplier {
			severity := models.AlertSeverityWarning
			if deviation > a.config.DeviationMultiplier*2 {
				severity = models.AlertSeverityCritical
			}
			a.generateAlert(ctx, service, metricType, severity, currentValue, mean, "deviation_detected")
		}
	}
}

func (a *Analyzer) generateAlert(
	ctx context.Context,
	service models.ServiceName,
	metricType models.MetricType,
	severity models.AlertSeverity,
	currentValue, threshold float64,
	alertType string,
) {
	// Generate deduplication key
	deduplicationKey := fmt.Sprintf("%s:%s:%s", service, metricType, alertType)

	// Check if we already sent this alert recently
	alreadySent, err := a.metricsStore.CheckAndSetAlertSent(ctx, deduplicationKey, 5*time.Minute)
	if err != nil {
		a.logger.Warn("failed to check alert deduplication", zap.Error(err))
	}
	if alreadySent {
		return
	}

	// Check cooldown
	inCooldown, err := a.metricsStore.CheckCooldown(ctx, string(service), string(metricType))
	if err != nil {
		a.logger.Warn("failed to check cooldown", zap.Error(err))
	}
	if inCooldown {
		return
	}

	// Create alert
	alert := &models.Alert{
		ID:           uuid.New().String(),
		ServiceName:  service,
		MetricType:   metricType,
		Severity:     severity,
		Title:        a.generateAlertTitle(alertType, service, metricType),
		Message:      a.generateAlertMessage(alertType, service, metricType, currentValue, threshold),
		CurrentValue: currentValue,
		Threshold:    threshold,
		Timestamp:    time.Now(),
		Labels: map[string]string{
			"alert_type": alertType,
			"service":    string(service),
			"metric":     string(metricType),
		},
	}

	// Publish alert
	if err := a.alertPublisher.PublishAlert(ctx, alert); err != nil {
		a.logger.Error("failed to publish alert",
			zap.String("alert_id", alert.ID),
			zap.Error(err),
		)
		return
	}

	// Set cooldown
	if err := a.metricsStore.SetCooldown(ctx, string(service), string(metricType), a.config.DefaultCooldownPeriod); err != nil {
		a.logger.Warn("failed to set cooldown", zap.Error(err))
	}

	a.logger.Info("alert generated",
		zap.String("alert_id", alert.ID),
		zap.String("service", string(service)),
		zap.String("metric_type", string(metricType)),
		zap.String("severity", string(severity)),
		zap.Float64("current_value", currentValue),
		zap.Float64("threshold", threshold),
	)
}

func (a *Analyzer) generateAlertTitle(alertType string, service models.ServiceName, metricType models.MetricType) string {
	switch alertType {
	case "threshold_violation":
		return fmt.Sprintf("[%s] %s threshold exceeded for %s", strings.ToUpper(string(service)), metricType, service)
	case "deviation_detected":
		return fmt.Sprintf("[%s] Anomaly detected in %s for %s", strings.ToUpper(string(service)), metricType, service)
	default:
		return fmt.Sprintf("[%s] Alert for %s: %s", strings.ToUpper(string(service)), metricType, alertType)
	}
}

func (a *Analyzer) generateAlertMessage(alertType string, service models.ServiceName, metricType models.MetricType, currentValue, threshold float64) string {
	switch alertType {
	case "threshold_violation":
		return fmt.Sprintf(
			"The %s metric for service %s has exceeded the threshold.\n\nCurrent Value: %.2f\nThreshold: %.2f\n\nPlease investigate immediately.",
			metricType, service, currentValue, threshold,
		)
	case "deviation_detected":
		return fmt.Sprintf(
			"An anomaly has been detected in %s for service %s.\n\nCurrent Value: %.2f\nExpected (Mean): %.2f\n\nThe current value deviates significantly from the historical average.",
			metricType, service, currentValue, threshold,
		)
	default:
		return fmt.Sprintf("Alert for %s: %s (current: %.2f, reference: %.2f)", service, metricType, currentValue, threshold)
	}
}

// AnalyzeMetric immediately analyzes a single metric (useful for real-time processing).
func (a *Analyzer) AnalyzeMetric(ctx context.Context, metric *models.ServiceMetric) error {
	// Get relevant rules for this service
	rules, err := a.rulesStore.GetRulesByService(ctx, metric.ServiceName)
	if err != nil {
		return err
	}

	// Store the metric first
	if err := a.metricsStore.AddMetric(ctx, metric); err != nil {
		a.logger.Warn("failed to store metric for analysis", zap.Error(err))
	}

	// Get historical metrics for deviation analysis
	history, err := a.metricsStore.GetMetricsWindow(ctx, metric.ServiceName, a.config.SlidingWindowSize)
	if err != nil {
		a.logger.Warn("failed to get historical metrics", zap.Error(err))
		history = []*models.ServiceMetric{metric}
	}

	// Check threshold rules
	for _, rule := range rules {
		if rule.Enabled {
			a.checkThresholdRule(ctx, rule, metric, history)
		}
	}

	// Check deviations if we have enough history
	if len(history) >= a.config.MinSamplesForDeviation {
		a.checkDeviations(ctx, metric.ServiceName, metric, history)
	}

	return nil
}
