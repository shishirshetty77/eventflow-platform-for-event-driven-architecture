#!/bin/bash
# =============================================================================
# EventFlow Platform - Health Check Script
# Verifies all services are running and healthy
# =============================================================================

set -e

echo "=========================================="
echo "EventFlow Platform - Health Check"
echo "=========================================="
echo ""

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Services to check
declare -A SERVICES
SERVICES=(
    ["Auth Service"]="http://localhost:8001/health"
    ["Orders Service"]="http://localhost:8002/health"
    ["Payments Service"]="http://localhost:8003/health"
    ["Notification Service"]="http://localhost:8004/health"
    ["Analyzer Service"]="http://localhost:8005/health"
    ["Alert Engine"]="http://localhost:8006/health"
    ["UI Backend"]="http://localhost:8007/api/health"
    ["Dashboard"]="http://localhost:3001"
    ["Prometheus"]="http://localhost:9090/-/healthy"
    ["Grafana"]="http://localhost:3000/api/health"
    ["Kafka UI"]="http://localhost:8080"
)

# Infrastructure checks
declare -A INFRA
INFRA=(
    ["Kafka"]="localhost:9092"
    ["Redis"]="localhost:6379"
    ["Zookeeper"]="localhost:2181"
)

echo "Checking infrastructure..."
echo "----------------------------------------"

for name in "${!INFRA[@]}"; do
    addr=${INFRA[$name]}
    host=$(echo $addr | cut -d: -f1)
    port=$(echo $addr | cut -d: -f2)
    
    if nc -z "$host" "$port" 2>/dev/null; then
        echo -e "  ${GREEN}✓${NC} $name ($addr)"
    else
        echo -e "  ${RED}✗${NC} $name ($addr) - NOT REACHABLE"
    fi
done

echo ""
echo "Checking services..."
echo "----------------------------------------"

FAILED=0

for name in "${!SERVICES[@]}"; do
    url=${SERVICES[$name]}
    
    # Make HTTP request with timeout
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 "$url" 2>/dev/null || echo "000")
    
    if [ "$HTTP_CODE" -ge 200 ] && [ "$HTTP_CODE" -lt 400 ]; then
        echo -e "  ${GREEN}✓${NC} $name (HTTP $HTTP_CODE)"
    elif [ "$HTTP_CODE" == "000" ]; then
        echo -e "  ${RED}✗${NC} $name - NOT REACHABLE"
        FAILED=$((FAILED + 1))
    else
        echo -e "  ${YELLOW}!${NC} $name (HTTP $HTTP_CODE)"
    fi
done

echo ""
echo "Checking Docker containers..."
echo "----------------------------------------"

docker ps --filter "name=eventflow-" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "Could not list Docker containers"

echo ""
echo "=========================================="

if [ $FAILED -gt 0 ]; then
    echo -e "${YELLOW}Some services are not reachable.${NC}"
    echo "Run 'docker-compose logs <service>' to check logs"
    exit 1
else
    echo -e "${GREEN}All services are healthy!${NC}"
fi

echo ""
echo "Access Points:"
echo "  - Dashboard:     http://localhost:3001"
echo "  - UI Backend:    http://localhost:8007"
echo "  - Kafka UI:      http://localhost:8080"
echo "  - Redis UI:      http://localhost:8081"
echo "  - Grafana:       http://localhost:3000 (admin/admin)"
echo "  - Prometheus:    http://localhost:9090"
echo "  - Jaeger:        http://localhost:16686"
echo ""
echo "Default credentials:"
echo "  - Dashboard:     admin / admin"
echo "  - Grafana:       admin / admin"
echo "  - Redis UI:      admin / admin"
