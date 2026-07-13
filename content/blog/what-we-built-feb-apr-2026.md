---
title: "What We Built: February – April 2026"
date: 2026-05-03
description: A look at everything we shipped over the last three months — a full public API, MCP server, built-in authentication, Nix runtime support, networking improvements, and self-hosted billing.
author-name: Savas Vedova
author-tw: @savasvedova
author-img: https://pbs.twimg.com/profile_images/1993991074138779648/Up6HP-Jw_reasonably_small.jpg
---

The past three months have been some of our most productive. We shipped a full public API, built an MCP server for AI agents, introduced a hosted authentication layer, added Nix runtime support, hardened our networking stack, and gave self-hosted customers a complete billing lifecycle. Here's the full rundown.

## A Public API Worth Building On

The single biggest theme of this period was making Stormkit programmable. We shipped a consistent `WithAPIKey` middleware layer and built out a full suite of REST endpoints:

- **Apps**: list, retrieve, and create applications via API
- **Environments**: list and update environment configuration
- **Deployments**: list, retrieve, trigger, restart, stop, delete, and publish deployments
- **Volumes**: upload files to environment storage programmatically
- **Deployment status**: a polling endpoint so CI/CD pipelines can wait for a build to complete

All endpoints use the same API key authentication model, with keys scoped to either a team, an environment, or a user. Speaking of which — **user-level API keys** are now available from your account settings, giving individuals programmatic access to everything in their Stormkit account without sharing team credentials.

On the security side, API keys are now stored as **SHA-256 hashes** in the database. The raw token is shown only once at creation time. Existing keys continue to work without any migration required.

## MCP Server

We shipped an **MCP (Model Context Protocol) server** endpoint, making Stormkit natively accessible to AI agents. The first tool, `create_app`, lets an agent provision a new application directly — no UI required. With the deployment and environment endpoints already in place, an agent can now manage the full app lifecycle: create, configure, deploy, and publish.

This is the foundation for a genuinely agent-friendly hosting platform.

## Authentication as a Feature

We built **SkAuth** — a hosted authentication layer that any application running on Stormkit can plug into. No third-party auth service required. This quarter's work included:

- Email/password registration via `/_stormkit/auth/register`
- OAuth2 providers with PKCE verification, including X (Twitter)
- A guided flow that walks users through connecting providers
- A UI for managing authenticated users, with a secondary navigation panel
- A `GET /v1/auth/users` public API endpoint for retrieving your user list

> SkAuth is currently behind a feature flag. Reach out if you'd like early access.

## Nix Runtime Support

Server-side deployments now support **Nix** as a runtime environment:

- When a `flake.nix` is detected, all commands are automatically wrapped with `nix develop`
- The Nix store is persisted across deployments to avoid redundant downloads
- `flake.nix` is copied to the server output directory so environments are fully reproducible
- Status checks also run inside `nix develop` when a flake is present

The `mise` integration also improved significantly this quarter: tool paths are injected into the CI environment, duplicate `.bashrc` activation was eliminated, and `mise trust` now runs non-interactively so it never blocks a headless build.

## Networking and Proxy Improvements

The HTTP layer received some serious hardening:

- **PROXY protocol** support for L4 load balancers
- **`X-Forwarded-For` spoofing** is now blocked at the edge
- **Streaming uploads** no longer time out — the proxy body is streamed through rather than buffered, and the proxy timeout is disabled for streaming requests
- `remoteAddress` and `remotePort` are injected into lambda invocations
- A `STORMKIT_HTTP_PROXY_TIMEOUT` environment variable lets you configure the proxy timeout
- A custom `httpsServe` implementation replaced `certmagic.HTTPS`, removing hardcoded server timeouts
- Connection timeouts were tightened across the board
- SSE streaming endpoints are no longer affected by `http.TimeoutHandler`

## Self-Hosting: Billing and Licensing

Self-hosted instances can now handle their own billing lifecycle end-to-end:

- Stripe checkout is routed through an `api.stormkit.io` deep link
- Licenses are **generated automatically** after a successful Stripe checkout and delivered via email
- Cloud and self-hosted Stripe products are separated so pricing is independent
- Non-admin users see a clear info modal when they encounter upgrade prompts
- **Artifact retention days** are now configurable per environment, so you can tune storage usage to your needs
- SMTP configuration and a mailer utility ship with the platform for transactional email

## Service Discovery Reliability

One subtle but important fix: stale service discovery entries are now evicted via a **TTL heartbeat**. Previously, a crashed instance could linger in the registry indefinitely, causing traffic to be routed to a dead node. The heartbeat interval and TTL are both configurable. On top of that, a nil-pointer panic when restarting a failed deployment was fixed, and redirect rule validation was added to environment updates.

## Everything Else

- Switched to **CalVer** (`YYYY.MM.DD.MICRO`) for release naming
- The audit log now tracks deployment stop and publish actions
- CodeMirror no longer overflows its container in the editor
- AWS-chunked encoding is disabled for S3-compatible storage backends
- Production environments can now be deleted
- The "What's New" dialog was migrated from an embedded iframe to a local changelog API endpoint

---

Three months, a lot of surface area. The public API and MCP work in particular set the foundation for what's coming next: making Stormkit a first-class target for AI-driven deployment workflows.
