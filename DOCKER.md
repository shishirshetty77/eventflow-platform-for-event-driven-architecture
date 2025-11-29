# EventFlow Platform - Docker Deployment Guide

## 🚀 Quick Start

```bash
# Start the entire platform with a single command
make up

# Or using docker-compose directly
# Or using docker-compose directly
docker-compose up --build
```

### 🐣 Lite Mode (Low Resource)

For environments with limited resources (e.g., micro instances), use Lite Mode. This disables observability tools (Prometheus, Grafana, Jaeger) and tunes JVM heap sizes.

```bash
# Start in lite mode
make up-lite

# Or using docker-compose
docker-compose -f docker-compose.lite.yml up --build
```

That's it! The platform will be available at:

- **Dashboard**: http://localhost:3001
- **Grafana**: http://localhost:3000 (admin/admin)
- **Kafka UI**: http://localhost:8080
- **Prometheus**: http://localhost:9090
- **Jaeger**: http://localhost:16686

## 📋 Prerequisites

- Docker Engine 20.10+
- Docker Compose v2.0+
- At least 8GB RAM available for Docker
- macOS, Linux, or Windows with WSL2

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              EventFlow Platform                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                           Frontend Layer                                 │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │ │
│  │  │  Dashboard  │  │  Kafka UI   │  │  Redis UI   │  │   Grafana   │     │ │
│  │  │    :3001    │  │    :8080    │  │    :8081    │  │    :3000    │     │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘     │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                      │                                        │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                          Backend Services                                │ │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌──────────────┐  │ │
│  │  │  Auth   │  │ Orders  │  │Payments │  │ Notif.  │  │  UI Backend  │  │ │
│  │  │  :8001  │  │  :8002  │  │  :8003  │  │  :8004  │  │    :8007     │  │ │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘  └──────────────┘  │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                      │                                        │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                         Analytics Services                               │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐  │ │
│  │  │    Analyzer     │  │  Alert Engine   │  │      Prometheus         │  │ │
│  │  │     :8005       │  │     :8006       │  │        :9090            │  │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────────┘  │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                      │                                        │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                         Infrastructure Layer                             │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐  │ │
│  │  │      Kafka      │  │      Redis      │  │        Jaeger           │  │ │
│  │  │      :9092      │  │      :6379      │  │        :16686           │  │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────────┘  │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                               │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 🐳 Docker Commands

### Primary Commands

| Command          | Description                                    |
| ---------------- | ---------------------------------------------- |
| `make up`        | Start entire platform (build + run)            |
| `make down`      | Stop entire platform                           |
| `make restart`   | Restart all services                           |
| `make logs`      | Follow logs from all services                  |
| `make ps`        | Show running containers                        |
| `make status`    | Show detailed status                           |
| `make health`    | Run health check on all services               |
| `make up-lite`   | Start platform in Lite Mode (no observability) |
| `make down-lite` | Stop Lite Mode platform                        |

### Build Commands

| Command             | Description              |
| ------------------- | ------------------------ |
| `make build-images` | Build all Docker images  |
| `make rebuild`      | Force rebuild (no cache) |
| `make pull`         | Pull latest base images  |

### Cleanup Commands

| Command                 | Description                     |
| ----------------------- | ------------------------------- |
| `make docker-clean`     | Stop containers, remove network |
| `make docker-clean-all` | Full cleanup (volumes + images) |
| `make docker-prune`     | Remove dangling resources       |

### Service-Specific Commands

```bash
# Restart specific service
make restart-auth
make restart-orders
make restart-payments
make restart-notification
make restart-analyzer
make restart-alert-engine
make restart-ui-backend
make restart-dashboard

# View specific service logs
make logs-auth
make logs-orders
make logs-kafka
make logs-redis

# Open shell in container
make shell-auth
make shell-kafka
make shell-redis
```

### Utility Commands

```bash
# Seed Redis with default threshold rules
make seed-redis

# Verify Kafka topics
make verify-kafka

# Create Kafka topics manually
make create-topics

# Open UIs in browser (macOS)
make open-dashboard
make open-grafana
make open-all
```

## 🔧 Configuration

### Environment Variables

Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

Key variables:

