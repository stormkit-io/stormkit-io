---
title: Environments API
description: API Documentation for managing environments through Stormkit API.
---

# Environments API

## Overview

The Environments API lets you create, update, and delete environments, and pull environment variable values programmatically.

---

## POST /v1/env

Creates a new environment for an application.

**Base URL:** `https://api.stormkit.io`

**Authentication:** API key with at least environment-level scope. Environment-, app-, team-, or user-level keys are accepted as long as they have access to the app that owns the environment and a valid `envId` is provided.

### Request body

| Field                | Type                    | Required | Description                                                                                                 |
| -------------------- | ----------------------- | -------- | ----------------------------------------------------------------------------------------------------------- |
| `name`               | string                  | **Yes**  | Environment name. Only alphanumeric characters and hyphens are allowed. Double hyphens (`--`) are reserved. |
| `branch`             | string                  | **Yes**  | Default git branch for this environment.                                                                    |
| `apiFolder`          | string                  | No       | Repository folder containing serverless API functions.                                                      |
| `apiPathPrefix`      | string                  | No       | URL path prefix for API calls (default: `/api`).                                                            |
| `autoDeploy`         | boolean                 | No       | Whether to trigger automatic deployments.                                                                   |
| `autoDeployBranches` | string                  | No       | Glob/regex pattern to filter which branches trigger auto-deploys. Setting this enables `autoDeploy`.        |
| `autoDeployCommits`  | string                  | No       | Glob/regex pattern to filter which commit messages trigger auto-deploys. Setting this enables `autoDeploy`. |
| `autoPublish`        | boolean                 | No       | Whether to automatically publish successful deployments.                                                    |
| `buildCmd`           | string                  | No       | Command to build the application.                                                                           |
| `distFolder`         | string                  | No       | Output folder containing the build artifacts.                                                               |
| `envVars`            | `Record<string,string>` | No       | Environment variables to inject into deployments.                                                           |
| `errorFile`          | string                  | No       | File served on errors. Must be inside `distFolder`.                                                         |
| `headers`            | string                  | No       | Inline custom HTTP response headers.                                                                        |
| `headersFile`        | string                  | No       | Path to the custom HTTP headers file.                                                                       |
| `installCmd`         | string                  | No       | Command to install dependencies.                                                                            |
| `previewLinks`       | boolean                 | No       | Whether Stormkit posts a preview URL on pull/merge requests.                                                |
| `redirects`          | `Redirect[]`            | No       | Inline redirect/rewrite rules. See the Redirects API for the `Redirect` object shape.                       |
| `redirectsFile`      | string                  | No       | Path to a file containing redirect/rewrite rules.                                                           |
| `serverCmd`          | string                  | No       | Command to start the server (self-hosted only).                                                             |
| `serverFolder`       | string                  | No       | Server-side upload folder.                                                                                  |
| `statusChecks`       | `StatusCheck[]`         | No       | Post-deployment commands to run. See `StatusCheck` object below.                                            |

**`StatusCheck` object:**

| Field         | Type   | Description                             |
| ------------- | ------ | --------------------------------------- |
| `name`        | string | Human-readable name of the check.       |
| `cmd`         | string | Shell command to execute.               |
| `description` | string | Description of what the check verifies. |

### Response — 201 Created

| Field   | Type   | Description                          |
| ------- | ------ | ------------------------------------ |
| `envId` | string | ID of the newly created environment. |

### Error responses

| Status | Condition                                            |
| ------ | ---------------------------------------------------- |
| `400`  | Missing or invalid fields (see `errors` array).      |
| `403`  | Missing/invalid API key or insufficient permissions. |
| `409`  | An environment with the same name already exists.    |
| `500`  | Internal server error.                               |

### Example

```bash
curl -X POST \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/json' \
     -d '{"appId":"1510","branch":"main","name":"staging","envVars":{"NODE_ENV":"staging"}}' \
     'https://api.stormkit.io/v1/env'
```

```json
// Example response
{
  "envId": "305"
}
```

---

## PUT /v1/env

Updates the configuration of an existing environment. Only the fields provided in the request body are changed — all other fields retain their current values.

**Base URL:** `https://api.stormkit.io`

**Authentication:** Environment-level API key. When using an app- or higher-level key, `envId` must be provided in the request body.

### Request body

All fields are **optional**. Only the fields you include will be updated.

