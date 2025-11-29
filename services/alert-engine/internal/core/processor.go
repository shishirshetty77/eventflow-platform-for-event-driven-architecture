// Package core provides the core alert processing logic.
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	sharedkafka "github.com/microservices-platform/pkg/shared/kafka"
	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/metrics"
	"github.com/microservices-platform/pkg/shared/models"
	"github.com/microservices-platform/services/alert-engine/internal/ports"
)

// ProcessorConfig holds configuration for the alert processor.
type ProcessorConfig struct {
	MaxRetries               int
	RetryDelaySeconds        int
	GroupingWindowSeconds    int
	SuppressionWindowSeconds int
	MaxAlertsPerGroup        int
	BatchSize                int
	BatchTimeout             time.Duration
}

// DefaultProcessorConfig returns the default configuration.
func DefaultProcessorConfig() *ProcessorConfig {
	return &ProcessorConfig{
		MaxRetries:               3,
		RetryDelaySeconds:        5,
		GroupingWindowSeconds:    60,
		SuppressionWindowSeconds: 300,
		MaxAlertsPerGroup:        10,
		BatchSize:                10,
		BatchTimeout:             5 * time.Second,
	}
}

// AlertProcessor processes alerts from Kafka.
type AlertProcessor struct {
	config      *ProcessorConfig
	consumer    *sharedkafka.Consumer
	dlqProducer *sharedkafka.Producer
	dispatchers []ports.AlertDispatcher
	logger      *logging.Logger
	metrics     *metrics.Metrics

	// Grouping
	alertGroups map[string]*ports.AlertGroup
	groupMu     sync.Mutex

	// Suppression
	suppressions map[string]time.Time
	suppressMu   sync.Mutex

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewAlertProcessor creates a new AlertProcessor.
func NewAlertProcessor(
	config *ProcessorConfig,
	brokers []string,
	alertsTopic, dlqTopic, consumerGroup string,
	dispatchers []ports.AlertDispatcher,
	logger *logging.Logger,
	m *metrics.Metrics,
) (*AlertProcessor, error) {
	if config == nil {
		config = DefaultProcessorConfig()
	}

	// Create consumer
	consumerConfig := sharedkafka.DefaultConsumerConfig(brokers, alertsTopic, consumerGroup)
	consumerConfig.StartOffset = kafka.LastOffset
	consumer, err := sharedkafka.NewConsumer(consumerConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Create DLQ producer
	dlqConfig := sharedkafka.DefaultProducerConfig(brokers, dlqTopic)
	dlqProducer, err := sharedkafka.NewProducer(dlqConfig, logger)
	if err != nil {
		consumer.Close()
		return nil, fmt.Errorf("failed to create DLQ producer: %w", err)
	}

	return &AlertProcessor{
		config:       config,
		consumer:     consumer,
		dlqProducer:  dlqProducer,
		dispatchers:  dispatchers,
		logger:       logger,
		metrics:      m,
		alertGroups:  make(map[string]*ports.AlertGroup),
		suppressions: make(map[string]time.Time),
	}, nil
}

// Start starts the alert processor.
func (p *AlertProcessor) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = true
	p.stopCh = make(chan struct{})
	p.mu.Unlock()

	p.logger.Info("starting alert processor")

	// Start consumer loop
	p.wg.Add(1)
	go p.consumeLoop(ctx)

	// Start grouping flush loop
	p.wg.Add(1)
	go p.groupingFlushLoop(ctx)

	// Start suppression cleanup loop
	p.wg.Add(1)
	go p.suppressionCleanupLoop(ctx)

	return nil
}

// Stop stops the alert processor.
func (p *AlertProcessor) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	close(p.stopCh)
	p.mu.Unlock()

	p.wg.Wait()

	if err := p.consumer.Close(); err != nil {
		p.logger.Error("failed to close consumer", zap.Error(err))
	}
	if err := p.dlqProducer.Close(); err != nil {
		p.logger.Error("failed to close DLQ producer", zap.Error(err))
	}

	p.logger.Info("alert processor stopped")
	return nil
}

