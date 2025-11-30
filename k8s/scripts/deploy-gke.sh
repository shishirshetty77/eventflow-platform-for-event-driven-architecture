#!/bin/bash
# =============================================================================
# EventFlow Platform - GKE Deployment Script
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

# Configuration - UPDATE THESE
GCP_PROJECT="${GCP_PROJECT:-your-gcp-project}"
GCP_REGION="${GCP_REGION:-us-central1}"
GKE_CLUSTER="${GKE_CLUSTER:-eventflow-cluster}"
DOCKER_REGISTRY="${DOCKER_REGISTRY:-gcr.io/${GCP_PROJECT}}"

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║       EventFlow Platform - GKE Deployment                     ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

if ! command -v gcloud &> /dev/null; then
    echo -e "${RED}Error: gcloud is not installed${NC}"
    echo "Install from: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}Error: kubectl is not installed${NC}"
    exit 1
fi

if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: docker is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}✓ All prerequisites met${NC}"
echo ""

# Check GCP authentication
echo -e "${YELLOW}Checking GCP authentication...${NC}"
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | head -n 1 &> /dev/null; then
    echo -e "${YELLOW}Please authenticate with GCP:${NC}"
    gcloud auth login
fi
echo -e "${GREEN}✓ GCP authenticated${NC}"

# Set project
echo -e "${YELLOW}Setting GCP project to ${GCP_PROJECT}...${NC}"
gcloud config set project $GCP_PROJECT

# Get GKE credentials
echo -e "${YELLOW}Getting GKE cluster credentials...${NC}"
gcloud container clusters get-credentials $GKE_CLUSTER --region $GCP_REGION

# Verify connection
kubectl cluster-info
echo ""

# Configure Docker for GCR
echo -e "${YELLOW}Configuring Docker for GCR...${NC}"
gcloud auth configure-docker gcr.io --quiet
echo -e "${GREEN}✓ Docker configured${NC}"
echo ""

# Build and push Docker images
echo -e "${YELLOW}Building and pushing Docker images...${NC}"
cd "$PROJECT_ROOT"

SERVICES="auth orders payments notification analyzer alert-engine ui-backend"
for SERVICE in $SERVICES; do
    echo -e "${BLUE}Building ${SERVICE}...${NC}"
    docker build -t ${DOCKER_REGISTRY}/eventflow-${SERVICE}:latest -f services/${SERVICE}/Dockerfile .
    docker push ${DOCKER_REGISTRY}/eventflow-${SERVICE}:latest
done

echo -e "${BLUE}Building dashboard...${NC}"
docker build -t ${DOCKER_REGISTRY}/eventflow-dashboard:latest -f dashboard/Dockerfile dashboard/
docker push ${DOCKER_REGISTRY}/eventflow-dashboard:latest

echo -e "${GREEN}✓ Images built and pushed${NC}"
echo ""

# Create image patch for GKE overlay
echo -e "${YELLOW}Creating image patches for GKE...${NC}"
cat > "$PROJECT_ROOT/k8s/overlays/gke/images.yaml" << EOF
# Auto-generated image patches for GKE
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

images:
  - name: eventflow/auth
    newName: ${DOCKER_REGISTRY}/eventflow-auth
    newTag: latest
  - name: eventflow/orders
    newName: ${DOCKER_REGISTRY}/eventflow-orders
    newTag: latest
  - name: eventflow/payments
    newName: ${DOCKER_REGISTRY}/eventflow-payments
    newTag: latest
  - name: eventflow/notification
    newName: ${DOCKER_REGISTRY}/eventflow-notification
    newTag: latest
  - name: eventflow/analyzer
    newName: ${DOCKER_REGISTRY}/eventflow-analyzer
    newTag: latest
  - name: eventflow/alert-engine
    newName: ${DOCKER_REGISTRY}/eventflow-alert-engine
    newTag: latest
  - name: eventflow/ui-backend
    newName: ${DOCKER_REGISTRY}/eventflow-ui-backend
    newTag: latest
  - name: eventflow/dashboard
    newName: ${DOCKER_REGISTRY}/eventflow-dashboard
    newTag: latest
EOF

# Install NGINX Ingress Controller if not exists
echo -e "${YELLOW}Checking NGINX Ingress Controller...${NC}"
if ! kubectl get namespace ingress-nginx &> /dev/null; then
    echo -e "${YELLOW}Installing NGINX Ingress Controller...${NC}"
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml
    
    echo -e "${YELLOW}Waiting for Ingress Controller...${NC}"
    kubectl wait --namespace ingress-nginx \
      --for=condition=ready pod \
      --selector=app.kubernetes.io/component=controller \
      --timeout=300s
fi
echo -e "${GREEN}✓ Ingress Controller ready${NC}"
echo ""

# Deploy using Kustomize
echo -e "${YELLOW}Deploying EventFlow Platform to GKE...${NC}"
kubectl apply -k "$PROJECT_ROOT/k8s/overlays/gke"
echo -e "${GREEN}✓ Manifests applied${NC}"
echo ""

# Wait for deployments
echo -e "${YELLOW}Waiting for deployments to be ready...${NC}"
echo "(This may take several minutes)"

kubectl -n eventflow wait --for=condition=ready pod -l app.kubernetes.io/name=zookeeper --timeout=300s || true
kubectl -n eventflow wait --for=condition=ready pod -l app.kubernetes.io/name=kafka --timeout=600s || true
kubectl -n eventflow wait --for=condition=ready pod -l app.kubernetes.io/name=redis --timeout=180s || true
kubectl -n eventflow wait --for=condition=available deployment --all --timeout=600s || true

echo ""

# Get external IP
echo -e "${YELLOW}Getting external IP...${NC}"
EXTERNAL_IP=$(kubectl -n ingress-nginx get svc ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo ""
echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║       EventFlow Platform Deployed to GKE!                     ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}External IP:${NC} ${EXTERNAL_IP}"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo "  1. Update DNS to point 'eventflow.example.com' to ${EXTERNAL_IP}"
echo "  2. Or add to /etc/hosts: ${EXTERNAL_IP} eventflow.example.com"
echo ""
echo -e "${BLUE}Access Points:${NC}"
echo "  Dashboard:     http://eventflow.example.com"
echo "  API:           http://eventflow.example.com/api"
echo "  Grafana:       http://eventflow.example.com/grafana"
echo ""
echo -e "${BLUE}Check Status:${NC}"
echo "  kubectl -n eventflow get pods"
echo "  kubectl -n eventflow get svc"
echo "  kubectl -n eventflow get ingress"
echo ""
