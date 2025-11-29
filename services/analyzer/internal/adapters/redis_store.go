// Package adapters provides Redis implementations for the analyzer service.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/models"
	"github.com/microservices-platform/pkg/shared/utils"
	"github.com/microservices-platform/services/analyzer/internal/ports"
)

// RedisMetricsStore implements MetricsStore using Redis sorted sets.
type RedisMetricsStore struct {
	client *redis.Client
	logger *logging.Logger
}

// NewRedisMetricsStore creates a new RedisMetricsStore.
func NewRedisMetricsStore(client *redis.Client, logger *logging.Logger) ports.MetricsStore {
	return &RedisMetricsStore{
		client: client,
		logger: logger,
	}
}

// metricsKey generates a Redis key for metrics.
func (s *RedisMetricsStore) metricsKey(serviceName models.ServiceName, metricType models.MetricType) string {
	return fmt.Sprintf("metrics:%s:%s", serviceName, metricType)
}

// AddMetric adds a metric to the sliding window.
func (s *RedisMetricsStore) AddMetric(ctx context.Context, metric *models.ServiceMetric) error {
	key := s.metricsKey(metric.ServiceName, metric.MetricType)

	data, err := metric.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize metric: %w", err)
	}

	// Use timestamp as score for sorted set
	score := float64(metric.Timestamp.UnixNano())

	err = s.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to add metric to Redis: %w", err)
	}

	// Also store the latest metric separately for quick access
	latestKey := fmt.Sprintf("metrics:latest:%s:%s", metric.ServiceName, metric.MetricType)
	err = s.client.Set(ctx, latestKey, string(data), 10*time.Minute).Err()
	if err != nil {
		s.logger.Warn("failed to set latest metric", zap.Error(err))
	}

	return nil
}

// GetMetricsInWindow retrieves metrics within the sliding window.
func (s *RedisMetricsStore) GetMetricsInWindow(ctx context.Context, serviceName models.ServiceName, metricType models.MetricType, windowSize time.Duration) ([]*models.ServiceMetric, error) {
	key := s.metricsKey(serviceName, metricType)

	minTime := time.Now().Add(-windowSize).UnixNano()
	maxTime := time.Now().UnixNano()

	results, err := s.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: strconv.FormatInt(minTime, 10),
		Max: strconv.FormatInt(maxTime, 10),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics from Redis: %w", err)
	}

	metrics := make([]*models.ServiceMetric, 0, len(results))
	for _, data := range results {
		var metric models.ServiceMetric
		if err := json.Unmarshal([]byte(data), &metric); err != nil {
			s.logger.Warn("failed to deserialize metric", zap.Error(err))
			continue
		}
		metrics = append(metrics, &metric)
	}

	// Sort by timestamp
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Timestamp.Before(metrics[j].Timestamp)
	})

	return metrics, nil
}

// GetRollingAverage calculates the rolling average for a service and metric type.
func (s *RedisMetricsStore) GetRollingAverage(ctx context.Context, serviceName models.ServiceName, metricType models.MetricType, windowSize time.Duration) (float64, error) {
	metrics, err := s.GetMetricsInWindow(ctx, serviceName, metricType, windowSize)
	if err != nil {
		return 0, err
	}

	if len(metrics) == 0 {
		return 0, nil
	}

	var sum float64
	for _, m := range metrics {
		sum += m.Value
	}

	return sum / float64(len(metrics)), nil
}

// GetLatestMetric retrieves the most recent metric.
func (s *RedisMetricsStore) GetLatestMetric(ctx context.Context, serviceName models.ServiceName, metricType models.MetricType) (*models.ServiceMetric, error) {
	latestKey := fmt.Sprintf("metrics:latest:%s:%s", serviceName, metricType)

	data, err := s.client.Get(ctx, latestKey).Result()
	if err == redis.Nil {
		return nil, utils.ErrNotFound("metric")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest metric: %w", err)
	}

	var metric models.ServiceMetric
	if err := json.Unmarshal([]byte(data), &metric); err != nil {
		return nil, fmt.Errorf("failed to deserialize metric: %w", err)
	}

	return &metric, nil
}

// CleanupOldMetrics removes metrics older than the retention period.
func (s *RedisMetricsStore) CleanupOldMetrics(ctx context.Context, retentionPeriod time.Duration) error {
	// Get all metric keys
	keys, err := s.client.Keys(ctx, "metrics:*:*").Result()
	if err != nil {
		return fmt.Errorf("failed to get metric keys: %w", err)
	}

	cutoffTime := time.Now().Add(-retentionPeriod).UnixNano()

	for _, key := range keys {
		// Skip latest keys
		if len(key) > 15 && key[8:14] == "latest" {
			continue
		}

		err := s.client.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(cutoffTime, 10)).Err()
		if err != nil {
			s.logger.Warn("failed to cleanup old metrics",
				zap.String("key", key),
				zap.Error(err),
			)
		}
	}

	return nil
}