func (p *AlertProcessor) consumeLoop(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		default:
			msg, err := p.consumer.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				p.logger.Error("failed to fetch message", zap.Error(err))
				time.Sleep(100 * time.Millisecond)
				continue
			}

			p.processMessage(ctx, msg)
			p.consumer.CommitMessages(ctx, msg)
		}
	}
}

func (p *AlertProcessor) processMessage(ctx context.Context, msg kafka.Message) {
	var alert models.Alert
	if err := json.Unmarshal(msg.Value, &alert); err != nil {
		p.logger.Warn("failed to deserialize alert",
			zap.Error(err),
			zap.String("value", string(msg.Value)),
		)
		return
	}

	p.logger.Debug("processing alert",
		zap.String("alert_id", alert.ID),
		zap.String("service", string(alert.ServiceName)),
	)

	// Check suppression
	if p.isSuppressed(&alert) {
		p.logger.Debug("alert suppressed",
			zap.String("alert_id", alert.ID),
		)
		return
	}

	// Add to group
	p.addToGroup(&alert)
}

func (p *AlertProcessor) isSuppressed(alert *models.Alert) bool {
	key := p.getSuppressionKey(alert)

	p.suppressMu.Lock()
	defer p.suppressMu.Unlock()

	if expiry, exists := p.suppressions[key]; exists {
		if time.Now().Before(expiry) {
			return true
		}
		delete(p.suppressions, key)
	}
	return false
}

func (p *AlertProcessor) addSuppression(alert *models.Alert) {
	key := p.getSuppressionKey(alert)
	duration := time.Duration(p.config.SuppressionWindowSeconds) * time.Second

	p.suppressMu.Lock()
	defer p.suppressMu.Unlock()

	p.suppressions[key] = time.Now().Add(duration)
}

func (p *AlertProcessor) getSuppressionKey(alert *models.Alert) string {
	return fmt.Sprintf("%s:%s:%s", alert.ServiceName, alert.MetricType, alert.Severity)
}

func (p *AlertProcessor) addToGroup(alert *models.Alert) {
	groupKey := p.getGroupKey(alert)

	p.groupMu.Lock()
	defer p.groupMu.Unlock()

	group, exists := p.alertGroups[groupKey]
	if !exists {
		group = &ports.AlertGroup{
			ID:          uuid.New().String(),
			GroupKey:    groupKey,
			Alerts:      make([]*models.Alert, 0),
			FirstSeen:   time.Now().Unix(),
			ServiceName: alert.ServiceName,
			Severity:    alert.Severity,
		}
		p.alertGroups[groupKey] = group
	}

	if len(group.Alerts) < p.config.MaxAlertsPerGroup {
		group.Alerts = append(group.Alerts, alert)
	}
	group.LastSeen = time.Now().Unix()
	group.Count++

	// Update severity to highest
	if compareSeverity(alert.Severity, group.Severity) > 0 {
		group.Severity = alert.Severity
	}
}

func (p *AlertProcessor) getGroupKey(alert *models.Alert) string {
	return fmt.Sprintf("%s:%s", alert.ServiceName, alert.MetricType)
}

func (p *AlertProcessor) groupingFlushLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(time.Duration(p.config.GroupingWindowSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.flushGroups(ctx)
		}
	}
}

func (p *AlertProcessor) flushGroups(ctx context.Context) {
	p.groupMu.Lock()
	groups := make([]*ports.AlertGroup, 0, len(p.alertGroups))
	for _, group := range p.alertGroups {
		groups = append(groups, group)
	}
	p.alertGroups = make(map[string]*ports.AlertGroup)
	p.groupMu.Unlock()

	for _, group := range groups {
		if len(group.Alerts) == 0 {
			continue
		}

		// Create a summary alert for the group
		summaryAlert := p.createGroupSummary(group)

		// Dispatch to all enabled dispatchers
		p.dispatchAlert(ctx, summaryAlert)

		// Add suppression for this group
		p.addSuppression(summaryAlert)
	}
}

