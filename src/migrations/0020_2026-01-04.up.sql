ALTER TABLE skitapi.apps_build_conf ADD COLUMN IF NOT EXISTS auth_conf bytea;

CREATE TABLE IF NOT EXISTS skitapi.oauth_configs (
    provider_id serial primary key NOT NULL,
    provider_name text NOT NULL,
    provider_client_id text NOT NULL,
    provider_client_secret text NOT NULL,
    provider_redirect_url text NOT NULL,
    provider_scopes text[] NOT NULL,
    provider_status boolean NOT NULL DEFAULT FALSE,
    env_id bigint NOT NULL,
    app_id bigint NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS oauth_configs_env_provider_unique_key ON skitapi.oauth_configs (env_id, provider_name);

DO $$
BEGIN
  BEGIN

        ALTER TABLE ONLY skitapi.oauth_configs
            ADD CONSTRAINT oauth_configs_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;

        ALTER TABLE ONLY skitapi.oauth_configs
            ADD CONSTRAINT oauth_configs_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;

  EXCEPTION
    WHEN duplicate_table THEN 
    WHEN duplicate_object THEN
      RAISE NOTICE 'Table constraint already exists';
  END;

END $$;