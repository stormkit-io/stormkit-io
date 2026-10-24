-- Drop the copy of auth_wall taken before Auth Wall logins moved to teams
-- (0030). The migration shipped in v2026.10.22.1 and no rollback is needed.
DROP TABLE IF EXISTS auth_wall_backup;

-- Campaign attribution read from the landing URL before its query is stripped
-- from request_path. Nullable with no default, so the ALTERs are metadata-only.
ALTER TABLE analytics ADD COLUMN IF NOT EXISTS utm_source text;
ALTER TABLE analytics ADD COLUMN IF NOT EXISTS utm_medium text;
ALTER TABLE analytics ADD COLUMN IF NOT EXISTS utm_campaign text;
ALTER TABLE analytics ADD COLUMN IF NOT EXISTS ref text;

-- Daily page views and unique visitors per path. Visitor ids rotate daily, so
-- unique_visitors is exact per day and a multi-day total is a sum of daily
-- uniques. server_views and client_views split page_views by source.
CREATE TABLE IF NOT EXISTS analytics_paths_agg (
    aggregate_date date NOT NULL,
    domain_id bigint NOT NULL,
    request_path text NOT NULL,
    request_path_hash bytea NOT NULL,
    unique_visitors bigint NOT NULL,
    page_views bigint NOT NULL,
    server_views bigint NOT NULL,
    client_views bigint NOT NULL,
    -- Foreground time measured by the client script: the sum of seconds and
    -- the number of page views that reported it.
    engaged_seconds bigint NOT NULL DEFAULT 0,
    engaged_views bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (domain_id, aggregate_date, request_path_hash)
);

-- Daily unique visitors per path and country.
CREATE TABLE IF NOT EXISTS analytics_path_countries_agg (
    aggregate_date date NOT NULL,
    domain_id bigint NOT NULL,
    request_path text NOT NULL,
    request_path_hash bytea NOT NULL,
    country_iso_code text NOT NULL,
    unique_visitors bigint NOT NULL,
    PRIMARY KEY (domain_id, aggregate_date, request_path_hash, country_iso_code)
);

-- Daily visits per campaign and landing path. Only rows carrying at least one
-- UTM parameter or ref are aggregated; empty strings stand for absent values.
CREATE TABLE IF NOT EXISTS analytics_campaigns_agg (
    aggregate_date date NOT NULL,
    domain_id bigint NOT NULL,
    utm_source text NOT NULL,
    utm_medium text NOT NULL,
    utm_campaign text NOT NULL,
    ref text NOT NULL,
    campaign_hash bytea NOT NULL,
    request_path text NOT NULL,
    request_path_hash bytea NOT NULL,
    unique_visitors bigint NOT NULL,
    visits bigint NOT NULL,
    PRIMARY KEY (domain_id, aggregate_date, campaign_hash, request_path_hash)
);

-- Daily visits per path and referrer host, with the referrer's channel
-- (search, ai, social, email, referral, direct, internal) assigned by the
-- rollup job. Reports group by the stored channel instead of classifying
-- referrers on every read, and the key leads with the domain so a report
-- reads one site's rows only.
CREATE TABLE IF NOT EXISTS analytics_channels_agg (
    aggregate_date date NOT NULL,
    domain_id bigint NOT NULL,
    request_path text NOT NULL,
    request_path_hash bytea NOT NULL,
    referrer text NOT NULL,
    referrer_hash bytea NOT NULL,
    channel text NOT NULL,
    visits bigint NOT NULL,
    PRIMARY KEY (domain_id, aggregate_date, request_path_hash, referrer_hash)
);

-- The channel rollup looks up the domains of each page's environment to tell
-- internal referrers apart, and the per-environment domain listing filters on
-- it too; without an index both scan the whole table.
CREATE INDEX IF NOT EXISTS idx_domains_env_id ON domains (env_id);

-- The path rollups delete the days they recompute. The aggregate tables are
-- keyed by domain first, so without these indexes each delete would scan the
-- whole table.
CREATE INDEX IF NOT EXISTS idx_analytics_paths_agg_date ON analytics_paths_agg (aggregate_date);
CREATE INDEX IF NOT EXISTS idx_analytics_path_countries_agg_date ON analytics_path_countries_agg (aggregate_date);
CREATE INDEX IF NOT EXISTS idx_analytics_campaigns_agg_date ON analytics_campaigns_agg (aggregate_date);
CREATE INDEX IF NOT EXISTS idx_analytics_channels_agg_date ON analytics_channels_agg (aggregate_date);

-- Days whose path rollups are final. The hourly job keeps rebuilding
-- yesterday until a run well after midnight commits it, so page views ingested
-- late are not lost when runs are missed around midnight.
CREATE TABLE IF NOT EXISTS analytics_rollup_days (
    aggregate_date date PRIMARY KEY,
    finalized_at timestamp without time zone NOT NULL DEFAULT now()
);

-- Foreground time per page view, reported by the client script each time a
-- page is hidden or left, so one page view (view_id) can report several times.
-- Retained like the analytics table and rolled up into analytics_paths_agg.
CREATE TABLE IF NOT EXISTS analytics_engagement (
    engagement_ts timestamp without time zone NOT NULL,
    domain_id bigint NOT NULL,
    request_path text NOT NULL,
    view_id text NOT NULL,
    seconds integer NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_analytics_engagement_ts ON analytics_engagement (engagement_ts);
