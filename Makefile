.PHONY: docker-build docker-up docker-down monitor-check load-test-quick load-test-full fresh-start

APP_NAME := manifold-api

docker-build:
	docker build -t $(APP_NAME) .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

load-test-quick: 
	@go build -o bin/load_test ./cmd/load_test
	@echo "Running load test (50 requests, 10 concurrent workers)..."
	@./bin/load_test quick

load-test-full:
	@go build -o bin/load_test ./cmd/load_test
	@echo "Running load test (5000 requests, 100 concurrent workers)..."
	@./bin/load_test

fresh-start:
	docker-compose down -v

	rm -rf bin/

	-docker exec -it $$(docker-compose ps -q redis) redis-cli FLUSHALL

	docker-compose build --no-cache

	docker-compose up -d
