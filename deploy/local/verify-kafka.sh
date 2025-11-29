#!/bin/bash
# =============================================================================
# EventFlow Platform - Kafka Verification Script
# Verifies Kafka connection and topic configuration
# =============================================================================

set -e

KAFKA_BOOTSTRAP="${KAFKA_BOOTSTRAP:-localhost:9092}"

echo "=========================================="
echo "EventFlow Platform - Kafka Verification"
echo "=========================================="
echo "Kafka Bootstrap: $KAFKA_BOOTSTRAP"
echo ""

# Check if kafka-topics command is available
if ! command -v kafka-topics &> /dev/null; then
    echo "WARNING: kafka-topics command not found"
    echo "Using docker exec to run kafka commands..."
    
    KAFKA_CMD="docker exec eventflow-kafka kafka-topics"
    KAFKA_BOOTSTRAP_INTERNAL="kafka:29092"
else
    KAFKA_CMD="kafka-topics"
    KAFKA_BOOTSTRAP_INTERNAL="$KAFKA_BOOTSTRAP"
fi

# Wait for Kafka to be ready
echo "Checking Kafka connection..."
MAX_RETRIES=30
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if $KAFKA_CMD --bootstrap-server "$KAFKA_BOOTSTRAP_INTERNAL" --list > /dev/null 2>&1; then
        echo "✓ Kafka connection successful"
        break
    fi
    
    RETRY_COUNT=$((RETRY_COUNT + 1))
    echo "  Waiting for Kafka... (attempt $RETRY_COUNT/$MAX_RETRIES)"
    sleep 2
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo "ERROR: Could not connect to Kafka after $MAX_RETRIES attempts"
    exit 1
fi

echo ""
echo "Listing topics..."
echo "----------------------------------------"
TOPICS=$($KAFKA_CMD --bootstrap-server "$KAFKA_BOOTSTRAP_INTERNAL" --list)
echo "$TOPICS"
echo "----------------------------------------"
echo ""

# Required topics
REQUIRED_TOPICS=("service-metrics" "service-logs" "alerts" "alerts-dlq")

echo "Verifying required topics..."
MISSING_TOPICS=()

for topic in "${REQUIRED_TOPICS[@]}"; do
    if echo "$TOPICS" | grep -q "^$topic$"; then
        echo "  ✓ $topic"
    else
        echo "  ✗ $topic (MISSING)"
        MISSING_TOPICS+=("$topic")
    fi
done

echo ""

if [ ${#MISSING_TOPICS[@]} -gt 0 ]; then
    echo "=========================================="
    echo "WARNING: Missing topics detected!"
    echo "=========================================="
    echo ""
    echo "The following topics are missing:"
    for topic in "${MISSING_TOPICS[@]}"; do
        echo "  - $topic"
    done
    echo ""
    echo "Run 'docker-compose up kafka-init' to create missing topics"
    exit 1
fi

# Show topic details
echo "=========================================="
echo "Topic Details"
echo "=========================================="

for topic in "${REQUIRED_TOPICS[@]}"; do
    echo ""
    echo "--- $topic ---"
    $KAFKA_CMD --bootstrap-server "$KAFKA_BOOTSTRAP_INTERNAL" --describe --topic "$topic" 2>/dev/null || echo "  (could not describe topic)"
done

echo ""
echo "=========================================="
echo "Kafka verification completed successfully!"
echo "=========================================="
