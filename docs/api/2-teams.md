---
title: Teams API
description: Retrieve the list of teams that belong to your Stormkit account using the Teams API.
---

# Teams API

## Overview

The Teams API lets you list the teams your account belongs to. Use the returned `id` values as `teamId` in other API endpoints such as [Apps](/docs/api/apps) and [Environments](/docs/api/environments).

---

## GET /v1/teams

Returns all teams the authenticated user belongs to. On Community Edition (self-hosted) instances only the default (personal) team is returned; Enterprise Edition instances return all teams.

**Base URL:** `https://api.stormkit.io`

**Authentication:** User-level API key passed as the `Authorization` header. To generate one: **Profile → Account → API Keys**.

### Response — 200 OK

| Field    | Type     | Description              |
| -------- | -------- | ------------------------ |
| `teams`  | `Team[]` | Array of team objects.   |

**`Team` object:**

| Field             | Type    | Description                                                                   |
| ----------------- | ------- | ----------------------------------------------------------------------------- |
| `id`              | string  | Unique team ID. Use this as `teamId` in other endpoints.                      |
| `name`            | string  | Human-readable team name.                                                     |
| `slug`            | string  | URL-friendly team identifier. Unique within the response; suffixed with `-id` when two teams share the same slug. |
| `isDefault`       | boolean | `true` for the personal (default) team automatically created with the account. |
| `currentUserRole` | string  | Role of the authenticated user in this team: `owner`, `admin`, or `developer`. |

### Error responses

| Status | Condition                             |
| ------ | ------------------------------------- |
| `403`  | Missing or invalid API key.           |
| `500`  | Internal server error.                |

### Example

```bash
curl -X GET \
     -H 'Authorization: <user_api_key>' \
     'https://api.stormkit.io/v1/teams'
```

```json
// Example response
{
  "teams": [
    {
      "id": "7",
      "name": "Personal",
      "slug": "personal",
      "isDefault": true,
      "currentUserRole": "owner"
    },
    {
      "id": "42",
      "name": "Acme Corp",
      "slug": "acme-corp",
      "isDefault": false,
      "currentUserRole": "admin"
    }
  ]
}
```
