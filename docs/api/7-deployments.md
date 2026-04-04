---
title: Deployments API
description: Trigger new deployments programmatically using the Stormkit Deployments API.
---

# Deployments API

## Overview

The Deployments API lets you trigger new deployments, poll their status, and publish them — all from any CI/CD pipeline, script, or agent. At minimum an environment-level API key is required. App, team, and user-level keys are also accepted provided an `envId` is supplied in the request body.

---

## POST /v1/deploy

Triggers a new deployment for the environment associated with the API key.

**Base URL:** `https://api.stormkit.io`

**Authentication:** At least an environment-level API key passed as the `Authorization` header. To generate one: **Your App** → **Your Environment** → **Config** → **Other** → **API Keys**. App, team, or user-level keys are also accepted — in those cases `envId` must be included in the request body to identify the target environment.

### Request body

| Field     | Type    | Required    | Default                      | Description                                                                                                                                                           |
| --------- | ------- | ----------- | ---------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `branch`  | string  | No          | Environment's default branch | The git branch to check out and build.                                                                                                                                |
| `publish` | boolean | No          | `false`                      | When `true`, the deployment is published (made live) after a successful build, overriding the environment's `autoPublish` setting.                                    |
| `envId`   | string  | Conditional | —                            | Required when using an app, team, or user-level API key. Ignored when using an environment-level key — the environment is always derived from the token in that case. |

### Response — 201 Created

Returns the newly created deployment object.

| Field               | Type    | Description                                                                      |
| ------------------- | ------- | -------------------------------------------------------------------------------- |
| `id`                | string  | Unique deployment ID.                                                            |
| `appId`             | string  | ID of the application being deployed.                                            |
| `envId`             | string  | ID of the target environment.                                                    |
| `branch`            | string  | The git branch being deployed.                                                   |
| `checkoutRepo`      | string  | The repository that was checked out (e.g. `github/org/repo`).                    |
| `shouldPublish`     | boolean | Whether this deployment will be published once it succeeds.                      |
| `isAutoDeploy`      | boolean | `true` when this deployment was triggered automatically (e.g. via webhook).      |
| `pullRequestNumber` | number  | Pull request number, if this deployment was triggered by a PR. `null` otherwise. |
| `isFork`            | boolean | `true` when the branch originates from a forked repository.                      |
| `commit`            | object  | Commit metadata — see **`Commit` object** below.                                 |
| `createdAt`         | string  | Unix timestamp (seconds) when the deployment was created.                        |
| `stoppedAt`         | string  | Unix timestamp when the deployment stopped. `null` while still running.          |

**`Commit` object:**

| Field     | Type   | Description                                     |
| --------- | ------ | ----------------------------------------------- |
| `author`  | string | Name of the commit author. `null` if unknown.   |
| `message` | string | Commit message. `null` if unknown.              |
| `sha`     | string | Full commit SHA. `null` until the build starts. |

### Error responses

| Status | Condition                                                                                        |
| ------ | ------------------------------------------------------------------------------------------------ |
| `403`  | Missing/invalid API key, team token does not own the app, or user is not a team member.          |
| `404`  | The environment associated with the API key no longer exists, or the repository is inaccessible. |
| `402`  | Build minutes limit exceeded (Stormkit Cloud only).                                              |
| `500`  | Internal server error.                                                                           |

### Examples

```bash
# Trigger a deployment on the default branch
curl -X POST \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/json' \
     'https://api.stormkit.io/v1/deploy'
```

```bash
# Deploy a specific branch and publish immediately
curl -X POST \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/json' \
     -d '{"branch":"release/2.0","publish":true}' \
     'https://api.stormkit.io/v1/deploy'
```

```json
// Example response
{
  "id": "8241",
  "appId": "1510",
  "envId": "305",
  "branch": "release/2.0",
  "checkoutRepo": "github/acme/my-app",
  "shouldPublish": true,
  "isAutoDeploy": false,
  "pullRequestNumber": null,
  "isFork": false,
  "commit": {
    "author": null,
    "message": null,
    "sha": null
  },
  "createdAt": "1742300400",
  "stoppedAt": null
}
```

### Notes

- Deployments run **asynchronously**. The `201` response confirms the deployment was queued — it does not mean the build has completed. Use `GET /v1/deployments/{id}/poll` for an efficient status check, or `GET /v1/deployments/{id}` to fetch the full deployment object.
- When using an **environment-level key**, the `envId` field in the request body has no effect — the environment is always the one the key was issued for. To target a different environment, use that environment's own key.
- When using an **app, team, or user-level key**, `envId` is required. The request will return `404` if the environment does not exist or is not accessible to the key.

---

## GET /v1/deployments/{id}

Retrieves a single deployment by its ID. The deployment must belong to the environment associated with the API key.

**Base URL:** `https://api.stormkit.io`

**Authentication:** At least an environment-level API key passed as the `Authorization` header.

