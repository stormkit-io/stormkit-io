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
