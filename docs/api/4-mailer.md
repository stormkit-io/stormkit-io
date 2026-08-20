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
  delivered: boolean // false when the environment has no SMTP configuration
}
```

An environment without a mailer still records the email without sending it, so
`ok` alone does not mean the message was handed to an SMTP server. Check
`delivered` when you are using this endpoint to verify that a mailer works.

```bash
# Example

curl -X POST \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/mail' \
     -d '{ "to": "joe@example.org", "from": "Jane Doe <jane@example.org>", "subject": "Hello Joe", "body": "Hi,<br/>How are you?" }'
```

</details>

<details>

<summary>
  <span>GET </span><span>/v1/mailer/config</span>
</summary>

Return the SMTP configuration of an environment.

The password is never returned. When one is stored, `password` contains the
placeholder `****-****-****-****`; when no mailer is configured at all,
`config` is `null`.

```typescript
interface Response {
  config: {
    host: string
    port: string
    username: string
    password: string // always the placeholder
  } | null
}
```

```bash
# Example

curl -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/mailer/config?appId=<app_id>&envId=<env_id>'
```

</details>

<details>

<summary>
  <span>POST </span><span>/v1/mailer/config</span>
</summary>

Update the SMTP configuration of an environment. Only the fields you provide
are changed; omitted fields keep their stored value.

`password` is write-only. Omit it — or send the `****-****-****-****`
placeholder returned by the GET endpoint — to keep the stored password. `host`,
`username` and a password must all be set before email can be sent.

Changing `smtpHost` or `username` **clears the stored password**, because a
credential you cannot read must not follow the configuration to a different
server or account. Send a new `password` in the same request when you change
either; otherwise the request is rejected with `Password is a required field.`
and nothing is written.

```typescript
interface Request {
  smtpHost?: string
  smtpPort?: string // defaults to 587
  username?: string
  password?: string
}

interface Response {
  config: {
    host: string
    port: string
    username: string
    password: string // always the placeholder
  }
}
```

```bash
# Example

curl -X POST \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/json' \
     'https://api.stormkit.io/v1/mailer/config' \
     -d '{ "appId": "<app_id>", "envId": "<env_id>", "smtpHost": "smtp.example.org", "smtpPort": "587", "username": "jane@example.org", "password": "<smtp_password>" }'
```

</details>

<details>

<summary>
  <span>GET </span><span>/v1/mailer/emails</span>
</summary>

Return the last 100 emails recorded for an environment, newest first.

Message bodies are not included, and recipient addresses are masked
(`j***@example.com`). The mailer log stores magic-link emails verbatim, so a
body contains a sign-in link — single-use and valid for 15 minutes — and the
full recipient list is the app's end-user mailing list. The remaining metadata
is enough to confirm that an email was sent. Full bodies and addresses remain
visible in the dashboard, behind a session login.

```typescript
interface Response {
  emails: Array<{
    id: string
    envId: string
    from: string
    to: string
    subject: string
    sentAt: number
  }>
}
```

```bash
# Example

curl -H 'Authorization: <api_key>' \
     'https://api.stormkit.io/v1/mailer/emails?appId=<app_id>&envId=<env_id>'
```

</details>
