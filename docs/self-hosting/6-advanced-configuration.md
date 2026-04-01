---
title: "Advanced Configuration"
description: Advanced configuration options for self-hosted Stormkit instances, including HTTP timeout tuning and queue configuration.
---

# Advanced Configuration

## Hosting Queue

The hosting queue is a Redis list used to buffer incoming analytics, logs, and usage metrics before they are written to the database. A background worker drains this queue every 5 seconds.

| Variable | Default | Description |
| --- | --- | --- |
| `STORMKIT_HOSTING_QUEUE_BATCH_SIZE` | `1000` | Number of items consumed from the hosting queue per worker run. Increase this value if the queue grows faster than it is being drained. |

## HTTP Timeouts

The following environment variables control the HTTP server timeouts. Values are parsed as Go duration strings; you should include a unit suffix (e.g. `30s`, `1m`, `500ms`). Bare integers without a unit (e.g. `30`) are interpreted as nanoseconds (e.g. `30` → `30ns`), which results in an extremely short timeout and is almost never desired. When unset, the defaults shown below are used.

| Variable | Default | Description |
| --- | --- | --- |
| `STORMKIT_HTTP_READ_TIMEOUT` | `30s` | Maximum time to read an entire request, including the body. |
| `STORMKIT_HTTP_WRITE_TIMEOUT` | `30s` | Maximum time to write a response. |
| `STORMKIT_HTTP_IDLE_TIMEOUT` | `60s` | Maximum time to wait for the next request on a keep-alive connection. |