func (p *AlertProcessor) createGroupSummary(group *ports.AlertGroup) *models.Alert {
	firstAlert := group.Alerts[0]

	title := firstAlert.Title
	message := firstAlert.Message

	if group.Count > 1 {
		title = fmt.Sprintf("%s (+%d more)", firstAlert.Title, group.Count-1)
		message = fmt.Sprintf("%s\n\n--- %d similar alerts were grouped ---", firstAlert.Message, group.Count)
	}

	return &models.Alert{
		ID:           group.ID,
		ServiceName:  group.ServiceName,
		MetricType:   firstAlert.MetricType,
		Severity:     group.Severity,
		Title:        title,
		Message:      message,
		CurrentValue: firstAlert.CurrentValue,
		Threshold:    firstAlert.Threshold,
		Timestamp:    time.Now(),
		Labels: map[string]string{
			"group_id":    group.ID,
			"group_count": fmt.Sprintf("%d", group.Count),
			"first_seen":  time.Unix(group.FirstSeen, 0).Format(time.RFC3339),
		},
	}
}

func (p *AlertProcessor) dispatchAlert(ctx context.Context, alert *models.Alert) {
	for _, dispatcher := range p.dispatchers {
		if !dispatcher.Enabled() {
			continue
		}

		var lastErr error
		for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
			err := dispatcher.Dispatch(ctx, alert)
			if err == nil {
				p.logger.Info("alert dispatched successfully",
					zap.String("alert_id", alert.ID),
					zap.String("dispatcher", dispatcher.Name()),
				)
				break
			}

			lastErr = err
			p.logger.Warn("dispatch failed, retrying",
				zap.String("alert_id", alert.ID),
				zap.String("dispatcher", dispatcher.Name()),
				zap.Int("attempt", attempt+1),
				zap.Error(err),
			)

			if attempt < p.config.MaxRetries {
				delay := time.Duration(p.config.RetryDelaySeconds*(attempt+1)) * time.Second
				time.Sleep(delay)
			}
		}

		if lastErr != nil {
			p.logger.Error("dispatch failed after retries, sending to DLQ",
				zap.String("alert_id", alert.ID),
				zap.String("dispatcher", dispatcher.Name()),
				zap.Error(lastErr),
			)
			p.sendToDLQ(ctx, alert, dispatcher.Name(), lastErr)
		}
	}
}

func (p *AlertProcessor) sendToDLQ(ctx context.Context, alert *models.Alert, dispatcher string, dispatchErr error) {
	dlqEntry := map[string]interface{}{
		"alert":      alert,
		"dispatcher": dispatcher,
		"error":      dispatchErr.Error(),
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(dlqEntry)
	if err != nil {
		p.logger.Error("failed to marshal DLQ entry", zap.Error(err))
		return
	}

	if err := p.dlqProducer.Publish(ctx, []byte(alert.ID), data); err != nil {
		p.logger.Error("failed to send to DLQ",
			zap.String("alert_id", alert.ID),
			zap.Error(err),
		)
	}
}

func (p *AlertProcessor) suppressionCleanupLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.cleanupSuppressions()
		}
	}
}

func (p *AlertProcessor) cleanupSuppressions() {
	p.suppressMu.Lock()
	defer p.suppressMu.Unlock()

	now := time.Now()
	for key, expiry := range p.suppressions {
		if now.After(expiry) {
			delete(p.suppressions, key)
		}
	}
}

func compareSeverity(a, b models.AlertSeverity) int {
	order := map[models.AlertSeverity]int{
		models.AlertSeverityInfo:     0,
		models.AlertSeverityWarning:  1,
		models.AlertSeverityCritical: 2,
	}
	return order[a] - order[b]
}

// MockAlertProcessor for testing without Kafka.
type MockAlertProcessor struct {
	config      *ProcessorConfig
	dispatchers []ports.AlertDispatcher
	logger      *logging.Logger

	alertGroups map[string]*ports.AlertGroup
	groupMu     sync.Mutex

	suppressions map[string]time.Time
	suppressMu   sync.Mutex

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewMockAlertProcessor creates a new MockAlertProcessor.
func NewMockAlertProcessor(
	config *ProcessorConfig,
	dispatchers []ports.AlertDispatcher,
	logger *logging.Logger,
) *MockAlertProcessor {
	if config == nil {
		config = DefaultProcessorConfig()
	}

	return &MockAlertProcessor{
		config:       config,
		dispatchers:  dispatchers,
		logger:       logger,
		alertGroups:  make(map[string]*ports.AlertGroup),
		suppressions: make(map[string]time.Time),
	}
}

// Start starts the mock processor.
func (p *MockAlertProcessor) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = true
	p.stopCh = make(chan struct{})
	p.mu.Unlock()

	p.logger.Info("starting mock alert processor")

	// Start grouping flush loop
	p.wg.Add(1)
	go p.groupingFlushLoop(ctx)

	return nil
}

