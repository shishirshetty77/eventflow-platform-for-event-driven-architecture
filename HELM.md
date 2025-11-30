# EventFlow Platform - Helm Deployment Guide

This guide details how to deploy the EventFlow Platform using the provided Helm chart.

## Prerequisites

- **Kubernetes Cluster**: A running Kubernetes cluster (e.g., Kind, GKE, EKS).
- **Helm**: Helm v3+ installed on your local machine.
- **kubectl**: Configured to communicate with your cluster.

## Configuration

The main configuration file is `charts/eventflow/values.yaml`.

### Global Settings

- `global.externalIp`: **CRITICAL**. Set this to the public IP address of your node or load balancer. This ensures the frontend can communicate with the backend and CORS is configured correctly.
- `global.environment`: Set to `production` or `development`.

### Services

Each service (dashboard, uiBackend, auth, etc.) has its own section where you can configure:

- `image`: Docker image to use.
- `replicas`: Number of pods.
- `service.port`: Internal service port.

### Secrets

Sensitive data is configured in the `secrets` section:

- `jwtSecret`: Secret key for JWT signing. **Change this in production.**
- `redisPassword`: Password for Redis (if enabled).
- `grafanaPassword`: Admin password for Grafana.

## Deployment Steps

### 1. Update Configuration

Open `charts/eventflow/values.yaml` and update the `externalIp`:

```yaml
global:
  externalIp: "YOUR_PUBLIC_IP"
```

### 2. Install/Upgrade the Chart

Run the following command from the root of the repository to install or upgrade the application:

```bash
helm upgrade --install eventflow ./charts/eventflow --create-namespace --namespace eventflow
```

- `eventflow`: The release name.
- `./charts/eventflow`: Path to the chart directory.
- `--create-namespace`: Creates the `eventflow` namespace if it doesn't exist.
- `--namespace eventflow`: Deploys into the `eventflow` namespace.

### 3. Verify Deployment

Check the status of the pods:

```bash
kubectl get pods -n eventflow
```

Wait until all pods are in the `Running` state.

### 4. Access the Application

The application exposes the following services:

- **Dashboard**: NodePort 30000 (mapped to port 3000)
- **UI Backend**: NodePort 30001 (mapped to port 8080)

If you are running on a cloud VM (like EC2) without a LoadBalancer, you may need to port-forward:

```bash
# Expose Dashboard on port 3000
kubectl port-forward --address 0.0.0.0 svc/dashboard 3000:3000 -n eventflow &

# Expose Backend on port 8080
kubectl port-forward --address 0.0.0.0 svc/ui-backend 8080:8080 -n eventflow &
```

Access the dashboard at: `http://<YOUR_PUBLIC_IP>:3000`

## Uninstalling

To remove the deployment:

```bash
helm uninstall eventflow -n eventflow
```
