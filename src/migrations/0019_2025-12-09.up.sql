ALTER TABLE skitapi.apps_build_conf ADD COLUMN IF NOT EXISTS schema_conf BYTEA;
ALTER TABLE skitapi.deployments ADD COLUMN IF NOT EXISTS migrations_path TEXT;
