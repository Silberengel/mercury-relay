# Mercury Relay Makefile

.PHONY: build clean test run dev docker-build docker-up docker-down help

# Variables
BINARY_NAME=mercury-relay
ADMIN_BINARY=mercury-admin
TEST_GEN_BINARY=test-data-gen
DOCKER_COMPOSE=docker-compose
GO=go

# Build all binaries
build:
	$(GO) build -o $(BINARY_NAME) ./cmd/mercury-relay
	$(GO) build -o $(ADMIN_BINARY) ./cmd/mercury-admin
	$(GO) build -o $(TEST_GEN_BINARY) ./cmd/test-data-gen

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME) $(ADMIN_BINARY) $(TEST_GEN_BINARY)
	$(GO) clean

# Run tests
test:
	$(GO) test ./...

# Run the relay locally
run:
	$(GO) run ./cmd/mercury-relay/main.go

# Run admin interface
admin:
	$(GO) run ./cmd/mercury-admin/main.go --tui

# Generate test data
test-data:
	$(GO) run ./cmd/test-data-gen/main.go --count 100 --persona random

# Development mode (start services)
dev:
	$(DOCKER_COMPOSE) up -d rabbitmq redis postgres xftp-server tor i2p
	@echo "Services started. Run 'make run' to start the relay."

# Docker build
docker-build:
	$(DOCKER_COMPOSE) build

# Start all services
docker-up:
	$(DOCKER_COMPOSE) up -d

# Stop all services
docker-down:
	$(DOCKER_COMPOSE) down

# View logs
logs:
	$(DOCKER_COMPOSE) logs -f

# View relay logs
logs-relay:
	$(DOCKER_COMPOSE) logs -f mercury-relay

# Check service status
status:
	$(DOCKER_COMPOSE) ps

# Get Tor .onion address
tor-address:
	$(DOCKER_COMPOSE) exec tor cat /var/lib/tor/mercury_relay/hostname 2>/dev/null || echo "Tor hidden service not ready"

# Get I2P address
i2p-address:
	$(DOCKER_COMPOSE) logs i2p | grep "Tunnel" | tail -1 || echo "I2P tunnel not ready"

# Show all addresses
addresses:
	@echo "=== Relay Addresses ==="
	@echo "Tor .onion:"
	@make tor-address
	@echo ""
	@echo "I2P address:"
	@make i2p-address
	@echo ""
	@echo "Direct IP: http://localhost:8080"
	@echo "Admin API: http://localhost:8081"

# Install dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Format code
fmt:
	$(GO) fmt ./...

# Lint code
lint:
	$(GO) vet ./...

# Security scan
security:
	$(GO) list -json -deps ./... | nancy sleuth

# Full test suite
test-all: test lint security

# Reset everything
reset: docker-down clean
	$(DOCKER_COMPOSE) down -v
	docker system prune -f

# Help
help:
	@echo "Mercury Relay - Available Commands:"
	@echo ""
	@echo "Build Commands:"
	@echo "  build          Build all binaries"
	@echo "  clean          Clean build artifacts"
	@echo "  docker-build   Build Docker images"
	@echo ""
	@echo "Development:"
	@echo "  dev            Start infrastructure services"
	@echo "  run            Run relay locally"
	@echo "  admin          Run admin interface"
	@echo "  test-data      Generate test data"
	@echo ""
	@echo "Docker Commands:"
	@echo "  docker-up      Start all services"
	@echo "  docker-down    Stop all services"
	@echo "  logs           View all logs"
	@echo "  logs-relay     View relay logs"
	@echo "  status         Check service status"
	@echo ""
	@echo "Utilities:"
	@echo "  addresses      Show all relay addresses"
	@echo "  tor-address    Get Tor .onion address"
	@echo "  i2p-address    Get I2P address"
	@echo ""
	@echo "Testing:"
	@echo "  test           Run tests"
	@echo "  test-all       Run full test suite"
	@echo "  lint           Lint code"
	@echo "  security       Security scan"
	@echo ""
	@echo "Maintenance:"
	@echo "  deps           Install dependencies"
	@echo "  fmt            Format code"
	@echo "  reset          Reset everything"
	@echo "  help           Show this help"
