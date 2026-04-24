---
title: "Advanced Configuration"
description: Advanced configuration options for self-hosted Stormkit instances, including HTTP timeout tuning and queue configuration.
---

# Advanced Configuration

## Hosting Queue

The hosting queue is a Redis list used to buffer incoming analytics, logs, and usage metrics before they are written to the database. A background worker drains this queue every 5 seconds.

| Variable                            | Default | Description                                                                                                                             |
| ----------------------------------- | ------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `STORMKIT_HOSTING_QUEUE_BATCH_SIZE` | `1000`  | Number of items consumed from the hosting queue per worker run. Increase this value if the queue grows faster than it is being drained. |

## Reverse Proxy / Load Balancer

By default Stormkit assumes it is the public edge and rewrites `X-Forwarded-For` and `X-Real-IP` with the real socket address on every proxied request, preventing clients from spoofing those headers.

If Stormkit sits behind a trusted reverse proxy or load balancer that already sets `X-Forwarded-For` correctly, enable the following variable so that the upstream chain is preserved and the real socket address is appended rather than replaced:

| Variable                          | Default | Description                                                                                                                                                      |
| --------------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `STORMKIT_TRUST_PROXY_HEADERS`    | `false` | Set to `true` when Stormkit runs behind a trusted upstream proxy. The existing `X-Forwarded-For` header is preserved and the connecting address is appended to it. When `false` (default), `X-Forwarded-For` and `X-Real-IP` are always overwritten with the real socket address. |

## HTTP Timeouts

The following environment variables control the HTTP server timeouts. Values are parsed as Go duration strings; you should include a unit suffix (e.g. `30s`, `1m`, `500ms`). Bare integers without a unit (e.g. `30`) are interpreted as nanoseconds (e.g. `30` → `30ns`), which results in an extremely short timeout and is almost never desired. When unset, the defaults shown below are used.

| Variable                             | Default | Description                                                                                                                                                                                                                    |
| ------------------------------------ | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `STORMKIT_HTTP_READ_TIMEOUT`         | `30s`   | Maximum time to read an entire request, including the body. Used only for the API server; the hosting server uses `STORMKIT_HTTP_CLIENT_BODY_TIMEOUT` instead.                                                                 |
| `STORMKIT_HTTP_IDLE_TIMEOUT`         | `60s`   | Maximum time to wait for the next request on a keep-alive connection.                                                                                                                                                          |
| `STORMKIT_HTTP_CLIENT_BODY_TIMEOUT`  | `60s`   | Maximum idle time between successive reads of an incoming request body. Equivalent to nginx's `client_body_timeout`. If no bytes arrive within this window, body reads time out. Set to `0` to disable.                       |
| `STORMKIT_HTTP_PROXY_TIMEOUT`        | `30s`   | Maximum time the upstream server has to start sending response headers after the proxy finishes sending the request. Does not cap total upload duration or idle time while reading the response body. Set to `0` to disable.  |
