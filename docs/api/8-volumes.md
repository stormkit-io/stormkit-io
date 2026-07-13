---
title: Volumes API
description: Upload files to your environment's file storage using the Stormkit Volumes API.
---

# Volumes API

## Overview

The Volumes API lets you upload files to your environment's configured file storage (filesystem or S3-compatible object storage) programmatically. An environment-level API key is required.

---

## POST /v1/volumes

Uploads one or more files to the environment associated with the API key.

**Base URL:** `https://api.stormkit.io`

**Authentication:** An environment-level API key passed as the `Authorization` header. To generate one: **Your App** → **Your Environment** → **Config** → **Other** → **API Keys**. App, team, or user-level keys are also accepted — in those cases `envId` must be included in the request body or query string to identify the target environment.

**Content-Type:** `multipart/form-data`

### Request fields

| Field   | Type   | Required | Description                                                                                                                                                                                        |
| ------- | ------ | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `files` | file   | Yes      | One or more files to upload. Repeat the field for multiple files. Files with the same name within a single request are deduplicated — the last one wins.                                           |
| `envId` | string | No       | Target environment ID. Required when using app, team, or user-level API keys. May be provided as a multipart form field or as an `envId` query parameter. Not required for environment-level keys. |

### Response — 200 OK

| Field    | Type            | Description                                                                          |
| -------- | --------------- | ------------------------------------------------------------------------------------ |
| `files`  | array of object | Successfully uploaded files. See **File object** below.                              |
| `failed` | object          | Map of `filename → error message` for files that could not be uploaded. May be `{}`. |

**File object:**

| Field        | Type    | Description                                                                        |
| ------------ | ------- | ---------------------------------------------------------------------------------- |
| `id`         | string  | Unique file ID.                                                                    |
| `name`       | string  | File name including any directory structure (e.g. `images/logo.png`).              |
| `size`       | number  | File size in bytes.                                                                |
| `isPublic`   | boolean | Whether the file is publicly accessible.                                           |
| `publicLink` | string  | Publicly accessible URL. Present only when `isPublic` is `true`.                   |
| `createdAt`  | number  | Unix timestamp (seconds) when the file was first uploaded.                         |
| `updatedAt`  | number  | Unix timestamp (seconds) when the file was last replaced. Omitted on first upload. |

### Upload limits

| Limit                                 | Default | Environment variable                           |
| ------------------------------------- | ------- | ---------------------------------------------- |
| Maximum upload size (per request)     | 50 MB   | `STORMKIT_VOLUMES_MAX_UPLOAD_SIZE` (bytes)     |
| Memory buffer before spilling to disk | 100 MB  | `STORMKIT_VOLUMES_UPLOAD_MEMORY_LIMIT` (bytes) |

On **Stormkit Cloud** the maximum upload size is fixed. On **self-hosted** instances you can raise or lower both limits by setting the environment variables on your Stormkit server process. For example, to allow 200 MB uploads:

```bash
STORMKIT_VOLUMES_MAX_UPLOAD_SIZE=209715200   # 200 MB
STORMKIT_VOLUMES_UPLOAD_MEMORY_LIMIT=209715200
```

### Error responses

| Status | Condition                                                           |
| ------ | ------------------------------------------------------------------- |
| `400`  | No files provided or storage not configured.                        |
| `403`  | Missing or invalid API key.                                         |
| `404`  | The environment identified by the API key or `envId` was not found. |
| `413`  | Payload exceeds the maximum upload size (50 MB on Stormkit Cloud).  |

---

### Example

```bash
curl -X POST https://api.stormkit.io/v1/volumes \
  -H "Authorization: SK_your_env_key" \
  -F "files=@./logo.png" \
  -F "files=@./assets/banner.jpg"
```

**Response:**

```json
{
  "files": [
    {
      "id": "123",
      "name": "logo.png",
      "size": 48210,
      "isPublic": false,
      "createdAt": 1710000000
    },
    {
      "id": "124",
      "name": "assets/banner.jpg",
      "size": 102400,
      "isPublic": false,
      "createdAt": 1710000001
    }
  ],
  "failed": {}
}
```
