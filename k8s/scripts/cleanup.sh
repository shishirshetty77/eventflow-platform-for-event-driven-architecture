#!/bin/bash
# =============================================================================
# EventFlow Platform - Cleanup Script
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║       EventFlow Platform - Cleanup                            ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Parse arguments
DELETE_CLUSTER=false
DELETE_PVCS=false
ENVIRONMENT="local"

while [[ $# -gt 0 ]]; do
    case $1 in
        --delete-cluster)
            DELETE_CLUSTER=true
            shift
            ;;
        --delete-pvcs)
            DELETE_PVCS=true
            shift
            ;;
        --env)
            ENVIRONMENT="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo -e "${YELLOW}Environment: ${ENVIRONMENT}${NC}"
echo ""

# Delete Kubernetes resources
echo -e "${YELLOW}Deleting EventFlow resources...${NC}"
kubectl delete -k "$PROJECT_ROOT/k8s/overlays/${ENVIRONMENT}" --ignore-not-found=true || true
echo -e "${GREEN}✓ Resources deleted${NC}"

# Delete PVCs if requested
if [ "$DELETE_PVCS" = true ]; then
    echo -e "${YELLOW}Deleting PVCs...${NC}"
    kubectl -n eventflow delete pvc --all --ignore-not-found=true || true
    echo -e "${GREEN}✓ PVCs deleted${NC}"
fi

# Delete namespace
echo -e "${YELLOW}Deleting namespace...${NC}"
kubectl delete namespace eventflow --ignore-not-found=true || true
echo -e "${GREEN}✓ Namespace deleted${NC}"

# Delete Kind cluster if requested and using local environment
if [ "$DELETE_CLUSTER" = true ] && [ "$ENVIRONMENT" = "local" ]; then
    echo -e "${YELLOW}Deleting Kind cluster...${NC}"
    kind delete cluster --name eventflow || true
    echo -e "${GREEN}✓ Kind cluster deleted${NC}"
fi

echo ""
echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║       Cleanup Complete!                                       ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""
