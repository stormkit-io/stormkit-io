---
title: MCP Server
description: Connect Claude and other MCP clients to Stormkit to deploy and manage apps from your agent.
---

# MCP Server

## Overview

Stormkit ships a built-in [Model Context Protocol](https://modelcontextprotocol.io) (MCP) server. It exposes Stormkit operations — deployments, apps, environments, domains, periodic triggers and teams — as tools that MCP clients such as Claude Code can call directly.

The server speaks the **Streamable HTTP** transport (protocol revision `2025-11-25`): a single `POST /v1/mcp` endpoint handles all client→server messages, and an optional `GET /v1/mcp` SSE stream carries server→client keep-alives.

---

## Connecting

**Base URL:** `https://api.stormkit.io` (Stormkit Cloud). Self-hosted instances expose the MCP server at `https://<your-host>/v1/mcp`.

**Authentication:** a Stormkit API key passed as a Bearer token in the `Authorization` header. A user-level key works for every tool; team-, app- and env-scoped keys are limited to resources within their scope. Create a key from **User Settings → API Keys**.

```
Authorization: Bearer SK_xxxxxxxx
```

### Claude Code plugin (recommended)

The quickest way to connect Claude Code is the official plugin:

```
/plugin marketplace add stormkit-io/stormkit-io
/plugin install stormkit@stormkit
```

Then set your credentials as environment variables before launching Claude Code:

```bash
# Stormkit Cloud — only the key is needed
export STORMKIT_API_KEY="SK_xxxxxxxx"

# Self-hosted — also point at your instance (base URL only, no /v1/mcp suffix)
export STORMKIT_HOST="https://stormkit.mycompany.com"
export STORMKIT_API_KEY="SK_xxxxxxxx"
```

### Manual configuration

Any MCP client that supports remote HTTP servers can connect directly. For Claude Code without the plugin:

```bash
claude mcp add --transport http stormkit https://api.stormkit.io/v1/mcp \
  --header "Authorization: Bearer SK_xxxxxxxx"
```

Or as raw `.mcp.json`:

```json
{
  "mcpServers": {
    "stormkit": {
      "type": "http",
      "url": "https://api.stormkit.io/v1/mcp",
      "headers": { "Authorization": "Bearer SK_xxxxxxxx" }
    }
  }
}
```

---

## Tools

All tools return JSON. Errors are reported as MCP `isError` content while the transport stays HTTP 200, per JSON-RPC convention.

> **Environment variable values are masked.** Tools that return environments (such as `list_environments`) blank out variable _values_ for security, matching the REST API. See the [Environments API](/docs/api/environments) for the dedicated pull endpoint.

### Deployments

| Tool                    | Description                                                                 |
| ----------------------- | --------------------------------------------------------------------------- |
| `deploy`                | Trigger a new deployment for an environment. Returns the deployment object. |
| `get_deployment`        | Return metadata and status for a deployment. Poll until success/failed.     |
| `get_runtime_logs`      | Return runtime logs (SSR and API function output) produced by a deployment. |
| `list_deployments`      | Paginated list of deployments for an environment.                           |
| `publish_deployment`    | Publish a successfully built deployment, making it live.                    |
| `restart_deployment`    | Restart a failed deployment.                                                |
| `stop_deployment`       | Stop a running deployment.                                                  |
| `prioritize_deployment` | Move a queued deployment to the front of the build queue.                   |
| `delete_deployment`     | Delete a deployment and its artifacts.                                      |

### Access logs

| Tool              | Description                                                                    |
| ----------------- | ------------------------------------------------------------------------------ |
| `get_access_logs` | Return raw HTTP access logs for an environment, newest first (last 24h by default, 100 entries per page — pass `limit` for up to 1000). |

### Apps & environments

| Tool                 | Description                                                       |
| -------------------- | --------------------------------------------------------------- |
| `create_app`         | Connect a git repository as a new Stormkit app.                  |
| `list_apps`          | List apps the API key can access.                                |
| `create_environment` | Create an environment for an app.                                |
| `update_environment` | Update an environment's build config.                            |
| `list_environments`  | List environments for an app (env-var values masked).            |

### Domains

| Tool            | Description                              |
| --------------- | --------------------------------------- |
| `list_domains`  | List domains attached to an environment.|
| `create_domain` | Attach a custom domain to an environment.|
| `delete_domain` | Remove a domain from an environment.    |

### Triggers

| Tool               | Description                                                                  |
| ------------------ | --------------------------------------------------------------------------- |
| `list_triggers`    | List periodic triggers configured for an environment.                       |
| `create_trigger`   | Create a periodic trigger that calls a URL on a cron schedule (UTC).         |
| `update_trigger`   | Update an existing periodic trigger.                                         |
| `delete_trigger`   | Delete a periodic trigger.                                                   |
| `invoke_trigger`   | Run a trigger immediately and return the execution log.                      |
| `get_trigger_logs` | Return the last 25 executions (scheduled or manual) of a trigger.           |

### Teams

| Tool          | Description                          |
| ------------- | ----------------------------------- |
| `list_teams`  | List teams the user belongs to.     |
| `create_team` | Create a new team.                  |

### Database integration (self-hosted only)

| Tool                             | Description                                                      |
| -------------------------------- | --------------------------------------------------------------- |
| `enable_database_integration`    | Provision a Postgres schema for an environment.                 |
| `configure_database_integration` | Toggle migrations and env-var injection for a provisioned schema.|

For the exact input parameters of each tool, call `tools/list` on your instance.
