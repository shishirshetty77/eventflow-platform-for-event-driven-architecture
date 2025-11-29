# =============================================================================
# Payments Service - Production Dockerfile
# =============================================================================

FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata && update-ca-certificates
RUN adduser -D -g '' -s /sbin/nologin -u 10001 appuser

WORKDIR /build

COPY pkg/shared ./pkg/shared
COPY services/payments/go.mod services/payments/go.sum* ./services/payments/
WORKDIR /build/services/payments
RUN go mod download

WORKDIR /build
COPY services/payments ./services/payments

WORKDIR /build/services/payments
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o /app/payments ./cmd/...

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata curl && rm -rf /var/cache/apk/*

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

WORKDIR /app
COPY --from=builder /app/payments .

USER appuser:appuser
EXPOSE 8080 9090

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENV ENVIRONMENT=production \
    SERVICE_NAME=payments \
    HTTP_PORT=8080 \
    METRICS_PORT=9090 \
    LOG_LEVEL=info

ENTRYPOINT ["./payments"]
