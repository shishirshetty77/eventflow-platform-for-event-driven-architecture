# Multi-stage build for Auth service
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Copy go workspace files
COPY go.work go.work.sum ./
COPY pkg/shared/go.mod pkg/shared/go.sum ./pkg/shared/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/

# Download dependencies
RUN cd services/auth && go mod download

# Copy source code
COPY pkg/shared ./pkg/shared
COPY services/auth ./services/auth

# Build
RUN cd services/auth && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /auth ./cmd/...

# Final image
FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /auth .

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080 9090

CMD ["./auth"]
