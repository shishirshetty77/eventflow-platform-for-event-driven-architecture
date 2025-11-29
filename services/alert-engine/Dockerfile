# Multi-stage build for Alert Engine service
FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.work go.work.sum ./
COPY pkg/shared/go.mod pkg/shared/go.sum ./pkg/shared/
COPY services/alert-engine/go.mod services/alert-engine/go.sum ./services/alert-engine/

RUN cd services/alert-engine && go mod download

COPY pkg/shared ./pkg/shared
COPY services/alert-engine ./services/alert-engine

RUN cd services/alert-engine && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /alert-engine ./cmd/...

FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /alert-engine .

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080 9090

CMD ["./alert-engine"]
