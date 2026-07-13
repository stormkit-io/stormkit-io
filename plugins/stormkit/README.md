# Stormkit plugin for Claude Code

Deploy and manage Stormkit apps, environments, domains and deployments from
Claude Code. Talks to Stormkit's hosted MCP server (`/v1/mcp`).

## Install

```
/plugin marketplace add stormkit-io/stormkit-io
/plugin install stormkit@stormkit
```

## Configure

The connection is driven by two environment variables:

| Variable            | Required           | Default                     | Notes                                            |
| ------------------- | ------------------ | --------------------------- | ------------------------------------------------ |
| `STORMKIT_API_KEY`  | yes                | —                           | Stormkit API key (`SK_...`), from User Settings. |
| `STORMKIT_HOST`     | self-hosted only   | `https://api.stormkit.io`   | Base URL only; `/v1/mcp` is appended for you.    |

```bash
# Stormkit Cloud
export STORMKIT_API_KEY="SK_xxxxxxxx"

# Self-hosted
export STORMKIT_HOST="https://stormkit.mycompany.com"
export STORMKIT_API_KEY="SK_xxxxxxxx"
```

Restart Claude Code, then run `/mcp` to confirm the `stormkit` server is
connected. Run `/stormkit:setup` for a guided walkthrough.
