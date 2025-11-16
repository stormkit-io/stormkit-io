# Welcome

[Stormkit](https://www.stormkit.io) is a hosting solution for seamless deployment and management of modern web applications.

![Stormkit](./.github/assets/deployment-page.png)

## Cloud Edition

For those who prefer a managed solution, Stormkit offers a Cloud Edition that can be accessed at [app.stormkit.io](app.stormkit.io). The Cloud Edition handles all the hosting, scaling, and maintenance tasks for you, allowing you to focus solely on building and improving your applications.

## Self-Hosted Edition

The Self-Hosted Edition of Stormkit gives you the flexibility to host your own instance of Stormkit on your infrastructure. This version is ideal for organizations that require more control over their hosting environment, need to comply with specific regulatory requirements, or prefer to manage their own infrastructure.

## Getting Started

To get started with the Self-Hosted Edition of Stormkit, you can choose to use either the provided binaries or Docker images.

### Using docker containers

You can use Docker images to run the Self-Hosted Edition. The following images are available:

- ghcr.io/stormkit-io/workerserver:latest
- ghcr.io/stormkit-io/hosting:latest

## Additional services

In addition to the Stormkit's microservices, a PostgreSQL database and a Redis Instance is also required for Stormkit to function properly.

## Local Development

To run Stormkit locally for development purposes:

### Prerequisites

- Go 1.25+
- Node.js 24+
- PostgreSQL 17+
- Redis 7+
- tmux 3+
- [Overmind (Procfile process manager)](https://github.com/DarthSim/overmind) binary
- Docker Engine with Compose v2 plugin

You can install `go` and `node` using [mise](https://mise.jdx.dev/), which is a polyglot tool version manager.

> [!NOTE]
> Debian/Ubuntu `overmind` packages install the SaaS CLI, not the Procfile process manager that `scripts/start.sh` expects. The start script now checks for the SaaS CLI and, on Linux, automatically downloads the Procfile release into `./bin/overmind` when needed. You can still download the release yourself from [github.com/DarthSim/overmind/releases](https://github.com/DarthSim/overmind/releases), place it somewhere on your `PATH` (for example `~/bin/overmind`), or set the `OVERMIND_BIN` environment variable to point to the binary if you prefer to manage it manually. If `tmux` is missing on Debian/Ubuntu, the script tries to install it with `apt-get`; otherwise, install it manually via `sudo apt install tmux`.

> [!TIP]
> Example setup for Ubuntu/Debian:
> ```bash
> sudo apt update
> sudo apt install curl gnupg ca-certificates libvips-dev pkg-config
> # Docker Engine + Compose v2 (official repo)
> curl -fsSL https://download.docker.com/linux/debian/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
> echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
> sudo apt update
> sudo apt install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
> sudo usermod -aG docker $USER && newgrp docker
> ```
> After installing these packages, `./scripts/start.sh` will find Docker/Compose, tmux, and libvips on Debian/Ubuntu systems without extra setup.

### Update environment variables

- Copy [.env.example](./.env.example) and create an `.env` file. Provide the missing variables.
- Generate a 32 random token and set the `STORMKIT_APP_SECRET` environment variable.

### Running the services

```bash
# Clone the repository
git clone https://github.com/stormkit-io/stormkit-io.git
cd stormkit-io

# Trust the dependencies specified in `mise.toml` and install them
# NOTE: you may need to install mise first: https://mise.jdx.dev/getting-started.html
mise trust && mise install

# Start all services (includes database setup and migrations)
./scripts/start.sh
```

If you keep the Procfile Overmind binary outside of your `PATH`, point the script to it with `OVERMIND_BIN=/path/to/overmind ./scripts/start.sh`.

After starting the services:

- The application will be available at `https://localhost:5400`
- The API will be available at `http://api.localhost:8888`

## Project Structure

```
stormkit-io/
├── src/
│   ├── ce/                   # Community Edition (AGPL-3.0)
│   │   ├── api/              # REST API server
│   │   ├── hosting/          # Hosting service
│   │   ├── runner/           # Build and deployment runner
│   │   └── workerserver/     # Background job processing
│   ├── ee/                   # Enterprise Edition (Commercial)
│   │   ├── api/              # Enterprise API features
│   │   ├── hosting/          # Enterprise hosting features
│   │   └── workerserver/     # Enterprise background services
│   ├── lib/                  # Shared libraries and utilities
│   ├── migrations/           # Database migrations
│   ├── mocks/                # Test mocks and fixtures
│   └── ui/                   # Frontend React application
├── scripts/                  # Build and deployment scripts
```

### Component Overview

- **Community Edition (`src/ce/`)**: Open source components under AGPL-3.0
- **Enterprise Edition (`src/ee/`)**: Commercial features requiring a license
- **Shared Libraries (`src/lib/`)**: Common utilities used by both editions
- **Frontend (`src/ui/`)**: React-based web interface

## Testing

Tests require PostgreSQL with a test database named `sktest` and Redis to be running.

### Setup

```bash
# Start services
docker compose up -d db redis

# Create test database
docker compose exec db createdb -U ${POSTGRES_USER} sktest
```

### Running Tests

```bash
# Run all tests (sequential execution required)
go test -p 1 ./...

# With verbose output
go test -p 1 -v ./...

# With coverage
go test -p 1 -coverprofile=coverage.out ./...

# Custom timeout
go test -p 1 -timeout 30m ./...
```

## Troubleshooting

### `go: command not found` after running `mise install`

**Problem:** After running `mise install`, which reports "all tools are installed", running `./scripts/start.sh` fails with:

```
./scripts/start.sh: line 29: go: command not found
```

**Solution:** The mise tools aren't activated in your shell. You need to add mise activation to your shell configuration:

```bash
# Add mise activation to your shell config
echo 'eval "$(mise activate zsh)"' >> ~/.zshrc

# Reload the configuration
source ~/.zshrc

# Verify go is now available
which go
```

For other shells, replace `zsh` with your shell (e.g., `bash`, `fish`). See [mise activation docs](https://mise.jdx.dev/getting-started.html#_2-activate-mise) for more details.

### `pkg-config: executable file not found` - hosting service crashes

**Problem:** After starting the services, the hosting service immediately crashes with:

```
hosting      | github.com/h2non/bimg: exec: "pkg-config": executable file not found in $PATH
hosting      | Exited with code 1
services     | Interrupting...
```

**Cause:** This error only occurs if the hosting service was built with image optimization enabled (using `-tags=imageopt`). Image optimization requires `libvips` and `pkg-config` system libraries.

**Solution 1 - Install the required dependencies (if you need image optimization):**

On macOS:
```bash
# Install libvips and pkg-config via Homebrew
brew install vips pkg-config

# Verify installation
pkg-config --modversion vips

# Restart services
./scripts/start.sh
```

On Ubuntu/Debian:
```bash
apt-get update
apt-get install -y libvips-dev pkg-config
```

**Solution 2 - Build without image optimization (recommended for development):**

By default, Stormkit is built without image optimization to avoid requiring additional dependencies. If your build includes the `imageopt` tag, rebuild without it:

```bash
# Build without image optimization
go build ./src/ce/hosting
```

See [docs/IMAGE_OPTIMIZATION.md](docs/IMAGE_OPTIMIZATION.md) for more details on enabling and using image optimization.


### API endpoints return 500 errors - `/api/auth/providers` and `/api/instance`

**Problem:** When accessing the application at `https://localhost:5400`, the auth page may fail to load properly and you may see 500 Internal Server Error responses on API endpoints like `/api/auth/providers` and `/api/instance` in the browser's Network tab.

**Solution:**

```bash
# Add api.localhost to your hosts file
echo "127.0.0.1       api.localhost" | sudo tee -a /etc/hosts

# Verify it resolves correctly
ping -c 1 api.localhost

# Restart the services
./scripts/start.sh
```

After applying this fix, the API proxy will work correctly and the endpoints will return proper responses.
