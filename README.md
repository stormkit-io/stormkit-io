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

You can install `go` and `node` using [mise](https://mise.jdx.dev/), which is a polyglot tool version manager.

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

**Cause:** The hosting service uses `github.com/h2non/bimg` for image processing, which requires `libvips` and `pkg-config` system libraries.

**Solution on macOS:**

```bash
# Install libvips and pkg-config via Homebrew
brew install vips pkg-config

# Verify installation
pkg-config --modversion vips

# Restart services
./scripts/start.sh
```

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
