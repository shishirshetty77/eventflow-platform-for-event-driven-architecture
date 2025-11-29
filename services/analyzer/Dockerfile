# Multi-stage build for Analyzer service
FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.work go.work.sum ./
COPY pkg/shared/go.mod pkg/shared/go.sum ./pkg/shared/
COPY services/analyzer/go.mod services/analyzer/go.sum ./services/analyzer/

RUN cd services/analyzer && go mod download

COPY pkg/shared ./pkg/shared
COPY services/analyzer ./services/analyzer

RUN cd services/analyzer && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /analyzer ./cmd/...

FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /analyzer .

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080 9090

CMD ["./analyzer"]
