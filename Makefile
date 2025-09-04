.PHONY: help dev build test clean install-deps check lint

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

dev: ## Start development servers
	@echo "Starting development servers..."
	@make -j2 dev-backend dev-frontend

dev-backend: ## Start backend development server with hot reload
	cd backend && $$(go env GOPATH)/bin/air

dev-frontend: ## Start frontend development server
	cd frontend && npm run dev

build: ## Build the application
	@echo "Building application..."
	docker-compose build

build-backend: ## Build backend binary
	cd backend && CGO_ENABLED=1 go build -o bin/server cmd/server/main.go

build-frontend: ## Build frontend
	cd frontend && npm run build

test: ## Run tests
	@echo "Running tests..."
	cd backend && go test ./...
	cd frontend && npm test

check: ## Run type checking and linting
	@echo "Running checks..."
	cd backend && go vet ./...
	cd backend && go mod tidy
	cd frontend && npm run type-check
	cd frontend && npm run lint

lint: ## Run linters
	@echo "Running linters..."
	cd backend && golangci-lint run || echo "golangci-lint not installed"
	cd frontend && npm run lint

install-deps: ## Install dependencies
	@echo "Installing dependencies..."
	cd backend && go mod download
	cd frontend && npm install

clean: ## Clean build artifacts
	@echo "Cleaning..."
	cd backend && rm -rf bin/
	cd frontend && rm -rf .next/ dist/
	docker-compose down --volumes --remove-orphans
	docker system prune -f

setup: ## Set up the development environment
	@echo "Setting up development environment..."
	@make install-deps
	@if [ ! -f .env ]; then cp .env.example .env; echo "Created .env file from .env.example"; fi
	@echo "Setup complete! Please edit .env with your configuration."

docker-dev: ## Start development environment with Docker
	docker-compose up --build

docker-prod: ## Start production environment with Docker
	docker-compose --profile production up -d

logs: ## Show application logs
	docker-compose logs -f app

shell: ## Open shell in running container
	docker-compose exec app sh

# Database migrations (if needed in future)
migrate-up: ## Run database migrations up
	@echo "No migrations yet"

migrate-down: ## Run database migrations down
	@echo "No migrations yet"

# Deployment helpers
deploy-staging: ## Deploy to staging
	@echo "Deploying to staging..."
	# Add your staging deployment commands here

deploy-prod: ## Deploy to production
	@echo "Deploying to production..."
	# Add your production deployment commands here

# Security scanning
security-scan: ## Run security scans
	@echo "Running security scans..."
	cd backend && gosec ./... || echo "gosec not installed"
	cd frontend && npm audit

# Performance testing
perf-test: ## Run performance tests
	@echo "Running performance tests..."
	# Add performance testing commands here
