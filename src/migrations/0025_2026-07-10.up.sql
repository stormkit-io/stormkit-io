-- Maintenance mode (issue #21): a per-environment toggle that blocks public
-- traffic and serves a maintenance page. Stored as jsonb so follow-up
-- iterations (custom redirect, custom HTML) extend the same config object.
ALTER TABLE apps_build_conf ADD COLUMN IF NOT EXISTS maintenance_conf jsonb;
