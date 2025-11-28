.PHONY: build run test clean docker-build docker-up docker-down dev

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
BINARY_NAME=datawatch

# Build the binary
build:
	CGO_ENABLED=1 $(GOBUILD) -o $(BINARY_NAME) ./cmd/datawatch

# Run locally
run: build
	ENVIRONMENT=development ./$(BINARY_NAME)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Docker commands
docker-build:
	docker build -f docker/Dockerfile -t datawatch:latest .

docker-up:
	docker-compose -f docker/docker-compose.yml up -d

docker-down:
	docker-compose -f docker/docker-compose.yml down

docker-logs:
	docker-compose -f docker/docker-compose.yml logs -f datawatch

# Development
dev:
	ENVIRONMENT=development $(GOCMD) run ./cmd/datawatch

# Lint (requires golangci-lint)
lint:
	golangci-lint run ./...

# Generate (if needed)
generate:
	$(GOCMD) generate ./...

# Install development tools
tools:
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Help
help:
	@echo "Available commands:"
	@echo "  make build        - Build the binary"
	@echo "  make run          - Build and run locally"
	@echo "  make dev          - Run in development mode"
	@echo "  make test         - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make deps         - Download dependencies"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-up    - Start Docker Compose services"
	@echo "  make docker-down  - Stop Docker Compose services"
	@echo "  make docker-logs  - View Docker logs"
	@echo "  make lint         - Run linter"
