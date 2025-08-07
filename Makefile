.PHONY: build run test docker-build docker-up docker-down

APP_NAME := manifold-api

build: ## Build the Go application
	go mod tidy
	go build -o bin/$(APP_NAME) ./cmd/api

run: build ## Run the application locally
	./bin/$(APP_NAME)

test: ## Run tests
	go test ./...

docker-build: ## Build Docker image
	docker build -t $(APP_NAME) .

docker-up: ## Start services via Docker Compose
	docker-compose up -d

docker-down: ## Stop services
	docker-compose down

fresh-start: ## Project-local reset: stops and resets only this app
	docker-compose down -v

	rm -rf bin/

	-docker exec -it $$(docker-compose ps -q redis) redis-cli FLUSHALL

	docker-compose build --no-cache

	docker-compose up -d


