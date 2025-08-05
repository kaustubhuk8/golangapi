.PHONY: build run test clean docker-build docker-up docker-down help

# Variables
APP_NAME := manifold-api
DOCKER_COMPOSE := docker-compose
GO_FILES := $(shell find . -name '*.go' -type f)

# Default target
help: ## Show this help message
	@echo "🚀 MANIFOLD API - AVAILABLE COMMANDS"
	@echo "====================================="
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-20s %s\n", $$1, $$2}'
	@echo ""
	@echo "🎯 INTERVIEW PREPARATION WORKFLOW:"
	@echo "1. make fresh-start     # Complete reset and fresh start"
	@echo "2. make load-test-quick # Quick test (5 minutes)"
	@echo "3. make load-test-full  # Full assessment test (60-90 minutes)"
	@echo ""

build: ## Build the Go application
	@echo "Building $(APP_NAME)..."
	@go mod tidy
	@go build -o bin/$(APP_NAME) ./cmd/api
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
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o bin/$(APP_NAME) ./cmd/api
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

# Load testing
load-test: ## Run quick load test (100 requests, 20 concurrent)
	@echo "Running quick load test..."
	@chmod +x load_test.sh
	@./load_test.sh

load-test-quick: ## Run very quick load test (50 requests, 10 concurrent) - ~5 minutes
	@echo "Building load test tool..."
	@go build -o bin/load_test ./cmd/load_test
	@echo "Running quick load test (50 requests, 10 concurrent workers)..."
	@./bin/load_test quick

load-test-full: ## Run full load test (5000 requests, 100 concurrent) - ~60-90 minutes
	@echo "Building load test tool..."
	@go build -o bin/load_test ./cmd/load_test
	@echo "Running full load test (5000 requests, 100 concurrent workers)..."
	@echo "⚠️  This will take 60-90 minutes due to 1-minute streaming per request"
	@./bin/load_test

load-test-custom: ## Run custom load test (usage: make load-test-custom REQUESTS=1000 CONCURRENT=50)
	@echo "Running custom load test..."
	@chmod +x load_test.sh
	@REQUESTS=$${REQUESTS:-100} CONCURRENT=$${CONCURRENT:-20} ./load_test.sh

# Fresh start for load testing
fresh-start: ## Complete reset and fresh start for load testing
	@echo "🧹 MANIFOLD API - FRESH START FOR LOAD TESTING"
	@echo "=============================================="
	@echo ""
	@echo "Stopping all services..."
	@docker-compose down -v 2>/dev/null || true
	@echo "✅ Services stopped"
	@echo ""
	@echo "Cleaning up Docker resources..."
	@docker system prune -f 2>/dev/null || true
	@docker volume prune -f 2>/dev/null || true
	@echo "✅ Docker cleanup completed"
	@echo ""
	@echo "Building fresh application..."
	@docker-compose build --no-cache app
	@echo "✅ Application built successfully"
	@echo ""
	@echo "Starting fresh services..."
	@docker-compose up -d
	@echo "✅ Services started"
	@echo ""
	@echo "Waiting for services to be ready..."
	@sleep 15
	@echo ""
	@echo "Clearing Redis cache..."
	@docker exec -it manifoldtest-redis-1 redis-cli FLUSHALL 2>/dev/null || true
	@echo "✅ Redis cache cleared"
	@echo ""
	@echo "Verifying fresh start..."
	@echo ""
	@echo "API Health:"
	@curl -s http://localhost:8080/health | jq . 2>/dev/null || echo "API not ready yet, waiting..."
	@sleep 5
	@echo ""
	@echo "User Stats (user1):"
	@curl -s -H "X-User-Id: user1" http://localhost:8080/user/stats | jq . 2>/dev/null || echo "User stats not ready yet"
	@echo ""
	@echo "=============================================="
	@echo "🚀 FRESH START COMPLETE!"
	@echo ""
	@echo "Your application is ready for testing!"
	@echo ""
	@echo "📊 Quick load test (5 minutes):"
	@echo "   make load-test-quick"
	@echo ""
	@echo "🎯 Full assessment test (60-90 minutes):"
	@echo "   make load-test-full"
	@echo ""
	@echo "🔧 Manual testing:"
	@echo "   curl -X POST -H 'X-User-Id: user1' --no-buffer http://localhost:8080/generate-data"
	@echo ""
	@echo "📈 Check stats:"
	@echo "   curl -H 'X-User-Id: user1' http://localhost:8080/user/stats"
	@echo ""
	@echo "=============================================="

# All-in-one commands
quick-start: check-deps docker-up ## Quick start everything
	@echo "Waiting for services to be ready..."
	@sleep 10
	@make stats
	@echo "\n🚀 Manifold API is ready!"
	@echo "📋 Try: curl -X POST -H 'X-User-Id: user1' --no-buffer http://localhost:8080/generate-data"
	@echo "📊 Run load test: make load-test"

full-test: docker-up load-test docker-down ## Run full test suite including load test