| Field                | Type                    | Description                                                                                                                                  |
| -------------------- | ----------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `envId`              | string                  | ID of the environment to update. **Required when using an app-, team-, or user-level API key; ignored when using an environment-level key.** |
| `name`               | string                  | Environment name. Only alphanumeric characters and hyphens are allowed. Double hyphens (`--`) are reserved.                                  |
| `branch`             | string                  | Default git branch for this environment.                                                                                                     |
| `apiFolder`          | string                  | Repository folder containing serverless API functions.                                                                                       |
| `apiPathPrefix`      | string                  | URL path prefix for API calls (default: `/api`).                                                                                             |
| `autoDeploy`         | boolean                 | Whether to trigger automatic deployments.                                                                                                    |
| `autoDeployBranches` | string                  | Glob/regex pattern to filter which branches trigger auto-deploys. Setting this enables `autoDeploy`.                                         |
| `autoDeployCommits`  | string                  | Glob/regex pattern to filter which commit messages trigger auto-deploys. Setting this enables `autoDeploy`.                                  |
| `autoPublish`        | boolean                 | Whether to automatically publish successful deployments.                                                                                     |
| `buildCmd`           | string                  | Command to build the application.                                                                                                            |
| `distFolder`         | string                  | Output folder containing the build artifacts.                                                                                                |
| `envVars`            | `Record<string,string>` | Environment variables to inject into deployments. Replaces all existing variables.                                                           |
| `errorFile`          | string                  | File served on errors. Must be inside `distFolder`.                                                                                          |
| `headers`            | string                  | Inline custom HTTP response headers.                                                                                                         |
| `headersFile`        | string                  | Path to the custom HTTP headers file.                                                                                                        |
| `installCmd`         | string                  | Command to install dependencies.                                                                                                             |
| `previewLinks`       | boolean                 | Whether Stormkit posts a preview URL on pull/merge requests.                                                                                 |
| `redirects`          | `Redirect[]`            | Inline redirect/rewrite rules. Replaces all existing inline rules. See the Redirects API for the `Redirect` object shape.                    |
| `redirectsFile`      | string                  | Path to a file containing redirect/rewrite rules.                                                                                            |
| `serverCmd`          | string                  | Command to start the server (self-hosted only).                                                                                              |
| `serverFolder`       | string                  | Server-side upload folder.                                                                                                                   |
| `statusChecks`       | `StatusCheck[]`         | Post-deployment commands to run. Replaces all existing checks. See `StatusCheck` in `POST /v1/env`.                                          |

### Response — 200 OK

| Field | Type    | Description                   |
| ----- | ------- | ----------------------------- |
| `ok`  | boolean | `true` when update succeeded. |

### Error responses

| Status | Condition                                                      |
| ------ | -------------------------------------------------------------- |
| `400`  | Invalid field values (e.g. malformed headers, duplicate name). |
| `403`  | Missing/invalid API key or insufficient permissions.           |
| `404`  | Environment not found.                                         |
| `500`  | Internal server error.                                         |

### Example

```bash
curl -X PUT \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/json' \
     -d '{"envId":"305","buildCmd":"npm run build:prod","distFolder":"dist","autoPublish":true}' \
     'https://api.stormkit.io/v1/env'
```

---

## DELETE /v1/env

Deletes an environment by its ID.

**Base URL:** `https://api.stormkit.io`

**Authentication:** App-level API key (the key must be scoped to the app that owns the environment).

### Query parameters

| Parameter | Type   | Required | Description                                                               |
| --------- | ------ | -------- | ------------------------------------------------------------------------- |
| `envId`   | number | **Yes**  | The ID of the environment to delete. Alternatively `id` is also accepted. |

### Response — 200 OK

| Field | Type    | Description                     |
| ----- | ------- | ------------------------------- |
| `ok`  | boolean | `true` when deletion succeeded. |

### Error responses

| Status | Condition                                            |
| ------ | ---------------------------------------------------- |
| `403`  | Missing/invalid API key or insufficient permissions. |
| `404`  | No environment found with the given ID.              |
| `500`  | Internal server error.                               |

### Example

```bash
curl -X DELETE \
     -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/env?envId=305'
```

---

## GET /v1/env/pull

Returns all environment variables for the specified environment. The response is a flat JSON object where each key is a variable name and each value is the variable's value.

**Base URL:** `https://api.stormkit.io`

**Authentication:** Environment-level API key passed as the `Authorization` header. To generate one: **Your App** → **Your Environment** → **Config** → **Other** → **API Keys**. When using an app or higher-level key, `envId` must be provided as a query parameter.

### Query parameters

| Parameter | Type   | Required                                            | Description                                                                                                                                          |
| --------- | ------ | --------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `envId`   | number | **Yes** (unless using an environment-level API key) | The ID of the environment whose variables to retrieve. Not required when using an environment-level key — the environment is derived from the token. |

### Response — 200 OK

A flat JSON object of environment variable key/value pairs.

```json
{
  "NODE_ENV": "production",
  "API_URL": "https://api.my-app.com"
}
```

### Error responses

| Status | Condition                                            |
| ------ | ---------------------------------------------------- |
| `403`  | Missing/invalid API key or insufficient permissions. |
| `500`  | Internal server error.                               |

### Examples

```bash
# Using an environment-level key (envId derived from token)
curl -X GET \
     -H 'Authorization: <env_api_key>' \
     'https://api.stormkit.io/v1/env/pull'
```

```bash
# Using an app or higher-level key (envId required as query parameter)
curl -X GET \
     -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/env/pull?envId=305'
```
