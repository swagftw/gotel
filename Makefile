# GoTel - OpenTelemetry Metrics for Go
# Makefile for common development tasks

.PHONY: help build test clean run-examples start-stack stop-stack

# Default target
help:
	@echo "GoTel - OpenTelemetry Metrics for Go"
	@echo "Available targets:"
	@echo "  build         - Build all packages"
	@echo "  test          - Run all tests"
	@echo "  clean         - Clean build artifacts"
	@echo "  run-examples  - Run example applications"
	@echo "  start-stack   - Start OpenTelemetry, Prometheus, and Grafana stack"
	@echo "  stop-stack    - Stop the observability stack"
	@echo "  lint          - Run go fmt and go vet"

# Build all packages
build:
	@echo "Building GoTel packages..."
	go build -v ./...
	@echo "Build complete."

# Run all tests
test:
	@echo "Running tests..."
	go test ./pkg/... -v -timeout=30s
	@echo "All tests passed."

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test ./pkg/... -v -timeout=30s -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	go clean ./...
	rm -f coverage.out coverage.html
	rm -f examples/simple_usage/simple_usage
	rm -f examples/stress_demo/stress_demo
	@echo "Clean complete."

# Run example applications
run-examples: build
	@echo "Running simple usage example..."
	cd examples/simple_usage && go run main.go

# Run stress test example
run-stress: build
	@echo "Running stress test example..."
	cd examples/stress_demo && go run main.go

# Start the observability stack (OpenTelemetry Collector, Prometheus, Grafana)
start-stack:
	@echo "Starting OpenTelemetry, Prometheus, and Grafana stack..."
	cd setup && docker-compose up -d
	@echo "Stack started. Services available at:"
	@echo "  OpenTelemetry Collector: http://localhost:4318"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Grafana: http://localhost:3000 (admin/admin)"

# Stop the observability stack
stop-stack:
	@echo "Stopping observability stack..."
	cd setup && docker-compose down
	@echo "Stack stopped."

# Lint code
lint:
	@echo "Running go fmt..."
	go fmt ./...
	@echo "Running go vet..."
	go vet ./...
	@echo "Linting complete."

# Tidy dependencies
tidy:
	@echo "Tidying go modules..."
	go mod tidy
	@echo "Dependencies tidied."

# Run all checks (build, test, lint)
check: lint build test
	@echo "All checks passed."

# Development setup
dev-setup: tidy
	@echo "Setting up development environment..."
	@echo "Installing required tools..."
	go install golang.org/x/tools/cmd/cover@latest
	@echo "Development setup complete."

# Show module information
info:
	@echo "Module: github.com/swagftw/gotel"
	@echo "Go version: $(shell go version)"
	@echo "Dependencies:"
	@go list -m all
