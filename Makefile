# Makefile for CEL Go Scanner

# Variables
GO_VERSION = 1.24.0


# Default target
.PHONY: all
all: test


# Run unit tests
.PHONY: test-unit
test:
	@echo "Running tests..."
	go test ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Run integration tests
.PHONY: test-integration
test-integration:
	@echo "Integration tests not implemented yet"

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	go mod tidy


# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build & Test:"
	@echo "  test-unit           - Run unit tests"
	@echo "  test-coverage       - Run tests with coverage report"
	@echo "  test-integration    - Run integration tests (noop)"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  tidy          - Tidy dependencies"
	@echo ""
	@echo "Other:"
	@echo "  clean         - Clean build artifacts"
	@echo "  help          - Show this help message" 