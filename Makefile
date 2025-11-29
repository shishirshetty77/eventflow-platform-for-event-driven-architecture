# Microservices Platform Makefile
# Production-grade distributed event-driven microservices system

.PHONY: all build test clean run-all stop-all docker-build docker-up docker-down \
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
	@echo "$(BLUE)Microservices Platform$(NC)"
	@echo ""
	@echo "$(GREEN)Available targets:$(NC)"
	@echo "  $(YELLOW)all$(NC)              - Run deps, tidy, build, and test"
	@echo "  $(YELLOW)build$(NC)            - Build all services"
	@echo "  $(YELLOW)build-<service>$(NC)  - Build specific service (auth, orders, payments, notification, analyzer, alert-engine, ui-backend)"
	@echo "  $(YELLOW)test$(NC)             - Run all tests"
	@echo "  $(YELLOW)test-coverage$(NC)    - Run tests with coverage"
	@echo "  $(YELLOW)lint$(NC)             - Run linter"
	@echo "  $(YELLOW)fmt$(NC)              - Format code"
	@echo "  $(YELLOW)tidy$(NC)             - Tidy go modules"
	@echo "  $(YELLOW)deps$(NC)             - Download dependencies"
	@echo "  $(YELLOW)clean$(NC)            - Clean build artifacts"
	@echo ""
	@echo "$(GREEN)Run targets:$(NC)"
	@echo "  $(YELLOW)run-<service>$(NC)    - Run specific service"
	@echo "  $(YELLOW)run-all$(NC)          - Run all services (requires tmux)"
	@echo "  $(YELLOW)stop-all$(NC)         - Stop all services"
	@echo ""
	@echo "$(GREEN)Docker targets:$(NC)"
	@echo "  $(YELLOW)docker-build$(NC)     - Build all Docker images"
	@echo "  $(YELLOW)docker-up$(NC)        - Start all services with docker-compose"
	@echo "  $(YELLOW)docker-down$(NC)      - Stop all services"
	@echo "  $(YELLOW)docker-logs$(NC)      - View logs from all services"
	@echo ""
	@echo "$(GREEN)Infrastructure targets:$(NC)"
	@echo "  $(YELLOW)infra-up$(NC)         - Start infrastructure (Kafka, Redis, etc.)"
	@echo "  $(YELLOW)infra-down$(NC)       - Stop infrastructure"

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