### Path parameters

| Parameter | Type   | Description        |
| --------- | ------ | ------------------ |
| `id`      | string | The deployment ID. |

### Query parameters

| Parameter | Type    | Required    | Default | Description                                                                                                                                                                                      |
| --------- | ------- | ----------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `envId`   | string  | Conditional | —       | Required when using an app, team, or user-level API key to identify the target environment.                                                                                                      |
| `logs`    | boolean | No          | `false` | When `true`, includes the `logs` and `statusChecks` arrays in the response. Omit or set to `false` when polling for deployment status to avoid transferring large log payloads on every request. |

### Response — 200 OK

Returns the deployment object wrapped in a `deployment` key.

| Field                | Type         | Description                                                                               |
| -------------------- | ------------ | ----------------------------------------------------------------------------------------- |
| `id`                 | string       | Unique deployment ID.                                                                     |
| `appId`              | string       | ID of the application.                                                                    |
| `envId`              | string       | ID of the environment.                                                                    |
| `envName`            | string       | Name of the environment.                                                                  |
| `displayName`        | string       | Display name of the application.                                                          |
| `repo`               | string       | Repository that was checked out (e.g. `github/org/repo`).                                 |
| `branch`             | string       | Git branch that was deployed.                                                             |
| `status`             | string       | Current status: `running`, `success`, `failed`, or `stopped`.                             |
| `error`              | string       | Error message if the deployment failed. Empty string otherwise.                           |
| `isAutoDeploy`       | boolean      | `true` when triggered automatically (e.g. via webhook).                                   |
| `isAutoPublish`      | boolean      | Whether this deployment was configured to publish automatically on success.               |
| `stoppedManually`    | boolean      | `true` when the deployment was stopped by a user action.                                  |
| `statusChecksPassed` | boolean      | `true` when all configured status checks passed.                                          |
| `apiPathPrefix`      | string       | API path prefix, if configured.                                                           |
| `previewUrl`         | string       | Preview URL for this specific deployment.                                                 |
| `detailsUrl`         | string       | Dashboard path to the deployment details page.                                            |
| `duration`           | number       | Build duration in seconds. `0` while still running.                                       |
| `createdAt`          | string       | Unix timestamp (seconds) when the deployment was created.                                 |
| `stoppedAt`          | string       | Unix timestamp when the deployment stopped. `null` while still running.                   |
| `commit`             | object       | Commit metadata — see **`Commit` object** below.                                          |
| `snapshot`           | object       | Snapshot of the environment configuration used for this deployment.                       |
| `logs`               | array\|null  | Array of log entries — see **`Log` object** below. `null` unless `?logs=true` is passed.  |
| `statusChecks`       | array\|null  | Array of status-check log entries. `null` unless `?logs=true` is passed.                  |
| `published`          | array        | Environments where this deployment is currently live — see **`Published` object** below.  |
| `uploadResult`       | object\|null | Upload size breakdown — see **`UploadResult` object** below. `null` if not yet available. |

**`Commit` object:**

| Field     | Type   | Description                                     |
| --------- | ------ | ----------------------------------------------- |
| `author`  | string | Name of the commit author. `null` if unknown.   |
| `message` | string | Commit message. `null` if unknown.              |
| `sha`     | string | Full commit SHA. `null` until the build starts. |

**`Log` object:**

| Field      | Type    | Description                    |
| ---------- | ------- | ------------------------------ |
| `title`    | string  | Step title.                    |
| `message`  | string  | Log output for this step.      |
| `status`   | boolean | `true` if this step succeeded. |
| `duration` | number  | Step duration in seconds.      |

**`Published` object:**

| Field        | Type   | Description                                           |
| ------------ | ------ | ----------------------------------------------------- |
| `envId`      | string | Environment ID where this deployment is published.    |
| `percentage` | number | Traffic percentage routed to this deployment (0–100). |

**`UploadResult` object:**

| Field             | Type   | Description                           |
| ----------------- | ------ | ------------------------------------- |
| `clientBytes`     | number | Size of client-side assets in bytes.  |
| `serverBytes`     | number | Size of server-side assets in bytes.  |
| `serverlessBytes` | number | Size of serverless function in bytes. |

### Error responses

| Status | Condition                                                                               |
| ------ | --------------------------------------------------------------------------------------- |
| `403`  | Missing/invalid API key, team token does not own the app, or user is not a team member. |
| `404`  | Deployment not found, or it does not belong to the environment of the API key.          |
| `500`  | Internal server error.                                                                  |

### Examples

```bash
# Poll deployment status (no logs)
curl -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/deployments/8241'
```

