---
title: "Monitoring"
description: Run Prometheus and Grafana against your self-hosted Stormkit instance with ready-made dashboards for CPU, memory, disk, PostgreSQL and Redis.
---

# Monitoring

Stormkit ships an optional monitoring stack: Prometheus, Grafana, and the
standard exporters for the host, PostgreSQL and Redis. It answers the questions
an operator asks about the box:

- Is the machine running out of CPU, memory or disk?
- Is PostgreSQL near its connection limit, and is Stormkit waiting on its own
  connection pool?
- Is Redis close to `maxmemory`, and is it evicting keys?

This is **instance health**, not per-app analytics. Traffic, response times and
deployment history for individual applications already live in the Stormkit UI
under Access Logs and Analytics; these dashboards deliberately have no app
dimension.

Nothing here runs unless you ask for it. The configuration files sit on disk
after installation, and a Docker Compose profile decides whether the containers
start.

## Enabling it

Two commands from the directory holding your `docker-compose.yaml`:

```bash
sed -i 's/^PROMETHEUS_METRICS=.*/PROMETHEUS_METRICS=true/' .env
docker compose --profile monitoring up -d
```

The second command does double duty. It starts the monitoring containers, and
it recreates `hosting` and `workerserver` so they pick up the new environment
variable. There is no separate restart step.

`install.sh` generates a `GRAFANA_ADMIN_PASSWORD` for you. If you are enabling
this on an instance installed before monitoring existed, set one yourself — the
profile refuses to start without it rather than falling back to `admin/admin`:

```bash
echo "GRAFANA_ADMIN_PASSWORD='$(openssl rand -base64 18 | tr -dc 'a-zA-Z0-9' | head -c 24)'" >> .env
```

## Reaching Grafana

Grafana is published on loopback only, so it is not reachable from the internet.
Open an SSH tunnel:

```bash
ssh -L 3000:127.0.0.1:3000 you@your-server
```

Then visit `http://localhost:3000` and sign in as `admin` with the password from
your `.env`. The dashboards are already provisioned under the **Stormkit**
folder — there is nothing to import.

`GRAFANA_ADMIN_PASSWORD` is read only on Grafana's **first** start. Once the
admin user exists in the `grafana` volume, editing that variable and restarting
has no effect — the old password keeps working. This matters if you are rotating
after a suspected leak: change it in Grafana itself, or run

```bash
docker compose exec grafana grafana-cli admin reset-admin-password <new-password>
```

If you would rather expose Grafana properly, put your own TLS termination and
authentication in front of it. Do not simply change the published address to
`0.0.0.0`.

## The dashboards

**Stormkit — Host.** Disk free and used per filesystem, CPU by mode, memory,
load average against core count, and disk I/O. Container layer mounts (`tmpfs`,
`overlay`) are filtered out so the disk figures describe real disks. It also
charts resident memory and goroutine counts for the two Stormkit processes,
where steady monotonic growth is the signature of a leak.

**Stormkit — Dependencies.** PostgreSQL connections against `max_connections`,
database size, deadlocks, cache hit ratio and transaction rates; Redis memory
against `maxmemory`, evictions, hit ratio and client counts. It also has a row
for Stormkit's own connection pool, described below.

**Stormkit — Requests.** Requests per minute, status codes, request rate by
method, response time percentiles and an Apdex score, for the traffic served by
the `hosting` service. Like the other two it has no per-application dimension —
per-app traffic lives in the Stormkit UI under Analytics. Status codes are
recorded exactly for `200` and `304` and collapsed to a class (`4xx`, `5xx`)
otherwise, and every method other than `GET` and the body-carrying verbs shows
up as `OTHER`.

The first panel on the Host dashboard is **Scrape targets up**, expected to read
5. If it reads 3, the two `stormkit-*` targets are down and
`PROMETHEUS_METRICS` is almost certainly still `false`.

## Stormkit's own metrics

Almost everything on these dashboards comes from the standard exporters. The one
thing Stormkit exports itself is its database connection pool:

| Metric | Meaning |
| --- | --- |
| `stormkit_db_connections{state}` | Connections in the pool, `in_use` or `idle` |
| `stormkit_db_connections_max` | The pool's configured maximum |
| `stormkit_db_wait_total` | Times a caller had to wait for a free connection |
| `stormkit_db_wait_seconds_total` | Total time spent waiting |
| `stormkit_db_closed_total{reason}` | Connections closed, by the limit that closed them |

