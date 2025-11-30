# EventFlow Platform - Kubernetes Deployment

This directory contains Kubernetes manifests for deploying the EventFlow Platform using Kustomize.

## 📁 Directory Structure

```
k8s/
├── base/                           # Base manifests (shared across all environments)
│   ├── namespace.yaml              # Namespace definition
│   ├── configmaps/                 # ConfigMaps
│   │   └── eventflow-config.yaml
│   ├── secrets/                    # Secrets (templates)
│   │   └── eventflow-secrets.yaml
│   ├── infrastructure/             # Infrastructure components
│   │   ├── zookeeper/
│   │   ├── kafka/
│   │   └── redis/
│   ├── services/                   # Application services
│   │   ├── auth/
│   │   ├── orders/
│   │   ├── payments/
│   │   ├── notification/
│   │   ├── analyzer/
│   │   ├── alert-engine/
│   │   ├── ui-backend/
│   │   └── dashboard/
│   ├── observability/              # Monitoring stack
│   │   ├── prometheus/
│   │   ├── grafana/
│   │   └── jaeger/
│   ├── ingress/                    # Ingress configuration
│   │   └── ingress.yaml
│   └── kustomization.yaml          # Base kustomization
├── overlays/                       # Environment-specific overrides
│   ├── local/                      # For Kind/Minikube
│   │   └── kustomization.yaml
│   ├── gke/                        # For Google Kubernetes Engine
│   │   └── kustomization.yaml
│   └── production/                 # For production deployment
│       └── kustomization.yaml
└── README.md
```

## 🚀 Quick Start

### Prerequisites

- `kubectl` installed and configured
- `kustomize` installed (or use `kubectl -k`)
- A Kubernetes cluster (Kind, Minikube, GKE, EKS, etc.)
- Docker images built and pushed to a registry

### Build Docker Images First

```bash
# From the project root
cd ..

# Build all images
docker build -t eventflow/auth:latest -f services/auth/Dockerfile .
docker build -t eventflow/orders:latest -f services/orders/Dockerfile .
docker build -t eventflow/payments:latest -f services/payments/Dockerfile .
docker build -t eventflow/notification:latest -f services/notification/Dockerfile .
docker build -t eventflow/analyzer:latest -f services/analyzer/Dockerfile .
docker build -t eventflow/alert-engine:latest -f services/alert-engine/Dockerfile .
docker build -t eventflow/ui-backend:latest -f services/ui-backend/Dockerfile .
docker build -t eventflow/dashboard:latest -f dashboard/Dockerfile dashboard/

# For Kind: Load images into the cluster
kind load docker-image eventflow/auth:latest
kind load docker-image eventflow/orders:latest
kind load docker-image eventflow/payments:latest
kind load docker-image eventflow/notification:latest
kind load docker-image eventflow/analyzer:latest
kind load docker-image eventflow/alert-engine:latest
kind load docker-image eventflow/ui-backend:latest
kind load docker-image eventflow/dashboard:latest
```

### Deploy to Local (Kind/Minikube)

```bash
# Preview what will be deployed
kubectl kustomize k8s/overlays/local

# Deploy
kubectl apply -k k8s/overlays/local

# Watch deployment progress
kubectl -n eventflow get pods -w

# Check all resources
kubectl -n eventflow get all
```

### Deploy to GKE

```bash
# Preview
kubectl kustomize k8s/overlays/gke

# Deploy
kubectl apply -k k8s/overlays/gke

# Watch deployment
kubectl -n eventflow get pods -w
```

### Deploy to Production

```bash
# Preview
kubectl kustomize k8s/overlays/production

# Deploy
kubectl apply -k k8s/overlays/production
```

## 🔧 Configuration

### Environment Variables

Configuration is managed via ConfigMaps and Secrets:

| ConfigMap Key | Description | Default |
|---------------|-------------|---------|
| `ENVIRONMENT` | Environment name | `production` |
| `LOG_LEVEL` | Logging level | `info` |
| `KAFKA_BROKERS` | Kafka broker addresses | `kafka-0.kafka-headless:9092` |
| `REDIS_ADDR` | Redis address | `redis:6379` |
| `JWT_EXPIRATION` | JWT token expiration (hours) | `24` |
| `ALLOWED_ORIGINS` | CORS allowed origins | `http://localhost:3001` |

