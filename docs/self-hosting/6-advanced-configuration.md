---
title: "Advanced Configuration"
description: Advanced configuration options for self-hosted Stormkit instances, including HTTP timeout tuning, load balancer setup, and queue configuration.
---

# Advanced Configuration

## Hosting Queue

The hosting queue is a Redis list used to buffer incoming analytics, logs, and usage metrics before they are written to the database. A background worker drains this queue every 5 seconds.

| Variable                            | Default | Description                                                                                                                             |
| ----------------------------------- | ------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `STORMKIT_HOSTING_QUEUE_BATCH_SIZE` | `1000`  | Number of items consumed from the hosting queue per worker run. Increase this value if the queue grows faster than it is being drained. |

## Reverse Proxy / Load Balancer

By default Stormkit assumes it is the public edge: `X-Forwarded-For` is always overwritten with the real socket address, and `X-Real-IP` is overwritten if the client supplied one.

Choose the right mode based on the type of load balancer in front of Stormkit:

| Setup                                                    | Variable to enable                  |
| -------------------------------------------------------- | ----------------------------------- |
| No load balancer (Stormkit is the public edge)           | neither (default)                   |
| L7 HTTP load balancer (e.g. nginx, AWS ALB, Cloudflare)  | `STORMKIT_TRUST_PROXY_HEADERS=true` |
| L4 TCP load balancer (e.g. AWS NLB, HAProxy in TCP mode) | `STORMKIT_PROXY_PROTOCOL=true`      |

### L7 HTTP load balancer

If Stormkit sits behind a trusted HTTP reverse proxy that sets `X-Forwarded-For` / `X-Real-IP` correctly, enable the following variable so that those headers are passed through unchanged:

| Variable                       | Default | Description                                                                                                                                                                                                                          |
| ------------------------------ | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `STORMKIT_TRUST_PROXY_HEADERS` | `false` | Set to `true` when Stormkit runs behind a trusted upstream HTTP proxy. `X-Forwarded-For` and `X-Real-IP` are passed through unchanged. When `false` (default), `X-Forwarded-For` is always overwritten with the real socket address. |

### L4 TCP load balancer (PROXY protocol)

TCP load balancers forward connections at the network level and cannot inject HTTP headers, so `X-Forwarded-For` will never be set. Instead, configure the load balancer to emit a [PROXY protocol](https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt) header (v1 or v2) at the start of each TCP connection, and enable the corresponding Stormkit flag:

| Variable                  | Default | Description                                                                                                                                                        |
| ------------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `STORMKIT_PROXY_PROTOCOL` | `false` | Set to `true` when Stormkit runs behind a TCP load balancer that emits PROXY protocol headers. The real client IP is read from the TCP layer before TLS handshake. |

> **Security note:** PROXY protocol headers are only trustworthy when Stormkit can be reached **only** through the trusted L4 load balancer. If the Stormkit port is directly reachable, a client can send a forged PROXY header and spoof the source IP. When `STORMKIT_PROXY_PROTOCOL=true`, restrict inbound traffic to the Stormkit origin so that only the load balancer can connect (for example with security groups, firewall rules, or equivalent network controls).

> **Security note:** It is not advised to enable `STORMKIT_PROXY_PROTOCOL` and `STORMKIT_TRUST_PROXY_HEADERS` at the same time as it will allow a client to spoof its source IP by injecting arbitrary `X-Forwarded-For` headers alongside a PROXY protocol header.

## HTTP Timeouts

The following environment variables control the HTTP server timeouts. Values are parsed as Go duration strings; you should include a unit suffix (e.g. `30s`, `1m`, `500ms`). Bare integers without a unit (e.g. `30`) are interpreted as nanoseconds (e.g. `30` → `30ns`), which results in an extremely short timeout and is almost never desired. When unset, the defaults shown below are used.

| Variable                            | Default | Description                                                                                                                                                                                                                  |
| ----------------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `STORMKIT_HTTP_READ_TIMEOUT`        | `30s`   | Maximum time to read an entire request, including the body. Used only for the API server; the hosting server uses `STORMKIT_HTTP_CLIENT_BODY_TIMEOUT` instead.                                                               |
| `STORMKIT_HTTP_IDLE_TIMEOUT`        | `60s`   | Maximum time to wait for the next request on a keep-alive connection.                                                                                                                                                        |
| `STORMKIT_HTTP_CLIENT_BODY_TIMEOUT` | `60s`   | Maximum idle time between successive reads of an incoming request body. Equivalent to nginx's `client_body_timeout`. If no bytes arrive within this window, body reads time out. Set to `0` to disable.                      |
| `STORMKIT_HTTP_PROXY_TIMEOUT`       | `30s`   | Maximum time the upstream server has to start sending response headers after the proxy finishes sending the request. Does not cap total upload duration or idle time while reading the response body. Set to `0` to disable. |
