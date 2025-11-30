#!/bin/bash
# =============================================================================
# EventFlow Platform - Kind Deployment Script
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║       EventFlow Platform - Kind Deployment                    ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

if ! command -v kind &> /dev/null; then
    echo -e "${RED}Error: kind is not installed${NC}"
    echo "Install with: brew install kind"
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}Error: kubectl is not installed${NC}"
    echo "Install with: brew install kubectl"
    exit 1
fi

if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: docker is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}✓ All prerequisites met${NC}"
echo ""

# Create Kind cluster if it doesn't exist
CLUSTER_NAME="eventflow"
if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo -e "${YELLOW}Creating Kind cluster '${CLUSTER_NAME}'...${NC}"
    cat <<EOF | kind create cluster --name $CLUSTER_NAME --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress-ready=true"
    extraPortMappings:
      - containerPort: 80
        hostPort: 80
        protocol: TCP
      - containerPort: 443
        hostPort: 443
        protocol: TCP
      - containerPort: 30000
        hostPort: 30000
        protocol: TCP
  - role: worker
  - role: worker
EOF
    echo -e "${GREEN}✓ Kind cluster created${NC}"
else
    echo -e "${GREEN}✓ Kind cluster '${CLUSTER_NAME}' already exists${NC}"
fi

# Set kubectl context
kubectl cluster-info --context kind-$CLUSTER_NAME
echo ""

# Install NGINX Ingress Controller
echo -e "${YELLOW}Installing NGINX Ingress Controller...${NC}"
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

echo -e "${YELLOW}Waiting for Ingress Controller to be ready...${NC}"
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s || true
echo -e "${GREEN}✓ Ingress Controller installed${NC}"
echo ""

# Build Docker images
echo -e "${YELLOW}Building Docker images...${NC}"
cd "$PROJECT_ROOT"

docker build -t eventflow/auth:latest -f services/auth/Dockerfile . &
docker build -t eventflow/orders:latest -f services/orders/Dockerfile . &
docker build -t eventflow/payments:latest -f services/payments/Dockerfile . &
docker build -t eventflow/notification:latest -f services/notification/Dockerfile . &
wait

docker build -t eventflow/analyzer:latest -f services/analyzer/Dockerfile . &
docker build -t eventflow/alert-engine:latest -f services/alert-engine/Dockerfile . &
docker build -t eventflow/ui-backend:latest -f services/ui-backend/Dockerfile . &
wait

docker build -t eventflow/dashboard:latest -f dashboard/Dockerfile dashboard/

echo -e "${GREEN}✓ Docker images built${NC}"
echo ""

# Load images into Kind
echo -e "${YELLOW}Loading images into Kind cluster...${NC}"
kind load docker-image eventflow/auth:latest --name $CLUSTER_NAME &
kind load docker-image eventflow/orders:latest --name $CLUSTER_NAME &
kind load docker-image eventflow/payments:latest --name $CLUSTER_NAME &
kind load docker-image eventflow/notification:latest --name $CLUSTER_NAME &
wait

kind load docker-image eventflow/analyzer:latest --name $CLUSTER_NAME &
kind load docker-image eventflow/alert-engine:latest --name $CLUSTER_NAME &
kind load docker-image eventflow/ui-backend:latest --name $CLUSTER_NAME &
kind load docker-image eventflow/dashboard:latest --name $CLUSTER_NAME &
wait

echo -e "${GREEN}✓ Images loaded into Kind${NC}"
echo ""

# Deploy using Kustomize
echo -e "${YELLOW}Deploying EventFlow Platform...${NC}"
kubectl apply -k "$PROJECT_ROOT/k8s/overlays/local"
echo -e "${GREEN}✓ Manifests applied${NC}"
echo ""

# Wait for deployments
echo -e "${YELLOW}Waiting for deployments to be ready...${NC}"
echo "(This may take a few minutes for Kafka to start)"
echo ""

# Wait for infrastructure first
kubectl -n eventflow wait --for=condition=ready pod -l app.kubernetes.io/name=zookeeper --timeout=180s || true
kubectl -n eventflow wait --for=condition=ready pod -l app.kubernetes.io/name=kafka --timeout=300s || true
kubectl -n eventflow wait --for=condition=ready pod -l app.kubernetes.io/name=redis --timeout=120s || true

# Wait for services
kubectl -n eventflow wait --for=condition=available deployment --all --timeout=300s || true

echo ""
echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║       EventFlow Platform Deployed Successfully!              ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Access Points:${NC}"
echo "  Dashboard:     http://eventflow.local (add to /etc/hosts: 127.0.0.1 eventflow.local)"
echo ""
echo -e "${BLUE}Port Forwarding (alternative):${NC}"
echo "  kubectl -n eventflow port-forward svc/dashboard 3001:3000"
echo "  kubectl -n eventflow port-forward svc/ui-backend 8007:8080"
echo "  kubectl -n eventflow port-forward svc/grafana 3000:3000"
echo ""
echo -e "${BLUE}Check Status:${NC}"
echo "  kubectl -n eventflow get pods"
echo "  kubectl -n eventflow get svc"
echo ""
echo -e "${YELLOW}Tip: Add to /etc/hosts:${NC}"
echo "  echo '127.0.0.1 eventflow.local' | sudo tee -a /etc/hosts"
echo ""
