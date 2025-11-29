# =============================================================================
# EventFlow Platform Makefile
# Production-grade distributed event-driven microservices system
# =============================================================================

.PHONY: all build test clean run-all stop-all docker-build docker-up docker-down \
        docker-restart docker-logs docker-ps docker-clean docker-clean-all \
        up down restart logs ps build-images status health seed-redis verify-kafka \
        lint fmt tidy deps help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Binary output directory
BIN_DIR=./bin

# Services
SERVICES=auth orders payments notification analyzer alert-engine ui-backend

# Docker
DOCKER_COMPOSE=docker-compose
DOCKER=docker

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[0;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

# Default target
all: deps tidy build test

# Help
help:
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo "$(BLUE)         EventFlow Platform - Command Reference$(NC)"
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo ""
	@echo "$(GREEN)🐳 Docker Commands (Primary):$(NC)"
	@echo "  $(YELLOW)make up$(NC)              - Start entire platform (docker-compose up --build)"
	@echo "  $(YELLOW)make down$(NC)            - Stop entire platform"
	@echo "  $(YELLOW)make restart$(NC)         - Restart all services"
	@echo "  $(YELLOW)make logs$(NC)            - View logs from all services (follow)"
	@echo "  $(YELLOW)make logs-<service>$(NC)  - View logs for specific service"
	@echo "  $(YELLOW)make ps$(NC)              - Show running containers"
	@echo "  $(YELLOW)make status$(NC)          - Show detailed container status"
	@echo "  $(YELLOW)make health$(NC)          - Run health check on all services"
	@echo ""
	@echo "$(GREEN)🔧 Docker Build Commands:$(NC)"
	@echo "  $(YELLOW)make build-images$(NC)    - Build all Docker images"
	@echo "  $(YELLOW)make rebuild$(NC)         - Force rebuild all images (no cache)"
	@echo "  $(YELLOW)make pull$(NC)            - Pull latest base images"
	@echo ""
	@echo "$(GREEN)🧹 Docker Cleanup Commands:$(NC)"
	@echo "  $(YELLOW)make docker-clean$(NC)    - Stop containers and remove network"
	@echo "  $(YELLOW)make docker-clean-all$(NC)- Full cleanup (volumes + images)"
	@echo "  $(YELLOW)make docker-prune$(NC)    - Remove dangling images/containers"
	@echo ""
	@echo "$(GREEN)🔌 Service Commands:$(NC)"
	@echo "  $(YELLOW)make restart-<svc>$(NC)   - Restart specific service"
	@echo "  $(YELLOW)make shell-<svc>$(NC)     - Open shell in service container"
	@echo "  Services: auth, orders, payments, notification, analyzer, alert-engine, ui-backend, dashboard"
	@echo ""
	@echo "$(GREEN)⚙️  Utility Commands:$(NC)"
	@echo "  $(YELLOW)make seed-redis$(NC)      - Seed Redis with default threshold rules"
	@echo "  $(YELLOW)make verify-kafka$(NC)    - Verify Kafka topics are created"
	@echo "  $(YELLOW)make wait$(NC)            - Wait for all services to be ready"
	@echo ""
	@echo "$(GREEN)📦 Local Development:$(NC)"
	@echo "  $(YELLOW)make build$(NC)           - Build all services locally"
	@echo "  $(YELLOW)make test$(NC)            - Run all tests"
	@echo "  $(YELLOW)make test-coverage$(NC)   - Run tests with coverage"
	@echo "  $(YELLOW)make lint$(NC)            - Run linter"
	@echo "  $(YELLOW)make fmt$(NC)             - Format code"
	@echo "  $(YELLOW)make deps$(NC)            - Download dependencies"
	@echo "  $(YELLOW)make tidy$(NC)            - Tidy go modules"
	@echo ""
	@echo "$(GREEN)🖥️  Dashboard Commands:$(NC)"
	@echo "  $(YELLOW)make dashboard-install$(NC)  - Install dashboard dependencies"
	@echo "  $(YELLOW)make dashboard-dev$(NC)      - Run dashboard in dev mode"
	@echo "  $(YELLOW)make dashboard-build$(NC)    - Build dashboard for production"
	@echo ""
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo "$(GREEN)Quick Start:$(NC) make up"
	@echo "$(GREEN)Access:$(NC)"
	@echo "  Dashboard:  http://localhost:3001"
	@echo "  Grafana:    http://localhost:3000 (admin/admin)"
	@echo "  Kafka UI:   http://localhost:8080"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Jaeger:     http://localhost:16686"
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"

# Dependencies
deps:
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	cd pkg/shared && $(GOMOD) download
	@for service in $(SERVICES); do \
		echo "$(YELLOW)Downloading deps for $$service...$(NC)"; \
		cd services/$$service && $(GOMOD) download && cd ../..; \
	done
	@echo "$(GREEN)Dependencies downloaded$(NC)"

# Tidy modules
tidy:
	@echo "$(BLUE)Tidying modules...$(NC)"
	cd pkg/shared && $(GOMOD) tidy
	@for service in $(SERVICES); do \
		echo "$(YELLOW)Tidying $$service...$(NC)"; \
		cd services/$$service && $(GOMOD) tidy && cd ../..; \
	done
	@echo "$(GREEN)Modules tidied$(NC)"

# Build all services
build: $(SERVICES)

$(SERVICES):
	@echo "$(BLUE)Building $@...$(NC)"
	@mkdir -p $(BIN_DIR)
	cd services/$@ && $(GOBUILD) -ldflags="-s -w" -o ../../$(BIN_DIR)/$@ ./cmd/...
	@echo "$(GREEN)Built $@$(NC)"

# Build specific service
build-auth:
	@$(MAKE) auth

build-orders:
	@$(MAKE) orders

build-payments:
	@$(MAKE) payments

build-notification:
	@$(MAKE) notification

build-analyzer:
	@$(MAKE) analyzer

build-alert-engine:
	@$(MAKE) alert-engine

build-ui-backend:
	@$(MAKE) ui-backend

# Test
test:
	@echo "$(BLUE)Running tests...$(NC)"
	$(GOTEST) -v -race ./pkg/shared/...
	@for service in $(SERVICES); do \
		echo "$(YELLOW)Testing $$service...$(NC)"; \
		cd services/$$service && $(GOTEST) -v -race ./... && cd ../..; \
	done
	@echo "$(GREEN)All tests passed$(NC)"

# Test with coverage
test-coverage:
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@mkdir -p coverage
	$(GOTEST) -v -race -coverprofile=coverage/shared.out ./pkg/shared/...
	@for service in $(SERVICES); do \
		echo "$(YELLOW)Testing $$service with coverage...$(NC)"; \
		cd services/$$service && $(GOTEST) -v -race -coverprofile=../../coverage/$$service.out ./... && cd ../..; \
	done
	@echo "$(BLUE)Merging coverage reports...$(NC)"
	@echo "mode: atomic" > coverage/coverage.out
	@for f in coverage/*.out; do \
		tail -n +2 $$f >> coverage/coverage.out 2>/dev/null || true; \
	done
	$(GOCMD) tool cover -func=coverage/coverage.out
	@echo "$(GREEN)Coverage report generated$(NC)"

# Lint
lint:
	@echo "$(BLUE)Running linter...$(NC)"
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		cd pkg/shared && $(GOLINT) run ./...; \
		for service in $(SERVICES); do \
			echo "$(YELLOW)Linting $$service...$(NC)"; \
			cd services/$$service && $(GOLINT) run ./... && cd ../..; \
		done; \
	else \
		echo "$(RED)golangci-lint not installed. Install with: brew install golangci-lint$(NC)"; \
	fi

# Format
fmt:
	@echo "$(BLUE)Formatting code...$(NC)"
	$(GOFMT) -s -w ./pkg/shared
	@for service in $(SERVICES); do \
		$(GOFMT) -s -w ./services/$$service; \
	done
	@echo "$(GREEN)Code formatted$(NC)"

# Clean
clean:
	@echo "$(BLUE)Cleaning...$(NC)"
	rm -rf $(BIN_DIR)
	rm -rf coverage
	$(GOCLEAN) ./...
	@echo "$(GREEN)Cleaned$(NC)"

# Run individual services
run-auth:
	@echo "$(BLUE)Starting auth service...$(NC)"
	cd services/auth && $(GOCMD) run ./cmd/...

run-orders:
	@echo "$(BLUE)Starting orders service...$(NC)"
	cd services/orders && $(GOCMD) run ./cmd/...

run-payments:
	@echo "$(BLUE)Starting payments service...$(NC)"
	cd services/payments && $(GOCMD) run ./cmd/...

run-notification:
	@echo "$(BLUE)Starting notification service...$(NC)"
	cd services/notification && $(GOCMD) run ./cmd/...

run-analyzer:
	@echo "$(BLUE)Starting analyzer service...$(NC)"
	cd services/analyzer && $(GOCMD) run ./cmd/...

run-alert-engine:
	@echo "$(BLUE)Starting alert-engine service...$(NC)"
	cd services/alert-engine && $(GOCMD) run ./cmd/...

run-ui-backend:
	@echo "$(BLUE)Starting ui-backend service...$(NC)"
	cd services/ui-backend && $(GOCMD) run ./cmd/...

# Run all services (using background processes)
run-all:
	@echo "$(BLUE)Starting all services...$(NC)"
	@$(MAKE) run-auth &
	@sleep 1
	@$(MAKE) run-orders &
	@sleep 1
	@$(MAKE) run-payments &
	@sleep 1
	@$(MAKE) run-notification &
	@sleep 1
	@$(MAKE) run-analyzer &
	@sleep 1
	@$(MAKE) run-alert-engine &
	@sleep 1
	@$(MAKE) run-ui-backend &
	@echo "$(GREEN)All services started$(NC)"
	@echo "$(YELLOW)Press Ctrl+C to stop all services$(NC)"
	@wait

# Stop all services
stop-all:
	@echo "$(BLUE)Stopping all services...$(NC)"
	@pkill -f "go run.*auth" 2>/dev/null || true
	@pkill -f "go run.*orders" 2>/dev/null || true
	@pkill -f "go run.*payments" 2>/dev/null || true
	@pkill -f "go run.*notification" 2>/dev/null || true
	@pkill -f "go run.*analyzer" 2>/dev/null || true
	@pkill -f "go run.*alert-engine" 2>/dev/null || true
	@pkill -f "go run.*ui-backend" 2>/dev/null || true
	@echo "$(GREEN)All services stopped$(NC)"

# Docker targets
docker-build:
	@echo "$(BLUE)Building Docker images...$(NC)"
	@for service in $(SERVICES); do \
		echo "$(YELLOW)Building $$service image...$(NC)"; \
		$(DOCKER) build -t microservices-platform/$$service:latest -f services/$$service/Dockerfile .; \
	done
	@echo "$(GREEN)Docker images built$(NC)"

docker-up:
	@echo "$(BLUE)Starting services with docker-compose...$(NC)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)Services started$(NC)"

docker-down:
	@echo "$(BLUE)Stopping services...$(NC)"
	$(DOCKER_COMPOSE) down
	@echo "$(GREEN)Services stopped$(NC)"

docker-up-lite:
	@echo "$(BLUE)Starting services in lite mode...$(NC)"
	$(DOCKER_COMPOSE) -f docker-compose.lite.yml up -d
	@echo "$(GREEN)Services started (Lite Mode)$(NC)"

docker-down-lite:
	@echo "$(BLUE)Stopping services (Lite Mode)...$(NC)"
	$(DOCKER_COMPOSE) -f docker-compose.lite.yml down
	@echo "$(GREEN)Services stopped$(NC)"

docker-logs:
	$(DOCKER_COMPOSE) logs -f

# Infrastructure targets
infra-up:
	@echo "$(BLUE)Starting infrastructure...$(NC)"
	$(DOCKER_COMPOSE) -f docker-compose.infra.yml up -d
	@echo "$(GREEN)Infrastructure started$(NC)"
	@echo "$(YELLOW)Kafka: localhost:9092$(NC)"
	@echo "$(YELLOW)Redis: localhost:6379$(NC)"
	@echo "$(YELLOW)Kafka UI: localhost:8080$(NC)"

infra-down:
	@echo "$(BLUE)Stopping infrastructure...$(NC)"
	$(DOCKER_COMPOSE) -f docker-compose.infra.yml down
	@echo "$(GREEN)Infrastructure stopped$(NC)"

# Dashboard targets
dashboard-install:
	@echo "$(BLUE)Installing dashboard dependencies...$(NC)"
	cd dashboard && npm install
	@echo "$(GREEN)Dependencies installed$(NC)"

dashboard-dev:
	@echo "$(BLUE)Starting dashboard in dev mode...$(NC)"
	cd dashboard && npm run dev

dashboard-build:
	@echo "$(BLUE)Building dashboard...$(NC)"
	cd dashboard && npm run build
	@echo "$(GREEN)Dashboard built$(NC)"

# Generate proto files (if using protobuf)
proto:
	@echo "$(BLUE)Generating protobuf files...$(NC)"
	@if [ -d "proto" ]; then \
		protoc --go_out=. --go-grpc_out=. proto/*.proto; \
		echo "$(GREEN)Proto files generated$(NC)"; \
	else \
		echo "$(YELLOW)No proto directory found$(NC)"; \
	fi

# Verify everything works
verify: deps tidy fmt lint test
	@echo "$(GREEN)All verifications passed$(NC)"

# Create release
release: verify build
	@echo "$(BLUE)Creating release...$(NC)"
	@mkdir -p release
	@for service in $(SERVICES); do \
		cp $(BIN_DIR)/$$service release/; \
	done
	@tar -czvf release/microservices-platform.tar.gz -C release $(SERVICES)
	@echo "$(GREEN)Release created at release/microservices-platform.tar.gz$(NC)"

# =============================================================================
# PRIMARY DOCKER TARGETS (Recommended for running the platform)
# =============================================================================

# Start the entire platform (main command)
up:
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo "$(BLUE)       Starting EventFlow Platform$(NC)"
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo ""
	@echo "$(YELLOW)Building and starting all services...$(NC)"
	$(DOCKER_COMPOSE) up --build -d
	@echo ""
	@echo "$(GREEN)Platform started successfully!$(NC)"
	@echo ""
	@echo "$(BLUE)Services are starting up. Run 'make health' to check status.$(NC)"
	@echo ""
	@echo "$(GREEN)Access Points:$(NC)"
	@echo "  Dashboard:     http://localhost:3001"
	@echo "  UI Backend:    http://localhost:8007"
	@echo "  Kafka UI:      http://localhost:8080"
	@echo "  Redis UI:      http://localhost:8081"
	@echo "  Grafana:       http://localhost:3000 (admin/admin)"
	@echo "  Prometheus:    http://localhost:9090"
	@echo "  Jaeger:        http://localhost:16686"
	@echo ""
	@echo "$(YELLOW)Tip: Run 'make logs' to follow service logs$(NC)"

# Stop the entire platform
down:
	@echo "$(BLUE)Stopping EventFlow Platform...$(NC)"
	$(DOCKER_COMPOSE) down
	@echo "$(GREEN)Platform stopped$(NC)"

# Start the platform in lite mode (low resource)
up-lite:
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo "$(BLUE)       Starting EventFlow Platform (Lite Mode)$(NC)"
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo ""
	@echo "$(YELLOW)Building and starting essential services...$(NC)"
	$(DOCKER_COMPOSE) -f docker-compose.lite.yml up --build -d
	@echo ""
	@echo "$(GREEN)Platform started successfully!$(NC)"
	@echo ""
	@echo "$(GREEN)Access Points:$(NC)"
	@echo "  Dashboard:     http://localhost:3001"
	@echo "  UI Backend:    http://localhost:8007"
	@echo ""
	@echo "$(YELLOW)Note: Observability stack is disabled in Lite Mode$(NC)"

# Stop the platform (lite mode)
down-lite:
	@echo "$(BLUE)Stopping EventFlow Platform (Lite Mode)...$(NC)"
	$(DOCKER_COMPOSE) -f docker-compose.lite.yml down
	@echo "$(GREEN)Platform stopped$(NC)"

# Restart all services
restart:
	@echo "$(BLUE)Restarting EventFlow Platform...$(NC)"
	$(DOCKER_COMPOSE) restart
	@echo "$(GREEN)Platform restarted$(NC)"

# View logs (follow mode)
logs:
	$(DOCKER_COMPOSE) logs -f

# Show running containers
ps:
	@echo "$(BLUE)Running Containers:$(NC)"
	$(DOCKER_COMPOSE) ps

# Detailed status
status:
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo "$(BLUE)       EventFlow Platform Status$(NC)"
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo ""
	@docker ps --filter "name=eventflow-" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "No containers running"
	@echo ""

# Health check
health:
	@chmod +x ./deploy/local/health-check.sh
	@./deploy/local/health-check.sh

# Wait for services
wait:
	@chmod +x ./deploy/local/wait-for-services.sh
	@./deploy/local/wait-for-services.sh

# =============================================================================
# DOCKER BUILD TARGETS
# =============================================================================

# Build all Docker images
build-images:
	@echo "$(BLUE)Building Docker images...$(NC)"
	$(DOCKER_COMPOSE) build
	@echo "$(GREEN)Docker images built$(NC)"

# Force rebuild all images (no cache)
rebuild:
	@echo "$(BLUE)Rebuilding Docker images (no cache)...$(NC)"
	$(DOCKER_COMPOSE) build --no-cache
	@echo "$(GREEN)Docker images rebuilt$(NC)"

# Pull latest base images
pull:
	@echo "$(BLUE)Pulling latest base images...$(NC)"
	$(DOCKER) pull golang:1.21-alpine
	$(DOCKER) pull alpine:3.19
	$(DOCKER) pull node:20-alpine
	$(DOCKER) pull confluentinc/cp-zookeeper:7.5.0
	$(DOCKER) pull confluentinc/cp-kafka:7.5.0
	$(DOCKER) pull redis:7.2-alpine
	$(DOCKER) pull prom/prometheus:v2.47.0
	$(DOCKER) pull grafana/grafana:10.2.0
	$(DOCKER) pull jaegertracing/all-in-one:1.51
	@echo "$(GREEN)Base images pulled$(NC)"

# =============================================================================
# DOCKER CLEANUP TARGETS
# =============================================================================

# Stop and remove containers/network
docker-clean:
	@echo "$(BLUE)Cleaning up containers...$(NC)"
	$(DOCKER_COMPOSE) down --remove-orphans
	@echo "$(GREEN)Cleanup complete$(NC)"

# Full cleanup including volumes and images
docker-clean-all:
	@chmod +x ./deploy/local/cleanup.sh
	@./deploy/local/cleanup.sh --all --force

# Prune dangling resources
docker-prune:
	@echo "$(BLUE)Pruning unused Docker resources...$(NC)"
	$(DOCKER) system prune -f
	@echo "$(GREEN)Pruning complete$(NC)"

# =============================================================================
# SERVICE-SPECIFIC DOCKER TARGETS
# =============================================================================

# Restart individual services
restart-auth:
	$(DOCKER_COMPOSE) restart auth

restart-orders:
	$(DOCKER_COMPOSE) restart orders

restart-payments:
	$(DOCKER_COMPOSE) restart payments

restart-notification:
	$(DOCKER_COMPOSE) restart notification

restart-analyzer:
	$(DOCKER_COMPOSE) restart analyzer

restart-alert-engine:
	$(DOCKER_COMPOSE) restart alert-engine

restart-ui-backend:
	$(DOCKER_COMPOSE) restart ui-backend

restart-dashboard:
	$(DOCKER_COMPOSE) restart dashboard

# View individual service logs
logs-auth:
	$(DOCKER_COMPOSE) logs -f auth

logs-orders:
	$(DOCKER_COMPOSE) logs -f orders

logs-payments:
	$(DOCKER_COMPOSE) logs -f payments

logs-notification:
	$(DOCKER_COMPOSE) logs -f notification

logs-analyzer:
	$(DOCKER_COMPOSE) logs -f analyzer

logs-alert-engine:
	$(DOCKER_COMPOSE) logs -f alert-engine

logs-ui-backend:
	$(DOCKER_COMPOSE) logs -f ui-backend

logs-dashboard:
	$(DOCKER_COMPOSE) logs -f dashboard

logs-kafka:
	$(DOCKER_COMPOSE) logs -f kafka

logs-redis:
	$(DOCKER_COMPOSE) logs -f redis

# Shell into containers
shell-auth:
	$(DOCKER_COMPOSE) exec auth sh

shell-orders:
	$(DOCKER_COMPOSE) exec orders sh

shell-payments:
	$(DOCKER_COMPOSE) exec payments sh

shell-notification:
	$(DOCKER_COMPOSE) exec notification sh

shell-analyzer:
	$(DOCKER_COMPOSE) exec analyzer sh

shell-alert-engine:
	$(DOCKER_COMPOSE) exec alert-engine sh

shell-ui-backend:
	$(DOCKER_COMPOSE) exec ui-backend sh

shell-dashboard:
	$(DOCKER_COMPOSE) exec dashboard sh

shell-kafka:
	$(DOCKER_COMPOSE) exec kafka bash

shell-redis:
	$(DOCKER_COMPOSE) exec redis sh

# =============================================================================
# UTILITY TARGETS
# =============================================================================

# Seed Redis with default threshold rules
seed-redis:
	@echo "$(BLUE)Seeding Redis with default threshold rules...$(NC)"
	@chmod +x ./deploy/local/seed-redis.sh
	@./deploy/local/seed-redis.sh
	@echo "$(GREEN)Redis seeded$(NC)"

# Verify Kafka topics
verify-kafka:
	@echo "$(BLUE)Verifying Kafka topics...$(NC)"
	@chmod +x ./deploy/local/verify-kafka.sh
	@./deploy/local/verify-kafka.sh
	@echo "$(GREEN)Kafka verified$(NC)"

# Create Kafka topics manually
create-topics:
	@echo "$(BLUE)Creating Kafka topics...$(NC)"
	docker exec eventflow-kafka kafka-topics --create --if-not-exists --topic service-metrics --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1
	docker exec eventflow-kafka kafka-topics --create --if-not-exists --topic service-logs --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1
	docker exec eventflow-kafka kafka-topics --create --if-not-exists --topic alerts --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
	docker exec eventflow-kafka kafka-topics --create --if-not-exists --topic alerts-dlq --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
	@echo "$(GREEN)Kafka topics created$(NC)"

# List Kafka topics
list-topics:
	docker exec eventflow-kafka kafka-topics --list --bootstrap-server localhost:9092

# Open URLs in browser (macOS)
open-dashboard:
	open http://localhost:3001

open-grafana:
	open http://localhost:3000

open-kafka-ui:
	open http://localhost:8080

open-prometheus:
	open http://localhost:9090

open-jaeger:
	open http://localhost:16686

open-all:
	@echo "$(BLUE)Opening all UIs in browser...$(NC)"
	open http://localhost:3001
	open http://localhost:3000
	open http://localhost:8080
	open http://localhost:9090
	open http://localhost:16686
