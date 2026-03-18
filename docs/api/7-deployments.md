---
title: Deployments API
description: Trigger new deployments programmatically using the Stormkit Deployments API.
---

# Deployments API

## Overview

The Deployments API lets you trigger new deployments from any CI/CD pipeline, script, or agent. At minimum an environment-level API key is required. App, team, and user-level keys are also accepted provided an `envId` is supplied in the request body.

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

- Deployments run **asynchronously**. The `201` response confirms the deployment was queued — it does not mean the build has completed. Poll the deployment status using the deployment `id` once a GET endpoint is available.
- When using an **environment-level key**, the `envId` field in the request body has no effect — the environment is always the one the key was issued for. To target a different environment, use that environment's own key.
- When using an **app, team, or user-level key**, `envId` is required. The request will return `404` if the environment does not exist or is not accessible to the key.
