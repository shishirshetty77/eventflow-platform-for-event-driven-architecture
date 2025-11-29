// Package store provides Redis storage for the UI backend.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/models"
)

// RedisStore provides Redis-based storage for the UI backend.
type RedisStore struct {
	client *redis.Client
	logger *logging.Logger
}

// NewRedisStore creates a new RedisStore.
func NewRedisStore(addr, password string, db int, logger *logging.Logger) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{
		client: client,
		logger: logger,
	}, nil
}

// Close closes the Redis connection.
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Ping checks Redis connectivity.
func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// GetMetrics returns metrics for a service within a time window.
func (s *RedisStore) GetMetrics(ctx context.Context, service models.ServiceName, window time.Duration) ([]*models.ServiceMetric, error) {
	key := fmt.Sprintf("metrics:%s", service)
	now := time.Now()
	minScore := float64(now.Add(-window).UnixNano())

	results, err := s.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", minScore),
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, err
	}

	metrics := make([]*models.ServiceMetric, 0, len(results))
	for _, r := range results {
		var metric models.ServiceMetric
		if err := json.Unmarshal([]byte(r), &metric); err != nil {
			continue
		}
		metrics = append(metrics, &metric)
	}

	return metrics, nil
}

// GetLatestMetric returns the latest aggregated metric for a service.
func (s *RedisStore) GetLatestMetric(ctx context.Context, service models.ServiceName) (*models.ServiceMetric, error) {
	// The analyzer stores metrics by type, so we need to aggregate them
	metricTypes := []models.MetricType{
		models.MetricTypeCPU,
		models.MetricTypeMemory,
		models.MetricTypeLatency,
		models.MetricTypeError,
	}

	result := &models.ServiceMetric{
		ServiceName: service,
		Timestamp:   time.Now(),
	}

	foundAny := false

	for _, metricType := range metricTypes {
		// Try the latest key first (set by analyzer)
		latestKey := fmt.Sprintf("metrics:latest:%s:%s", service, metricType)
		data, err := s.client.Get(ctx, latestKey).Result()
		if err == nil && data != "" {
			var metric models.ServiceMetric
			if err := json.Unmarshal([]byte(data), &metric); err == nil {
				foundAny = true
				switch metricType {
				case models.MetricTypeCPU:
					result.CPUUsage = metric.Value
				case models.MetricTypeMemory:
					result.MemoryUsage = metric.Value
				case models.MetricTypeLatency:
					result.LatencyP95 = metric.Value
				case models.MetricTypeError:
					result.ErrorRate = metric.Value
				}
				if metric.Timestamp.After(result.Timestamp) || result.Timestamp.IsZero() {
					result.Timestamp = metric.Timestamp
				}
			}
			continue
		}

		// Fallback to sorted set
		sortedKey := fmt.Sprintf("metrics:%s:%s", service, metricType)
		results, err := s.client.ZRevRange(ctx, sortedKey, 0, 0).Result()
		if err == nil && len(results) > 0 {
			var metric models.ServiceMetric
			if err := json.Unmarshal([]byte(results[0]), &metric); err == nil {
				foundAny = true
				switch metricType {
				case models.MetricTypeCPU:
					result.CPUUsage = metric.Value
				case models.MetricTypeMemory:
					result.MemoryUsage = metric.Value
				case models.MetricTypeLatency:
					result.LatencyP95 = metric.Value
				case models.MetricTypeError:
					result.ErrorRate = metric.Value
				}
				if metric.Timestamp.After(result.Timestamp) || result.Timestamp.IsZero() {
					result.Timestamp = metric.Timestamp
				}
			}
		}
	}

	if !foundAny {
		return nil, nil
	}

	return result, nil
}

// StoreMetric stores a metric.
func (s *RedisStore) StoreMetric(ctx context.Context, metric *models.ServiceMetric) error {
	key := fmt.Sprintf("metrics:%s", metric.ServiceName)

	data, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	score := float64(metric.Timestamp.UnixNano())

	if err := s.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: string(data),
	}).Err(); err != nil {
		return err
	}

	cutoff := float64(time.Now().Add(-24 * time.Hour).UnixNano())
	s.client.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%f", cutoff))

	return nil
}

// GetAlerts returns alerts with pagination.
func (s *RedisStore) GetAlerts(ctx context.Context, serviceName, severity string, page, limit int) ([]*models.Alert, int, error) {
	key := "alerts"

	results, err := s.client.ZRevRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, 0, err
	}

	alerts := make([]*models.Alert, 0)
	for _, r := range results {
		var alert models.Alert
		if err := json.Unmarshal([]byte(r), &alert); err != nil {
			continue
		}

		if serviceName != "" && string(alert.ServiceName) != serviceName {
			continue
		}
		if severity != "" && string(alert.Severity) != severity {
			continue
		}

		alerts = append(alerts, &alert)
	}

	total := len(alerts)

	start := (page - 1) * limit
	if start >= len(alerts) {
		return []*models.Alert{}, total, nil
	}

	end := start + limit
	if end > len(alerts) {
		end = len(alerts)
	}

	return alerts[start:end], total, nil
}

// GetAlert returns a single alert by ID.
func (s *RedisStore) GetAlert(ctx context.Context, alertID string) (*models.Alert, error) {
	key := fmt.Sprintf("alert:%s", alertID)

	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var alert models.Alert
	if err := json.Unmarshal([]byte(data), &alert); err != nil {
		return nil, err
	}

	return &alert, nil
}

