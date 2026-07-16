---
title: Expose your app to MCP clients with the OAuth 2.1 server
description: Turn the application you host on Stormkit into an OAuth 2.1 authorization server so MCP connectors like Claude and ChatGPT can connect to it with your end users' identities.
---

# OAuth 2.1 Server

## Overview

The **OAuth 2.1 server** turns the application you host on Stormkit into an
OAuth *authorization server*. External clients — most commonly **MCP
connectors** such as Claude (`claude.ai` / `claude.com`) and ChatGPT
(`chatgpt.com`), or native/CLI clients like Claude Code — register with your app
and obtain tokens that let them call your app's MCP endpoint **as your end
users**.

This is the opposite role from the Google / X provider logins described in
[Authentication](/docs/features/authentication). There, your app is a *client*
of an external identity provider. Here, your app *is* the authorization server
that other clients connect to.

The OAuth server is layered on top of **Stormkit Auth (SkAuth)**: it reuses
SkAuth's end-user identities and its signing secret. SkAuth must be enabled for
the OAuth server to work.

> **Requirements**
>
> - Available on **self-hosted** installations only.
> - **[Stormkit Auth](/docs/features/authentication)** must be enabled
>   (which in turn requires a **[database](/docs/features/database)** attached to
>   the environment). Identities and the token-signing secret come from SkAuth.

## How it works

All endpoints are served from your **app's own hosting domain** under the
`/_stormkit/oauth` path prefix. Stormkit intercepts these paths before your
deployment is served — there is nothing to install in your application.

### Endpoints

| Endpoint | Description |
| --- | --- |
| `POST /_stormkit/oauth/register` | [RFC 7591](https://www.rfc-editor.org/rfc/rfc7591) Dynamic Client Registration. Clients self-register as **public** clients; PKCE is required. |
| `GET /_stormkit/oauth/authorize` | Authorization request ([RFC 6749](https://www.rfc-editor.org/rfc/rfc6749) + PKCE [RFC 7636](https://www.rfc-editor.org/rfc/rfc7636)). If the user has no readable session, the server delegates to your app's configured **Login URL** (with a loop-guarded `return_to` param) rather than showing a Stormkit login page. |
| `POST /_stormkit/oauth/authorize` | Consent / grant submission. Issues the authorization code. |
| `POST /_stormkit/oauth/token` | Exchanges the authorization code + PKCE `code_verifier` for an access token, and — when the `offline_access` scope was requested — a rotating refresh token. |
| `POST /_stormkit/oauth/revoke` | [RFC 7009](https://www.rfc-editor.org/rfc/rfc7009) token revocation. Deletes a refresh token. Always returns `200`, even for unknown tokens (per RFC 7009 §2.2, to prevent probing). |

### Tokens

| Token | How it works |
| --- | --- |
| **Access token** | A stateless **HS256 JWT** signed with the environment's SkAuth secret. The edge validates it on every request. It carries an audience (`aud`) bound to the resource ([RFC 8707](https://www.rfc-editor.org/rfc/rfc8707)) — your app's origin plus the MCP resource path. Access tokens are **not individually revocable**; they expire on their own. |
| **Refresh token** | Stored server-side (Redis, SHA-256 keyed), **rotates on each use**, and is only issued when the `offline_access` scope is requested. Refresh tokens can be revoked via `/revoke`. |

### Discovery

The server publishes standard `/.well-known/` metadata so clients can discover
it automatically:

- **OAuth Authorization Server** metadata.
- **Protected Resource** metadata ([RFC 9728](https://www.rfc-editor.org/rfc/rfc9728)),
  including a per-path variant at
  `/.well-known/oauth-protected-resource/<path>` matching your MCP resource
  path.

The `/revoke` endpoint is advertised as `revocation_endpoint` in the metadata.
When an unauthenticated request hits the MCP endpoint, the auth challenge points
the client at these documents.

### Redirect URI validation

Redirect validation for the OAuth server does **not** use SkAuth's
[Allowed origins](/docs/features/authentication) list. Instead:

- A curated set of **connector presets** is trusted automatically when the OAuth
  server is enabled: `claude.ai`, `claude.com`, and `chatgpt.com`. No per-origin
  configuration is needed for these clients.
- **Native / CLI clients** that use [RFC 8252](https://www.rfc-editor.org/rfc/rfc8252)
  loopback redirects (`localhost` on an ephemeral port) are supported only when
  the **Allow loopback** toggle is on. Loopback redirects are matched on
  **scheme + path**, ignoring the port.

## Configuration

Configure the OAuth server under **Environment** > **Authentication**.

1. **Enable [Stormkit Auth](/docs/features/authentication) first.** This requires
   a database attached to the environment.
2. **Turn on the OAuth server.**
3. **Set the MCP resource path** (e.g. `/mcp`). This must exactly equal the path
   in the connector URL your users enter — a mismatch makes connectors fail
   silently.
4. Optionally enable **Allow loopback** to support native / CLI clients.
5. **Set a Login URL** — your app's own login page. `/authorize` delegates
   unauthenticated users here.

Remember to **Save** after changing these settings.

### Configuring via the API

The same settings can be set programmatically with `POST /v1/auth/config` using
a user- or environment-scoped API key, or via the `configure_auth` MCP tool. The
relevant fields are:

| Field | Meaning |
| --- | --- |
| `oauthServerEnabled` | Enable the OAuth server. |
| `oauthResourcePath` | The MCP resource path (e.g. `/mcp`). |
| `oauthAllowLoopback` | Allow RFC 8252 loopback redirects. |
| `loginUrl` | Your app's login page for `/authorize` delegation. |

See the [Authentication](/docs/features/authentication) docs for the full field
list.

## How a connector connects

1. The user enters your app's **MCP URL** (e.g.
   `https://app.example.com/mcp`) into Claude or ChatGPT.
2. The client discovers your OAuth server via the `/.well-known/` metadata.
3. The client **self-registers** at `/_stormkit/oauth/register`.
4. The user is sent to `/_stormkit/oauth/authorize`.
5. If they are **not signed in**, the server delegates them to your app's
   **Login URL**; after login they return to `/authorize`.
6. The user **consents**, and the server issues an authorization code.
7. The client exchanges the code + PKCE `code_verifier` at
   `/_stormkit/oauth/token` for an access token (and a refresh token if
   `offline_access` was requested).
8. The client calls your **MCP endpoint** with the access token as a bearer
   token; the edge validates it on every request.

## Troubleshooting

- **The connector fails silently**: the **MCP resource path** must exactly match
  the path in the URL the user entered (e.g. `/mcp`). A mismatch is the most
  common cause.
- **Nothing works even though the path is right**: make sure
  [Stormkit Auth](/docs/features/authentication) is enabled — the OAuth server
  depends on it for identities and the signing secret.
- **A native / CLI client can't complete the flow**: enable the **Allow
  loopback** toggle. Loopback redirects are only trusted when it is on.
- **Users get stuck in a login loop**: set the **Login URL** to your app's login
  page so `/authorize` can delegate unauthenticated users to it.
