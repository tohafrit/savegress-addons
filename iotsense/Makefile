.PHONY: build run test clean docker-build docker-run lint fmt

# Binary name
BINARY=iotsense
BUILD_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Build flags
LDFLAGS=-ldflags "-s -w"

all: build

build:
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/iotsense

run: build
	@echo "Running $(BINARY)..."
	./$(BUILD_DIR)/$(BINARY)

test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

lint:
	@echo "Running linter..."
	@golangci-lint run ./... || true

fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

docker-build:
	@echo "Building Docker image..."
	docker build -t iotsense:latest -f docker/Dockerfile .

docker-run:
	@echo "Running with Docker Compose..."
	docker-compose -f docker/docker-compose.yml up -d

docker-stop:
	@echo "Stopping Docker Compose..."
	docker-compose -f docker/docker-compose.yml down

docker-logs:
	docker-compose -f docker/docker-compose.yml logs -f iotsense

# Development helpers
dev: build
	ENVIRONMENT=development ./$(BUILD_DIR)/$(BINARY)

# API testing
test-health:
	@curl -s http://localhost:3006/health | jq .

test-register-device:
	@curl -s -X POST http://localhost:3006/api/v1/devices \
		-H "Content-Type: application/json" \
		-d '{"name":"Temperature Sensor 1","type":"sensor","model":"TMP36","manufacturer":"Analog Devices"}' | jq .

test-send-telemetry:
	@curl -s -X POST http://localhost:3006/api/v1/telemetry \
		-H "Content-Type: application/json" \
		-d '{"device_id":"$(DEVICE_ID)","metric":"temperature","value":23.5,"unit":"celsius"}' | jq .

test-create-rule:
	@curl -s -X POST http://localhost:3006/api/v1/rules \
		-H "Content-Type: application/json" \
		-d '{"name":"High Temperature","metric":"temperature","condition":{"type":"threshold","operator":"gt","value":30},"severity":"warning","cooldown":"5m","enabled":true}' | jq .
