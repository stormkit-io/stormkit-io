# ==============================================================================
# Platform Detection
# ==============================================================================

ifeq ($(OS),Windows_NT)
  ifeq '$(findstring ;,$(PATH))' ';'
    UNIX_LIKE    := FALSE
    DETECTED_OS  := Windows
  else
    UNIX_LIKE    := TRUE
    DETECTED_OS  := WSL
  endif
else
  UNIX_LIKE      := TRUE
  DETECTED_OS    := Unix
endif

# ==============================================================================
# Configuration
# ==============================================================================

# Set shell for Windows
ifeq ($(UNIX_LIKE),FALSE)
  # Suppress mise warning on PowerShell 5
  export MISE_PWSH_CHPWD_WARNING := 0
  export GO_BUILD_TAGS := windows

  SHELL       := powershell.exe
  .SHELLFLAGS := -NoProfile -NoLogo
  RUNNER_BIN  := runner.exe
else
  RUNNER_BIN := runner
endif

export STORMKIT_PROJECT_ROOT := $(CURDIR)
export STORMKIT_DEPLOYER_DIR := ${CURDIR}/build
export STORMKIT_DEPLOYER_EXECUTABLE := $(CURDIR)/bin/$(RUNNER_BIN)

# ==============================================================================
# Dependency Check Messages
# ==============================================================================

DOCKER_ERROR        := Docker is not installed. Please install Docker from https://docs.docker.com/get-started/get-docker/
DOCKER_OK           := Docker is installed
DOCKER_NOT_RUNNING  := Docker is not running. Please start Docker Desktop.
DOCKER_RUNNING      := Docker daemon is running
MISE_ERROR          := Mise is not installed. Please install Mise from https://mise.jdx.dev/getting-started.html
MISE_OK             := Mise is installed
ENV_CREATED         := .env file created from .env.example
ENV_OK              := .env file exists

# ==============================================================================
# Helper Functions (Windows PowerShell)
# ==============================================================================

ifeq ($(UNIX_LIKE),FALSE)

  define check_command
    Get-Command $(1) -ErrorAction SilentlyContinue | Out-Null; \
    if ($$?) { \
      Write-Host -NoNewline '[OK] ' -ForegroundColor Green; \
      Write-Host '$(3)' \
    } else { \
      Write-Host -NoNewline '[ERROR] ' -ForegroundColor Red; \
      Write-Host '$(2)'; \
      exit 1 \
    }
  endef

  define check_docker_running
    docker ps 2>&1 | Out-Null; \
    if ($$?) { \
      Write-Host -NoNewline '[OK] ' -ForegroundColor Green; \
      Write-Host '$(2)' \
    } else { \
      Write-Host -NoNewline '[ERROR] ' -ForegroundColor Red; \
      Write-Host '$(1)'; \
      exit 1 \
    }
  endef

  define check_env_file
    if (!(Test-Path '.env')) { \
      Copy-Item '.env.example' '.env'; \
      Write-Host -NoNewline '[CREATED] ' -ForegroundColor Yellow; \
      Write-Host '$(1)' \
    } else { \
      Write-Host -NoNewline '[OK] ' -ForegroundColor Green; \
      Write-Host '$(2)' \
    }
  endef

  define wait_for_service
    @$$maxAttempts = 30; \
    $$attempt = 0; \
    while ($$attempt -lt $$maxAttempts) { \
      $$attempt++; \
      try { \
        docker compose exec -T $(1) $(2) 2>&1 | Out-Null; \
        if ($$?) { \
          Write-Host "[OK] $(3) is ready" -ForegroundColor Green; \
          break; \
        } \
      } catch {} \
      if ($$attempt -lt $$maxAttempts) { \
        Write-Host "Waiting for $(3)... (attempt $$attempt/$$maxAttempts)" -ForegroundColor Yellow; \
        Start-Sleep -Seconds 2; \
      } else { \
        Write-Host "[ERROR] $(3) failed to become ready" -ForegroundColor Red; \
        exit 1; \
      } \
    }
  endef

endif

# ==============================================================================
# Helper Functions (Unix/Linux/macOS)
# ==============================================================================