```json
// Example response (logs omitted by default)
{
  "deployment": {
    "id": "8241",
    "appId": "1510",
    "envId": "305",
    "envName": "production",
    "displayName": "my-app",
    "repo": "github/acme/my-app",
    "branch": "release/2.0",
    "status": "success",
    "error": "",
    "isAutoDeploy": false,
    "isAutoPublish": true,
    "stoppedManually": false,
    "statusChecksPassed": true,
    "apiPathPrefix": "",
    "previewUrl": "https://8241--my-app.stormkit.dev",
    "detailsUrl": "/apps/1510/environments/305/deployments/8241",
    "duration": 42,
    "createdAt": "1742300400",
    "stoppedAt": "1742300442",
    "commit": {
      "author": "Jane Doe",
      "message": "fix: improve performance",
      "sha": "16ab41e8"
    },
    "logs": null,
    "statusChecks": null,
    "published": [{ "envId": "305", "percentage": 100 }],
    "uploadResult": {
      "clientBytes": 204800,
      "serverBytes": 0,
      "serverlessBytes": 0
    }
  }
}
```

```bash
# Fetch deployment with build logs
curl -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/deployments/8241?logs=true'
```

```json
// Example response (with logs)
{
  "deployment": {
    "id": "8241",
    "...": "...",
    "logs": [
      {
        "title": "Install dependencies",
        "message": "...",
        "status": true,
        "duration": 12
      }
    ],
    "statusChecks": []
  }
}
```

---

## GET /v1/deployments/{id}/poll

Returns only the current status of a deployment. Use this for lightweight polling loops — it avoids fetching the full deployment object on every tick.

**Base URL:** `https://api.stormkit.io`

**Authentication:** At least an environment-level API key passed as the `Authorization` header.

### Path parameters

| Parameter | Type   | Description        |
| --------- | ------ | ------------------ |
| `id`      | string | The deployment ID. |

### Query parameters

| Parameter | Type   | Required    | Description                                                                                                                                                         |
| --------- | ------ | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `envId`   | string | Conditional | Required when using an app, team, or user-level API key. Identifies the environment that owns the deployment. Environment-level keys do not require this parameter. |

### Response — 200 OK

| Field    | Type   | Description                                        |
| -------- | ------ | -------------------------------------------------- |
| `status` | string | Current status: `running`, `success`, or `failed`. |

### Error responses

| Status | Condition                                                                               |
| ------ | --------------------------------------------------------------------------------------- |
| `403`  | Missing/invalid API key, or token does not have access to the deployment's environment. |
| `404`  | Deployment not found, or it does not belong to the environment of the API key.          |
| `500`  | Internal server error.                                                                  |

### Example

```bash
# Poll until the deployment is no longer running
while true; do
  STATUS=$(curl -s -H 'Authorization: <api_key>' \
    'https://api.stormkit.io/v1/deployments/8241/poll' \
    | jq -r '.status')
  echo "Status: $STATUS"
  [ "$STATUS" != "running" ] && break
  sleep 5
done
```

```json
// Example response
{ "status": "running" }
```

---

## POST /v1/deployments/{id}/publish

Makes a deployment live for the environment associated with the API key.

**Base URL:** `https://api.stormkit.io`

**Authentication:** At least an environment-level API key passed as the `Authorization` header.

### Path parameters

| Parameter | Type   | Description        |
| --------- | ------ | ------------------ |
| `id`      | string | The deployment ID. |

### Response — 200 OK

| Field | Type    | Description               |
| ----- | ------- | ------------------------- |
| `ok`  | boolean | Always `true` on success. |

### Error responses

| Status | Condition                                                                               |
| ------ | --------------------------------------------------------------------------------------- |
| `400`  | Deployment does not have a successful build and cannot be published.                    |
| `403`  | Missing/invalid API key, or token does not have access to the deployment's environment. |
| `404`  | Deployment not found, or it does not belong to the environment of the API key.          |
| `500`  | Internal server error.                                                                  |

### Examples

```bash
# Publish a deployment at 100%
curl -X POST \
     -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/deployments/8241/publish'
```

```json
// Example response
{ "ok": true }
```

---

## DELETE /v1/deployments/{id}

Soft-deletes a deployment and resets the environment cache. The deployment must belong to the environment of the API key.

**Base URL:** `https://api.stormkit.io`

**Authentication:** At least an environment-level API key passed as the `Authorization` header.

### Path parameters

| Parameter | Type   | Description        |
| --------- | ------ | ------------------ |
| `id`      | string | The deployment ID. |

### Response — 200 OK

| Field | Type    | Description               |
| ----- | ------- | ------------------------- |
| `ok`  | boolean | Always `true` on success. |

### Error responses

| Status | Condition                                                                               |
| ------ | --------------------------------------------------------------------------------------- |
| `403`  | Missing/invalid API key, or token does not have access to the deployment's environment. |
| `404`  | Deployment not found, or it does not belong to the environment of the API key.          |
| `500`  | Internal server error.                                                                  |

### Example

```bash
curl -X DELETE \
     -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/deployments/8241'
```

```json
{ "ok": true }
```
