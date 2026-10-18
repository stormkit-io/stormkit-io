---
title: Authentication
description: Documentation on accessing Stormkit API.
---

# API Authentication

You can access Stormkit API by using API Keys. Currently, there are three-level API keys:

## User Level API Key

1. Click on your profile picture on the top-right corner
1. Select **Account**
1. Scroll down to the **API Keys** section
1. Create a new API Key

This API Key will grant programmatic access to everything in your Stormkit account.

## Team Level API Key

1. Expand the Team toggle on top-left corner of the page
1. Select the `gear` icon of the Team you would like to access to
1. Create a new API Key

This API Key will grant access to all applications owned by the team.

## Environment Level API Key

1. Visit **Your App** > **Your Environment** > **Config** > **Other** > **API Keys**
1. Create a new API Key

This API Key will grant access to the specified environment.

## Read-only API keys

Any of the keys above can be created as **read only** by turning on the
**Read only** switch in the creation dialog. A read-only key:

- answers on `GET` and `HEAD` requests only, and only on endpoints that reveal
  no secrets. Everything else is refused with `403` and the code
  `read_only_api_key`.
- cannot pull plaintext environment variables (`GET /v1/env/pull`), and does
  not receive the private key of custom domain certificates.
- sees and calls only the read-only tools on the [MCP server](/docs/api/mcp),
  within the key's own team, app or environment.

Team members with the developer role can create and delete read-only team
keys; full-access team keys still need an owner or admin.

Use read-only keys for monitors, dashboards and agents that should never deploy,
change or delete anything. The setting cannot be changed after creation; delete
the key and create a new one instead.

> **Important:** The API key token is displayed **only once** immediately after creation. Make sure to copy it before closing the dialog — it cannot be retrieved afterwards. If you lose the key, delete it and create a new one.

## Authenticating

Once the API Key is obtained, add an `Authorization` header and use the API key. For example:

```bash
# Using the User Level API Key:

curl -X GET \
     -H 'Authorization: Bearer <api_key>' \
     -H 'Content-Type: application/json' \
     'https://api.stormkit.io/v1/snippets?envId=4151'
```

```bash
# Using the Team Level API Key:

curl -X GET \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/json' \
     'https://api.stormkit.io/v1/apps'
```

```bash
# Using the Environment Level API Key:

curl -X GET \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/json' \
     'https://api.stormkit.io/v1/redirects?appId=48961&envId=58181'
```

## OpenAPI specification

The full API surface is published as an OpenAPI 3.1 document. Every operation
carries a unique `operationId`, a description, typed parameters and response
schemas, so it can be loaded straight into an API client or turned into
function-calling tools for an agent.

- `https://api.stormkit.io/v1/openapi.json` — served by the API itself, no
  authentication required. On a self-hosted instance, use your own API host.
- `https://www.stormkit.io/openapi.json` — the same document on the website.

```bash
curl -s 'https://api.stormkit.io/v1/openapi.json' | jq '.paths | keys'
```

## Error responses

Every failing call answers with JSON:

```json
{
  "error": "The API key is missing, invalid, or does not grant access to this resource.",
  "code": "forbidden",
  "docs": "https://www.stormkit.io/docs/api/authentication"
}
```

| Field    | Description                                                                       |
| -------- | --------------------------------------------------------------------------------- |
| `error`  | Human-readable description of what went wrong.                                    |
| `code`   | Stable, machine-readable identifier — branch on this, not on the message.         |
| `docs`   | Present on authentication failures: the page explaining how to resolve them.      |
| `errors` | Present on validation failures: a message per rejected field, keyed by field name. |

Common codes: `forbidden` (missing, invalid or too narrowly scoped key),
`read_only_api_key` (a read-only key attempted a change),
`unauthorized` (no credentials at all), `not-found` (the addressed resource does
not exist or is not visible to the key), `unknown-endpoint` (no such path — check
the OpenAPI document), `method-not-allowed` (wrong HTTP method for the path).