`stormkit_db_wait_total` is the one worth watching. `postgres_exporter` reports
what the server sees; it cannot tell you that Stormkit is queuing for a slot in
its own pool. A sustained wait rate means the pool is undersized, and it can
happen while PostgreSQL itself looks completely idle.

Stormkit also exports HTTP response time (`stormkit_lb_response_time_ms`, the
histogram behind the Requests dashboard) and
the usual Go runtime metrics.

## Using your own Prometheus

You do not need the bundled stack. Set `PROMETHEUS_METRICS=true` and scrape
`hosting:2112` and `workerserver:2112` from wherever your Prometheus lives.

`PROMETHEUS_PORT` changes the port. It is used exactly as given and never
silently moves, so your scrape configuration cannot end up pointing at nothing.
If the port is taken or the value is not a valid port number, metrics are
disabled and an error is logged — the service itself keeps serving traffic, so
check the logs if a target is unexpectedly down. Changing this in the bundled
stack also means updating the targets in `monitoring/prometheus.yml`, which
addresses `hosting:2112` and `workerserver:2112` directly.

The dashboard JSON under `monitoring/grafana/dashboards/` imports into any
Grafana, as long as your Prometheus datasource has the uid
`stormkit-prometheus` or you remap it during import.

## Turning it off

```bash
docker compose --profile monitoring down
```

Add `-v` to discard the retained metrics data as well. Setting
`PROMETHEUS_METRICS=false` and recreating the services stops Stormkit exposing
metrics at all.

## Security notes

- The metrics endpoints on port 2112 are unauthenticated. They are never
  published to the host, so they are not reachable from outside the machine.
- Prometheus is not published to the host either. Its API is unauthenticated and
  allows arbitrary queries. Note that "not published" means not reachable from
  *outside*: the Compose network is flat, so any container on it can reach
  Prometheus by name — including `hosting`, where deployed application code runs
  as child processes. The same is true of the exporters and of Redis, which has
  no password of its own on a default install.
- Grafana is published on `127.0.0.1` only, disables anonymous access and
  disables sign-up. Inside the Compose network it still listens on all
  interfaces, so the loopback binding protects it from the internet, not from
  the other containers.
- Grafana's plugin catalogue is disabled, because installing a plugin from the
  UI runs its code inside the Grafana container. Treat this as defence in depth
  rather than a cap on what an admin session is worth: an admin can still point
  a new datasource at any address on the Compose network and query it through
  Grafana's datasource proxy. Guard the admin password accordingly, especially
  if you put Grafana behind a public hostname rather than the SSH tunnel.
- `node_exporter` mounts the host filesystem read-only. This is safe because no
  deployed application code runs in that container. Do not replicate that mount
  into `hosting` or `workerserver`, where deployments execute as child
  processes.
- `postgres_exporter` currently reuses the Stormkit database credentials. If you
  want to tighten that, create a dedicated monitoring role with
  `pg_monitor` and point `DATA_SOURCE_NAME` at it.

## Installing the files on an older instance

Instances installed before monitoring existed will not have the configuration on
disk. From the directory holding your `docker-compose.yaml`:

```bash
BASE=https://raw.githubusercontent.com/stormkit-io/stormkit-io/main/deploy/monitoring

mkdir -p monitoring/grafana/provisioning/datasources \
         monitoring/grafana/provisioning/dashboards \
         monitoring/grafana/dashboards

curl -sfo monitoring/prometheus.yml "$BASE/prometheus.yml"
curl -sfo monitoring/grafana/provisioning/datasources/prometheus.yml "$BASE/grafana/provisioning/datasources/prometheus.yml"
curl -sfo monitoring/grafana/provisioning/dashboards/dashboards.yml "$BASE/grafana/provisioning/dashboards/dashboards.yml"
curl -sfo monitoring/grafana/dashboards/stormkit-host.json "$BASE/grafana/dashboards/stormkit-host.json"
curl -sfo monitoring/grafana/dashboards/stormkit-dependencies.json "$BASE/grafana/dashboards/stormkit-dependencies.json"
curl -sfo monitoring/grafana/dashboards/stormkit-requests.json "$BASE/grafana/dashboards/stormkit-requests.json"
```

You will also need the monitoring services in your `docker-compose.yaml`. The
simplest route is to re-download it, since the file is not meant to be edited by
hand:

```bash
curl -so docker-compose.yaml https://raw.githubusercontent.com/stormkit-io/stormkit-io/main/deploy/docker-compose.yaml
```
