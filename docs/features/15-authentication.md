---
title: Add end-user authentication with Stormkit Auth
description: Learn how to configure Stormkit Auth to add sign-in to your application — email and password, magic links, and Google or X (Twitter) OAuth — and how to consume the session in your frontend and API.
---

# Authentication

## Overview

**Stormkit Auth** adds end-user authentication to the application you host on
Stormkit. Your visitors can register and sign in with **email and password**, a
**magic link**, or an OAuth provider (**Google**, **X / Twitter**), and your
app receives a signed session token it can use to identify the user on every
request.

This is different from the [Auth Wall](/docs/features/auth-wall), which gates
access to a deployment for *your* team. Stormkit Auth is for *your application's*
own users.

You configure it under **Environment** > **Authentication**.

> **Requirements**
>
> - Stormkit Auth is available on **self-hosted** installations only.
> - The environment must have a **[database](/docs/features/database)** attached —
>   users and sessions are stored there. If no database is attached, the
>   Authentication tab will prompt you to configure one first.
> - The **Magic Link** and **email / password** providers send email (sign-in
>   links and verification messages), so the environment's
>   **[Mailer](/docs/features/mailer)** must be configured for them to actually
>   deliver. The OAuth providers (Google, X) don't need the Mailer.

## How it works

All authentication endpoints are served from your **app's own hosting domain**
under the `/_stormkit/auth` path. There is nothing to install in your
application: Stormkit intercepts these paths before your deployment is served.

The high-level flow is:

1. The user starts a sign-in flow (submits a form, clicks a magic link, or is
   redirected to an OAuth provider).
2. On success, Stormkit issues a **session token** (a JWT) and stores it in the
   browser under the `skauth` key in `localStorage`.
3. The browser is redirected to your **Success callback URL**. The
   email-verification and same-origin magic-link flows append `?verified=true`;
   the OAuth and cross-origin flows redirect to the plain Success URL.
4. Your frontend reads the `skauth` token and sends it as a
   `Authorization: Bearer <token>` header on subsequent requests.
5. For every authenticated request, Stormkit validates the token and forwards
   `X-User-Id` and `X-User-Email` headers to your backend / API. These headers
   are always stripped from incoming requests first, so they cannot be spoofed.

## Configuration

The Authentication tab has two parts: a set of **global settings** and the list
of **providers**.

### Global settings

| Setting | Description |
| --- | --- |
| **Success callback URL** | A relative URL (e.g. `/auth/success`) the browser is redirected to after a successful sign-in. At this URL your browser code can read the `skauth` item from `localStorage`. |
| **Session TTL** | How long a session token stays valid. Accepts durations like `30min`, `12h`, `7d`, `1w`, `1mo`, `1y`. |
| **Allowed origins** | Optional. One origin per line (scheme + host, no path). Required only when your frontend runs on a **different domain** than the auth host (a decoupled frontend or a native app). Leave it empty for single-host setups. |

Remember to click **Save** after changing these settings.

### Providers

Open a provider from the list to configure and enable it. Disabling a provider
does **not** delete its existing users.

#### Email and password

No credentials to configure — just enable it. Users register and sign in
programmatically:

- `POST /_stormkit/auth/register` with a JSON body `{ "email": "...", "password": "..." }`. Returns a session JWT on success.
- `POST /_stormkit/auth/login` with the same body to sign an existing user in.
- `GET /_stormkit/auth/verify` is used to confirm email addresses. The
  verification email needs the [Mailer](/docs/features/mailer) configured; as
  with Magic Link, unsent emails are still viewable under **Mailer** > **Sent
  Emails**.

#### Magic Link

Passwordless sign-in via a one-time link sent by email.

- Configure the **From address** used as the `From` header for magic-link emails
  (e.g. `Acme <noreply@acme.com>`).
- Request a link: `GET /_stormkit/auth/magic?email=user@example.com`. The
  endpoint returns `201` with no content and emails the user a link.
- The emailed link points to `/_stormkit/auth/magic?token=<token>`, which
  exchanges the token for a session JWT stored in `localStorage` and redirects
  to your Success callback URL.

