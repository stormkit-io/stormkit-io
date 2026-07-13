---
title: Triggers API
description: Documentation on managing periodic function triggers through Stormkit API.
---

# Triggers API

Periodic triggers send an HTTP request to a URL on a cron schedule. Use them to
run recurring tasks, health checks, or scheduled calls to your functions.

All endpoints are scoped to an **environment**. Authenticate with an
environment-level API key, or pass `envId` (query string for `GET`/`DELETE`,
request body for `POST`/`PATCH`) when using an app-, team- or user-level key.

Cron expressions are evaluated in **UTC**. See [crontab.guru](https://crontab.guru)
for help writing them.

These endpoints are also available as MCP tools (`list_triggers`,
`create_trigger`, `update_trigger`, `delete_trigger`, `invoke_trigger`,
`get_trigger_logs`) — see the [MCP Server](/docs/api/mcp) docs.

> **Request header values are masked.** Triggers commonly carry secrets in their
> request headers (e.g. `Authorization`, `X-Api-Key`). For security, header
> _values_ are blanked out (returned as empty strings) in every response that
> lists triggers or their logs — the header _names_ are kept so you can see
> what is configured. The stored values are used unchanged when the trigger
> actually runs.

<details>

<summary>
  <span>POST </span><span>/v1/trigger</span>
</summary>

Create a trigger.

```typescript
interface Request {
  envId?: string // Required when the API key is not environment-scoped.
  cron: string // Cron expression, evaluated in UTC.
  status: boolean // Whether the trigger is active.
  options: {
    method: string // GET, POST, HEAD, PATCH or DELETE.
    url: string // http/https URL to call.
    payload?: string // Request body, sent for non-GET methods.
    headers?: Record<string, string> | string // Either a map or "key:value;key:value".
  }
}

interface Response {
  trigger: Trigger
}
```

```bash
# Example

curl -X POST \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/json' \
     'https://api.stormkit.io/v1/trigger' \
     -d '{
       "cron": "*/5 * * * *",
       "status": true,
       "options": {
         "method": "POST",
         "url": "https://www.example.org/cron",
         "payload": "{ \"hello\": \"world\" }",
         "headers": { "Content-Type": "application/json" }
       }
     }'
```

```json
{
  "trigger": {
    "id": "18914",
    "envId": "1500",
    "cron": "*/5 * * * *",
    "status": true,
    "nextRunAt": 1712418330,
    "options": {
      "method": "POST",
      "url": "https://www.example.org/cron",
      "payload": "{ \"hello\": \"world\" }",
      "headers": { "Content-Type": "" }
    }
  }
}
```

> Header values are masked in the response (see the note above); the value you
> sent is stored and used when the trigger runs.

Validation errors return `400` with a map of field errors, e.g.
`{ "cron": "Invalid cron format" }` or `{ "url": "Invalid URL" }`.

</details>

<details>

<summary>
  <span>PATCH </span><span>/v1/trigger</span>
</summary>

Update an existing trigger. The `id` must belong to the authenticated
environment.

> `options.headers` is replaced wholesale with what you send. Because list
> responses mask header values, always re-send the real values you want to keep
> — sending back the masked (blanked) values will overwrite the stored ones.

```typescript
interface Request {
  id: string
  envId?: string // Required when the API key is not environment-scoped.
  cron: string
  status: boolean
  options: {
    method: string
    url: string
    payload?: string
    headers?: Record<string, string> | string
  }
}

interface Response {
  ok: boolean
}
```

```bash
# Example

curl -X PATCH \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/json' \
     'https://api.stormkit.io/v1/trigger' \
     -d '{
       "id": "18914",
       "cron": "0 * * * *",
       "status": false,
       "options": { "method": "GET", "url": "https://www.example.org/cron" }
     }'
```

```json
{
  "ok": true
}
```

</details>

<details>

<summary>
  <span>DELETE </span><span>/v1/trigger</span>
</summary>

Delete a trigger by its id.

```typescript
interface QueryString {
  triggerId: string
  envId?: string // Required when the API key is not environment-scoped.
}

interface Response {
  ok: boolean
}
```

```bash
# Example

curl -X DELETE \
     -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/trigger?triggerId=18914'
```

```json
{
  "ok": true
}
```

</details>

<details>

<summary>
  <span>POST </span><span>/v1/trigger/invoke</span>
</summary>

Run a trigger immediately, regardless of its schedule or status. The request is
executed synchronously and the resulting log is returned and stored alongside
the scheduled runs.

```typescript
interface Request {
  id: string
  envId?: string // Required when the API key is not environment-scoped.
}

interface Response {
  log: TriggerLog
}
```

```bash
# Example

curl -X POST \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/json' \
     'https://api.stormkit.io/v1/trigger/invoke' \
     -d '{ "id": "18914" }'
```

```json
{
  "log": {
    "triggerId": "18914",
    "request": {
      "url": "https://www.example.org/cron",
      "method": "GET",
      "headers": null,
      "payload": ""
    },
    "response": {
      "code": 200,
      "body": "pong"
    }
  }
}
```

</details>

<details>

<summary>
  <span>GET </span><span>/v1/triggers</span>
</summary>

List the triggers configured for an environment.

```typescript
interface QueryString {
  envId?: string // Required when the API key is not environment-scoped.
}

interface Response {
  triggers: []Trigger
}
```

```bash
# Example

curl -X GET \
     -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/triggers'
```

```json
{
  "triggers": [
    {
      "id": "18914",
      "envId": "1500",
      "cron": "*/5 * * * *",
      "status": true,
      "nextRunAt": 1712418330,
      "options": {
        "method": "POST",
        "url": "https://www.example.org/cron",
        "payload": "{ \"hello\": \"world\" }",
        "headers": null
      }
    }
  ]
}
```

</details>

<details>

<summary>
  <span>GET </span><span>/v1/trigger/logs</span>
</summary>

Return the last 25 executions of a trigger, most recent first. Each scheduled
run and each `invoke` call is recorded.

```typescript
interface QueryString {
  triggerId: string
  envId?: string // Required when the API key is not environment-scoped.
}

interface Response {
  logs: []TriggerLog
}
```

```bash
# Example

curl -X GET \
     -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/trigger/logs?triggerId=18914'
```

```json
{
  "logs": [
    {
      "id": "5001",
      "triggerId": "18914",
      "request": {
        "url": "https://www.example.org/cron",
        "method": "GET",
        "headers": null,
        "payload": ""
      },
      "response": {
        "code": 200,
        "body": "pong"
      },
      "createdAt": 1712418330
    }
  ]
}
```

</details>

#### Syntax

```typescript
interface Trigger {
  id: string
  envId: string
  cron: string
  status: boolean
  nextRunAt: number // Unix timestamp of the next scheduled run.
  options: {
    method: string
    url: string
    payload: string
    headers: Record<string, string> | null
  }
}

interface TriggerLog {
  id: string
  triggerId: string
  request: {
    url: string
    method: string
    headers: Record<string, string> | null
    payload: string
  }
  response: {
    code?: number // HTTP status code of the response, when one was received.
    body?: string // Response body.
    error?: string // Set instead of code/body when the request failed.
  }
  createdAt: number
}
```

| Property    | Definition                                                                  |
| ----------- | --------------------------------------------------------------------------- |
| id          | The unique id of the trigger.                                               |
| envId       | The environment the trigger belongs to.                                     |
| cron        | The cron expression, evaluated in UTC.                                       |
| status      | Whether the trigger is active. Inactive triggers are not scheduled.         |
| nextRunAt   | Unix timestamp of the next scheduled run. `0` when the trigger is inactive. |
| options     | The HTTP request executed on each run.                                      |
