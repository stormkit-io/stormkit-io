---
title: Snippets API
description: Documentation on accessing snippets through Stormkit API.
---

# Snippets API

<details>

<summary>
  <span>POST </span><span>/v1/snippets</span>
</summary>

Add snippets.

```typescript
interface Request {
  snippets: []Snippet
}

interface Response {
  snippets: []Snippet
}
```

```bash
# Example

curl -X POST \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/snippets' \
     -d '{ "snippets": [{ "title": "Snippet title", "content":  "Hello World", "enabled":  true, "prepend":  false, "location": "head", "rules": { "hosts": ["example.org", "*.dev"], "path": "/my-path" } }] }'
```

| Response | Definition                                           |
| -------- | ---------------------------------------------------- |
| 201      | Snippet was created successfully.                    |
| 400      | Snippet is invalid. Check your parameters.           |
| 409      | Snippet content is duplicate within the environment. |

</details>

<details>

<summary>
  <span>GET </span><span>/v1/snippets</span>
</summary>

Return all snippets of an environment.

```typescript
interface QueryString {
  afterId?: string
  hosts?: string
  title?: string
}

interface Response {
  snippets: []Redirect
  pagination: Pagination
}
```

```bash
# Example

curl -X GET \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/snippets'
```

```json
{
  "snippets": [
    {
      "id": "1501",
      "title": "Snippet title",
      "content": "Hello World",
      "enabled": true,
      "prepend": false,
      "location": "head",
      "rules": {
        "hosts": ["example.org", "*.dev"],
        "path": "/my-path"
      }
    }
  ],
  "pagination": {
    "hasNextPage": false
  }
}
```

To filter snippets by hosts, you can use the `hosts` querystring parameter. Multiple hosts should be separated by comma (`,`).

To filter snippets by title, use the `title` querystring parameter. Ensure the querystring is properly encoded.

To paginate results, you can specify the `afterId` querystring parameter. This value is returned in the first `GET` request.

</details>

<details>

<summary>
  <span>PUT </span><span>/v1/snippets</span>
</summary>

Update the given snippet.

```typescript
interface Request {
  snippet: Snippet
}

interface Response {
  ok: boolean
}
```

```bash
# Example

curl -X PUT \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/snippets' \
     -d '{ "snippet": { "id": 1501, "title": "New title", "content":  "Hello World", "enabled":  true, "prepend":  false, "location": "head", "rules": { "hosts": ["example.org", "*.dev"], "path": "/my-path" } } }'
```

| Response | Definition                                           |
| -------- | ---------------------------------------------------- |
| 200      | Snippet was updated successfully.                    |
| 400      | Snippet is invalid. Check your parameters.           |
| 409      | Snippet content is duplicate within the environment. |

</details>

<details>

<summary>
  <span>DELETE </span><span>/v1/snippets</span>
</summary>

Delete Snippets with the given ids. You can delete maximum `100` snippets at a time.

```typescript
interface QueryString {
  ids: []string
}

interface Response {
  ok: boolean
}
```

```bash
# Example

curl -X PUT \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/snippets?ids=1501,5061'
```

</details>

#### Syntax

```typescript
interface Snippet {
  id: string
  title: string
  content: string
  enabled: boolean
  location: 'head' | 'body'
  prepend: boolean
  rules?: {
    hosts?: []string
    path?: string
  }
}

interface Pagination {
  hasNextPage: boolean
  afterId?: string
}
```

| Property    | Definition                                                                                               |
| ----------- | -------------------------------------------------------------------------------------------------------- |
| id          | The unique id of the snippet.                                                                            |
| title       | The title is used internally for describing what the snippet does.                                       |
| content     | The content that is going to be injected into the page.                                                  |
| enabled     | Whether the snippet is enabled or not.                                                                   |
| location    | The injection location for the snippet.                                                                  |
| prepend     | If true, the snippet will be injected as the first child to the `location`. Otherwise as the last child. |
| rules.hosts | Inject snippet only for the provided hosts.                                                              |
| rules.path  | If specified, snippet will be injected only when the path matches.                                       |

**Note:** To inject snippets for all development endpoints, specify `*.dev` for the `rules.hosts` property. Specifying an indivual deployment endpoint will enable the snippet for all development endpoints.
