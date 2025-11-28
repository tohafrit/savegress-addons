.PHONY: help dev-infra dev-backend dev-vscode dev-frontend docker-up docker-down docker-logs build test clean

# Default target
help:
	@echo "ChainLens Development Commands"
	@echo ""
	@echo "Development:"
	@echo "  make dev-infra      - Start PostgreSQL and Redis"
	@echo "  make dev-backend    - Run backend in development mode"
	@echo "  make dev-vscode     - Compile VS Code extension"
	@echo "  make dev-frontend   - Run frontend in development mode"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up      - Start all services with Docker"
	@echo "  make docker-down    - Stop all Docker services"
	@echo "  make docker-logs    - View Docker logs"
	@echo ""
	@echo "Build:"
	@echo "  make build          - Build all components"
	@echo "  make build-backend  - Build backend binary"
	@echo "  make build-vscode   - Build VS Code extension"
	@echo "  make build-frontend - Build frontend for production"
	@echo ""
	@echo "Test:"
	@echo "  make test           - Run all tests"
	@echo "  make test-backend   - Run backend tests"
	@echo "  make test-vscode    - Run VS Code extension tests"
	@echo ""
	@echo "Other:"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make migrate        - Run database migrations"

# Development
dev-infra:
	docker-compose -f docker/docker-compose.yml up -d postgres redis

dev-backend:
	cd backend && go run cmd/chainlens/main.go

dev-vscode:
	cd vscode-extension && npm install && npm run watch

dev-frontend:
	cd frontend && npm install && npm run dev

# Docker
docker-up:
	docker-compose -f docker/docker-compose.yml up -d

docker-down:
	docker-compose -f docker/docker-compose.yml down

docker-logs:
	docker-compose -f docker/docker-compose.yml logs -f

docker-build:
	docker-compose -f docker/docker-compose.yml build

# Build
build: build-backend build-vscode build-frontend

build-backend:
	cd backend && go build -o bin/chainlens cmd/chainlens/main.go

build-vscode:
	cd vscode-extension && npm install && npm run compile

build-frontend:
	cd frontend && npm install && npm run build

# Package VS Code extension
package-vscode:
	cd vscode-extension && npm run package

# Test
test: test-backend test-vscode

test-backend:
	cd backend && go test ./...

test-vscode:
	cd vscode-extension && npm test

# Database
migrate:
	psql $(DATABASE_URL) -f backend/migrations/001_init.sql

migrate-docker:
	docker exec -i chainlens-postgres psql -U chainlens -d chainlens < backend/migrations/001_init.sql

# Clean
clean:
	rm -rf backend/bin
	rm -rf vscode-extension/out
	rm -rf vscode-extension/node_modules
	rm -rf frontend/.next
	rm -rf frontend/node_modules

# Install dependencies
deps:
	cd backend && go mod download
	cd vscode-extension && npm install
	cd frontend && npm install

# Lint
lint: lint-backend lint-vscode

lint-backend:
	cd backend && golangci-lint run

lint-vscode:
	cd vscode-extension && npm run lint

# Format
fmt:
	cd backend && go fmt ./...
	cd vscode-extension && npm run lint -- --fix
