---
title: Redirects API
description: Documentation on accessing redirects through Stormkit API.
---

# Redirects API

<details>

<summary>
  <span>POST </span><span>/v1/redirects</span>
</summary>

Set Environment Level Redirects.

```typescript
interface Request {
  "redirects": []Redirect
}

interface Response {
  "redirects": []Redirect
}
```

```bash
# Example

curl -X POST \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/redirects` \
     -d '{ "redirects": [{ "from": "/path", "to": "/new-path" }] }'
```

</details>

<details>

<summary>
  <span>GET </span><span>/v1/redirects</span>
</summary>

Get Environment Level Redirects.

```typescript
interface Response {
  "redirects": []Redirect
}
```

```bash
# Example

curl -X GET \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/redirects'
```

```json
{
  "redirects": [{ "from": "*", "to": "index.html" }]
}
```

</details>

#### Syntax

```typescript
interface Redirect {
  from: string
  to: string
  status?: 200 | 301 | 302
  assets?: boolean
  hosts?: string[]
}
```
