// Package adapters provides Kafka implementations for the analyzer service.
package adapters

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	sharedkafka "github.com/microservices-platform/pkg/shared/kafka"
	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/metrics"
	"github.com/microservices-platform/pkg/shared/models"
	"github.com/microservices-platform/services/analyzer/internal/ports"
)

// KafkaMetricsConsumer consumes metrics from Kafka.
type KafkaMetricsConsumer struct {
	metricsConsumer *sharedkafka.Consumer
	logsConsumer    *sharedkafka.Consumer
	metricsStore    ports.MetricsStore
	logger          *logging.Logger
	metrics         *metrics.Metrics

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewKafkaMetricsConsumer creates a new KafkaMetricsConsumer.
func NewKafkaMetricsConsumer(
	brokers []string,
	metricsTopic, logsTopic, consumerGroup string,
	metricsStore ports.MetricsStore,
	logger *logging.Logger,
	m *metrics.Metrics,
) (*KafkaMetricsConsumer, error) {
	metricsConfig := sharedkafka.DefaultConsumerConfig(brokers, metricsTopic, consumerGroup+"-metrics")
	metricsConfig.StartOffset = kafka.LastOffset
	metricsConsumer, err := sharedkafka.NewConsumer(metricsConfig, logger)
	if err != nil {
		return nil, err
	}

	logsConfig := sharedkafka.DefaultConsumerConfig(brokers, logsTopic, consumerGroup+"-logs")
	logsConfig.StartOffset = kafka.LastOffset
	logsConsumer, err := sharedkafka.NewConsumer(logsConfig, logger)
	if err != nil {
		metricsConsumer.Close()
		return nil, err
	}

	return &KafkaMetricsConsumer{
		metricsConsumer: metricsConsumer,
		logsConsumer:    logsConsumer,
		metricsStore:    metricsStore,
		logger:          logger,
		metrics:         m,
	}, nil
}

// Start starts consuming metrics.
func (c *KafkaMetricsConsumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = true
	c.stopCh = make(chan struct{})
	c.mu.Unlock()

	c.logger.Info("starting Kafka metrics consumer")

	// Start metrics consumer
	c.wg.Add(1)
	go c.consumeMetrics(ctx)

	// Start logs consumer
	c.wg.Add(1)
	go c.consumeLogs(ctx)

	return nil
}

// Stop stops consuming metrics.
func (c *KafkaMetricsConsumer) Stop() error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = false
	close(c.stopCh)
	c.mu.Unlock()

	c.wg.Wait()

	if err := c.metricsConsumer.Close(); err != nil {
		c.logger.Error("failed to close metrics consumer", zap.Error(err))
	}
	if err := c.logsConsumer.Close(); err != nil {
		c.logger.Error("failed to close logs consumer", zap.Error(err))
	}

	c.logger.Info("Kafka metrics consumer stopped")
	return nil
}

func (c *KafkaMetricsConsumer) consumeMetrics(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		default:
			msg, err := c.metricsConsumer.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Error("failed to fetch metrics message", zap.Error(err))
				time.Sleep(100 * time.Millisecond)
				continue
			}

			var metric models.ServiceMetric
			if err := json.Unmarshal(msg.Value, &metric); err != nil {
				c.logger.Warn("failed to deserialize metric",
					zap.Error(err),
					zap.String("value", string(msg.Value)),
				)
				c.metricsConsumer.CommitMessages(ctx, msg)
				continue
			}

			if err := c.metricsStore.AddMetric(ctx, &metric); err != nil {
				c.logger.Error("failed to store metric",
					zap.Error(err),
					zap.String("metric_id", metric.ID),
				)
			}

			if c.metrics != nil {
				c.metrics.RecordKafkaConsume(sharedkafka.TopicServiceMetrics)
			}

			c.metricsConsumer.CommitMessages(ctx, msg)
		}
	}
}

func (c *KafkaMetricsConsumer) consumeLogs(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		default:
			msg, err := c.logsConsumer.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Error("failed to fetch logs message", zap.Error(err))
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// For now, just acknowledge logs - they could be stored or analyzed
			if c.metrics != nil {
				c.metrics.RecordKafkaConsume(sharedkafka.TopicServiceLogs)
			}

			c.logsConsumer.CommitMessages(ctx, msg)
		}
	}
}

// KafkaAlertPublisher publishes alerts to Kafka.
type KafkaAlertPublisher struct {
	producer *sharedkafka.Producer
	logger   *logging.Logger
	metrics  *metrics.Metrics
}

// NewKafkaAlertPublisher creates a new KafkaAlertPublisher.
func NewKafkaAlertPublisher(
	brokers []string,
	alertsTopic string,
	logger *logging.Logger,
	m *metrics.Metrics,
) (*KafkaAlertPublisher, error) {
	config := sharedkafka.DefaultProducerConfig(brokers, alertsTopic)
	producer, err := sharedkafka.NewProducer(config, logger)
	if err != nil {
		return nil, err
	}

	return &KafkaAlertPublisher{
		producer: producer,
		logger:   logger,
		metrics:  m,
	}, nil
}

// PublishAlert publishes an alert to Kafka.
func (p *KafkaAlertPublisher) PublishAlert(ctx context.Context, alert *models.Alert) error {
	timer := metrics.NewTimer()

	data, err := alert.ToJSON()
	if err != nil {
		return err
	}

	err = p.producer.Publish(ctx, []byte(alert.ID), data)
	if p.metrics != nil {
		p.metrics.RecordKafkaPublish(sharedkafka.TopicAlerts, timer.Elapsed(), err)
	}

	if err != nil {
		p.logger.Error("failed to publish alert",
			zap.String("alert_id", alert.ID),
			zap.Error(err),
		)
		return err
	}

	p.logger.Info("alert published",
		zap.String("alert_id", alert.ID),
		zap.String("service", string(alert.ServiceName)),
		zap.String("severity", string(alert.Severity)),
	)

	return nil
}

// Close closes the publisher.
func (p *KafkaAlertPublisher) Close() error {
	return p.producer.Close()
}

// MockAlertPublisher is a mock implementation for testing.
type MockAlertPublisher struct {
	logger *logging.Logger
	alerts []*models.Alert
	mu     sync.Mutex
}

// NewMockAlertPublisher creates a new MockAlertPublisher.
func NewMockAlertPublisher(logger *logging.Logger) *MockAlertPublisher {
	return &MockAlertPublisher{
		logger: logger,
		alerts: make([]*models.Alert, 0),
	}
}

// PublishAlert stores the alert in memory.
func (p *MockAlertPublisher) PublishAlert(ctx context.Context, alert *models.Alert) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.alerts = append(p.alerts, alert)
	p.logger.Info("mock: alert published",
		zap.String("alert_id", alert.ID),
		zap.String("service", string(alert.ServiceName)),
		zap.String("severity", string(alert.Severity)),
		zap.String("title", alert.Title),
	)
	return nil
}

// Close is a no-op.
func (p *MockAlertPublisher) Close() error {
	return nil
}

// GetAlerts returns all stored alerts.
func (p *MockAlertPublisher) GetAlerts() []*models.Alert {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.alerts
}