> Magic Link requires the [Mailer](/docs/features/mailer) to be configured to
> deliver links. If the Mailer is **not** configured, the email is still
> recorded and visible under **Environment** > **Mailer** > **Sent Emails** —
> but it is not actually delivered, so this is only useful for local testing.

#### Google / X (Twitter) OAuth

OAuth 2.0 providers require a **Client ID** and **Client Secret** from the
provider's developer console.

When you open an OAuth provider, the drawer shows the exact **Callback URL** to
register with the provider. Because an app can be served from several domains,
pick the domain from the dropdown and register the matching callback URL —
`https://<your-domain>/_stormkit/auth/callback` — for **every** domain you sign
in from.

- **Google:** create an *OAuth 2.0 Client ID* (Web application) in the
  [Google Cloud Console](https://console.developers.google.com/apis/credentials)
  and set the Authorized Redirect URI to the Callback URL shown in the drawer.
- **X / Twitter:** create a project in the
  [X Developer Portal](https://developer.x.com/en/portal/dashboard), enable
  *Request email from users*, choose *Web App, Automated App or Bot*, and set the
  Redirect URL to the Callback URL shown in the drawer.

Start the OAuth flow by redirecting the user to the **Authorization URL** on your
own domain, e.g. `https://app.example.com/_stormkit/auth/google`. Optionally
append `?redirect=<origin>` to control where the user returns afterwards;
otherwise the request's `Origin` / `Referer` is used.

## Consuming the session in your app

After a successful sign-in, read the token and attach it to your API calls:

```js
// After the redirect to your Success callback URL
const token = JSON.parse(localStorage.getItem("skauth"));

const res = await fetch("/api/me", {
  headers: { Authorization: `Bearer ${token}` },
});
```

On the server side, your deployment receives the validated identity as request
headers:

- `X-User-Id` — the user's external identifier.
- `X-User-Email` — the user's email address.

### Refreshing the token

To keep an active user signed in without forcing a new sign-in, exchange a still
valid token for a fresh one:

- `POST /_stormkit/auth/refresh` with the current `Authorization: Bearer <token>`
  header. Returns a new token with the same identity.

Call this proactively (for example on app open and on a periodic timer) so the
token never reaches its TTL while the user is active. **Expired** tokens are
rejected — the user must sign in again.

## Cross-origin and native apps

When your frontend is served from a **different origin** than the auth host (a
separately deployed SPA, or a native/mobile app), add that frontend's origin to
**Allowed origins**. Stormkit then completes sign-in by bouncing the browser
through a short-lived one-time code, landing the user back on your frontend with
the token injected into `localStorage` there — even though that frontend has no
auth configuration of its own.

> If a provider is enabled but **no allowed origins** are set and your frontend
> runs on a separate domain, sign-in will return users to the auth host instead
> of your app. The Authentication tab warns you about this.

## Handling errors

When a sign-in fails (an invalid, expired, or already-used token, for example),
the user is redirected back to your **Success callback URL** with a
`login_error` query parameter holding a human-readable message, instead of a
Stormkit error page. Read it on your callback page to show a friendly message:

```js
const params = new URLSearchParams(window.location.search);
const error = params.get("login_error");
```

## Managing users

Sign-ins from different providers that share the same email address are linked to
a **single user**, so a person who first signed in with Google and later via a
magic link remains one account.

You can review your application's users under **Environment** > **Auth Users**,
or list them programmatically with the public API:

- `GET /v1/auth/users?envId=<id>&from=<offset>` — requires a **user-scoped API
  key**. Returns a page of users and a `hasNextPage` flag.

## Troubleshooting

- **The Authentication tab asks for a database**: attach a
  [database](/docs/features/database) to the environment first. Stormkit Auth
  stores users and sessions there.
- **Sign-in returns users to the wrong place**: add your frontend's origin to
  **Allowed origins**, and double-check the **Success callback URL** is the
  relative path you expect.
- **OAuth fails with a redirect URI mismatch**: the Callback URL registered with
  the provider must exactly match `https://<your-domain>/_stormkit/auth/callback`
  for the domain the user signs in on. Register it for every domain you use.
- **Users are signed out unexpectedly**: increase the **Session TTL**, or call
  `POST /_stormkit/auth/refresh` periodically so active sessions are renewed
  before they expire.
