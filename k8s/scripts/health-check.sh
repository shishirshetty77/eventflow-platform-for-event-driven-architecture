#!/bin/bash
# =============================================================================
# Health check script for EventFlow Platform
# =============================================================================

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║       EventFlow Platform - Health Check                       ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check namespace exists
if ! kubectl get namespace eventflow &> /dev/null; then
    echo -e "${RED}Error: eventflow namespace does not exist${NC}"
    exit 1
fi

echo -e "${YELLOW}Pod Status:${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
kubectl -n eventflow get pods -o wide
echo ""

echo -e "${YELLOW}Service Status:${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
kubectl -n eventflow get svc
echo ""

echo -e "${YELLOW}PVC Status:${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
kubectl -n eventflow get pvc
echo ""

echo -e "${YELLOW}Ingress Status:${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
kubectl -n eventflow get ingress
echo ""

# Check individual services
echo -e "${YELLOW}Service Health:${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

SERVICES="zookeeper kafka redis auth orders payments notification analyzer alert-engine ui-backend dashboard"

for SERVICE in $SERVICES; do
    POD=$(kubectl -n eventflow get pod -l app.kubernetes.io/name=$SERVICE -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$POD" ]; then
        STATUS=$(kubectl -n eventflow get pod $POD -o jsonpath='{.status.phase}')
        READY=$(kubectl -n eventflow get pod $POD -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}')
        
        if [ "$STATUS" = "Running" ] && [ "$READY" = "True" ]; then
            echo -e "  ${SERVICE}: ${GREEN}✓ Running${NC}"
        else
            echo -e "  ${SERVICE}: ${YELLOW}⚠ ${STATUS} (Ready: ${READY})${NC}"
        fi
    else
        echo -e "  ${SERVICE}: ${RED}✗ Not found${NC}"
    fi
done

echo ""
echo -e "${YELLOW}Recent Events:${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
kubectl -n eventflow get events --sort-by='.lastTimestamp' | tail -10
echo ""
