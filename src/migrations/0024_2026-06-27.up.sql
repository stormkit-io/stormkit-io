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
