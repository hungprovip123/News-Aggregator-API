.PHONY: help build test clean docker-build docker-up docker-down deps

# Variables
DOCKER_COMPOSE_FILE = docker-compose.yml
SERVICES = auth-service news-api news-scraper api-gateway

help:
	@echo "Available commands:"
	@echo "  deps        - Download Go dependencies"
	@echo "  build       - Build all services"
	@echo "  test        - Run tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  docker-build - Build Docker images"
	@echo "  docker-up   - Start all services with Docker Compose"
	@echo "  docker-down - Stop all services"
	@echo "  logs        - Show Docker logs"
	@echo "  dev         - Start development environment"

deps:
	@echo "Downloading Go dependencies..."
	go mod tidy
	go mod download

build: deps
	@echo "Building all services..."
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		go build -o bin/$$service ./cmd/$$service; \
	done

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	docker system prune -f

docker-build:
	@echo "Building Docker images..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) build

docker-up:
	@echo "Starting services with Docker Compose..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

docker-down:
	@echo "Stopping services..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) down

logs:
	@echo "Showing Docker logs..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

dev: docker-down
	@echo "Starting development environment..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) up --build

# Development commands
dev-auth:
	@echo "Starting auth service in development mode..."
	go run ./cmd/auth-service

dev-news-api:
	@echo "Starting news API service in development mode..."
	go run ./cmd/news-api

dev-scraper:
	@echo "Starting news scraper service in development mode..."
	go run ./cmd/news-scraper

dev-gateway:
	@echo "Starting API gateway in development mode..."
	go run ./cmd/api-gateway

# Health check
health:
	@echo "Checking service health..."
	@curl -s http://localhost:8080/health | jq || echo "API Gateway not responding"

# Generate go.sum if it doesn't exist
init:
	@if [ ! -f go.sum ]; then \
		echo "Initializing Go module..."; \
		go mod tidy; \
	fi 