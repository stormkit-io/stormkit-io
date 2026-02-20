CREATE TABLE IF NOT EXISTS skitapi.licenses (
    license_id serial primary key NOT NULL,
    license_key text NOT NULL,
    license_version text NOT NULL DEFAULT '2024-06-10',
    is_premium boolean NOT NULL DEFAULT FALSE,
    is_ultimate boolean NOT NULL DEFAULT FALSE,
    number_of_seats integer NOT NULL,
    user_id bigint NULL,
    metadata jsonb NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.sk_system_snapshots (
    snapshot_identifier text,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.user_metrics (
    metric_id bigserial primary key NOT NULL,
    user_id bigint NOT NULL,
    bandwidth_bytes bigint DEFAULT 0,
    storage_bytes bigint DEFAULT 0,
    build_minutes bigint DEFAULT 0,
    function_invocations bigint DEFAULT 0,
    year integer NOT NULL,
    month integer NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS user_metrics_user_id_year_month_unique_key ON skitapi.user_metrics USING btree (user_id, year, month);

CREATE UNIQUE INDEX IF NOT EXISTS licenses_user_id_unique_key ON skitapi.licenses USING btree (user_id) WHERE user_id IS NOT NULL;

DO $$
BEGIN
  BEGIN

    ALTER TABLE ONLY skitapi.licenses
        ADD CONSTRAINT licenses_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.user_metrics
        ADD CONSTRAINT user_metrics_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;

  EXCEPTION
    WHEN duplicate_table THEN  -- postgres raises duplicate_table at surprising times. Ex.: for UNIQUE constraints.
    WHEN duplicate_object THEN
      RAISE NOTICE 'Table constraint already exists';
  END;
END $$;
