#!/bin/bash
# =============================================================================
# EventFlow Platform - Redis Seed Script
# Seeds initial threshold rules into Redis
# =============================================================================

set -e

REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_DB="${REDIS_DB:-2}"

echo "=========================================="
echo "EventFlow Platform - Redis Seed Script"
echo "=========================================="
echo "Redis Host: $REDIS_HOST:$REDIS_PORT"
echo "Redis DB: $REDIS_DB"
echo ""

# Function to execute Redis commands
redis_cmd() {
    redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -n "$REDIS_DB" "$@"
}

# Check Redis connection
echo "Checking Redis connection..."
if ! redis_cmd PING > /dev/null 2>&1; then
    echo "ERROR: Cannot connect to Redis at $REDIS_HOST:$REDIS_PORT"
    exit 1
fi
echo "✓ Redis connection successful"
echo ""

# Seed threshold rules
echo "Seeding threshold rules..."

# Rule 1: High CPU Usage - Critical
RULE_1_ID="rule-cpu-critical"
RULE_1=$(cat <<EOF
{
  "id": "$RULE_1_ID",
  "name": "High CPU Usage - Critical",
  "description": "Alert when CPU usage exceeds 90% for any service",
  "service_name": "",
  "metric_type": "cpu",
  "threshold": 90,
  "operator": "gt",
  "severity": "critical",
  "enabled": true,
  "cooldown_seconds": 300,
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
)
redis_cmd HSET rules "$RULE_1_ID" "$RULE_1" > /dev/null
echo "  ✓ Created rule: High CPU Usage - Critical"

# Rule 2: High CPU Usage - Warning
RULE_2_ID="rule-cpu-warning"
RULE_2=$(cat <<EOF
{
  "id": "$RULE_2_ID",
  "name": "High CPU Usage - Warning",
  "description": "Alert when CPU usage exceeds 75% for any service",
  "service_name": "",
  "metric_type": "cpu",
  "threshold": 75,
  "operator": "gt",
  "severity": "warning",
  "enabled": true,
  "cooldown_seconds": 300,
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
)
redis_cmd HSET rules "$RULE_2_ID" "$RULE_2" > /dev/null
echo "  ✓ Created rule: High CPU Usage - Warning"

# Rule 3: High Memory Usage - Critical
RULE_3_ID="rule-memory-critical"
RULE_3=$(cat <<EOF
{
  "id": "$RULE_3_ID",
  "name": "High Memory Usage - Critical",
  "description": "Alert when memory usage exceeds 90% for any service",
  "service_name": "",
  "metric_type": "memory",
  "threshold": 90,
  "operator": "gt",
  "severity": "critical",
  "enabled": true,
  "cooldown_seconds": 300,
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
)
redis_cmd HSET rules "$RULE_3_ID" "$RULE_3" > /dev/null
echo "  ✓ Created rule: High Memory Usage - Critical"

# Rule 4: High Latency - Warning
RULE_4_ID="rule-latency-warning"
RULE_4=$(cat <<EOF
{
  "id": "$RULE_4_ID",
  "name": "High Latency - Warning",
  "description": "Alert when P95 latency exceeds 500ms for any service",
  "service_name": "",
  "metric_type": "latency",
  "threshold": 500,
  "operator": "gt",
  "severity": "warning",
  "enabled": true,
  "cooldown_seconds": 180,
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
)
redis_cmd HSET rules "$RULE_4_ID" "$RULE_4" > /dev/null
echo "  ✓ Created rule: High Latency - Warning"

# Rule 5: High Error Rate - Critical
RULE_5_ID="rule-error-critical"
RULE_5=$(cat <<EOF
{
  "id": "$RULE_5_ID",
  "name": "High Error Rate - Critical",
  "description": "Alert when error rate exceeds 10% for any service",
  "service_name": "",
  "metric_type": "error_rate",
  "threshold": 10,
  "operator": "gt",
  "severity": "critical",
  "enabled": true,
  "cooldown_seconds": 120,
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
)
redis_cmd HSET rules "$RULE_5_ID" "$RULE_5" > /dev/null
echo "  ✓ Created rule: High Error Rate - Critical"

# Rule 6: Payment Service - High Latency
RULE_6_ID="rule-payments-latency"
RULE_6=$(cat <<EOF
{
  "id": "$RULE_6_ID",
  "name": "Payments Service - High Latency",
  "description": "Alert when Payments service latency exceeds 300ms",
  "service_name": "payments",
  "metric_type": "latency",
  "threshold": 300,
  "operator": "gt",
  "severity": "warning",
  "enabled": true,
  "cooldown_seconds": 180,
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
)
redis_cmd HSET rules "$RULE_6_ID" "$RULE_6" > /dev/null
echo "  ✓ Created rule: Payments Service - High Latency"

echo ""
echo "=========================================="
echo "Seed completed successfully!"
echo "=========================================="
echo ""
echo "Total rules created: 6"
echo ""
echo "You can verify by running:"
echo "  redis-cli -h $REDIS_HOST -p $REDIS_PORT -n $REDIS_DB HGETALL rules"