// GetMetricsWindow retrieves all metrics for a service within a time window.
func (s *RedisMetricsStore) GetMetricsWindow(ctx context.Context, serviceName models.ServiceName, windowSize time.Duration) ([]*models.ServiceMetric, error) {
	// Get metrics for all metric types for this service
	metricTypes := []models.MetricType{
		models.MetricTypeCPU,
		models.MetricTypeMemory,
		models.MetricTypeLatencyP95,
		models.MetricTypeLatencyP99,
		models.MetricTypeErrorRate,
		models.MetricTypeRequestRate,
	}

	var allMetrics []*models.ServiceMetric
	for _, metricType := range metricTypes {
		metrics, err := s.GetMetricsInWindow(ctx, serviceName, metricType, windowSize)
		if err != nil {
			s.logger.Warn("failed to get metrics for type",
				zap.String("metric_type", string(metricType)),
				zap.Error(err),
			)
			continue
		}
		allMetrics = append(allMetrics, metrics...)
	}

	// Also try the generic metrics key
	key := fmt.Sprintf("metrics:%s", serviceName)
	minTime := time.Now().Add(-windowSize).UnixNano()
	maxTime := time.Now().UnixNano()

	results, err := s.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: strconv.FormatInt(minTime, 10),
		Max: strconv.FormatInt(maxTime, 10),
	}).Result()
	if err == nil {
		for _, data := range results {
			var metric models.ServiceMetric
			if err := json.Unmarshal([]byte(data), &metric); err == nil {
				allMetrics = append(allMetrics, &metric)
			}
		}
	}

	// Sort by timestamp
	sort.Slice(allMetrics, func(i, j int) bool {
		return allMetrics[i].Timestamp.Before(allMetrics[j].Timestamp)
	})

	return allMetrics, nil
}

// CheckAndSetAlertSent checks if an alert was recently sent and marks it as sent.
func (s *RedisMetricsStore) CheckAndSetAlertSent(ctx context.Context, deduplicationKey string, ttl time.Duration) (bool, error) {
	key := fmt.Sprintf("alert:sent:%s", deduplicationKey)

	// Try to set the key only if it doesn't exist
	result, err := s.client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check/set alert sent: %w", err)
	}

	// If SetNX returns false, the key already existed (alert was already sent)
	return !result, nil
}

// CheckCooldown checks if a service/metric is in cooldown.
func (s *RedisMetricsStore) CheckCooldown(ctx context.Context, serviceName, metricType string) (bool, error) {
	key := fmt.Sprintf("cooldown:%s:%s", serviceName, metricType)

	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check cooldown: %w", err)
	}

	return exists > 0, nil
}

// SetCooldown sets a cooldown period for a service/metric.
func (s *RedisMetricsStore) SetCooldown(ctx context.Context, serviceName, metricType string, duration time.Duration) error {
	key := fmt.Sprintf("cooldown:%s:%s", serviceName, metricType)

	err := s.client.Set(ctx, key, "1", duration).Err()
	if err != nil {
		return fmt.Errorf("failed to set cooldown: %w", err)
	}

	return nil
}

// RedisRuleStore implements RuleStore using Redis.
type RedisRuleStore struct {
	client *redis.Client
	logger *logging.Logger
}

// NewRedisRuleStore creates a new RedisRuleStore.
func NewRedisRuleStore(client *redis.Client, logger *logging.Logger) ports.RuleStore {
	return &RedisRuleStore{
		client: client,
		logger: logger,
	}
}

const rulesKey = "analyzer:rules"