// StoreAlert stores an alert.
func (s *RedisStore) StoreAlert(ctx context.Context, alert *models.Alert) error {
	data, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("alert:%s", alert.ID)
	if err := s.client.Set(ctx, key, data, 7*24*time.Hour).Err(); err != nil {
		return err
	}

	listKey := "alerts"
	score := float64(alert.Timestamp.Unix())
	return s.client.ZAdd(ctx, listKey, redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
}

// AcknowledgeAlert acknowledges an alert.
func (s *RedisStore) AcknowledgeAlert(ctx context.Context, alertID, userID string) error {
	key := fmt.Sprintf("alert:%s", alertID)

	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	var alert models.Alert
	if err := json.Unmarshal([]byte(data), &alert); err != nil {
		return err
	}

	alert.Acknowledged = true
	alert.AcknowledgedBy = userID
	now := time.Now()
	alert.AcknowledgedAt = &now

	newData, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, newData, 7*24*time.Hour).Err()
}

// GetRules returns all threshold rules.
func (s *RedisStore) GetRules(ctx context.Context) ([]*models.ThresholdRule, error) {
	key := "rules"

	results, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	rules := make([]*models.ThresholdRule, 0, len(results))
	for _, v := range results {
		var rule models.ThresholdRule
		if err := json.Unmarshal([]byte(v), &rule); err != nil {
			continue
		}
		rules = append(rules, &rule)
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].CreatedAt.After(rules[j].CreatedAt)
	})

	return rules, nil
}

// GetRule returns a single rule by ID.
func (s *RedisStore) GetRule(ctx context.Context, ruleID string) (*models.ThresholdRule, error) {
	key := "rules"

	data, err := s.client.HGet(ctx, key, ruleID).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var rule models.ThresholdRule
	if err := json.Unmarshal([]byte(data), &rule); err != nil {
		return nil, err
	}

	return &rule, nil
}

// CreateRule creates a new threshold rule.
func (s *RedisStore) CreateRule(ctx context.Context, rule *models.ThresholdRule) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	data, err := json.Marshal(rule)
	if err != nil {
		return err
	}

	return s.client.HSet(ctx, "rules", rule.ID, string(data)).Err()
}

// UpdateRule updates a threshold rule.
func (s *RedisStore) UpdateRule(ctx context.Context, rule *models.ThresholdRule) error {
	data, err := json.Marshal(rule)
	if err != nil {
		return err
	}

	return s.client.HSet(ctx, "rules", rule.ID, string(data)).Err()
}

// DeleteRule deletes a threshold rule.
func (s *RedisStore) DeleteRule(ctx context.Context, ruleID string) error {
	return s.client.HDel(ctx, "rules", ruleID).Err()
}

// DashboardStats represents dashboard statistics.
type DashboardStats struct {
	TotalServices   int                      `json:"total_services"`
	HealthyServices int                      `json:"healthy_services"`
	TotalAlerts     int                      `json:"total_alerts"`
	CriticalAlerts  int                      `json:"critical_alerts"`
	WarningAlerts   int                      `json:"warning_alerts"`
	ActiveRules     int                      `json:"active_rules"`
	ServiceStats    map[string]*ServiceStats `json:"service_stats"`
	RecentAlerts    []*models.Alert          `json:"recent_alerts"`
}

// ServiceStats represents statistics for a service.
type ServiceStats struct {
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	LatencyP95  float64 `json:"latency_p95"`
	ErrorRate   float64 `json:"error_rate"`
	LastUpdate  string  `json:"last_update"`
}

// GetDashboardStats returns dashboard statistics.
func (s *RedisStore) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	stats := &DashboardStats{
		TotalServices:   4,
		HealthyServices: 4,
		ServiceStats:    make(map[string]*ServiceStats),
	}

	services := []models.ServiceName{
		models.ServiceNameAuth,
		models.ServiceNameOrders,
		models.ServiceNamePayments,
		models.ServiceNameNotification,
	}

	for _, service := range services {
		metric, err := s.GetLatestMetric(ctx, service)
		if err != nil {
			continue
		}

		status := "healthy"
		if metric != nil {
			if metric.CPUUsage > 80 || metric.MemoryUsage > 80 || metric.ErrorRate > 5 {
				status = "warning"
				stats.HealthyServices--
			}

			stats.ServiceStats[string(service)] = &ServiceStats{
				Name:        string(service),
				Status:      status,
				CPUUsage:    metric.CPUUsage,
				MemoryUsage: metric.MemoryUsage,
				LatencyP95:  metric.LatencyP95,
				ErrorRate:   metric.ErrorRate,
				LastUpdate:  metric.Timestamp.Format(time.RFC3339),
			}
		}
	}

	alerts, total, _ := s.GetAlerts(ctx, "", "", 1, 1000)
	stats.TotalAlerts = total

	for _, alert := range alerts {
		switch alert.Severity {
		case models.AlertSeverityCritical:
			stats.CriticalAlerts++
		case models.AlertSeverityWarning:
			stats.WarningAlerts++
		}
	}

	recentAlerts, _, _ := s.GetAlerts(ctx, "", "", 1, 5)
	stats.RecentAlerts = recentAlerts

	rules, _ := s.GetRules(ctx)
	for _, rule := range rules {
		if rule.Enabled {
			stats.ActiveRules++
		}
	}

	return stats, nil
}
