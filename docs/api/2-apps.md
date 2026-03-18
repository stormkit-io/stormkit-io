---
title: Apps API
description: Retrieve the list of applications that belong to your Stormkit account using the Apps API.
---

# Apps API

## Overview

The Apps API lets you list applications in your Stormkit account. It is useful for discovering `appId` values that are required by other API endpoints.

---

## GET /v1/apps

Returns a paginated list of applications belonging to the authenticated user.

**Base URL:** `https://api.stormkit.io`

**Authentication:** User or Team level API key passed as the `Authorization` header (no `Bearer` prefix required). To generate one: Profile → Account → API Keys.

### Query parameters

| Parameter     | Type   | Required | Default | Description                                                                   |
| ------------- | ------ | -------- | ------- | ----------------------------------------------------------------------------- |
| `teamId`      | number | **Yes**  | —       | ID of the team to scope results to. The API key owner must be a member.       |
| `from`        | number | No       | `0`     | Pagination offset. Must be ≥ 0.                                               |
| `repo`        | string | No       | —       | Exact case-insensitive match on the repository path (e.g. `github/org/repo`). |
| `displayName` | string | No       | —       | Exact case-insensitive match on the application display name.                 |

> `repo` and `displayName` can be combined and are applied as AND conditions.

### Response — 200 OK

| Field         | Type    | Description                                                                                      |
| ------------- | ------- | ------------------------------------------------------------------------------------------------ |
| `apps`        | `App[]` | Array of application objects for the current page.                                               |
| `hasNextPage` | boolean | `true` when more results exist. To fetch the next page, add `from=<current_offset + len(apps)>`. |

**`App` object:**

| Field          | Type    | Description                                                                         |
| -------------- | ------- | ----------------------------------------------------------------------------------- |
| `id`           | string  | Unique application ID. Use this as `appId` in other endpoints.                      |
| `displayName`  | string  | Human-readable name of the application.                                             |
| `repo`         | string  | Repository path (e.g. `github/org/repo`). Empty when `isBare` is `true`.            |
| `isBare`       | boolean | `true` when no repository is linked.                                                |
| `userId`       | string  | ID of the user who owns the application.                                            |
| `teamId`       | string  | ID of the team the application belongs to.                                          |
| `defaultEnvId` | string  | ID of the default environment. Use this as `envId` in environment-scoped endpoints. |
| `createdAt`    | string  | Unix timestamp (seconds) when the application was created.                          |

### Error responses

| Status | Condition                                                           |
| ------ | ------------------------------------------------------------------- |
| `400`  | Missing or invalid parameter (e.g. `teamId` omitted, non-integer).  |
| `403`  | Missing/invalid API key, or user is not a member of the given team. |
| `500`  | Internal server error.                                              |

### Pagination

The page size is fixed at 20 items. To iterate all pages:

1. Start with `from=0`.
2. If `hasNextPage` is `true`, set `from = from + len(apps)` and repeat.
3. Stop when `hasNextPage` is `false`.

### Examples

```bash
# First page
curl -X GET \
     -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/apps?teamId=7'
```

```bash
# Second page
curl -X GET \
     -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/apps?teamId=7&from=20'
```

```bash
# Exact match by repository
curl -X GET \
     -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/apps?teamId=7&repo=github/acme/my-app'
```

```bash
# Exact match by display name
curl -X GET \
     -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/apps?teamId=7&displayName=my-app-abc123'
```

```json
// Example response
{
  "apps": [
    {
      "id": "1510",
      "displayName": "my-app-abc123",
      "repo": "github/acme/my-app",
      "isBare": false,
      "userId": "42",
      "teamId": "7",
      "defaultEnvId": "305",
      "createdAt": "1700489144"
    }
  ],
  "hasNextPage": false
}
```

---

## GET /v1/app

Returns a single application.

**Base URL:** `https://api.stormkit.io`

**Authentication:** App-level, user-level, or team-level API key passed as the `Authorization` header. When using a user or team-level key, `appId` must be provided as a query parameter and the authenticated principal must be a member of the team that owns the application.

### Query parameters

| Parameter | Type   | Required                                    | Description                                                                                                       |
| --------- | ------ | ------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `appId`   | number | **Yes** (unless using an app-level API key) | The ID of the application to fetch. Not required when using an app-level key — the app is derived from the token. |

### Response — 200 OK

| Field | Type  | Description             |
| ----- | ----- | ----------------------- |
| `app` | `App` | The application object. |

