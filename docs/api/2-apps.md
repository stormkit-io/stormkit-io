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

**Authentication:** User-level API key passed as the `Authorization` header (no `Bearer` prefix required). To generate one: Profile → Account → API Keys.

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
