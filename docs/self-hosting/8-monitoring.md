---
title: "Monitoring"
description: See CPU, memory and disk usage for the machines running your self-hosted Stormkit instance, and optionally connect your own Prometheus.
---

# Monitoring

The **Health** tab in the Admin Interface shows CPU, memory, disk and load for
every machine running your instance, along with PostgreSQL and Redis health.
Stormkit keeps 24 hours of history.

## Why an exporter is needed

Stormkit runs inside a container, where it can only see the filesystems mounted
into that container — not every disk on the machine. Giving it full visibility
would mean mounting the host filesystem into the `hosting` service, which also
runs deployed applications as child processes. That would expose the host to
every application anyone deploys.

Instead, Stormkit reads machine stats from
[node_exporter](https://github.com/prometheus/node_exporter), which holds the
host mounts in a container of its own where no user code runs.

## Setup

Run this on each machine you want to monitor:

```bash
docker run -d --name node-exporter --pid host --net host \
  --restart unless-stopped -v /:/host:ro,rslave \
  prom/node-exporter --path.rootfs=/host
```

Machines running Stormkit are discovered automatically — each instance
registers itself, so a new machine appears within a few seconds with no
configuration.

In a deployment spanning several machines, set `STORMKIT_ADVERTISE_HOST` on each
one to an address the others can reach it on. Inside a single compose setup the
container hostname resolves on the shared network, so this can be left unset.

### Machines without Stormkit

A machine that runs an exporter but no Stormkit process — a dedicated database
host, for example — will not register itself. Add it under **Manual targets** at
the bottom of the Health page, one `host:port` per line.

## Security

`node_exporter` has no authentication. Keep port `9100` on a private interface
or firewall it; anyone who can reach it can read your machine's resource usage
and topology.

## Bringing your own Prometheus

The Health page is self-contained and needs nothing else. If you already run
Prometheus and want Stormkit's own application metrics — request latency,
deployment storage, Go runtime — enable them:

```bash
PROMETHEUS_METRICS=true
PROMETHEUS_PORT=2112       # optional, this is the default
```

Then scrape `<host>:2112/metrics`. Stormkit exports:

| Metric                                     | Description                                        |
| ------------------------------------------ | -------------------------------------------------- |
| `stormkit_lb_response_time_ms`              | Request latency histogram by method and status      |
| `stormkit_storage_deployments_free_bytes`   | Free space on the filesystem holding deployments    |
| `stormkit_storage_deployments_total_bytes`  | Total size of that filesystem                       |
| `go_*`, `process_*`                         | Go runtime and process metrics                      |

The deployment storage gauges are worth pairing with node_exporter's
`node_filesystem_*`: node_exporter reports every filesystem, but only Stormkit
knows which one its deployment artifacts land on.

This endpoint is also unauthenticated and is not published to the host in the
compose file. Keep it on a private interface.
