# ==============================================================================
# Platform Detection
# ==============================================================================

ifeq ($(OS),Windows_NT)
  ifeq '$(findstring ;,$(PATH))' ';'
    UNIX_LIKE := FALSE
    DETECTED_OS := Windows
  else
    UNIX_LIKE := TRUE
    DETECTED_OS := WSL
  endif
else
  UNIX_LIKE := TRUE
  DETECTED_OS := Unix
endif

# ==============================================================================
# Configuration
# ==============================================================================

# Set shell for Windows
ifeq ($(UNIX_LIKE),FALSE)
  # Suppress mise warning on PowerShell 5
  export MISE_PWSH_CHPWD_WARNING := 0

  SHELL := powershell.exe
  .SHELLFLAGS := -NoProfile -NoLogo
endif

# ==============================================================================
# Dependency Check Messages
# ==============================================================================

DOCKER_ERROR := Docker is not installed. Please install Docker from https://docs.docker.com/get-started/get-docker/
DOCKER_OK := Docker is installed
DOCKER_NOT_RUNNING := Docker is not running. Please start Docker Desktop.
DOCKER_RUNNING := Docker daemon is running
MISE_ERROR := Mise is not installed. Please install Mise from https://mise.jdx.dev/getting-started.html
MISE_OK := Mise is installed
ENV_CREATED := .env file created from .env.example
ENV_OK := .env file exists

# ==============================================================================
# Helper Functions (Windows PowerShell)
# ==============================================================================

ifeq ($(UNIX_LIKE),FALSE)
  define check_command
    @Get-Command $(1) -ErrorAction SilentlyContinue | Out-Null; \
    if ($$?) { \
      Write-Host -NoNewline '[OK] ' -ForegroundColor Green; \
      Write-Host '$(3)' \
    } else { \
      Write-Host -NoNewline '[ERROR] ' -ForegroundColor Red; \
      Write-Host '$(2)' \
    }
  endef
  
  define check_docker_running
    @docker ps 2>&1 | Out-Null; \
    if ($$?) { \
      Write-Host -NoNewline '[OK] ' -ForegroundColor Green; \
      Write-Host '$(2)' \
    } else { \
      Write-Host -NoNewline '[ERROR] ' -ForegroundColor Red; \
      Write-Host '$(1)' \
    }
  endef
  
  define check_env_file
    @if (!(Test-Path '.env')) { \
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
# Phony Targets
# ==============================================================================

.PHONY: check-deps start print-env

# ==============================================================================
# Tasks
# ==============================================================================

print-env:
	@echo "=== Environment Variables ==="
	@echo "OS: $(OS)"
	@echo "DETECTED_OS: $(DETECTED_OS)"
	@echo "UNAME_S: $(UNAME_S)"
	@echo "SHELL: $(SHELL)"
	@echo "PATH: $(PATH)"
	@echo "============================"

check-deps:
	@echo "Checking dependencies on $(DETECTED_OS)..."
	$(call check_command,docker,$(DOCKER_ERROR),$(DOCKER_OK))
	$(call check_docker_running,$(DOCKER_NOT_RUNNING),$(DOCKER_RUNNING))
	$(call check_command,mise,$(MISE_ERROR),$(MISE_OK))
	$(call check_env_file,$(ENV_CREATED),$(ENV_OK))

start:
	@echo "Starting on $(DETECTED_OS)..."
	docker compose up --build -d
	@echo "Waiting for services to be ready..."
	$(call wait_for_service,db,pg_isready -U postgres,PostgreSQL)
	$(call wait_for_service,redis,redis-cli ping,Redis)
	@echo "All services are ready!"

dev: check-deps start
