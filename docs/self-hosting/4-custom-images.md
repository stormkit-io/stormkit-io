---
title: 'Self-Hosting with Stormkit: Customizing Docker Images'
description: Learn how to customize Docker images for your self-hosted Stormkit instance. Build custom images with specific runtimes, dependencies, and configurations to meet your deployment requirements.
---

# Customizing Docker Images

<section id="custom-images">

Self-hosted Stormkit instances provide the flexibility to customize Docker images according to your specific deployment needs. You can build custom images that include additional runtimes, dependencies, tools, or configurations that your applications require.

## Understanding Stormkit's Runtime Management

Stormkit uses **`mise`** (formerly `rtx`) to dynamically install and manage runtimes during application deployment. This approach provides several benefits:

- **Flexible runtime versions** - Each project can specify its own runtime version
- **Automatic dependency management** - Runtimes are installed on-demand
- **Persistent storage** - Dependencies are cached in the home folder for reuse

### Runtime vs Build-time Dependencies

It's important to understand the distinction between different types of dependencies:

**Runtime Dependencies:**

- Programming language runtimes (Node.js, Python, Go, Rust, etc.)
- Language-specific package managers (`npm`, `pip`, `cargo`, etc.)
- Dependencies installed via `mise` configuration

**Build-time Dependencies:**

- System packages requiring root access
- Custom build tools and utilities
- Base system configurations

### Persisting Dependencies

Stormkit stores all runtime dependencies in the home folder. To persist these dependencies across upgrades and restarts, you can mount the home directory:

```yaml
# docker-compose.yml
services:
  hosting:
    image: ghcr.io/stormkit-io/hosting:latest
    volumes:
      - hosting:/home/stormkit
    # Additional configuration...

  workerserver:
    image: ghcr.io/stormkit-io/workerserver:latest
    volumes:
      - workerserver:/home/stormkit
    # Additional configuration...

volumes:
  hosting:
  workerserver:
```

## Official Stormkit Images

Before creating custom images, it's important to understand the official Stormkit images available:

### Available Images

**ghcr.io/stormkit-io/hosting:latest**

- Responsible for serving the Stormkit API
- Handles serving your deployed applications
- Contains the web interface and API endpoints
- Optimized for production web serving

**ghcr.io/stormkit-io/workerserver:latest**

- Responsible for running background jobs
- Handles application deployments and builds
- Processes build queues and deployment pipelines

### Using Official Images as Base

When creating custom images, you can extend these official images:

```dockerfile
# For custom hosting image
FROM ghcr.io/stormkit-io/hosting:latest
# Add your customizations here

# For custom workerserver image
FROM ghcr.io/stormkit-io/workerserver:latest
# Add your build tools and dependencies here
```

## When to Customize Images

Custom images are primarily needed for:

- **System-level dependencies** requiring root access
- **Custom build tools** not available through standard package managers
- **Security configurations** and hardening
- **Pre-installed system utilities** and libraries

For most runtime dependencies (Node.js versions, npm packages, etc.), Stormkit's dynamic installation via `mise` is sufficient and preferred.

### Example: System Dependencies

```dockerfile
FROM ghcr.io/stormkit-io/workerserver:latest

# Switch to root to install system packages
USER root

# Install system packages that require root access
RUN apt-get update && apt-get install -y \
    imagemagick \
    graphicsmagick \
    libvips-dev \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

# Install custom system tools
RUN wget -O /usr/local/bin/custom-tool https://example.com/custom-tool \
    && chmod +x /usr/local/bin/custom-tool

# Switch back to stormkit user
USER stormkit

# No need to specify the cmd as it will be inherited from the base image
```

## Build and Registry Configuration

### Building Custom Images

Build your custom image locally:

```bash
# Build the image
docker build -t my-custom-stormkit:latest .
```

Optionally, push it to your registry:

```bash
# Tag for your registry
docker tag my-custom-stormkit:latest your-registry.com/my-custom-stormkit:latest

# Push to registry
docker push your-registry.com/my-custom-stormkit:latest
```

### Configuring Docker Compose to Use Custom Images

Update your `docker-compose.yaml` to use custom images.

```yaml
hosting:
  # Instead of the image, provide the build configuration
  # and docker compose will build and use the image instead.
  # image: ghcr.io/stormkit-io/hosting:latest
  build:
    context: .
    dockerfile: MyCustomDockerFile
```

## Upgrading images

Pass the `--build` flag to the `up` command and Docker Compose will rebuild the image whenever there is a new version.

```bash
docker compose pull && docker compose down hosting workerserver && docker compose up --build -d hosting workerserver
```

## Security Considerations

When creating custom images, follow these security best practices:

- **Use official base images** from trusted sources
- **Keep images updated** with latest security patches
- **Minimize attack surface** by removing unnecessary packages
- **Always switch back to `stormkit` user** after root operations
- **Scan images** for vulnerabilities before deployment

## Restart your Stormkit Instance

After configuring custom images, restart your Stormkit instance for the changes to take effect:

```bash
docker compose down workerserver hosting && docker compose up workerserver hosting -d
```

</section>
