.PHONY: build run test clean docker-build docker-run docker-stop lint fmt tidy

BINARY_NAME=healthsync
BUILD_DIR=bin
DOCKER_IMAGE=savegress/healthsync

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Build flags
LDFLAGS=-ldflags "-w -s"

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/healthsync

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

tidy:
	@echo "Tidying modules..."
	$(GOMOD) tidy

fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

lint:
	@echo "Linting code..."
	$(GOLINT) run ./...

# Docker commands
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):latest -f docker/Dockerfile .

docker-run:
	@echo "Starting services with docker-compose..."
	docker-compose -f docker/docker-compose.yml up -d

docker-stop:
	@echo "Stopping services..."
	docker-compose -f docker/docker-compose.yml down

docker-logs:
	docker-compose -f docker/docker-compose.yml logs -f

docker-clean:
	@echo "Cleaning Docker resources..."
	docker-compose -f docker/docker-compose.yml down -v --rmi local

# Development
dev:
	@echo "Starting development server..."
	HEALTHSYNC_CONFIG=docker/healthsync.yaml $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/healthsync && ./$(BUILD_DIR)/$(BINARY_NAME)

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOGET) -v ./...

# Generate mocks (if needed)
generate:
	@echo "Generating code..."
	$(GOCMD) generate ./...

help:
	@echo "HealthSync - Healthcare Data Compliance"
	@echo ""
	@echo "Usage:"
	@echo "  make build         Build the binary"
	@echo "  make run           Build and run"
	@echo "  make test          Run tests"
	@echo "  make test-coverage Run tests with coverage report"
	@echo "  make clean         Clean build artifacts"
	@echo "  make tidy          Tidy Go modules"
	@echo "  make fmt           Format code"
	@echo "  make lint          Run linter"
	@echo "  make docker-build  Build Docker image"
	@echo "  make docker-run    Start with docker-compose"
	@echo "  make docker-stop   Stop docker-compose services"
	@echo "  make docker-logs   View docker-compose logs"
	@echo "  make docker-clean  Remove Docker resources"
	@echo "  make dev           Run in development mode"
	@echo "  make deps          Install dependencies"
	@echo "  make help          Show this help"
