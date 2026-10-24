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
