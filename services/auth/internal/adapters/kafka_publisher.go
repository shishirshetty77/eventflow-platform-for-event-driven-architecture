// Package adapters provides implementations of ports interfaces.
package adapters

import (
	"context"

	"go.uber.org/zap"

	"github.com/microservices-platform/pkg/shared/kafka"
	"github.com/microservices-platform/pkg/shared/logging"
	sharedmetrics "github.com/microservices-platform/pkg/shared/metrics"
	"github.com/microservices-platform/pkg/shared/models"
	"github.com/microservices-platform/services/auth/internal/ports"
)

// KafkaMetricsPublisher publishes metrics and logs to Kafka.
type KafkaMetricsPublisher struct {
	metricsProducer *kafka.Producer
	logsProducer    *kafka.Producer
	logger          *logging.Logger
	metrics         *sharedmetrics.Metrics
}

// NewKafkaMetricsPublisher creates a new KafkaMetricsPublisher.
func NewKafkaMetricsPublisher(
	brokers []string,
	metricsTopic, logsTopic string,
	logger *logging.Logger,
	metrics *sharedmetrics.Metrics,
) (ports.MetricsPublisher, error) {
	// Create metrics producer
	metricsConfig := kafka.DefaultProducerConfig(brokers, metricsTopic)
	metricsProducer, err := kafka.NewProducer(metricsConfig, logger)
	if err != nil {
		return nil, err
	}

	// Create logs producer
	logsConfig := kafka.DefaultProducerConfig(brokers, logsTopic)
	logsProducer, err := kafka.NewProducer(logsConfig, logger)
	if err != nil {
		metricsProducer.Close()
		return nil, err
	}

	return &KafkaMetricsPublisher{
		metricsProducer: metricsProducer,
		logsProducer:    logsProducer,
		logger:          logger,
		metrics:         metrics,
	}, nil
}

// PublishMetric publishes a metric to Kafka.
func (p *KafkaMetricsPublisher) PublishMetric(ctx context.Context, metric *models.ServiceMetric) error {
	timer := sharedmetrics.NewTimer()

	// Validate metric
	if err := models.Validate(metric); err != nil {
		p.logger.Warn("invalid metric",
			zap.String("metric_id", metric.ID),
			zap.Error(err),
		)
		return err
	}

	// Serialize metric
	data, err := metric.ToJSON()
	if err != nil {
		p.logger.Error("failed to serialize metric",
			zap.String("metric_id", metric.ID),
			zap.Error(err),
		)
		return err
	}

	// Publish to Kafka
	err = p.metricsProducer.Publish(ctx, []byte(metric.ID), data)

	// Record metrics
	if p.metrics != nil {
		p.metrics.RecordKafkaPublish(kafka.TopicServiceMetrics, timer.Elapsed(), err)
	}

	if err != nil {
		p.logger.Error("failed to publish metric to kafka",
			zap.String("metric_id", metric.ID),
			zap.Error(err),
		)
		return err
	}

	p.logger.Debug("metric published to kafka",
		zap.String("metric_id", metric.ID),
		zap.String("metric_type", string(metric.MetricType)),
		zap.Float64("value", metric.Value),
	)

	return nil
}

// PublishLog publishes a log entry to Kafka.
func (p *KafkaMetricsPublisher) PublishLog(ctx context.Context, log *models.ServiceLog) error {
	timer := sharedmetrics.NewTimer()

	// Validate log
	if err := models.Validate(log); err != nil {
		p.logger.Warn("invalid log",
			zap.String("log_id", log.ID),
			zap.Error(err),
		)
		return err
	}

	// Serialize log
	data, err := log.ToJSON()
	if err != nil {
		p.logger.Error("failed to serialize log",
			zap.String("log_id", log.ID),
			zap.Error(err),
		)
		return err
	}

	// Publish to Kafka
	err = p.logsProducer.Publish(ctx, []byte(log.ID), data)

	// Record metrics
	if p.metrics != nil {
		p.metrics.RecordKafkaPublish(kafka.TopicServiceLogs, timer.Elapsed(), err)
	}

	if err != nil {
		p.logger.Error("failed to publish log to kafka",
			zap.String("log_id", log.ID),
			zap.Error(err),
		)
		return err
	}

	p.logger.Debug("log published to kafka",
		zap.String("log_id", log.ID),
		zap.String("level", string(log.Level)),
	)

	return nil
}

// Close closes all Kafka producers.
func (p *KafkaMetricsPublisher) Close() error {
	var lastErr error

	if err := p.metricsProducer.Close(); err != nil {
		p.logger.Error("failed to close metrics producer", zap.Error(err))
		lastErr = err
	}

	if err := p.logsProducer.Close(); err != nil {
		p.logger.Error("failed to close logs producer", zap.Error(err))
		lastErr = err
	}

	return lastErr
}

// MockMetricsPublisher is a mock implementation for testing without Kafka.
type MockMetricsPublisher struct {
	logger  *logging.Logger
	metrics []*models.ServiceMetric
	logs    []*models.ServiceLog
}

// NewMockMetricsPublisher creates a new MockMetricsPublisher.
func NewMockMetricsPublisher(logger *logging.Logger) ports.MetricsPublisher {
	return &MockMetricsPublisher{
		logger:  logger,
		metrics: make([]*models.ServiceMetric, 0),
		logs:    make([]*models.ServiceLog, 0),
	}
}

// PublishMetric stores the metric in memory.
func (p *MockMetricsPublisher) PublishMetric(ctx context.Context, metric *models.ServiceMetric) error {
	p.metrics = append(p.metrics, metric)
	p.logger.Info("mock: metric published",
		zap.String("metric_type", string(metric.MetricType)),
		zap.Float64("value", metric.Value),
		zap.String("unit", metric.Unit),
		zap.Time("timestamp", metric.Timestamp),
	)
	return nil
}

// PublishLog stores the log in memory.
func (p *MockMetricsPublisher) PublishLog(ctx context.Context, log *models.ServiceLog) error {
	p.logs = append(p.logs, log)
	p.logger.Info("mock: log published",
		zap.String("level", string(log.Level)),
		zap.String("message", log.Message),
		zap.Time("timestamp", log.Timestamp),
	)
	return nil
}

// Close is a no-op for the mock publisher.
func (p *MockMetricsPublisher) Close() error {
	return nil
}

// GetMetrics returns all stored metrics.
func (p *MockMetricsPublisher) GetMetrics() []*models.ServiceMetric {
	return p.metrics
}

// GetLogs returns all stored logs.
func (p *MockMetricsPublisher) GetLogs() []*models.ServiceLog {
	return p.logs
}
