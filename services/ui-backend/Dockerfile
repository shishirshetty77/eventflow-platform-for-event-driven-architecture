# =============================================================================
# UI Backend Service - Production Dockerfile
# =============================================================================

FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata && update-ca-certificates
RUN adduser -D -g '' -s /sbin/nologin -u 10001 appuser

WORKDIR /build

COPY pkg/shared ./pkg/shared
COPY services/ui-backend/go.mod services/ui-backend/go.sum* ./services/ui-backend/
WORKDIR /build/services/ui-backend
RUN go mod download

WORKDIR /build
COPY services/ui-backend ./services/ui-backend

WORKDIR /build/services/ui-backend
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o /app/ui-backend ./cmd/...

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata curl && rm -rf /var/cache/apk/*

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

WORKDIR /app
COPY --from=builder /app/ui-backend .

USER appuser:appuser
EXPOSE 8080 9090

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

ENV ENVIRONMENT=production \
    SERVICE_NAME=ui-backend \
    HTTP_PORT=8080 \
    METRICS_PORT=9090 \
    REDIS_DB=2 \
    LOG_LEVEL=info

ENTRYPOINT ["./ui-backend"]