// GetRule retrieves a rule by ID.
func (s *RedisRuleStore) GetRule(ctx context.Context, id string) (*models.ThresholdRule, error) {
	data, err := s.client.HGet(ctx, rulesKey, id).Result()
	if err == redis.Nil {
		return nil, utils.ErrNotFound("rule")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	var rule models.ThresholdRule
	if err := json.Unmarshal([]byte(data), &rule); err != nil {
		return nil, fmt.Errorf("failed to deserialize rule: %w", err)
	}

	return &rule, nil
}

// GetAllRules retrieves all rules.
func (s *RedisRuleStore) GetAllRules(ctx context.Context) ([]*models.ThresholdRule, error) {
	data, err := s.client.HGetAll(ctx, rulesKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	rules := make([]*models.ThresholdRule, 0, len(data))
	for _, ruleData := range data {
		var rule models.ThresholdRule
		if err := json.Unmarshal([]byte(ruleData), &rule); err != nil {
			s.logger.Warn("failed to deserialize rule", zap.Error(err))
			continue
		}
		rules = append(rules, &rule)
	}

	return rules, nil
}

// GetEnabledRules retrieves all enabled rules.
func (s *RedisRuleStore) GetEnabledRules(ctx context.Context) ([]*models.ThresholdRule, error) {
	allRules, err := s.GetAllRules(ctx)
	if err != nil {
		return nil, err
	}

	enabledRules := make([]*models.ThresholdRule, 0)
	for _, rule := range allRules {
		if rule.Enabled {
			enabledRules = append(enabledRules, rule)
		}
	}

	return enabledRules, nil
}

// GetRulesForService retrieves all rules for a specific service.
func (s *RedisRuleStore) GetRulesForService(ctx context.Context, serviceName models.ServiceName) ([]*models.ThresholdRule, error) {
	allRules, err := s.GetAllRules(ctx)
	if err != nil {
		return nil, err
	}

	rules := make([]*models.ThresholdRule, 0)
	for _, rule := range allRules {
		if rule.ServiceName == serviceName {
			rules = append(rules, rule)
		}
	}

	return rules, nil
}

// CreateRule creates a new rule.
func (s *RedisRuleStore) CreateRule(ctx context.Context, rule *models.ThresholdRule) error {
	data, err := rule.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize rule: %w", err)
	}

	err = s.client.HSet(ctx, rulesKey, rule.ID, string(data)).Err()
	if err != nil {
		return fmt.Errorf("failed to create rule: %w", err)
	}

	return nil
}

// UpdateRule updates an existing rule.
func (s *RedisRuleStore) UpdateRule(ctx context.Context, rule *models.ThresholdRule) error {
	// Check if rule exists
	exists, err := s.client.HExists(ctx, rulesKey, rule.ID).Result()
	if err != nil {
		return fmt.Errorf("failed to check rule existence: %w", err)
	}
	if !exists {
		return utils.ErrNotFound("rule")
	}

	rule.UpdatedAt = time.Now().UTC()
	return s.CreateRule(ctx, rule)
}

// DeleteRule deletes a rule.
func (s *RedisRuleStore) DeleteRule(ctx context.Context, id string) error {
	result, err := s.client.HDel(ctx, rulesKey, id).Result()
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}
	if result == 0 {
		return utils.ErrNotFound("rule")
	}
	return nil
}

// RedisAlertStore implements AlertStore using Redis.
type RedisAlertStore struct {
	client *redis.Client
	logger *logging.Logger
}

// NewRedisAlertStore creates a new RedisAlertStore.
func NewRedisAlertStore(client *redis.Client, logger *logging.Logger) ports.AlertStore {
	return &RedisAlertStore{
		client: client,
		logger: logger,
	}
}

const alertsKey = "analyzer:alerts"

// RecordAlert records an alert for deduplication.
func (s *RedisAlertStore) RecordAlert(ctx context.Context, alert *models.Alert) error {
	data, err := alert.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize alert: %w", err)
	}

	score := float64(alert.Timestamp.UnixNano())
	err = s.client.ZAdd(ctx, alertsKey, redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to record alert: %w", err)
	}

	return nil
}

// IsAlertInCooldown checks if an alert is in cooldown period.
func (s *RedisAlertStore) IsAlertInCooldown(ctx context.Context, serviceName models.ServiceName, metricType models.MetricType, ruleID string) (bool, error) {
	key := fmt.Sprintf("alert:cooldown:%s:%s:%s", serviceName, metricType, ruleID)

	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check cooldown: %w", err)
	}

	return exists > 0, nil
}

// SetAlertCooldown sets a cooldown for an alert.
func (s *RedisAlertStore) SetAlertCooldown(ctx context.Context, serviceName models.ServiceName, metricType models.MetricType, ruleID string, duration time.Duration) error {
	key := fmt.Sprintf("alert:cooldown:%s:%s:%s", serviceName, metricType, ruleID)

	err := s.client.Set(ctx, key, "1", duration).Err()
	if err != nil {
		return fmt.Errorf("failed to set cooldown: %w", err)
	}

	return nil
}

// GetRecentAlerts retrieves recent alerts.
func (s *RedisAlertStore) GetRecentAlerts(ctx context.Context, limit int) ([]*models.Alert, error) {
	results, err := s.client.ZRevRange(ctx, alertsKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get recent alerts: %w", err)
	}

	alerts := make([]*models.Alert, 0, len(results))
	for _, data := range results {
		var alert models.Alert
		if err := json.Unmarshal([]byte(data), &alert); err != nil {
			s.logger.Warn("failed to deserialize alert", zap.Error(err))
			continue
		}
		alerts = append(alerts, &alert)
	}

	return alerts, nil
}