**`App` object:**

| Field          | Type    | Description                                                                         |
| -------------- | ------- | ----------------------------------------------------------------------------------- |
| `id`           | string  | Unique application ID.                                                              |
| `displayName`  | string  | Human-readable name of the application.                                             |
| `repo`         | string  | Repository path (e.g. `github/org/repo`). Empty when `isBare` is `true`.            |
| `isBare`       | boolean | `true` when no repository is linked.                                                |
| `userId`       | string  | ID of the user who owns the application.                                            |
| `teamId`       | string  | ID of the team the application belongs to.                                          |
| `defaultEnvId` | string  | ID of the default environment. Use this as `envId` in environment-scoped endpoints. |
| `createdAt`    | string  | Unix timestamp (seconds) when the application was created.                          |

### Error responses

| Status | Condition                                                                     |
| ------ | ----------------------------------------------------------------------------- |
| `403`  | Missing/invalid API key, or the authenticated principal is not a team member. |
| `404`  | No application found with the given ID.                                       |
| `500`  | Internal server error.                                                        |

### Examples

```bash
# Using an app-level key (appId is derived from the token)
curl -X GET \
     -H 'Authorization: <app_api_key>' \
     'https://api.stormkit.io/v1/app'
```

```bash
# Using a user or team-level key (appId must be provided as a query parameter)
curl -X GET \
     -H 'Authorization: <user_or_team_api_key>' \
     'https://api.stormkit.io/v1/app?appId=1510'
```

```json
// Example response
{
  "app": {
    "id": "1510",
    "displayName": "my-app-abc123",
    "repo": "github/acme/my-app",
    "isBare": false,
    "userId": "42",
    "teamId": "7",
    "defaultEnvId": "305",
    "createdAt": "1700489144"
  }
}
```

---

## GET /v1/app/config

Returns the runtime configuration for published deployment matching a given hostname. This endpoint is primarily used to debug the application config.

**Base URL:** `https://api.stormkit.io`

**Authentication:** App-level API key passed as the `Authorization` header. To generate one: **Your App** → **Your Environment** → **Config** → **Other** → **API Keys** (app scope).

### Query parameters

| Parameter  | Type   | Required | Description                                                           |
| ---------- | ------ | -------- | --------------------------------------------------------------------- |
| `hostName` | string | **Yes**  | The hostname of the deployed application (e.g. `my-app.stormkit.io`). |

### Response — 200 OK

Returns an array of configuration objects for all matching published deployments.

| Field     | Type       | Description                             |
| --------- | ---------- | --------------------------------------- |
| `configs` | `Config[]` | Array of runtime configuration objects. |

**`Config` object:**

| Field           | Type       | Description                                                                    |
| --------------- | ---------- | ------------------------------------------------------------------------------ |
| `deploymentId`  | string     | ID of the published deployment.                                                |
| `appId`         | string     | ID of the application.                                                         |
| `envId`         | string     | ID of the environment.                                                         |
| `percentage`    | number     | Traffic percentage routed to this deployment (0–100).                          |
| `apiPathPrefix` | string     | URL path prefix under which serverless API functions are served.               |
| `domains`       | `string[]` | Custom domains associated with this deployment. `null` if none are configured. |
| `staticFiles`   | object     | Map of URL paths to static file metadata.                                      |
| `envVariables`  | object     | Map of environment variable names to their values injected at runtime.         |
| `updatedAt`     | string     | Unix timestamp (seconds) of the last configuration update. `null` if not set.  |

### Error responses

| Status | Condition                                       |
| ------ | ----------------------------------------------- |
| `204`  | No published deployment found for the hostname. |
| `403`  | Missing or invalid API key.                     |
| `500`  | Internal server error.                          |

### Example

```bash
curl -X GET \
     -H 'Authorization: <app_api_key>' \
     'https://api.stormkit.io/v1/app/config?hostName=my-app.stormkit.io'
```

```json
// Example response
{
  "configs": [
    {
      "deploymentId": "8241",
      "appId": "1510",
      "envId": "305",
      "percentage": 100,
      "apiPathPrefix": "/api",
      "domains": null,
      "staticFiles": {
        "/index": {
          "fileName": "index",
          "headers": {
            "content-type": "text/html; charset=utf-8"
          }
        }
      },
      "envVariables": {
        "NODE_ENV": "production",
        "SK_APP_ID": "1510",
        "SK_ENV": "production"
      },
      "updatedAt": null
    }
  ]
}
```
