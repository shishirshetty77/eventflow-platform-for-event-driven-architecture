// Package kafka provides Kafka producer and consumer utilities with retries and backoff.
// It wraps segmentio/kafka-go to provide consistent Kafka operations across services.
package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/microservices-platform/pkg/shared/logging"
)

// Topic names for the microservices platform.
const (
	TopicServiceMetrics = "service-metrics"
	TopicServiceLogs    = "service-logs"
	TopicAlerts         = "alerts"
)

// ProducerConfig holds Kafka producer configuration.
type ProducerConfig struct {
	Brokers       []string      `json:"brokers"`
	Topic         string        `json:"topic"`
	BatchSize     int           `json:"batch_size"`
	BatchTimeout  time.Duration `json:"batch_timeout"`
	MaxRetries    int           `json:"max_retries"`
	RetryBackoff  time.Duration `json:"retry_backoff"`
	RequiredAcks  int           `json:"required_acks"`
	Async         bool          `json:"async"`
	TLS           *tls.Config   `json:"-"`
	SASLMechanism string        `json:"sasl_mechanism"`
	SASLUsername  string        `json:"sasl_username"`
	SASLPassword  string        `json:"sasl_password"`
}

// DefaultProducerConfig returns default producer configuration.
func DefaultProducerConfig(brokers []string, topic string) *ProducerConfig {
	return &ProducerConfig{
		Brokers:      brokers,
		Topic:        topic,
		BatchSize:    100,
		BatchTimeout: 1 * time.Second,
		MaxRetries:   5,
		RetryBackoff: 100 * time.Millisecond,
		RequiredAcks: 1, // Leader ack
		Async:        false,
	}
}

// Producer wraps kafka.Writer with retry logic and observability.
type Producer struct {
	writer *kafka.Writer
	config *ProducerConfig
	logger *logging.Logger
	mu     sync.RWMutex
	closed bool
}

// NewProducer creates a new Kafka producer with the given configuration.
func NewProducer(cfg *ProducerConfig, logger *logging.Logger) (*Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("at least one broker is required")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		RequiredAcks: kafka.RequiredAcks(cfg.RequiredAcks),
		Async:        cfg.Async,
		Compression:  kafka.Snappy,
	}

	if cfg.TLS != nil {
		writer.Transport = &kafka.Transport{
			TLS: cfg.TLS,
		}
	}

	return &Producer{
		writer: writer,
		config: cfg,
		logger: logger,
	}, nil
}

// Publish publishes a message to Kafka with retry logic and exponential backoff.
func (p *Producer) Publish(ctx context.Context, key, value []byte) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("producer is closed")
	}
	p.mu.RUnlock()

	msg := kafka.Message{
		Key:   key,
		Value: value,
		Time:  time.Now(),
	}

	return p.publishWithRetry(ctx, msg)
}

// PublishBatch publishes multiple messages to Kafka with retry logic.
func (p *Producer) PublishBatch(ctx context.Context, messages []kafka.Message) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("producer is closed")
	}
	p.mu.RUnlock()

	return p.publishBatchWithRetry(ctx, messages)
}

// publishWithRetry implements retry logic with exponential backoff.
func (p *Producer) publishWithRetry(ctx context.Context, msg kafka.Message) error {
	var lastErr error

	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := p.calculateBackoff(attempt)
			p.logger.Debug("retrying kafka publish",
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff),
				zap.String("topic", p.config.Topic),
			)

			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}

		err := p.writer.WriteMessages(ctx, msg)
		if err == nil {
			if attempt > 0 {
				p.logger.Info("kafka publish succeeded after retry",
					zap.Int("attempts", attempt+1),
					zap.String("topic", p.config.Topic),
				)
			}
			return nil
		}

		lastErr = err
		p.logger.Warn("kafka publish failed",
			zap.Error(err),
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", p.config.MaxRetries),
			zap.String("topic", p.config.Topic),
		)
	}

	return fmt.Errorf("failed to publish message after %d attempts: %w", p.config.MaxRetries+1, lastErr)
}

// publishBatchWithRetry implements retry logic for batch publishing.
func (p *Producer) publishBatchWithRetry(ctx context.Context, messages []kafka.Message) error {
	var lastErr error

	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := p.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}

		err := p.writer.WriteMessages(ctx, messages...)
		if err == nil {
			return nil
		}

		lastErr = err
		p.logger.Warn("kafka batch publish failed",
			zap.Error(err),
			zap.Int("attempt", attempt+1),
			zap.Int("message_count", len(messages)),
		)
	}

	return fmt.Errorf("failed to publish batch after %d attempts: %w", p.config.MaxRetries+1, lastErr)
}

