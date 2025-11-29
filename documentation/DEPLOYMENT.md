# EventFlow Deployment Guide

This guide covers deploying EventFlow on various environments.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Local Development](#local-development)
3. [Docker Deployment](#docker-deployment)
4. [AWS EC2 Deployment](#aws-ec2-deployment)
5. [Environment Variables](#environment-variables)
6. [Port Reference](#port-reference)

---

## Prerequisites

### Required Software

```
┌─────────────────────────────────────────────────────────────┐
│                    REQUIRED SOFTWARE                        │
├─────────────────────────────────────────────────────────────┤
│  Software        │  Min Version  │  Purpose                 │
├──────────────────┼───────────────┼──────────────────────────┤
│  Docker          │  24.0+        │  Container runtime       │
│  Docker Compose  │  2.20+        │  Service orchestration   │
│  Go              │  1.21+        │  Build microservices     │
│  Node.js         │  18+          │  Build dashboard         │
│  Git             │  2.0+         │  Version control         │
└─────────────────────────────────────────────────────────────┘
```

### System Requirements

```
┌─────────────────────────────────────────────────────────────┐
│              MINIMUM SYSTEM REQUIREMENTS                    │
├─────────────────────────────────────────────────────────────┤
│  Resource        │  Development  │  Production              │
├──────────────────┼───────────────┼──────────────────────────┤
│  CPU             │  4 cores      │  8+ cores                │
│  RAM             │  8 GB         │  16+ GB                  │
│  Storage         │  20 GB        │  100+ GB                 │
│  Network         │  Broadband    │  1 Gbps+                 │
└─────────────────────────────────────────────────────────────┘
```

---

## Local Development

### Setup Flow

```
┌─────────────────────────────────────────────────────────────┐
│                 LOCAL SETUP WORKFLOW                        │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
          ┌─────────────────────────────────┐
          │   1. Clone Repository            │
          │   git clone <repo-url>           │
          │   cd eventflow                   │
          └─────────────────────────────────┘
                           │
                           ▼
          ┌─────────────────────────────────┐
          │   2. Copy Environment File       │
          │   cp .env.example .env           │
          │   # Edit .env as needed          │
          └─────────────────────────────────┘
                           │
                           ▼
          ┌─────────────────────────────────┐
          │   3. Start Infrastructure        │
          │   docker compose up -d           │
          └─────────────────────────────────┘
                           │
                           ▼
          ┌─────────────────────────────────┐
          │   4. Wait for Health Checks      │
          │   docker compose ps              │
          └─────────────────────────────────┘
                           │
                           ▼
          ┌─────────────────────────────────┐
          │   5. Access Dashboard            │
          │   http://localhost:3001          │
          └─────────────────────────────────┘
```

### Quick Start Commands

```bash
# Clone and enter project
git clone <repository-url>
cd eventflow

# Setup environment
cp .env.example .env

# Start all services
docker compose up -d

# Check status
docker compose ps

# View logs
docker compose logs -f

# Stop services
docker compose down
```

---

## Docker Deployment

### Container Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        DOCKER NETWORK: eventflow                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │  Dashboard  │  │ UI-Backend  │  │   Grafana   │  │   Jaeger    │    │
│  │   :3001     │  │   :8007     │  │   :3003     │  │   :16686    │    │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘    │
│         │                │                │                │            │
│  ═══════╪════════════════╪════════════════╪════════════════╪═══════════ │
│         │                │                │                │            │
│  ┌──────┴──────┐  ┌──────┴──────┐  ┌──────┴──────┐  ┌──────┴──────┐    │
│  │    Auth     │  │   Orders    │  │  Payments   │  │ Notification│    │
│  │   :8001     │  │   :8002     │  │   :8003     │  │   :8004     │    │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘    │
│         │                │                │                │            │
│  ═══════╪════════════════╪════════════════╪════════════════╪═══════════ │
│         │                │                │                │            │
│  ┌──────┴──────┐  ┌──────┴──────┐  ┌──────┴──────┐  ┌──────┴──────┐    │
│  │   Kafka     │  │   Redis     │  │ Prometheus  │  │ PostgreSQL  │    │
│  │   :9192     │  │   :6379     │  │   :9190     │  │   :5432     │    │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Service Startup Order

```
┌─────────────────────────────────────────────────────────────┐
│                  SERVICE STARTUP ORDER                      │
└─────────────────────────────────────────────────────────────┘

Phase 1: Infrastructure (No Dependencies)
─────────────────────────────────────────
    ┌──────────┐  ┌──────────┐  ┌──────────┐
    │ Postgres │  │  Redis   │  │Zookeeper │
    └────┬─────┘  └────┬─────┘  └────┬─────┘
         │             │             │
         ▼             ▼             ▼
      Ready         Ready         Ready
                                     │
Phase 2: Messaging                   │
──────────────────────               │
                    ┌────────────────┘
                    ▼
              ┌──────────┐
              │  Kafka   │
              └────┬─────┘
                   │
                   ▼
                Ready

Phase 3: Monitoring
──────────────────────
    ┌──────────┐  ┌──────────┐  ┌──────────┐
    │Prometheus│  │  Jaeger  │  │ Grafana  │
    └────┬─────┘  └────┬─────┘  └────┬─────┘
         │             │             │
         ▼             ▼             ▼
      Ready         Ready         Ready

Phase 4: Business Services
──────────────────────────────
    ┌──────┐  ┌────────┐  ┌──────────┐  ┌────────────┐
    │ Auth │  │ Orders │  │ Payments │  │Notification│
    └──┬───┘  └───┬────┘  └────┬─────┘  └─────┬──────┘
       │          │            │              │
       ▼          ▼            ▼              ▼
    Ready      Ready        Ready          Ready

Phase 5: Analytics
──────────────────────
    ┌──────────┐  ┌─────────────┐
    │ Analyzer │  │Alert-Engine │
    └────┬─────┘  └──────┬──────┘
         │               │
         ▼               ▼
      Ready           Ready

Phase 6: Frontend
─────────────────────
    ┌────────────┐  ┌───────────┐
    │ UI-Backend │  │ Dashboard │
    └─────┬──────┘  └─────┬─────┘
          │               │
          ▼               ▼
       Ready           Ready
```

### Health Check Verification

```bash
# Check all containers are healthy
docker compose ps

# Expected output:
# NAME                STATUS
# auth-service        healthy
# orders-service      healthy
# payments-service    healthy
# notification        healthy
# analyzer            healthy
# alert-engine        healthy
# ui-backend          healthy
# dashboard           healthy
# kafka               healthy
# redis               healthy
# postgres            healthy
# prometheus          healthy
# grafana             healthy
# jaeger              healthy
```

---

## AWS EC2 Deployment

### EC2 Instance Setup

```
┌─────────────────────────────────────────────────────────────┐
│                 EC2 DEPLOYMENT TOPOLOGY                     │
└─────────────────────────────────────────────────────────────┘

                    ┌─────────────────────┐
                    │      Internet       │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │   Security Group    │
                    │   Inbound Rules:    │
                    │   - 22   (SSH)      │
                    │   - 80   (HTTP)     │
                    │   - 443  (HTTPS)    │
                    │   - 3001 (Dashboard)│
                    │   - 8007 (API)      │
                    │   - 3003 (Grafana)  │
                    │   - 16686 (Jaeger)  │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │     EC2 Instance    │
                    │   (t3.xlarge+)      │
                    │                     │
                    │  ┌───────────────┐  │
                    │  │    Docker     │  │
                    │  │   Compose     │  │
                    │  │               │  │
                    │  │  EventFlow    │  │
                    │  │   Services    │  │
                    │  └───────────────┘  │
                    │                     │
                    └─────────────────────┘
```

### Step-by-Step EC2 Setup

#### 1. Launch EC2 Instance

```bash
# Recommended: Amazon Linux 2023 or Ubuntu 22.04
# Instance type: t3.xlarge (4 vCPU, 16 GB RAM)
# Storage: 50 GB gp3

# Security Group Inbound Rules:
# - SSH (22) from your IP
# - Custom TCP (3001) from 0.0.0.0/0  # Dashboard
# - Custom TCP (8007) from 0.0.0.0/0  # API
# - Custom TCP (3003) from 0.0.0.0/0  # Grafana
# - Custom TCP (16686) from 0.0.0.0/0 # Jaeger
```

#### 2. Install Docker on EC2

```bash
# SSH into instance
ssh -i your-key.pem ec2-user@<public-ip>

# Update system (Amazon Linux 2023)
sudo dnf update -y

# Install Docker
sudo dnf install docker -y
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker $USER

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Re-login for group changes
exit
ssh -i your-key.pem ec2-user@<public-ip>
```

#### 3. Deploy EventFlow

```bash
# Clone repository
git clone <repository-url>
cd eventflow

# Configure environment for EC2
cat > .env << EOF
# Replace <EC2_PUBLIC_IP> with your actual EC2 public IP
NEXT_PUBLIC_API_URL=http://<EC2_PUBLIC_IP>:8007
NEXT_PUBLIC_WS_URL=ws://<EC2_PUBLIC_IP>:8007
EOF

# Start services
docker compose up -d

# Verify deployment
docker compose ps
```

#### 4. Access Services

```
┌─────────────────────────────────────────────────────────────┐
│                   EC2 ACCESS URLS                           │
├─────────────────────────────────────────────────────────────┤
│  Service         │  URL                                     │
├──────────────────┼──────────────────────────────────────────┤
│  Dashboard       │  http://<EC2_PUBLIC_IP>:3001             │
│  API Backend     │  http://<EC2_PUBLIC_IP>:8007             │
│  Grafana         │  http://<EC2_PUBLIC_IP>:3003             │
│  Jaeger          │  http://<EC2_PUBLIC_IP>:16686            │
│  Prometheus      │  http://<EC2_PUBLIC_IP>:9190             │
└─────────────────────────────────────────────────────────────┘
```

---

## Environment Variables

### Configuration Reference

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      ENVIRONMENT VARIABLES                              │
├─────────────────────────────────────────────────────────────────────────┤
│  Variable                │  Default                  │  Description     │
├──────────────────────────┼───────────────────────────┼──────────────────┤
│  NEXT_PUBLIC_API_URL     │  http://localhost:8007    │  Dashboard API   │
│  NEXT_PUBLIC_WS_URL      │  ws://localhost:8007      │  WebSocket URL   │
│  POSTGRES_USER           │  eventflow                │  DB username     │
│  POSTGRES_PASSWORD       │  eventflow123             │  DB password     │
│  POSTGRES_DB             │  eventflow                │  Database name   │
│  REDIS_URL               │  redis:6379               │  Redis address   │
│  KAFKA_BROKERS           │  kafka:9092               │  Kafka address   │
│  JAEGER_ENDPOINT         │  http://jaeger:14268      │  Tracing URL     │
└─────────────────────────────────────────────────────────────────────────┘
```

### Environment-Specific Setup

```bash
# Local Development (.env)
NEXT_PUBLIC_API_URL=http://localhost:8007
NEXT_PUBLIC_WS_URL=ws://localhost:8007

# EC2 Production (.env)
NEXT_PUBLIC_API_URL=http://<EC2_PUBLIC_IP>:8007
NEXT_PUBLIC_WS_URL=ws://<EC2_PUBLIC_IP>:8007

# Custom Domain with HTTPS (.env)
NEXT_PUBLIC_API_URL=https://api.yourdomain.com
NEXT_PUBLIC_WS_URL=wss://api.yourdomain.com
```

---

## Port Reference

### Complete Port Mapping

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        PORT MAPPING                                     │
├─────────────────────────────────────────────────────────────────────────┤
│  Service           │  Container Port  │  Host Port  │  Protocol        │
├────────────────────┼──────────────────┼─────────────┼──────────────────┤
│  Dashboard         │  3000            │  3001       │  HTTP            │
│  UI Backend        │  8080            │  8007       │  HTTP/WS         │
│  Auth Service      │  8080            │  8001       │  HTTP            │
│  Orders Service    │  8080            │  8002       │  HTTP            │
│  Payments Service  │  8080            │  8003       │  HTTP            │
│  Notification      │  8080            │  8004       │  HTTP            │
│  Analyzer          │  8080            │  8005       │  HTTP            │
│  Alert Engine      │  8080            │  8006       │  HTTP            │
│  Grafana           │  3000            │  3003       │  HTTP            │
│  Prometheus        │  9090            │  9190       │  HTTP            │
│  Jaeger UI         │  16686           │  16686      │  HTTP            │
│  Kafka             │  9092            │  9192       │  TCP             │
│  Redis             │  6379            │  6379       │  TCP             │
│  PostgreSQL        │  5432            │  5432       │  TCP             │
│  Zookeeper         │  2181            │  2181       │  TCP             │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Troubleshooting Deployment

### Common Issues

```
┌─────────────────────────────────────────────────────────────┐
│  Problem: Containers not starting                           │
├─────────────────────────────────────────────────────────────┤
│  Solution:                                                  │
│  1. Check Docker is running: docker info                    │
│  2. Check available memory: free -h                         │
│  3. Check disk space: df -h                                 │
│  4. View logs: docker compose logs <service>                │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  Problem: Dashboard shows "Failed to fetch"                 │
├─────────────────────────────────────────────────────────────┤
│  Solution:                                                  │
│  1. Verify .env has correct NEXT_PUBLIC_API_URL             │
│  2. Rebuild dashboard: docker compose build dashboard       │
│  3. Restart: docker compose up -d dashboard                 │
│  4. Check Security Group allows port 8007                   │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  Problem: Kafka connection errors                           │
├─────────────────────────────────────────────────────────────┤
│  Solution:                                                  │
│  1. Wait for Kafka to be fully ready (~60 seconds)          │
│  2. Check Kafka logs: docker compose logs kafka             │
│  3. Verify topics: docker exec -it kafka kafka-topics       │
│     --list --bootstrap-server localhost:9092                │
└─────────────────────────────────────────────────────────────┘
```

---

## Next Steps

- [Architecture Overview](./ARCHITECTURE.md)
- [Load Simulation Guide](./LOAD_SIMULATION.md)
- [API Reference](./API_REFERENCE.md)
- [Troubleshooting Guide](./TROUBLESHOOTING.md)
