---
title: Add end-user authentication with Stormkit Auth
description: Configure Stormkit Auth to add sign-in to your application — email and password, magic links, and Google or X (Twitter) OAuth — with cookie sessions for browsers and bearer tokens for native apps.
---

# Authentication

## Overview

**Stormkit Auth** adds end-user authentication to the application you host on
Stormkit. Your visitors can register and sign in with **email and password**, a
**magic link**, or an OAuth provider (**Google**, **X / Twitter**), and your
app receives a signed session it can use to identify the user on every request.

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
2. On success, Stormkit issues a **session token** (a JWT) and delivers it as a
   **cookie** to browsers, or directly to native apps (see
   [Sessions](#sessions) below).
3. The browser is redirected to your **Success callback URL**. The
   email-verification and same-origin magic-link flows append `?verified=true`;
   the OAuth and cross-origin flows redirect to the plain Success URL. A native
   app is instead redirected to its deep link carrying the token.
4. For every authenticated request, Stormkit validates the session and forwards
   `X-User-Id` and `X-User-Email` headers to your backend / API. These headers
   are always stripped from incoming requests first, so they cannot be spoofed.

## Sessions

Stormkit delivers the session in one of two ways, chosen automatically by the
kind of client:

- **Browsers get a cookie.** The session is stored in a cookie named
  `skauth_session` that is `HttpOnly`, `Secure`, and `SameSite=Lax`. Because it
  is `HttpOnly`, your JavaScript cannot read it — and does not need to. The
  browser attaches it automatically to same-origin requests, and you send it on
  cross-origin requests with `credentials: "include"`.
- **Native / mobile apps get a bearer token.** A client with no cookie jar
  receives the session token directly. Which mechanism applies depends on how the
  user signs in — see [Native apps](#native-apps) below. Either way the app ends
  up with a token it stores and sends back as an `Authorization: Bearer <token>`
  header.

> **Migrating from an older setup?** Earlier versions stashed the token in
> `localStorage` under a `skauth` key. That mode has been removed: browsers now
> always use the `HttpOnly` cookie. Update your frontend to call the API with
> `credentials: "include"` and read the user from `/_stormkit/auth/me` instead of
> reading a token from `localStorage`.

### Native apps

A native app has no usable cookie jar — the system sign-in browser
(`ASWebAuthenticationSession` on iOS, Custom Tabs on Android) discards
`Set-Cookie` when it closes. Stormkit therefore hands the token over in one of
two ways:

**1. JSON endpoints — the `X-Session-Delivery` header.** On
`/_stormkit/auth/login`, `/_stormkit/auth/register` and `/_stormkit/auth/refresh`,
send the header `X-Session-Delivery: bearer` and the token is returned in the
response body instead of a cookie:

```json
{ "token": "…", "email": "user@example.com", "userId": "…" }
```

At `/_stormkit/auth/refresh` this is auto-detected — a client that authenticated
with a bearer token and sent no cookie gets a new bearer token without the
header.

**2. Browser sign-in flows — a custom-scheme redirect plus PKCE.** Google / X
OAuth and Magic Link finish inside a browser, so there is no response body to
read. Instead, register your app's deep link as an **allowed origin** (scheme +
host, no path — e.g. `myapp://auth`) and pass it as the `redirect` target.

A private-use URI scheme is not exclusively yours — another app on the device can
register the same scheme — so the redirect must be assumed interceptable and
never carries the session token. It carries a **single-use code**, valid for one
minute, bound to a PKCE challenge only your app can answer:

```
302 Location: myapp://auth?code=<one-time-code>
```

Generate a random `code_verifier`, send `code_challenge = base64url(sha256(verifier))`
(unpadded, `code_challenge_method=S256`) when you start sign-in, and exchange the
code for the session token over TLS:

```
POST /_stormkit/auth/token
Content-Type: application/json

{ "code": "<one-time-code>", "code_verifier": "<verifier>" }
```

```json
{ "token": "…" }
```

The code is consumed on the first exchange attempt, so an interceptor without the
verifier gains nothing and cannot retry.

For OAuth, open the authorization URL with the redirect and the challenge:

```
https://app.example.com/_stormkit/auth/google?redirect=myapp://auth&code_challenge=<challenge>&code_challenge_method=S256
```

For Magic Link, pass the same values when requesting the link — they are
remembered and applied when the user taps it, so the binding survives the round
trip through the user's inbox:

```
GET /_stormkit/auth/magic?email=user@example.com&redirect=myapp://auth&code_challenge=<challenge>&code_challenge_method=S256
```

A custom-scheme redirect **without** a valid S256 challenge is rejected with
`400` — Stormkit will not fall back to putting the token in the URL.

Sign-in failures come back to the same deep link with a `login_error` parameter
(`myapp://auth?login_error=…`) rather than a `login_error` on your success page.
Expired links are included — the target is recovered from the link itself and
re-checked against the allow-list. A server-side fault (5xx) is the exception: it
renders an error page in the system browser, so keep a user-visible cancel path
in your sign-in sheet.

The code is only ever issued for an origin that is **on the allow-list**. An
unregistered scheme is rejected (Magic Link responds `403`; OAuth falls back to a
first-party cookie on your own domain), so nothing can leak to a scheme you did
not register. `http(s)` targets are unaffected and keep using the cookie.

> `X-Session-Delivery: bearer` applies **only** to `login`, `register` and
> `refresh`. The `magic` and OAuth `callback` landings are browser redirects and
> ignore the header — use the custom-scheme redirect for those. The `verify`
> (email-confirmation) landing has no native path at all: it always sets a
> first-party cookie, so a native app should call `login` with
> `X-Session-Delivery: bearer` once the address is confirmed.

### Cross-origin session protection

When Stormkit is about to set the session cookie (on `login`, `register`, and
`refresh`), it requires the request's `Origin` to be the auth host itself or a
value in **Allowed origins**. A missing or foreign `Origin` is rejected. This
blocks login CSRF / session fixation, where an attacker auto-submits a
cross-site form to plant their own session in the victim's browser.

## Configuration

The Authentication tab has two parts: a set of **global settings** and the list
of **providers**.

### Global settings

| Setting | Description |
| --- | --- |
| **Success callback URL** | A relative URL (e.g. `/auth/success`) the browser is redirected to after a successful sign-in. |
| **Session TTL** | How long a session token stays valid. Accepts durations like `30min`, `12h`, `7d`, `1w`, `1mo`, `1y`. |
| **Allowed origins** | Optional. One origin per line (scheme + host, no path). Required when your frontend runs on a **different domain** than the auth host (a decoupled frontend or a native app): those origins are permitted to initiate sign-in and set the session cookie. A native app's deep link goes here too — `myapp://auth` is valid, `myapp://` and `myapp://auth/callback` are not. Leave it empty for single-host setups. |
| **Cookie domain** | Optional. A parent domain (e.g. `.example.com`) to scope the session cookie to, so it is shared across subdomains — needed when your frontend and the auth host are different subdomains. Empty means a host-only cookie. |
| **Login URL** | Optional. Your app's own login page. Used by the [OAuth 2.1 server](/docs/features/oauth-server) to redirect unauthenticated users when an MCP connector asks them to authorize. |

Remember to click **Save** after changing these settings.

### Providers

Open a provider from the list to configure and enable it. Disabling a provider
does **not** delete its existing users.

#### Email and password

No credentials to configure — just enable it. Users register and sign in
programmatically:

- `POST /_stormkit/auth/register` with a JSON body `{ "email": "...", "password": "..." }`. Establishes a session on success.
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
  exchanges the token for a session and redirects to your Success callback URL.
- Optionally pass `redirect=<origin>` (query param or JSON body) on the request
  to control where the user lands after tapping the link. It must be on the
  **Allowed origins** list, and it is remembered with the link — so the redirect
  survives the round trip through the user's inbox. Native apps pass their deep
  link here, together with a PKCE `code_challenge`; see
  [Native apps](#native-apps).

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
otherwise the request's `Origin` / `Referer` is used. A cross-origin value must be
on the **Allowed origins** list. Native apps pass their deep link here — see
[Native apps](#native-apps).

> Sign-ins from providers that return an **unverified** email address are
> rejected before any account is created or linked, so a provider cannot be used
> to take over an account by asserting an address it hasn't verified.

## Consuming the session in your app

### In the browser

Because the session lives in an `HttpOnly` cookie, your frontend never handles a
token. Send your API calls with credentials and let the browser attach the
cookie:

```js
const res = await fetch("/api/me", {
  credentials: "include",
});
```

On the server side, your deployment receives the validated identity as request
headers:

- `X-User-Id` — the user's external identifier.
- `X-User-Email` — the user's email address.

These two headers are kept intentionally minimal so they stay cheap to forward on
every request. For the richer profile, use the user-info endpoint below.

### In a native app

Store the bearer token you received at sign-in — from the response body on
`login` / `register`, or from exchanging the `code` your deep link received after
an OAuth or Magic Link sign-in (see [Native apps](#native-apps)) — and attach it
to each request:

```
Authorization: Bearer <token>
```

### Fetching the full profile

`GET /_stormkit/auth/me` returns the full profile for the signed-in user. Send it
with the session cookie (browser) or the `Authorization` header (native):

```json
{
  "id": "…",
  "email": "user@example.com",
  "firstName": "John",
  "lastName": "Doe",
  "avatar": "https://…",
  "username": "johndoe",
  "profileUrl": "https://x.com/johndoe",
  "createdAt": 0,
  "lastLoginAt": 0
}
```

`username` and `profileUrl` are populated from the provider when available (for
example, the X handle and a link to the public profile); they are empty for
providers that don't supply them. The endpoint only ever returns the caller's own
record — identity is taken from the session, not from any parameter. Typically
your backend calls it once, on first sight of a new `X-User-Id`, to sync the
profile into its own store.

### Refreshing the session

To keep an active user signed in without forcing a new sign-in, exchange a still
valid session for a fresh one:

- `POST /_stormkit/auth/refresh`. In the browser send it with
  `credentials: "include"`; the response rotates the session cookie. A native
  client sends its current `Authorization: Bearer <token>` and receives a new
  token in the body.

Call this proactively (for example on app open and on a periodic timer) so the
session never reaches its TTL while the user is active. **Expired** sessions are
rejected — the user must sign in again.

### Signing out

- `POST /_stormkit/auth/logout` expires the session cookie. Because the cookie is
  `HttpOnly`, this endpoint is the only way to clear a browser session — your
  JavaScript cannot delete the cookie itself. It is `POST`-only, which prevents a
  cross-site `GET` from forcing a logout. Native apps simply discard their stored
  bearer token.

## Cross-origin and native apps

When your frontend is served from a **different origin** than the auth host (a
separately deployed SPA, or a native/mobile app):

- Add that frontend's origin to **Allowed origins**.
- For a browser SPA, set a shared **Cookie domain** (e.g. `.example.com`) so the
  `skauth_session` cookie is readable across your subdomains, and send every auth
  call with `credentials: "include"`.
- For a native app, add your deep link (e.g. `myapp://auth`) to **Allowed
  origins**, then send `X-Session-Delivery: bearer` on `login` / `register` /
  `refresh`, and pass `redirect=myapp://auth` plus a PKCE `code_challenge` for
  OAuth and Magic Link. See [Native apps](#native-apps).

> If a provider is enabled but **no allowed origins** are set and your frontend
> runs on a separate domain, sign-in will be rejected as cross-origin. The
> Authentication tab warns you about this.

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
magic link remains one account. Email addresses are normalized (lower-cased and
trimmed) so the same address in different casing never creates a duplicate.

You can review your application's users under **Environment** > **Auth Users**,
where you can also update or remove individual users.

## Configuring authentication via the API

The Authentication **settings** and the individual **sign-in providers** (not
the users) can be managed programmatically, which is handy for scripting an
environment or driving it from an AI agent. This mirrors the global settings in
the dashboard.

- `GET /v1/auth/config?envId=<id>` — returns the current configuration. Secrets
  (the signing secret and provider client secrets) are never included.
- `POST /v1/auth/config?envId=<id>` — updates the configuration. Only the fields
  you include are changed; omitted fields keep their stored value, so you can
  safely patch a single setting.

Both require a **user-** or **environment-scoped** [API key](/docs/api/authentication)
and are available on self-hosted installations only.

| Field | Type | Meaning |
| --- | --- | --- |
| `status` | boolean | Enable or disable Stormkit Auth for the environment. |
| `successUrl` | string | Relative Success callback URL (e.g. `/auth/success`). |
| `tokenTtl` | integer | Session lifetime, in minutes. |
| `allowedOrigins` | string[] | Allowed origins (scheme + host). Replaces the stored list. |
| `cookieDomain` | string | Parent domain for the session cookie (e.g. `.example.com`). |
| `loginUrl` | string | App login page for the OAuth 2.1 server. |
| `oauthServerEnabled` | boolean | Turn the [OAuth 2.1 server](/docs/features/oauth-server) on or off. |
| `oauthResourcePath` | string | MCP resource path (e.g. `/mcp`). |
| `oauthAllowLoopback` | boolean | Allow loopback redirects for native/CLI OAuth clients. |

Example:

```bash
curl -X POST \
     -H 'Authorization: Bearer <api_key>' \
     -H 'Content-Type: application/json' \
     -d '{"status": true, "successUrl": "/auth/success", "tokenTtl": 10080}' \
     'https://app.example.com/v1/auth/config?envId=<id>'
```

### Configuring providers via the API

The sign-in providers themselves are managed through a second pair of endpoints,
with the same key requirements and the same self-hosted-only availability.

- `GET /v1/auth/providers?envId=<id>` — returns the configured providers
  alongside the auth configuration. Client secrets are replaced with the
  `****-****-****-****` placeholder and never returned.
- `POST /v1/auth/providers` — enables or updates a single provider. `status` is
  optional: omitting it keeps the provider's current enabled flag, so rotating a
  secret cannot accidentally disable a live provider.

| Field | Type | Meaning |
| --- | --- | --- |
| `providerName` | string | `google`, `x`, `email` or `magiclink`. Separators are ignored, so `magic-link` also works. |
| `status` | boolean | Enable or disable the provider. Omitted keeps the stored value; a new provider defaults to enabled. |
| `clientId` | string | OAuth client id. Required for OAuth providers. |
| `clientSecret` | string | OAuth client secret. Write-only; omit it or send the placeholder to keep the stored one. |
| `fromAddress` | string | Sender address for the `email` and `magiclink` providers. Required for `magiclink`. |

Enabling a provider requires the environment to have a
[database schema](/docs/features/database) configured, and creates a default
auth configuration when the environment has none.

```bash
curl -X POST \
     -H 'Authorization: Bearer <api_key>' \
     -H 'Content-Type: application/json' \
     -d '{"envId": "<id>", "providerName": "magiclink", "fromAddress": "noreply@example.com"}' \
     'https://app.example.com/v1/auth/providers'
```

The same two operations are exposed to MCP clients as the `get_auth_config` and
`configure_auth` [MCP tools](/docs/features/oauth-server), so an agent can read
and adjust the configuration directly.

## Related

- [OAuth 2.1 server](/docs/features/oauth-server) — turn your app into an OAuth
  authorization server so MCP connectors (Claude, ChatGPT) can sign in your end
  users.

## Troubleshooting

- **The Authentication tab asks for a database**: attach a
  [database](/docs/features/database) to the environment first. Stormkit Auth
  stores users and sessions there.
- **Sign-in is rejected as cross-origin**: add your frontend's origin to
  **Allowed origins**, and make sure browser requests use
  `credentials: "include"`.
- **The browser isn't sending the session**: confirm your requests set
  `credentials: "include"`, that you're on HTTPS (the cookie is `Secure`), and —
  for a subdomain frontend — that **Cookie domain** is set to a shared parent.
- **OAuth fails with a redirect URI mismatch**: the Callback URL registered with
  the provider must exactly match `https://<your-domain>/_stormkit/auth/callback`
  for the domain the user signs in on. Register it for every domain you use.
- **Users are signed out unexpectedly**: increase the **Session TTL**, or call
  `POST /_stormkit/auth/refresh` periodically so active sessions are renewed
  before they expire.
