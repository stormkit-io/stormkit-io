---
title: 'Self-Hosting with Stormkit: Deploy Web Apps with Full Control'
description: Discover Stormkit, the self-hosted alternative to Vercel and Netlify. Deploy your frontend apps with full control over your infrastructure, offering powerful features like multiple environments, deployment previews, snippet injections, status checks and analytics.
---

# Self-Hosting

<section>

Stormkit is a deployment platform for frontend applications. It helps you focus on your product by providing a solution for most common technical challenges, such as deployments, logs, hosting, scaling tls certificates, and helps saving valuable time.

</section>

## Getting Started

<section>

Execute the following command to install Docker and Stormkit on your machine.

```bash
curl -sSL https://www.stormkit.io/install.sh | sh
```

[The installation script](https://github.com/stormkit-io/www-stormkit-io/blob/main/public/install.sh) has been tested on the following platforms:

- `Ubuntu (20, 22, 24)`
- `Darwin (Sonoma)`
- `Fedora (39, 40)`
- `Debian (11, 12)`
- `Rocky Linux (10)`

If you find an issue with the script, please [submit it here](https://github.com/stormkit-io/www-stormkit-io/issues).

## Environment variables

The installation script will prompt you to configure the necessary environment variables. However, if you wish to view the complete list of available environment variables, you can refer to the [docker-compose.yaml](https://github.com/stormkit-io/bin/blob/main/docker-compose.yaml) file. This file provides a comprehensive list of all environment variables, along with detailed descriptions for each.

</section>

## Installation Options

During the installation process, you will be prompted to choose between Docker Compose and Docker Swarm.

If you plan to deploy Stormkit on a single machine, we recommend using Docker Compose for its simplicity and ease of use.
For more advanced scenarios that may require scalability or clustering, we suggest opting for Docker Swarm or another container orchestration tool.

## Configuring GitHub Actions for Stormkit Deployments

By default, Stormkit runs deployments on the same machine where the `Workerserver` service is hosted. If you require better performance or more control over your Continuous Integration (CI) process, consider configuring GitHub Actions. For detailed instructions on how to set this up, refer to our [stormkit-io/runner](https://github.com/stormkit-io/runner) repository.

## Tutorials

<section>

- [Install Stormkit on Ubuntu Server and deploy a Next.js app](https://youtu.be/bLw0r3VKLrg)
- [How to Self-Host Stormkit on Hetzner Cloud](https://www.stormkit.io/tutorials/how-to-self-host-stormkit-on-hetzner-cloud)

</section>
