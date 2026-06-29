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

CREATE TABLE IF NOT EXISTS analytics_events_agg (
    aggregate_date date   NOT NULL,
    domain_id      bigint NOT NULL,
    event_name     text   NOT NULL,
    total_count    bigint NOT NULL,
    unique_actors  bigint NOT NULL,
    PRIMARY KEY (aggregate_date, domain_id, event_name)
);
