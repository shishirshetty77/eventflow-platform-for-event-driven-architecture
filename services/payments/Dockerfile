# Multi-stage build for Payments service
FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.work go.work.sum ./
COPY pkg/shared/go.mod pkg/shared/go.sum ./pkg/shared/
COPY services/payments/go.mod services/payments/go.sum ./services/payments/

RUN cd services/payments && go mod download

COPY pkg/shared ./pkg/shared
COPY services/payments ./services/payments

RUN cd services/payments && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /payments ./cmd/...

FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /payments .

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080 9090

CMD ["./payments"]
