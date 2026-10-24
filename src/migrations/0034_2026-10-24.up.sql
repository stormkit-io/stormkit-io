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
