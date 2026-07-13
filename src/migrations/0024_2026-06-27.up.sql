-- Speeds up the analytics rollup jobs, which filter raw rows by
-- request_timestamp/response_code and group by domain_id every few minutes.
-- Without this index those jobs sequentially scan the whole analytics table.
--
-- NOTE: built non-concurrently for migration-runner compatibility (the runner
-- executes each file in a single Exec). On very large existing tables, a DBA
-- may prefer to build it out-of-band with CREATE INDEX CONCURRENTLY and then
-- let this IF NOT EXISTS statement become a no-op.
CREATE INDEX IF NOT EXISTS idx_analytics_ts
    ON analytics (request_timestamp, response_code, domain_id);

-- request_id correlates a server-side analytics hit with the client-side events
-- emitted during the same page load (one request -> many client events). Reuses
-- the proxy's existing per-request uuid. Indexed for ad-hoc correlation lookups.
ALTER TABLE analytics ADD COLUMN IF NOT EXISTS request_id uuid;

CREATE INDEX IF NOT EXISTS idx_analytics_request_id
    ON analytics (request_id);

-- Custom events. Raw events are append-only (retained like the analytics table)
-- and rolled up hourly into analytics_events_agg, mirroring the visitor
-- aggregates. visitor_id is a cookieless daily hash; metadata holds optional
-- custom event properties; request_id links back to the originating server hit.
CREATE TABLE IF NOT EXISTS analytics_events (
    event_id     bigserial PRIMARY KEY,
    app_id       bigint NOT NULL REFERENCES apps(app_id) ON DELETE CASCADE,
    env_id       bigint NOT NULL REFERENCES apps_build_conf(env_id) ON DELETE CASCADE,
    domain_id    bigint,
    visitor_id   text,
    event_name   text NOT NULL,
    request_path text,
    event_ts     timestamp without time zone NOT NULL,
    request_id   uuid,
    metadata     jsonb
);

CREATE INDEX IF NOT EXISTS idx_analytics_events_ts
    ON analytics_events (event_ts, domain_id, event_name);

CREATE INDEX IF NOT EXISTS idx_analytics_events_request_id
    ON analytics_events (request_id);

-- Supports per-event property breakdown queries, which filter by domain+event
-- (equality) and range on time. Distinct from idx_analytics_events_ts, which
-- leads with event_ts for the rollup's "recent across everything" scan.
CREATE INDEX IF NOT EXISTS idx_analytics_events_breakdown
    ON analytics_events (domain_id, event_name, event_ts);

CREATE TABLE IF NOT EXISTS analytics_events_agg (
    aggregate_date date   NOT NULL,
    domain_id      bigint NOT NULL,
    event_name     text   NOT NULL,
    total_count    bigint NOT NULL,
    unique_actors  bigint NOT NULL,
    PRIMARY KEY (aggregate_date, domain_id, event_name)
);

-- source distinguishes server-recorded analytics hits ("server") from
-- client-side beacon pageviews ("client"), so the two can be told apart in the
-- dashboard. Nullable with no default to avoid rewriting the table; legacy rows
-- (NULL) are treated as server.
ALTER TABLE analytics ADD COLUMN IF NOT EXISTS source text;

-- snippet_interpolate opts a snippet into request-time interpolation of defined
-- system variables (e.g. {{SK_REQUEST_ID}}) at inject time. Off by default; only
-- flagged snippets are scanned, so unflagged user content is never rewritten.
ALTER TABLE snippets ADD COLUMN IF NOT EXISTS snippet_interpolate boolean NOT NULL DEFAULT false;

-- Raw, request-level access logs for every request hitting the hosting tier,
-- including bots. This is intentionally NOT the analytics table: it keeps the
-- raw client IP (unmasked), the full user-agent and path, and an is_bot flag
-- instead of dropping bot traffic.
--
-- Named request_logs (not access_logs): a defunct, code-unreferenced legacy
-- access_logs table from migration 0001 still exists and may hold historical
-- data, and migrations here are forward-only — so a fresh name avoids any
-- destructive drop while this feature owns the new partitioned table.
--
-- At ~200-500M rows/month the table is managed by native daily range
-- partitioning + DROP PARTITION retention (see MaintainAccessLogPartitions),
-- which makes 30-day purges instant and avoids the vacuum/bloat cost of the
-- batched ctid-DELETE used for the analytics table.
--
-- No foreign keys: cascading deletes over hundreds of millions of rows on app
-- deletion would be punishing, and the 30-day retention window bounds the data
-- regardless. The partition key (request_timestamp) is part of the primary key
-- because Postgres requires the partition column in every unique constraint.
--
-- NOTE: built non-concurrently for migration-runner compatibility (the runner
-- executes each file in a single Exec). Creating the parent, its DEFAULT
-- partition, a bootstrap window of daily partitions and the parent indexes in
-- one file is fine; ongoing partition lifecycle lives in the gocron job.
CREATE TABLE IF NOT EXISTS request_logs (
    log_id            bigint GENERATED ALWAYS AS IDENTITY,
    app_id            bigint NOT NULL,
    env_id            bigint NOT NULL,
    deployment_id     bigint,
    domain_id         bigint,
    host_name         text,
    request_timestamp timestamp without time zone NOT NULL,
    method            text,
    request_path      text,
    response_code     integer,
    client_ip         text,
    user_agent        text,
    referrer          text,
    is_bot            boolean NOT NULL DEFAULT false,
    bytes_sent        bigint,
    request_id        uuid,
    PRIMARY KEY (log_id, request_timestamp)
) PARTITION BY RANGE (request_timestamp);

-- Safety net so an insert never fails if the maintenance job lags behind (and
-- so tests with arbitrary timestamps insert cleanly). In steady state the
-- create-ahead job keeps this empty.
CREATE TABLE IF NOT EXISTS request_logs_default PARTITION OF request_logs DEFAULT;

-- Lean index set: every index is write amplification against ~16M inserts/day.
-- Defined on the parent so they propagate to existing and future partitions.
-- Both lead with the scoping column and include request_timestamp, which the
-- admin viewer always constrains (enabling partition pruning). Ad-hoc filters
-- (ip, path, user-agent) scan within the already-pruned day partitions rather
-- than carrying their own indexes.
CREATE INDEX IF NOT EXISTS idx_request_logs_app_ts
    ON request_logs (app_id, request_timestamp);

CREATE INDEX IF NOT EXISTS idx_request_logs_domain_ts
    ON request_logs (domain_id, request_timestamp);

-- Bootstrap daily partitions for a window around "now" so ingestion works
-- before the gocron maintenance job first runs.
DO $$
DECLARE
    d date;
BEGIN
    FOR d IN
        SELECT generate_series(current_date - 1, current_date + 7, interval '1 day')::date
    LOOP
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS request_logs_%s PARTITION OF request_logs FOR VALUES FROM (%L) TO (%L);',
            to_char(d, 'YYYYMMDD'), d, d + 1
        );
    END LOOP;
END $$;
