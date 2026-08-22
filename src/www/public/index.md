<!-- Source: https://www.stormkit.io/ -->
<!-- Title: Stormkit — self-hosted deployment platform for web applications -->
<!-- Description: Stormkit is a self-hostable deployment platform for web applications: deployments, environments, previews, a Postgres database, end-user auth, a mailer, cron triggers and analytics — on your own infrastructure, driveable by an agent over MCP. -->

# Stormkit

Stormkit is a deployment platform for web applications in any language. It gives
you the workflow of a managed platform — git-driven deployments, preview
environments, instant rollbacks — while running on infrastructure you own, so
there is no vendor lock-in and no usage-based surprise.

It is a product platform, not only hosting: a managed Postgres database,
end-user authentication, a transactional mailer, periodic (cron) triggers,
persistent volumes and built-in analytics ship with it.

## Install

```bash
# Interactive install
curl -sSL https://www.stormkit.io/install.sh | sh

# Hands-off install for an agent: provisions an owner admin plus an MCP-ready API key
curl -sSL https://www.stormkit.io/install.sh | sh -s -- --agent
```

Supports Ubuntu, Debian, Fedora and macOS. Stormkit Cloud is available at
[app.stormkit.io](https://app.stormkit.io) if you would rather not run it
yourself.

## What you can do with it

- **Deploy from git** — GitHub, GitLab and Bitbucket, with automatic deployments, preview links per pull request and post-deploy status checks.
- **Run any stack** — React, Vue, Angular, Next.js, Nuxt, SvelteKit, static site generators, and long-lived server processes in any language on self-hosted instances.
- **Own your data** — self-host for data sovereignty and regulatory compliance, with custom Docker images and your own certificates.
- **Ship product features** — database, auth, mailer, triggers, volumes, redirects, snippets, image optimisation and analytics without adding third-party services.
- **Automate everything** — a REST API and an MCP server cover the whole platform: provision a server, deploy, publish, read the logs when something breaks.

## Resources for developers and agents

| Resource | URL |
| --- | --- |
| Documentation | https://www.stormkit.io/docs/welcome/getting-started |
| API authentication | https://www.stormkit.io/docs/api/authentication |
| OpenAPI specification | https://www.stormkit.io/openapi.json |
| MCP server | https://www.stormkit.io/mcp |
| API reference (MCP endpoint) | https://api.stormkit.io/v1/mcp |
| Self-hosting guide | https://www.stormkit.io/docs/self-hosting/getting-started |
| llms.txt | https://www.stormkit.io/llms.txt |
| Sitemap | https://www.stormkit.io/sitemap.xml |
| Source code | https://github.com/stormkit-io/stormkit-io |

Every documentation, blog and tutorial page is also available as markdown: add
`.md` to its path, or send `Accept: text/markdown`.

## Pricing

Self-hosted Stormkit is free for a single user; paid plans start at $20 per user
per month. See [stormkit.io/#pricing](https://www.stormkit.io/#pricing).

## Contact

Email [hello@stormkit.io](mailto:hello@stormkit.io), join
[Discord](https://discord.com/invite/6yQWhyY), or use the
[contact form](https://www.stormkit.io/contact).
