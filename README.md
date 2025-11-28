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

To run Stormkit locally:

### Prerequisites

- Go 1.21+
- Node.js 22+
- PostgreSQL 14+
- Redis 7+
- [Mise](https://mise.jdx.dev/)
- [Docker](https://docs.docker.com/get-started/get-docker/)

You can install `go` and `node` using Mise, which is a polyglot tool version manager.

```bash
# Trust the dependencies specified in `mise.toml` and install them
mise trust && mise install
```

### Running the services

```bash
# Clone the repository
git clone https://github.com/stormkit-io/stormkit-io.git
cd stormkit-io

# Start all services (includes database setup and migrations)
make dev
```

After starting the services:

- The landing page will be available at `https://localhost:5500`
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
│   └── ui/                   # Frontend React
│   └── www/                  # Landing page React
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
# Run backend and frontend tests
make test

# Run only backend tests
make test-be

# Run only frontend tests
make test-fe
```

## Troubleshooting

For detailed troubleshooting steps, see our dedicated [troubleshooting guide](./docs/troubleshooting.md).