// Stop stops the mock processor.
func (p *MockAlertProcessor) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	close(p.stopCh)
	p.mu.Unlock()

	p.wg.Wait()

	p.logger.Info("mock alert processor stopped")
	return nil
}

// ProcessAlert processes a single alert directly.
func (p *MockAlertProcessor) ProcessAlert(ctx context.Context, alert *models.Alert) error {
	if p.isSuppressed(alert) {
		p.logger.Debug("alert suppressed",
			zap.String("alert_id", alert.ID),
		)
		return nil
	}

	p.addToGroup(alert)
	return nil
}

func (p *MockAlertProcessor) isSuppressed(alert *models.Alert) bool {
	key := fmt.Sprintf("%s:%s:%s", alert.ServiceName, alert.MetricType, alert.Severity)

	p.suppressMu.Lock()
	defer p.suppressMu.Unlock()

	if expiry, exists := p.suppressions[key]; exists {
		if time.Now().Before(expiry) {
			return true
		}
		delete(p.suppressions, key)
	}
	return false
}

func (p *MockAlertProcessor) addToGroup(alert *models.Alert) {
	groupKey := fmt.Sprintf("%s:%s", alert.ServiceName, alert.MetricType)

	p.groupMu.Lock()
	defer p.groupMu.Unlock()

	group, exists := p.alertGroups[groupKey]
	if !exists {
		group = &ports.AlertGroup{
			ID:          uuid.New().String(),
			GroupKey:    groupKey,
			Alerts:      make([]*models.Alert, 0),
			FirstSeen:   time.Now().Unix(),
			ServiceName: alert.ServiceName,
			Severity:    alert.Severity,
		}
		p.alertGroups[groupKey] = group
	}

	if len(group.Alerts) < p.config.MaxAlertsPerGroup {
		group.Alerts = append(group.Alerts, alert)
	}
	group.LastSeen = time.Now().Unix()
	group.Count++

	if compareSeverity(alert.Severity, group.Severity) > 0 {
		group.Severity = alert.Severity
	}
}

func (p *MockAlertProcessor) groupingFlushLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(time.Duration(p.config.GroupingWindowSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.flushGroups(ctx)
		}
	}
}

func (p *MockAlertProcessor) flushGroups(ctx context.Context) {
	p.groupMu.Lock()
	groups := make([]*ports.AlertGroup, 0, len(p.alertGroups))
	for _, group := range p.alertGroups {
		groups = append(groups, group)
	}
	p.alertGroups = make(map[string]*ports.AlertGroup)
	p.groupMu.Unlock()

	for _, group := range groups {
		if len(group.Alerts) == 0 {
			continue
		}

		firstAlert := group.Alerts[0]
		summaryAlert := &models.Alert{
			ID:           group.ID,
			ServiceName:  group.ServiceName,
			MetricType:   firstAlert.MetricType,
			Severity:     group.Severity,
			Title:        firstAlert.Title,
			Message:      firstAlert.Message,
			CurrentValue: firstAlert.CurrentValue,
			Threshold:    firstAlert.Threshold,
			Timestamp:    time.Now(),
		}

		if group.Count > 1 {
			summaryAlert.Title = fmt.Sprintf("%s (+%d more)", firstAlert.Title, group.Count-1)
		}

		for _, dispatcher := range p.dispatchers {
			if dispatcher.Enabled() {
				dispatcher.Dispatch(ctx, summaryAlert)
			}
		}

		// Add suppression
		key := fmt.Sprintf("%s:%s:%s", summaryAlert.ServiceName, summaryAlert.MetricType, summaryAlert.Severity)
		p.suppressMu.Lock()
		p.suppressions[key] = time.Now().Add(time.Duration(p.config.SuppressionWindowSeconds) * time.Second)
		p.suppressMu.Unlock()
	}
}