// calculateBackoff calculates exponential backoff duration.
func (p *Producer) calculateBackoff(attempt int) time.Duration {
	backoff := float64(p.config.RetryBackoff) * math.Pow(2, float64(attempt-1))
	maxBackoff := float64(30 * time.Second)
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	return time.Duration(backoff)
}

// Close closes the producer gracefully.
func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	return p.writer.Close()
}

// Stats returns the current producer statistics.
func (p *Producer) Stats() kafka.WriterStats {
	return p.writer.Stats()
}

// ConsumerConfig holds Kafka consumer configuration.
type ConsumerConfig struct {
	Brokers        []string      `json:"brokers"`
	Topic          string        `json:"topic"`
	GroupID        string        `json:"group_id"`
	MinBytes       int           `json:"min_bytes"`
	MaxBytes       int           `json:"max_bytes"`
	MaxWait        time.Duration `json:"max_wait"`
	StartOffset    int64         `json:"start_offset"`
	CommitInterval time.Duration `json:"commit_interval"`
	TLS            *tls.Config   `json:"-"`
}

// DefaultConsumerConfig returns default consumer configuration.
func DefaultConsumerConfig(brokers []string, topic, groupID string) *ConsumerConfig {
	return &ConsumerConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		MaxWait:        3 * time.Second,
		StartOffset:    kafka.LastOffset,
		CommitInterval: 1 * time.Second,
	}
}

// Consumer wraps kafka.Reader with observability.
type Consumer struct {
	reader *kafka.Reader
	config *ConsumerConfig
	logger *logging.Logger
	mu     sync.RWMutex
	closed bool
}

// NewConsumer creates a new Kafka consumer with the given configuration.
func NewConsumer(cfg *ConsumerConfig, logger *logging.Logger) (*Consumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("at least one broker is required")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("topic is required")
	}
	if cfg.GroupID == "" {
		return nil, fmt.Errorf("group ID is required")
	}

	readerConfig := kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MinBytes:       cfg.MinBytes,
		MaxBytes:       cfg.MaxBytes,
		MaxWait:        cfg.MaxWait,
		StartOffset:    cfg.StartOffset,
		CommitInterval: cfg.CommitInterval,
	}

	reader := kafka.NewReader(readerConfig)

	return &Consumer{
		reader: reader,
		config: cfg,
		logger: logger,
	}, nil
}

// ReadMessage reads a single message from Kafka.
func (c *Consumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return kafka.Message{}, fmt.Errorf("consumer is closed")
	}
	c.mu.RUnlock()

	return c.reader.ReadMessage(ctx)
}

// FetchMessage fetches a message without committing.
func (c *Consumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return kafka.Message{}, fmt.Errorf("consumer is closed")
	}
	c.mu.RUnlock()

	return c.reader.FetchMessage(ctx)
}

// CommitMessages commits the given messages.
func (c *Consumer) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	return c.reader.CommitMessages(ctx, msgs...)
}

// Close closes the consumer gracefully.
func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	return c.reader.Close()
}

// Stats returns the current consumer statistics.
func (c *Consumer) Stats() kafka.ReaderStats {
	return c.reader.Stats()
}

// MessageHandler is a function type for processing Kafka messages.
type MessageHandler func(ctx context.Context, msg kafka.Message) error

// ConsumeLoop starts a continuous consumption loop.
func (c *Consumer) ConsumeLoop(ctx context.Context, handler MessageHandler) error {
	c.logger.Info("starting kafka consumer loop",
		zap.String("topic", c.config.Topic),
		zap.String("group_id", c.config.GroupID),
	)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("kafka consumer loop stopped", zap.String("topic", c.config.Topic))
			return ctx.Err()
		default:
			msg, err := c.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				c.logger.Error("failed to fetch kafka message",
					zap.Error(err),
					zap.String("topic", c.config.Topic),
				)
				continue
			}

			if err := handler(ctx, msg); err != nil {
				c.logger.Error("failed to handle kafka message",
					zap.Error(err),
					zap.String("topic", c.config.Topic),
					zap.Int64("offset", msg.Offset),
				)
				// Don't commit on error - message will be reprocessed
				continue
			}

			if err := c.CommitMessages(ctx, msg); err != nil {
				c.logger.Error("failed to commit kafka message",
					zap.Error(err),
					zap.String("topic", c.config.Topic),
					zap.Int64("offset", msg.Offset),
				)
			}
		}
	}
}
