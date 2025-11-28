---
title: Mailer API
description: Send transactional emails programmatically using Stormkit API. Simple, efficient and free email service.
---

# Mailer API

<details>

<summary>
  <span>POST </span><span>/v1/mail</span>
</summary>

Requirements:

- Make sure the [mailer is configured](/docs/features/mailer) for your environment.
- Make sure you have generated an Environment-level API Key.

Send an email.

```typescript
interface Request {
  to: string
  from: string
  subject: string
  body: string
}

interface Response {
  ok: boolean
}
```

```bash
# Example

curl -X POST \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/mail' \
     -d '{ "to": "joe@example.org", "from": "Jane Doe <jane@example.org>", "subject": "Hello Joe", "body": "Hi,<br/>How are you?" }'
```

</details>
