# =============================================================================
# Analyzer Service - Production Dockerfile
# Multi-stage build with security best practices
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Builder
# -----------------------------------------------------------------------------
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    && update-ca-certificates

RUN adduser -D -g '' -s /sbin/nologin -u 10001 appuser

WORKDIR /build

COPY go.work ./
COPY go.work.sum* ./
COPY pkg/shared/go.mod pkg/shared/go.sum* ./pkg/shared/
COPY services/analyzer/go.mod services/analyzer/go.sum* ./services/analyzer/

WORKDIR /build/services/analyzer
RUN go mod download && go mod verify

WORKDIR /build
COPY pkg/shared ./pkg/shared
COPY services/analyzer ./services/analyzer

WORKDIR /build/services/analyzer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o /app/analyzer \
    ./cmd/...

# -----------------------------------------------------------------------------
# Stage 2: Production Runtime
# -----------------------------------------------------------------------------
FROM alpine:3.19

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    && rm -rf /var/cache/apk/*

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

WORKDIR /app

COPY --from=builder /app/analyzer .

USER appuser:appuser

EXPOSE 8080 9090

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENV ENVIRONMENT=production \
    SERVICE_NAME=analyzer \
    HTTP_PORT=8080 \
    METRICS_PORT=9090 \
    LOG_LEVEL=info \
    REDIS_DB=0

ENTRYPOINT ["./analyzer"]