- `JWT_SECRET` - Change in production!
- `GF_SECURITY_ADMIN_PASSWORD` - Grafana admin password
- `LOG_LEVEL` - Set to `debug` for troubleshooting

### Port Mapping

| Service         | Internal Port | External Port |
| --------------- | ------------- | ------------- |
| Auth            | 8080          | 8001          |
| Orders          | 8080          | 8002          |
| Payments        | 8080          | 8003          |
| Notification    | 8080          | 8004          |
| Analyzer        | 8080          | 8005          |
| Alert Engine    | 8080          | 8006          |
| UI Backend      | 8080          | 8007          |
| Dashboard       | 3000          | 3001          |
| Kafka           | 9092          | 9092          |
| Redis           | 6379          | 6379          |
| Prometheus      | 9090          | 9090          |
| Grafana         | 3000          | 3000          |
| Jaeger          | 16686         | 16686         |
| Kafka UI        | 8080          | 8080          |
| Redis Commander | 8081          | 8081          |

## 📊 Observability

### Grafana Dashboards

1. Open Grafana: http://localhost:3000
2. Login with `admin/admin`
3. Navigate to Dashboards > EventFlow Overview

### Prometheus Metrics

All services expose metrics on their `/metrics` endpoint:

- http://localhost:8001/metrics (Auth)
- http://localhost:8002/metrics (Orders)
- etc.

### Distributed Tracing

1. Open Jaeger: http://localhost:16686
2. Select a service from the dropdown
3. Click "Find Traces"

## 🔍 Troubleshooting

### Service Won't Start

```bash
# Check logs for specific service
make logs-<service>

# Example
make logs-auth
make logs-kafka
```

### Kafka Connection Issues

```bash
# Verify Kafka is ready
make verify-kafka

# List Kafka topics
make list-topics

# Check Kafka logs
make logs-kafka
```

### Redis Connection Issues

```bash
# Open Redis CLI
docker exec -it eventflow-redis redis-cli

# Check if Redis is responding
PING
```

### Container Resource Issues

```bash
# Check container stats
docker stats

# Increase Docker resources in Docker Desktop settings
```

### Clean Restart

```bash
# Full cleanup and restart
make docker-clean-all
make up
```

## 🗄️ Volumes

The platform uses named volumes for data persistence:

| Volume                      | Purpose                   |
| --------------------------- | ------------------------- |
| `eventflow-kafka-data`      | Kafka message storage     |
| `eventflow-zookeeper-data`  | Zookeeper data            |
| `eventflow-zookeeper-logs`  | Zookeeper logs            |
| `eventflow-redis-data`      | Redis data                |
| `eventflow-prometheus-data` | Prometheus metrics        |
| `eventflow-grafana-data`    | Grafana dashboards/config |

To clear all data:

```bash
make docker-clean-all
```

## 🌐 Network

All services communicate on the `eventflow-network` bridge network (172.28.0.0/16).

Service discovery uses Docker Compose service names:

- `kafka:9092`
- `redis:6379`
- `auth:8080`
- etc.

## 📝 Logs

### Log Aggregation

All services output structured JSON logs to stdout/stderr, which Docker captures automatically.

View logs:

```bash
# All services
make logs

# Specific service
docker-compose logs -f auth

# Last 100 lines
docker-compose logs --tail=100 auth
```

### Log Format

```json
{
  "level": "info",
  "ts": "2024-01-01T12:00:00.000Z",
  "caller": "auth/handler.go:42",
  "msg": "Request processed",
  "service": "auth",
  "method": "POST",
  "path": "/api/login",
  "duration_ms": 15,
  "status": 200
}
```

## 🔒 Security Notes

For production deployment:

1. **Change all default passwords**

   - JWT_SECRET
   - Grafana admin password
   - Redis password (enable AUTH)

2. **Enable TLS**

   - Configure reverse proxy (nginx/traefik)
   - Enable Kafka SSL
   - Enable Redis TLS

3. **Network isolation**

   - Don't expose infrastructure ports publicly
   - Use internal Docker network

4. **Secret management**
   - Use Docker secrets or external vault
   - Never commit `.env` files

## 📚 Additional Resources

- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Kafka Documentation](https://kafka.apache.org/documentation/)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