### Secrets

| Secret Key | Description |
|------------|-------------|
| `JWT_SECRET` | JWT signing secret (CHANGE IN PRODUCTION!) |
| `REDIS_PASSWORD` | Redis password |
| `GRAFANA_PASSWORD` | Grafana admin password |
| `SLACK_WEBHOOK_URL` | Slack notification webhook |

### Customizing for Your Environment

1. **Change domain name:**
   Edit `k8s/overlays/<env>/kustomization.yaml` and update the Ingress host patch.

2. **Change StorageClass:**
   The overlays already handle this for Kind (`standard`) and GKE (`standard-rwo`).

3. **Add TLS:**
   For production, add cert-manager annotations and TLS configuration to the Ingress.

## 📊 Accessing Services

### Port Forwarding (Local Development)

```bash
# Dashboard
kubectl -n eventflow port-forward svc/dashboard 3001:3000

# UI Backend API
kubectl -n eventflow port-forward svc/ui-backend 8007:8080

# Grafana
kubectl -n eventflow port-forward svc/grafana 3000:3000

# Prometheus
kubectl -n eventflow port-forward svc/prometheus 9090:9090

# Jaeger
kubectl -n eventflow port-forward svc/jaeger 16686:16686

# Kafka (for debugging)
kubectl -n eventflow port-forward svc/kafka 9092:9092

# Redis (for debugging)
kubectl -n eventflow port-forward svc/redis 6379:6379
```

### With Ingress (Kind)

```bash
# Add to /etc/hosts
127.0.0.1 eventflow.local

# Install NGINX Ingress Controller for Kind
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

# Wait for ingress controller
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s

# Access via browser
open http://eventflow.local
```

### With Ingress (GKE)

```bash
# Install NGINX Ingress Controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml

# Get the external IP
kubectl -n ingress-nginx get svc ingress-nginx-controller

# Update DNS to point to the external IP
# Then access via browser
open http://eventflow.example.com
```

## 🔍 Troubleshooting

### Check Pod Status

```bash
kubectl -n eventflow get pods
kubectl -n eventflow describe pod <pod-name>
kubectl -n eventflow logs <pod-name>
```

### Check Events

```bash
kubectl -n eventflow get events --sort-by='.lastTimestamp'
```

### Check PVC Status

```bash
kubectl -n eventflow get pvc
kubectl -n eventflow describe pvc <pvc-name>
```

### Common Issues

1. **Pods stuck in Pending:**
   - Check if PVCs are bound: `kubectl -n eventflow get pvc`
   - Check StorageClass exists: `kubectl get sc`
   - For Kind, ensure `standard` StorageClass exists

2. **Kafka not starting:**
   - Check Zookeeper is running: `kubectl -n eventflow logs zookeeper-0`
   - Ensure init containers completed

3. **Services can't connect to Kafka/Redis:**
   - Check init containers: `kubectl -n eventflow logs <pod> -c wait-for-kafka`
   - Verify services: `kubectl -n eventflow get svc`

4. **Ingress not working:**
   - Check Ingress controller is installed
   - Verify Ingress resource: `kubectl -n eventflow describe ingress`

## 🧹 Cleanup

```bash
# Delete all resources
kubectl delete -k k8s/overlays/local  # or gke, production

# Delete namespace (removes everything)
kubectl delete namespace eventflow

# Delete PVCs (if needed)
kubectl -n eventflow delete pvc --all
```

## 📈 Scaling

```bash
# Scale a deployment
kubectl -n eventflow scale deployment ui-backend --replicas=3

# Scale using HPA (if configured)
kubectl -n eventflow autoscale deployment ui-backend --min=2 --max=10 --cpu-percent=80
```

## 🔐 Security Notes

1. **Change default secrets** in production
2. **Use external secret management** (Vault, GCP Secret Manager, AWS Secrets Manager)
3. **Enable NetworkPolicies** for production
4. **Configure PodSecurityPolicies/Standards**
5. **Enable RBAC** properly for service accounts
