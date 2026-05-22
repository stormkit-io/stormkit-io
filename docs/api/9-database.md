---
title: Database API
description: Provision and configure per-environment Postgres schemas using the Stormkit Database API.
---

# Database API

## Overview

The Database API lets you provision a dedicated Postgres schema for an environment, fetch its metadata, configure migration and env-var-injection behavior, and remove it. Available on self-hosted and development deployments only.

When a schema is provisioned, Stormkit also creates two roles: an app user (used by your deployed code) and a migration user (used by the migrations runner). Credentials are stored on the environment's build config and surfaced as `DATABASE_URL` and related variables when `injectEnvVars` is enabled.

---

## GET /v1/schema

Returns the schema configured for an environment, including its tables and integration flags. When no schema has been provisioned, `schema` is `null`.

**Base URL:** `https://api.stormkit.io`

**Authentication:** Environment-level API key. When using an app-, team-, or user-level key, `envId` must be provided as a query parameter.

### Query parameters

| Parameter | Type   | Required                                            | Description                                          |
| --------- | ------ | --------------------------------------------------- | ---------------------------------------------------- |
| `envId`   | string | **Yes** (unless using an environment-level API key) | The ID of the environment whose schema to retrieve.  |

### Response — 200 OK

| Field    | Type           | Description                                                            |
| -------- | -------------- | ---------------------------------------------------------------------- |
| `schema` | object \| null | The schema object, or `null` when no schema has been provisioned yet.  |

**`schema` object:**

| Field               | Type    | Description                                                          |
| ------------------- | ------- | -------------------------------------------------------------------- |
| `name`              | string  | Postgres schema name, derived from the app and environment IDs.      |
| `tables`            | array   | Array of `{ name, rows, size }` entries describing the schema tables. |
| `migrationsEnabled` | boolean | Whether migrations run on each successful deployment.                 |
| `migrationsFolder`  | string  | Repository path containing migration SQL files.                       |
| `injectEnvVars`     | boolean | Whether `DATABASE_URL` and related vars are injected at build/runtime. |

### Error responses

| Status | Condition                                            |
| ------ | ---------------------------------------------------- |
| `400`  | Missing `envId`.                                     |
| `403`  | Missing/invalid API key or insufficient permissions. |
| `500`  | Internal server error.                               |

### Example

```bash
curl -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/schema?envId=305'
```

---

## POST /v1/schema

Provisions a new schema and roles for the environment, and stores the credentials on the environment's build config.

**Base URL:** `https://api.stormkit.io`

**Authentication:** Environment-level API key. When using an app-, team-, or user-level key, `envId` must be provided in the request body.

### Request body

| Field   | Type   | Required                                            | Description                                                |
| ------- | ------ | --------------------------------------------------- | ---------------------------------------------------------- |
| `envId` | string | **Yes** (unless using an environment-level API key) | Environment ID to provision the schema for.                |

### Response — 200 OK

| Field    | Type   | Description                                |
| -------- | ------ | ------------------------------------------ |
| `schema` | string | The provisioned schema name.               |

### Error responses

| Status | Condition                                                  |
| ------ | ---------------------------------------------------------- |
| `400`  | Missing `envId` or invalid schema name.                    |
| `403`  | Missing/invalid API key or insufficient permissions.       |
| `409`  | A schema has already been provisioned for the environment. |
| `500`  | Internal server error.                                     |

### Example

```bash
curl -X POST \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/json' \
     -d '{"envId":"305"}' \
     'https://api.stormkit.io/v1/schema'
```

---

## POST /v1/schema/configure

Updates the database integration flags for an environment that already has a schema provisioned.

**Base URL:** `https://api.stormkit.io`

**Authentication:** Environment-level API key. When using an app-, team-, or user-level key, `envId` must be provided in the request body.

### Request body

| Field               | Type    | Required                                            | Description                                                                              |
| ------------------- | ------- | --------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `envId`             | string  | **Yes** (unless using an environment-level API key) | Environment ID.                                                                          |
| `injectEnvVars`     | boolean | No                                                  | When true, inject `DATABASE_URL` and related variables into the build/runtime env vars.  |
| `migrationsEnabled` | boolean | No                                                  | When true, run migrations from `migrationsFolder` on every successful deployment.        |
| `migrationsFolder`  | string  | No                                                  | Repository path containing migration SQL files (relative to repo root).                  |

### Response — 200 OK

Empty body.

### Error responses

| Status | Condition                                            |
| ------ | ---------------------------------------------------- |
| `400`  | Missing `envId`.                                     |
| `403`  | Missing/invalid API key or insufficient permissions. |
| `500`  | Internal server error.                               |

### Example

```bash
curl -X POST \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/json' \
     -d '{"envId":"305","migrationsEnabled":true,"migrationsFolder":"db/migrations","injectEnvVars":true}' \
     'https://api.stormkit.io/v1/schema/configure'
```

---

## DELETE /v1/schema

Drops the environment's schema along with its associated roles and clears the configuration from the build config. The caller must have write access on the team.

**Base URL:** `https://api.stormkit.io`

**Authentication:** Environment-level API key with team write access. When using an app-, team-, or user-level key, `envId` must be provided as a query parameter.

### Query parameters

| Parameter | Type   | Required                                            | Description                            |
| --------- | ------ | --------------------------------------------------- | -------------------------------------- |
| `envId`   | string | **Yes** (unless using an environment-level API key) | Environment whose schema to delete.    |

### Response — 200 OK

Empty body.

### Error responses

| Status | Condition                                                                      |
| ------ | ------------------------------------------------------------------------------ |
| `400`  | Missing `envId`, or the environment has no schema configured.                  |
| `403`  | Missing/invalid API key, or the caller lacks team write access.                |
| `500`  | Internal server error.                                                         |

### Example

```bash
curl -X DELETE \
     -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/schema?envId=305'
```
