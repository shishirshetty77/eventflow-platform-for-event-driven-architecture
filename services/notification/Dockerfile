# Multi-stage build for Notification service
FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.work go.work.sum ./
COPY pkg/shared/go.mod pkg/shared/go.sum ./pkg/shared/
COPY services/notification/go.mod services/notification/go.sum ./services/notification/

RUN cd services/notification && go mod download

COPY pkg/shared ./pkg/shared
COPY services/notification ./services/notification

RUN cd services/notification && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /notification ./cmd/...

FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /notification .

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080 9090

CMD ["./notification"]
