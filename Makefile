.PHONY: build test test-verbose test-coverage clean run fmt lint help env-setup env-check

# Variables
BINARY_NAME=gotel
BUILD_DIR=bin
MAIN_PACKAGE=./cmd/gotel

# Load environment variables from .env file if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	@go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. ./...

# Run the application
run: build
	@echo "Starting $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

# Run in development mode (direct go run)
dev:
	@echo "Running in development mode..."
	@go run $(MAIN_PACKAGE)

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Update dependencies
deps:
	@echo "Updating dependencies..."
	@go mod tidy
	@go mod download

# Install dependencies
install-deps:
	@echo "Installing dependencies..."
	@go mod download

# Docker build
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):latest .

# Start monitoring stack
monitoring-up:
	@echo "Starting monitoring stack..."
	@cd setup && docker-compose up -d

# Stop monitoring stack
monitoring-down:
	@echo "Stopping monitoring stack..."
	@cd setup && docker-compose down

# Full test with monitoring stack
test-integration: monitoring-up
	@echo "Running integration tests..."
	@sleep 10  # Wait for services to start
	@$(MAKE) test
	@$(MAKE) monitoring-down

# Generate test data
test-load:
	@echo "Generating test load..."
	@for i in {1..10}; do \
		curl -s http://localhost:8080/ > /dev/null; \
		echo "Request $$i sent"; \
		sleep 1; \
	done

# Help
help:
	@echo "GoTel Build System"
	@echo "=================="
	@echo ""
	@echo "Available targets:"
	@echo "  build              Build the application"
	@echo "  test               Run tests"
	@echo "  test-verbose       Run tests with verbose output"
	@echo "  test-coverage      Run tests with coverage report"
	@echo "  bench              Run benchmarks"
	@echo "  run                Build and run the application"
	@echo "  dev                Run in development mode"
	@echo "  fmt                Format code"
	@echo "  lint               Lint code (requires golangci-lint)"
	@echo "  clean              Clean build artifacts"
	@echo "  deps               Update dependencies"
	@echo "  install-deps       Install dependencies"
	@echo "  docker-build       Build Docker image"
	@echo "  monitoring-up      Start monitoring stack"
	@echo "  monitoring-down    Stop monitoring stack"
	@echo "  test-integration   Run integration tests with monitoring"
	@echo "  test-load          Generate test load"
	@echo "  env-setup          Setup environment configuration"
	@echo "  env-check          Check environment configuration"
	@echo "  help               Show this help message"

# Environment setup and validation
env-setup:
	@echo "Setting up environment configuration..."
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env file from .env.example"; \
		echo "Please edit .env file to configure your settings"; \
	else \
		echo ".env file already exists"; \
	fi

env-check:
	@echo "Checking environment configuration..."
	@if [ -f .env ]; then \
		echo "Environment file found: .env"; \
		echo "Key configuration:"; \
		echo "  PORT: $${PORT:-8080}"; \
		echo "  PROMETHEUS_ENDPOINT: $${PROMETHEUS_ENDPOINT:-http://localhost:9090/api/v1/write}"; \
		echo "  DEBUG: $${DEBUG:-false}"; \
		echo "  ENVIRONMENT: $${ENVIRONMENT:-development}"; \
	else \
		echo "No .env file found. Run 'make env-setup' to create one."; \
	fi
