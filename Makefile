.PHONY: help test test-coverage test-be test-fe coverage-report coverage-html clean

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Test targets
test: test-be test-fe ## Run all tests (backend + frontend)

test-be: ## Run backend tests with coverage
	@echo "🧪 Running backend tests..."
	@go test -p 1 -v -coverprofile=coverage.out -covermode=atomic ./...
	@$(MAKE) coverage-report

test-fe: ## Run frontend tests with coverage
	@echo "🧪 Running frontend tests..."
	@cd src/ui && npm run test -- --coverage

test-coverage: ## Run tests and check coverage thresholds
	@echo "📊 Running tests with coverage checks..."
	@$(MAKE) test-be
	@$(MAKE) check-coverage-be
	@$(MAKE) test-fe

check-coverage-be: ## Check backend coverage threshold (80%)
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print substr($$3, 1, length($$3)-1)}'); \
	THRESHOLD=80.0; \
	echo "Backend coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < $$THRESHOLD" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below threshold $$THRESHOLD%"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets threshold $$THRESHOLD%"; \
	fi

coverage-report: ## Generate coverage report
	@echo "📊 Generating coverage report..."
	@go tool cover -func=coverage.out | tail -10

coverage-html: ## Generate HTML coverage report
	@echo "🌐 Generating HTML coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

coverage-summary: ## Show coverage summary
	@echo "📈 Coverage Summary:"
	@go tool cover -func=coverage.out | grep total

# Development targets
dev: ## Start development environment
	@echo "🚀 Starting development environment..."
	@./scripts/start.sh

restart: ## Restart development environment
	@echo "🔄 Restarting development environment..."
	@./scripts/restart.sh

# Database targets
db-migrate: ## Run database migrations
	@echo "🗄️  Running database migrations..."
	@go run src/migrations/migrate.go up

db-seed: ## Seed database with test data
	@echo "🌱 Seeding database..."
	@go run src/migrations/migrate.go seed

# Clean targets
clean: ## Clean build artifacts and coverage files
	@echo "🧹 Cleaning..."
	@rm -f coverage.out coverage.txt coverage.html
	@rm -rf src/ui/coverage/
	@rm -rf dist/
	@echo "✅ Clean complete"

clean-docker: ## Stop and remove Docker containers
	@echo "🐳 Cleaning Docker containers..."
	@docker compose down -v
	@echo "✅ Docker clean complete"

# Lint targets
lint: lint-be lint-fe ## Run all linters

lint-be: ## Run backend linter
	@echo "🔍 Running Go linter..."
	@golangci-lint run ./... || echo "⚠️  Please install golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

lint-fe: ## Run frontend linter
	@echo "🔍 Running frontend linter..."
	@cd src/ui && npm run lint

# Format targets
fmt: ## Format code
	@echo "✨ Formatting Go code..."
	@go fmt ./...
	@echo "✨ Formatting frontend code..."
	@cd src/ui && npm run prettier

# Build targets
build: ## Build all services
	@echo "🔨 Building services..."
	@go build -o bin/api src/ce/api/main.go
	@go build -o bin/hosting src/ce/hosting/main.go
	@go build -o bin/workerserver src/ce/workerserver/main.go
	@cd src/ui && npm run build

# Install targets
install: ## Install dependencies
	@echo "📦 Installing dependencies..."
	@go mod download
	@cd src/ui && npm install

# Git hooks
setup-hooks: ## Setup git hooks
	@echo "🪝 Setting up git hooks..."
	@git config core.hooksPath .githooks || true
	@chmod +x .githooks/* || true
	@echo "✅ Git hooks setup complete"
