.PHONY: build run test clean docker-build docker-up docker-down help

# Variables
APP_NAME := manifold-api
DOCKER_COMPOSE := docker-compose
GO_FILES := $(shell find . -name '*.go' -type f)

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the Go application
	@echo "Building $(APP_NAME)..."
	@go mod tidy
	@go build -o bin/$(APP_NAME) .
	@echo "Build complete: bin/$(APP_NAME)"

run: build ## Run the application locally
	@echo "Starting $(APP_NAME)..."
	@./bin/$(APP_NAME)

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning up..."
	@rm -rf bin/
	@docker system prune -f

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(APP_NAME) .

docker-up: ## Start all services with Docker Compose
	@echo "Starting services..."
	@$(DOCKER_COMPOSE) up -d
	@echo "Services started. API available at http://localhost:8080"
	@echo "Health check: curl http://localhost:8080/health"

docker-down: ## Stop all services
	@echo "Stopping services..."
	@$(DOCKER_COMPOSE) down

docker-logs: ## View logs from all services
	@$(DOCKER_COMPOSE) logs -f

docker-restart: docker-down docker-up ## Restart all services



# Development helpers
dev-setup: ## Setup development environment
	@echo "Setting up development environment..."
	@go mod download
	@go install github.com/air-verse/air@latest
	@echo "Development environment ready!"
	@echo "Run 'make dev' to start with hot reload"

dev: ## Run with hot reload (requires air)
	@air

check-deps: ## Check if external dependencies are available
	@echo "Checking dependencies..."
	@command -v docker >/dev/null 2>&1 || { echo "Docker is required but not installed"; exit 1; }
	@command -v docker-compose >/dev/null 2>&1 || { echo "Docker Compose is required but not installed"; exit 1; }
	@command -v go >/dev/null 2>&1 || { echo "Go is required but not installed"; exit 1; }
	@echo "All dependencies available!"

# Database helpers
db-shell: ## Connect to MySQL database
	@docker exec -it $$(docker-compose ps -q mysql) mysql -u root -prootpassword manifold

redis-shell: ## Connect to Redis
	@docker exec -it $$(docker-compose ps -q redis) redis-cli

# Monitoring and debugging
stats: ## Show API stats
	@echo "API Health:"
	@curl -s http://localhost:8080/health | jq .
	@echo "\nUser Stats (user1):"
	@curl -s -H "X-User-Id: user1" http://localhost:8080/user/stats | jq .

monitor: ## Monitor API performance
	@echo "Monitoring API (press Ctrl+C to stop)..."
	@while true; do \
		echo "=== $$(date) ==="; \
		curl -s http://localhost:8080/health | jq '.status, .database, .redis'; \
		sleep 5; \
	done

# Benchmarking
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Production helpers
production-build: ## Build for production
	@echo "Building for production..."
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o bin/$(APP_NAME) .
	@echo "Production build complete"

security-scan: ## Run security scan
	@echo "Running security scan..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@govulncheck ./...

lint: ## Run linter
	@echo "Running linter..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run

format: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@go mod tidy

# All-in-one commands
quick-start: check-deps docker-up ## Quick start everything
	@echo "Waiting for services to be ready..."
	@sleep 10
	@make stats
	@echo "\n🚀 Manifold API is ready!"
	@echo "📋 Try: curl -X POST -H 'X-User-Id: user1' http://localhost:8080/generate-data"

full-test: docker-up docker-down

