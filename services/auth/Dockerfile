# =============================================================================
# Auth Service - Production Dockerfile
# Multi-stage build with security best practices
# =============================================================================

FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata && update-ca-certificates
RUN adduser -D -g '' -s /sbin/nologin -u 10001 appuser

WORKDIR /build

# Copy shared package first
COPY pkg/shared ./pkg/shared

# Copy service module files and download dependencies
COPY services/auth/go.mod services/auth/go.sum* ./services/auth/
WORKDIR /build/services/auth
RUN go mod download

# Copy service source code and build
WORKDIR /build
COPY services/auth ./services/auth

WORKDIR /build/services/auth
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o /app/auth ./cmd/...

# -----------------------------------------------------------------------------
# Production Runtime
# -----------------------------------------------------------------------------
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata curl && rm -rf /var/cache/apk/*

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

WORKDIR /app
COPY --from=builder /app/auth .

USER appuser:appuser
EXPOSE 8080 9090

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENV ENVIRONMENT=production \
    SERVICE_NAME=auth \
    HTTP_PORT=8080 \
    METRICS_PORT=9090 \
    LOG_LEVEL=info

ENTRYPOINT ["./auth"]
