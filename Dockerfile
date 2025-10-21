FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mercury-relay ./cmd/mercury-relay
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mercury-admin ./cmd/mercury-admin
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o test-data-gen ./cmd/test-data-gen

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl

# Create app user
RUN adduser -D -s /bin/sh mercury

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/mercury-relay .
COPY --from=builder /app/mercury-admin .
COPY --from=builder /app/test-data-gen .

# Copy config
COPY config.yaml .

# Create logs directory
RUN mkdir -p /var/log/mercury-relay && chown mercury:mercury /var/log/mercury-relay

# Switch to non-root user
USER mercury

# Expose ports
EXPOSE 8080 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Default command
CMD ["./mercury-relay"]
