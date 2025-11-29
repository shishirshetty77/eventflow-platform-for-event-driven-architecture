# Multi-stage build for UI Backend service
FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.work go.work.sum ./
COPY pkg/shared/go.mod pkg/shared/go.sum ./pkg/shared/
COPY services/ui-backend/go.mod services/ui-backend/go.sum ./services/ui-backend/

RUN cd services/ui-backend && go mod download

COPY pkg/shared ./pkg/shared
COPY services/ui-backend ./services/ui-backend

RUN cd services/ui-backend && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ui-backend ./cmd/...

FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /ui-backend .

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080 9090

CMD ["./ui-backend"]
