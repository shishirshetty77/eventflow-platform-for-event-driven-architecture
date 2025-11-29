# Multi-stage build for Orders service
FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.work go.work.sum ./
COPY pkg/shared/go.mod pkg/shared/go.sum ./pkg/shared/
COPY services/orders/go.mod services/orders/go.sum ./services/orders/

RUN cd services/orders && go mod download

COPY pkg/shared ./pkg/shared
COPY services/orders ./services/orders

RUN cd services/orders && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /orders ./cmd/...

FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /orders .

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080 9090

CMD ["./orders"]
