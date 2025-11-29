#!/bin/bash
# =============================================================================
# EventFlow Platform - Wait for Services Script
# Waits for all critical services to be ready before proceeding
# =============================================================================

set -e

echo "=========================================="
echo "EventFlow Platform - Waiting for Services"
echo "=========================================="
echo ""

# Default timeout in seconds
TIMEOUT=${TIMEOUT:-300}
INTERVAL=${INTERVAL:-5}

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Function to wait for a TCP port
wait_for_port() {
    local host=$1
    local port=$2
    local name=$3
    local elapsed=0
    
    echo -n "Waiting for $name ($host:$port)..."
    
    while ! nc -z "$host" "$port" 2>/dev/null; do
        sleep $INTERVAL
        elapsed=$((elapsed + INTERVAL))
        echo -n "."
        
        if [ $elapsed -ge $TIMEOUT ]; then
            echo -e " ${RED}TIMEOUT${NC}"
            return 1
        fi
    done
    
    echo -e " ${GREEN}READY${NC}"
    return 0
}

# Function to wait for an HTTP endpoint
wait_for_http() {
    local url=$1
    local name=$2
    local expected_code=${3:-200}
    local elapsed=0
    
    echo -n "Waiting for $name ($url)..."
    
    while true; do
        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 "$url" 2>/dev/null || echo "000")
        
        if [ "$HTTP_CODE" -ge 200 ] && [ "$HTTP_CODE" -lt 400 ]; then
            echo -e " ${GREEN}READY${NC} (HTTP $HTTP_CODE)"
            return 0
        fi
        
        sleep $INTERVAL
        elapsed=$((elapsed + INTERVAL))
        echo -n "."
        
        if [ $elapsed -ge $TIMEOUT ]; then
            echo -e " ${RED}TIMEOUT${NC} (last: HTTP $HTTP_CODE)"
            return 1
        fi
    done
}

# Function to wait for Kafka
wait_for_kafka() {
    local elapsed=0
    
    echo -n "Waiting for Kafka (localhost:9092)..."
    
    while true; do
        # Try to list topics - if successful, Kafka is ready
        if docker exec eventflow-kafka kafka-topics --bootstrap-server localhost:9092 --list >/dev/null 2>&1; then
            echo -e " ${GREEN}READY${NC}"
            return 0
        fi
        
        sleep $INTERVAL
        elapsed=$((elapsed + INTERVAL))
        echo -n "."
        
        if [ $elapsed -ge $TIMEOUT ]; then
            echo -e " ${RED}TIMEOUT${NC}"
            return 1
        fi
    done
}

# Function to wait for Redis
wait_for_redis() {
    local elapsed=0
    
    echo -n "Waiting for Redis (localhost:6379)..."
    
    while true; do
        if docker exec eventflow-redis redis-cli ping 2>/dev/null | grep -q "PONG"; then
            echo -e " ${GREEN}READY${NC}"
            return 0
        fi
        
        sleep $INTERVAL
        elapsed=$((elapsed + INTERVAL))
        echo -n "."
        
        if [ $elapsed -ge $TIMEOUT ]; then
            echo -e " ${RED}TIMEOUT${NC}"
            return 1
        fi
    done
}

FAILED=0

echo "Phase 1: Infrastructure Services"
echo "----------------------------------------"

# Wait for Zookeeper first (Kafka depends on it)
wait_for_port "localhost" 2181 "Zookeeper" || FAILED=$((FAILED + 1))

# Wait for Kafka
wait_for_kafka || FAILED=$((FAILED + 1))

# Wait for Redis
wait_for_redis || FAILED=$((FAILED + 1))

echo ""
echo "Phase 2: Core Services"
echo "----------------------------------------"

# Wait for core microservices
wait_for_http "http://localhost:8001/health" "Auth Service" || FAILED=$((FAILED + 1))
wait_for_http "http://localhost:8002/health" "Orders Service" || FAILED=$((FAILED + 1))
wait_for_http "http://localhost:8003/health" "Payments Service" || FAILED=$((FAILED + 1))
wait_for_http "http://localhost:8004/health" "Notification Service" || FAILED=$((FAILED + 1))

echo ""
echo "Phase 3: Analytics Services"
echo "----------------------------------------"

wait_for_http "http://localhost:8005/health" "Analyzer Service" || FAILED=$((FAILED + 1))
wait_for_http "http://localhost:8006/health" "Alert Engine" || FAILED=$((FAILED + 1))
wait_for_http "http://localhost:8007/api/health" "UI Backend" || FAILED=$((FAILED + 1))

echo ""
echo "Phase 4: Observability"
echo "----------------------------------------"

wait_for_http "http://localhost:9090/-/healthy" "Prometheus" || FAILED=$((FAILED + 1))
wait_for_http "http://localhost:3000/api/health" "Grafana" || FAILED=$((FAILED + 1))
wait_for_port "localhost" 16686 "Jaeger" || FAILED=$((FAILED + 1))

echo ""
echo "Phase 5: UI Services"
echo "----------------------------------------"

wait_for_http "http://localhost:3001" "Dashboard" || FAILED=$((FAILED + 1))
wait_for_http "http://localhost:8080" "Kafka UI" || FAILED=$((FAILED + 1))

echo ""
echo "=========================================="

if [ $FAILED -gt 0 ]; then
    echo -e "${RED}$FAILED service(s) failed to start within timeout.${NC}"
    echo "Check logs with: docker-compose logs"
    exit 1
else
    echo -e "${GREEN}All services are ready!${NC}"
    echo ""
    echo "Platform is now accessible at:"
    echo "  Dashboard: http://localhost:3001"
    exit 0
fi
