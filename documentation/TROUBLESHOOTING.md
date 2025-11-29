# EventFlow Troubleshooting Guide

Common issues and their solutions for the EventFlow platform.

---

## Table of Contents

1. [Quick Diagnostics](#quick-diagnostics)
2. [Container Issues](#container-issues)
3. [Network Issues](#network-issues)
4. [Dashboard Issues](#dashboard-issues)
5. [Kafka Issues](#kafka-issues)
6. [Database Issues](#database-issues)
7. [Performance Issues](#performance-issues)
8. [Log Analysis](#log-analysis)

---

## Quick Diagnostics

### System Health Check

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    DIAGNOSTIC WORKFLOW                                  │
└─────────────────────────────────────────────────────────────────────────┘

                    Start Diagnosis
                          │
                          ▼
              ┌───────────────────────┐
              │  docker compose ps    │
              │  All containers       │
              │  showing "healthy"?   │
              └───────────┬───────────┘
                          │
               ┌──────────┴──────────┐
               │                     │
              YES                    NO
               │                     │
               ▼                     ▼
    ┌─────────────────┐   ┌─────────────────────┐
    │ Check Dashboard │   │ See Container Issues│
    │ connectivity    │   │ section below       │
    └─────────────────┘   └─────────────────────┘
```

### Quick Commands

```bash
# Check all container status
docker compose ps

# View logs for all services
docker compose logs --tail=100

# View logs for specific service
docker compose logs -f <service-name>

# Check resource usage
docker stats

# Restart all services
docker compose restart

# Full reset (destroys data)
docker compose down -v && docker compose up -d
```

---

## Container Issues

### Problem: Container Won't Start

```
┌─────────────────────────────────────────────────────────────────────────┐
│  SYMPTOM: Container exits immediately or shows "Exited" status          │
└─────────────────────────────────────────────────────────────────────────┘

  Diagnosis Steps:
  ─────────────────
  
  1. Check container logs:
     $ docker compose logs <service-name>
     
  2. Look for these common errors:
  
     ┌─────────────────────────────────────────────────────────────┐
     │  Error                      │  Solution                     │
     ├─────────────────────────────┼───────────────────────────────┤
     │  "port already in use"      │  Stop conflicting service     │
     │  "no space left on device"  │  docker system prune          │
     │  "connection refused"       │  Check dependency is running  │
     │  "permission denied"        │  Check file permissions       │
     └─────────────────────────────────────────────────────────────┘

  Solution Commands:
  ──────────────────
  
  # Find what's using a port
  $ lsof -i :<port>
  
  # Clean up Docker resources
  $ docker system prune -a
  
  # Check disk space
  $ df -h
  
  # Restart specific container
  $ docker compose up -d <service-name>
```

### Problem: Container Shows "Unhealthy"

```
┌─────────────────────────────────────────────────────────────────────────┐
│  SYMPTOM: Container running but health check failing                    │
└─────────────────────────────────────────────────────────────────────────┘

  Diagnosis:
  ──────────
  
  # Check health check output
  $ docker inspect <container-name> | grep -A 10 "Health"
  
  # Test health endpoint manually
  $ docker exec -it <container-name> curl localhost:<port>/health
  
  Common Causes & Solutions:
  ──────────────────────────
  
  ┌─────────────────────────────────────────────────────────────────────┐
  │                                                                     │
  │  Cause: Service still starting                                      │
  │  Solution: Wait 60-90 seconds for full initialization               │
  │                                                                     │
  │  Cause: Dependency not ready                                        │
  │  Solution: Check depends_on services are healthy                    │
  │                                                                     │
  │  Cause: Wrong health endpoint                                       │
  │  Solution: Verify healthcheck command matches service               │
  │                                                                     │
  └─────────────────────────────────────────────────────────────────────┘
```

### Problem: Out of Memory

```
┌─────────────────────────────────────────────────────────────────────────┐
│  SYMPTOM: Containers killed with OOMKilled status                       │
└─────────────────────────────────────────────────────────────────────────┘

  Diagnosis:
  ──────────
  
  # Check if OOM killed
  $ docker inspect <container> | grep OOMKilled
  
  # Check memory usage
  $ docker stats --no-stream
  
  Solutions:
  ──────────
  
  1. Increase Docker memory (Docker Desktop settings)
  
  2. Add memory limits to docker-compose.yml:
     
     services:
       service-name:
         deploy:
           resources:
             limits:
               memory: 512M
  
  3. Reduce number of running services:
     $ docker compose up -d service1 service2
```

---

## Network Issues

### Problem: Services Can't Communicate

```
┌─────────────────────────────────────────────────────────────────────────┐
│  SYMPTOM: "Connection refused" or "Host not found" errors               │
└─────────────────────────────────────────────────────────────────────────┘

  Network Architecture:
  ─────────────────────
  
  ┌───────────────────────────────────────────────────────────────────┐
  │                    Docker Network: eventflow                      │
  │                                                                   │
  │   ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐   │
  │   │  kafka   │    │  redis   │    │ postgres │    │ services │   │
  │   │          │    │          │    │          │    │          │   │
  │   │ Internal │    │ Internal │    │ Internal │    │ Internal │   │
  │   │kafka:9092│    │redis:6379│    │postgres: │    │ :8080    │   │
  │   │          │    │          │    │  5432    │    │          │   │
  │   └──────────┘    └──────────┘    └──────────┘    └──────────┘   │
  │                                                                   │
  └───────────────────────────────────────────────────────────────────┘
  
  Diagnosis:
  ──────────
  
  # List Docker networks
  $ docker network ls
  
  # Inspect network
  $ docker network inspect project_default
  
  # Test connectivity from container
  $ docker exec -it ui-backend ping redis
  
  Solutions:
  ──────────
  
  1. Ensure all services on same network:
     All services should be in docker-compose.yml
  
  2. Use service names, not localhost:
     ✗ localhost:6379
     ✓ redis:6379
  
  3. Recreate network:
     $ docker compose down
     $ docker network prune
     $ docker compose up -d
```

### Problem: Port Already in Use

```
┌─────────────────────────────────────────────────────────────────────────┐
│  SYMPTOM: "bind: address already in use" error                          │
└─────────────────────────────────────────────────────────────────────────┘

  Common Port Conflicts:
  ──────────────────────
  
  ┌─────────────────────────────────────────────────────────────────────┐
  │  Port   │  EventFlow Service  │  Common Conflict                    │
  ├─────────┼─────────────────────┼─────────────────────────────────────┤
  │  3001   │  Dashboard          │  Other Next.js apps                 │
  │  9190   │  Prometheus         │  Local Prometheus (9090)            │
  │  5432   │  PostgreSQL         │  Local PostgreSQL                   │
  │  6379   │  Redis              │  Local Redis                        │
  │  9192   │  Kafka              │  Local Kafka (9092)                 │
  └─────────────────────────────────────────────────────────────────────┘
  
  Solution:
  ─────────
  
  # Find process using port
  $ lsof -i :3001
  
  # Kill process
  $ kill -9 <PID>
  
  # Or stop local service
  $ brew services stop postgresql
  $ brew services stop redis
```

---

## Dashboard Issues

### Problem: "Failed to Fetch" Error

```
┌─────────────────────────────────────────────────────────────────────────┐
│  SYMPTOM: Dashboard shows "Failed to fetch" or blank data               │
└─────────────────────────────────────────────────────────────────────────┘

  Root Cause Diagram:
  ───────────────────
  
       Dashboard (Browser)
              │
              │ fetch(NEXT_PUBLIC_API_URL/api/metrics)
              ▼
       ┌─────────────┐
       │ API Request │
       └──────┬──────┘
              │
              │ ←── FAILURE POINT
              ▼
       ┌─────────────┐
       │ UI-Backend  │
       │ :8007       │
       └─────────────┘
  
  Diagnosis Steps:
  ────────────────
  
  1. Check environment variable:
     $ docker compose exec dashboard printenv | grep NEXT_PUBLIC
     
     Should show:
     NEXT_PUBLIC_API_URL=http://<correct-host>:8007
  
  2. Check UI-Backend is running:
     $ curl http://localhost:8007/api/health
  
  3. Check browser console for CORS errors
  
  Solutions:
  ──────────
  
  For Local Development:
  ──────────────────────
  
  # .env file
  NEXT_PUBLIC_API_URL=http://localhost:8007
  NEXT_PUBLIC_WS_URL=ws://localhost:8007
  
  # Rebuild dashboard
  $ docker compose build dashboard
  $ docker compose up -d dashboard
  
  For EC2/Remote:
  ───────────────
  
  # .env file (use public IP)
  NEXT_PUBLIC_API_URL=http://<EC2_PUBLIC_IP>:8007
  NEXT_PUBLIC_WS_URL=ws://<EC2_PUBLIC_IP>:8007
  
  # Rebuild and restart
  $ docker compose build dashboard
  $ docker compose up -d dashboard
  
  # Verify Security Group allows port 8007
```

### Problem: Login Not Working

```
┌─────────────────────────────────────────────────────────────────────────┐
│  SYMPTOM: Cannot log in to dashboard                                    │
└─────────────────────────────────────────────────────────────────────────┘

  Default Credentials:
  ────────────────────
  
  Email: admin@eventflow.io
  Password: admin123
  
  Diagnosis:
  ──────────
  
  1. Check auth service is running:
     $ docker compose ps auth-service
  
  2. Check auth service logs:
     $ docker compose logs auth-service
  
  3. Test auth endpoint directly:
     $ curl -X POST http://localhost:8001/api/auth/login \
       -H "Content-Type: application/json" \
       -d '{"email":"admin@eventflow.io","password":"admin123"}'
  
  Solutions:
  ──────────
  
  # Restart auth service
  $ docker compose restart auth-service
  
  # If database issue, reset:
  $ docker compose down -v
  $ docker compose up -d
```

### Problem: Real-time Updates Not Working

```
┌─────────────────────────────────────────────────────────────────────────┐
│  SYMPTOM: Dashboard not updating, WebSocket disconnected                │
└─────────────────────────────────────────────────────────────────────────┘

  WebSocket Flow:
  ───────────────
  
  Browser ──ws://host:8007/ws──► UI-Backend ──► Kafka Consumer
                                                      │
                                               ◄──────┘
                                          Metrics Update
  
  Diagnosis:
  ──────────
  
  1. Check browser console for WebSocket errors
  
  2. Verify WebSocket URL:
     Should match NEXT_PUBLIC_WS_URL in .env
  
  3. Check if data is flowing:
     $ docker compose logs -f analyzer
     
  Solutions:
  ──────────
  
  1. Ensure WS URL uses correct host:
     # Local
     NEXT_PUBLIC_WS_URL=ws://localhost:8007
     
     # EC2
     NEXT_PUBLIC_WS_URL=ws://<EC2_IP>:8007
  
  2. Restart services in order:
     $ docker compose restart kafka
     $ docker compose restart analyzer
     $ docker compose restart ui-backend
     $ docker compose restart dashboard
```

---

## Kafka Issues

### Problem: Kafka Not Starting

```
┌─────────────────────────────────────────────────────────────────────────┐
│  SYMPTOM: Kafka container fails to start or keeps restarting            │
└─────────────────────────────────────────────────────────────────────────┘

  Dependency Chain:
  ─────────────────
  
  Zookeeper (must start first)
       │
       ▼
  Kafka (depends on Zookeeper)
       │
       ▼
  Services (depend on Kafka)
  
  Diagnosis:
  ──────────
  
  # Check Zookeeper first
  $ docker compose logs zookeeper
  
  # Then check Kafka
  $ docker compose logs kafka
  
  # Test Kafka connectivity
  $ docker exec -it kafka kafka-topics \
      --list --bootstrap-server localhost:9092
  
  Solutions:
  ──────────
  
  # Start in correct order
  $ docker compose up -d zookeeper
  $ sleep 30
  $ docker compose up -d kafka
  
  # Or full restart
  $ docker compose down -v
  $ docker compose up -d
```

### Problem: Messages Not Being Consumed

```
┌─────────────────────────────────────────────────────────────────────────┐
│  SYMPTOM: Metrics not appearing, alerts not firing                      │
└─────────────────────────────────────────────────────────────────────────┘

  Message Flow:
  ─────────────
  
  Services ──► Topic: service-metrics ──► Analyzer
                                              │
                                              ▼
                                    Topic: alerts ──► Alert-Engine
  
  Diagnosis:
  ──────────
  
  # List topics
  $ docker exec -it kafka kafka-topics \
      --list --bootstrap-server localhost:9092
  
  Expected topics:
  - service-metrics
  - service-logs
  - alerts
  - alerts-dlq
  
  # Check consumer groups
  $ docker exec -it kafka kafka-consumer-groups \
      --list --bootstrap-server localhost:9092
  
  # Check consumer lag
  $ docker exec -it kafka kafka-consumer-groups \
      --describe --group analyzer-group \
      --bootstrap-server localhost:9092
  
  Solutions:
  ──────────
  
  # Create missing topics
  $ docker exec -it kafka kafka-topics \
      --create --topic service-metrics \
      --bootstrap-server localhost:9092 \
      --partitions 3 --replication-factor 1
  
  # Restart consumers
  $ docker compose restart analyzer alert-engine
```

---

## Database Issues

### Problem: PostgreSQL Connection Errors

```
┌─────────────────────────────────────────────────────────────────────────┐
│  SYMPTOM: "Connection refused" to PostgreSQL                            │
└─────────────────────────────────────────────────────────────────────────┘

  Diagnosis:
  ──────────
  
  # Check postgres is running
  $ docker compose ps postgres
  
  # Check logs
  $ docker compose logs postgres
  
  # Test connection
  $ docker exec -it postgres psql -U eventflow -d eventflow -c "\dt"
  
  Solutions:
  ──────────
  
  # Wait for initialization
  $ docker compose logs -f postgres
  # Look for "database system is ready to accept connections"
  
  # Reset database
  $ docker compose down -v
  $ docker compose up -d postgres
```

### Problem: Redis Connection Errors

```
┌─────────────────────────────────────────────────────────────────────────┐
│  SYMPTOM: "Connection refused" to Redis                                 │
└─────────────────────────────────────────────────────────────────────────┘

  Diagnosis:
  ──────────
  
  # Check Redis status
  $ docker compose ps redis
  
  # Test connection
  $ docker exec -it redis redis-cli ping
  # Should return: PONG
  
  # Check stored data
  $ docker exec -it redis redis-cli KEYS "*"
  
  Solutions:
  ──────────
  
  # Restart Redis
  $ docker compose restart redis
  
  # Clear and restart
  $ docker compose down redis
  $ docker volume rm project_redis_data
  $ docker compose up -d redis
```

---

## Performance Issues

### Problem: High CPU/Memory Usage

```
┌─────────────────────────────────────────────────────────────────────────┐
│  SYMPTOM: System slow, containers using excessive resources             │
└─────────────────────────────────────────────────────────────────────────┘

  Diagnosis:
  ──────────
  
  # Real-time resource monitoring
  $ docker stats
  
  # Check which service is consuming most
  NAME              CPU %     MEM USAGE
  kafka             45.2%     1.2GB
  prometheus        12.3%     500MB
  analyzer          8.5%      256MB
  
  Solutions:
  ──────────
  
  For Kafka:
  ──────────
  - Reduce retention period
  - Lower number of partitions
  - Increase heap size if needed
  
  For Prometheus:
  ───────────────
  - Reduce scrape interval
  - Lower retention time
  - Add memory limits
  
  For Services:
  ─────────────
  - Check for memory leaks in logs
  - Restart affected services
  - Add resource limits in docker-compose.yml
```

---

## Log Analysis

### Viewing Logs

```bash
# All services
docker compose logs

# Specific service
docker compose logs <service-name>

# Follow logs
docker compose logs -f <service-name>

# Last N lines
docker compose logs --tail=100 <service-name>

# With timestamps
docker compose logs -t <service-name>
```

### Common Error Patterns

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    COMMON ERROR PATTERNS                                │
├─────────────────────────────────────────────────────────────────────────┤
│  Pattern                        │  Likely Cause                        │
├─────────────────────────────────┼──────────────────────────────────────┤
│  "connection refused"           │  Target service not running          │
│  "no such host"                 │  Wrong hostname or DNS issue         │
│  "context deadline exceeded"    │  Service too slow or timeout         │
│  "EOF"                          │  Connection dropped unexpectedly     │
│  "permission denied"            │  File/volume permission issue        │
│  "out of memory"                │  Need more RAM or memory limit       │
│  "too many open files"          │  Increase ulimit                     │
└─────────────────────────────────────────────────────────────────────────┘
```

### Getting Help

If you're still stuck:

1. Check the full logs: `docker compose logs > logs.txt`
2. Review the [Architecture](./ARCHITECTURE.md) to understand dependencies
3. Check the [Deployment Guide](./DEPLOYMENT.md) for correct setup
4. Verify environment variables match your setup

---

## Related Documentation

- [Architecture Overview](./ARCHITECTURE.md)
- [Deployment Guide](./DEPLOYMENT.md)
- [API Reference](./API_REFERENCE.md)
- [Alert System](./ALERTING.md)