ifeq ($(UNIX_LIKE),TRUE)

  define check_command
    if command -v $(1) >/dev/null 2>&1; then \
      printf '\033[0;32m[OK]\033[0m $(3)\n'; \
      true; \
    else \
      printf '\033[0;31m[ERROR]\033[0m $(2)\n'; \
      false; \
    fi
  endef

  # The sleep, kill combo is used to implement a timeout for the docker ps command
  # on macOS, where docker ps may hang if Docker Desktop is paused.
  define check_docker_running
    ( docker ps >/dev/null 2>&1 ) & pid=$$!; \
    ( sleep 3; kill -9 $$pid 2>/dev/null ) & watcher=$$!; \
    if wait $$pid 2>/dev/null; then \
      kill -9 $$watcher 2>/dev/null; \
      wait $$watcher 2>/dev/null; \
      printf '\033[0;32m[OK]\033[0m $(2)\n'; \
      true; \
    else \
      printf '\033[0;31m[ERROR]\033[0m $(1)\n'; \
      false; \
    fi
  endef

  define check_env_file
    if [ ! -f .env ]; then \
      cp .env.example .env; \
      printf '\033[0;33m[CREATED]\033[0m $(1)\n'; \
    else \
      printf '\033[0;32m[OK]\033[0m $(2)\n'; \
    fi; \
    true
  endef

  define wait_for_service
    @max_attempts=30; \
    attempt=0; \
    while [ $$attempt -lt $$max_attempts ]; do \
      attempt=$$((attempt + 1)); \
      if docker compose exec -T $(1) $(2) >/dev/null 2>&1; then \
        printf '\033[0;32m[OK]\033[0m $(3) is ready\n'; \
        break; \
      fi; \
      if [ $$attempt -lt $$max_attempts ]; then \
        printf '\033[0;33mWaiting for $(3)... (attempt %s/%s)\033[0m\n' "$$attempt" "$$max_attempts"; \
        sleep 2; \
      else \
        printf '\033[0;31m[ERROR]\033[0m $(3) failed to become ready\n'; \
        exit 1; \
      fi; \
    done
  endef

endif

# ==============================================================================
# Phony Targets
# ==============================================================================

.PHONY: help check-deps start dev print-env test test-fe test-be test-fe-watch

# ==============================================================================
# Tasks
# ==============================================================================

# Default target - show help
help:
	@echo "Available targets:"
	@echo "  check-deps  - Verify all dependencies are installed and running"
	@echo "  start       - Start Docker services (db, redis)"
	@echo "  dev         - Run check-deps and start services"
	@echo "  print-env   - Display environment variables for debugging"

# Display environment information
print-env:
	@echo "=== Environment Variables ==="
	@echo "OS: $(OS)"
	@echo "DETECTED_OS: $(DETECTED_OS)"
	@echo "UNAME_S: $(UNAME_S)"
	@echo "SHELL: $(SHELL)"
	@echo "PATH: $(PATH)"
	@echo "RUNNER_BIN: $(RUNNER_BIN)"
	@echo "STORMKIT_PROJECT_ROOT: $(STORMKIT_PROJECT_ROOT)"
	@echo "============================"

# Check all dependencies
check-deps:
	@echo "Checking dependencies on $(DETECTED_OS)..."
	@$(call check_command,docker,$(DOCKER_ERROR),$(DOCKER_OK))
	@$(call check_docker_running,$(DOCKER_NOT_RUNNING),$(DOCKER_RUNNING))
	@$(call check_command,mise,$(MISE_ERROR),$(MISE_OK))
	@$(call check_env_file,$(ENV_CREATED),$(ENV_OK))

# Start Docker services
start:
	@echo "Starting on $(DETECTED_OS)..."
	docker compose up --build -d
	@echo "Waiting for services to be ready..."
	$(call wait_for_service,db,pg_isready -U postgres,PostgreSQL)
	$(call wait_for_service,redis,redis-cli ping,Redis)
	@echo "All services are ready!"
	go install github.com/mattn/goreman@latest
	go build -o ./bin/$(RUNNER_BIN) ./src/ee/runner
	goreman start

# Run all tests
test: test-fe test-be

# Test frontend services
test-fe:
	@echo "Running frontend tests..."
	cd src/ui && npm run test

# Test frontend services in watch mode
test-fe-watch:
	@echo "Running frontend tests in watch mode..."
	cd src/ui && npm run test:watch

# Test backend services
test-be:
	@echo "Running tests..."
	go test -tags=imageopt,alibaba -p 1 -v -failfast -coverprofile=coverage.out ./...
	go test -p 1 -v -failfast ./src/lib/integrations/

# Development workflow - check dependencies and start services
dev: check-deps start
