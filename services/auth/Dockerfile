# =============================================================================
# Auth Service - Production Dockerfile
# Multi-stage build with security best practices
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Builder
# -----------------------------------------------------------------------------
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Create appuser for the final image
RUN adduser -D -g '' -s /sbin/nologin -u 10001 appuser

WORKDIR /build

# Copy go workspace and module files first (better layer caching)
COPY go.work go.work.sum ./
COPY pkg/shared/go.mod pkg/shared/go.sum ./pkg/shared/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/

# Download dependencies
WORKDIR /build/services/auth
RUN go mod download && go mod verify

# Copy source code
WORKDIR /build
COPY pkg/shared ./pkg/shared
COPY services/auth ./services/auth

# Build the binary with optimizations
WORKDIR /build/services/auth
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o /app/auth \
    ./cmd/...

# -----------------------------------------------------------------------------
# Stage 2: Production Runtime
# -----------------------------------------------------------------------------
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    && rm -rf /var/cache/apk/*

# Import user and group from builder
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Set working directory
WORKDIR /app

# Copy the binary
COPY --from=builder /app/auth .

# Use non-root user
USER appuser:appuser

# Expose ports (HTTP and Metrics)
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set default environment variables
ENV ENVIRONMENT=production \
    SERVICE_NAME=auth \
    HTTP_PORT=8080 \
    METRICS_PORT=9090 \
    LOG_LEVEL=info

# Run the binary
ENTRYPOINT ["./auth"]
