-- =====================================
-- 0002_alter_domains_add_columns.up.sql
-- =====================================

ALTER TABLE skitapi.domains ADD COLUMN IF NOT EXISTS custom_cert_value TEXT;
ALTER TABLE skitapi.domains ADD COLUMN IF NOT EXISTS custom_cert_key TEXT;

-- ===============================================
-- 0003_alter_deployments_add_status_checks.up.sql
-- ===============================================

ALTER TABLE skitapi.deployments ADD COLUMN IF NOT EXISTS status_checks TEXT;
ALTER TABLE skitapi.deployments ADD COLUMN IF NOT EXISTS status_checks_passed BOOLEAN;
ALTER TABLE skitapi.deployments ADD COLUMN IF NOT EXISTS is_immutable BOOLEAN;

UPDATE skitapi.deployments SET is_immutable = TRUE;

-- =======================================
-- 0004_create_analytics_agg_tables.up.sql
-- =======================================

CREATE TABLE IF NOT EXISTS skitapi.analytics_visitors_agg_200 (
    aggregate_date date NOT NULL,
    domain_id bigint NOT NULL,
    unique_visitors bigint NOT NULL,
    total_visitors bigint NOT NULL,
    PRIMARY KEY (aggregate_date, domain_id)
);

CREATE TABLE IF NOT EXISTS skitapi.analytics_visitors_agg_404 (
    aggregate_date date NOT NULL,
    domain_id bigint NOT NULL,
    unique_visitors bigint NOT NULL,
    total_visitors bigint NOT NULL,
    PRIMARY KEY (aggregate_date, domain_id)
);

CREATE TABLE IF NOT EXISTS skitapi.analytics_visitors_agg_hourly_200 (
    aggregate_date timestamp without time zone NOT NULL,
    domain_id bigint NOT NULL,
    unique_visitors bigint NOT NULL,
    total_visitors bigint NOT NULL,
    PRIMARY KEY (aggregate_date, domain_id)
);

CREATE TABLE IF NOT EXISTS skitapi.analytics_visitors_agg_hourly_404 (
    aggregate_date timestamp without time zone NOT NULL,
    domain_id bigint NOT NULL,
    unique_visitors bigint NOT NULL,
    total_visitors bigint NOT NULL,
    PRIMARY KEY (aggregate_date, domain_id)
);

CREATE TABLE IF NOT EXISTS skitapi.analytics_referrers (
    aggregate_date date NOT NULL,
    referrer text NOT NULL,
    request_path text NOT NULL,
    domain_id bigint NOT NULL,
    visit_count bigint NOT NULL,
    PRIMARY KEY (aggregate_date, referrer, request_path, domain_id)
);

CREATE TABLE IF NOT EXISTS skitapi.analytics_visitors_by_countries (
    aggregate_date date NOT NULL,
    domain_id bigint NOT NULL,
    country_iso_code text NOT NULL,
    visit_count bigint NOT NULL,
    PRIMARY KEY (aggregate_date, country_iso_code, domain_id)
);

DO $$
BEGIN
  BEGIN
    -- We keep these inserts here because the if the foreign keys exist, the migration 
    -- is already completed and it will move to the catch block and skip these inserts.

    -- Insert the initial data into 200 and 304 table

    INSERT INTO skitapi.analytics_visitors_agg_200 (aggregate_date, domain_id, unique_visitors, total_visitors)
        SELECT DATE(a.request_timestamp) as req_date, domain_id, COUNT(DISTINCT a.visitor_ip) AS unique_visitors, COUNT(a.domain_id) as total_visitors
        FROM skitapi.analytics a
        WHERE a.response_code = ANY(array[200, 304])
        GROUP BY req_date, a.domain_id
        ORDER BY req_date DESC
    ON CONFLICT (aggregate_date, domain_id) DO UPDATE SET unique_visitors = EXCLUDED.unique_visitors, total_visitors = EXCLUDED.total_visitors;

    -- Insert the initial data into 404 table

    INSERT INTO skitapi.analytics_visitors_agg_404 (aggregate_date, domain_id, unique_visitors, total_visitors)
        SELECT DATE(a.request_timestamp) as req_date, domain_id, COUNT(DISTINCT a.visitor_ip) AS unique_visitors, COUNT(a.domain_id) as total_visitors
        FROM skitapi.analytics a
        WHERE a.response_code = 404
        GROUP BY req_date, a.domain_id
        ORDER BY req_date DESC
    ON CONFLICT (aggregate_date, domain_id) DO UPDATE SET unique_visitors = EXCLUDED.unique_visitors, total_visitors = EXCLUDED.total_visitors;

    -- Insert the initial data into 200 and 304 hourly table

    INSERT INTO skitapi.analytics_visitors_agg_hourly_200 (aggregate_date, domain_id, unique_visitors, total_visitors)
        SELECT TO_CHAR(DATE_TRUNC('hour', a.request_timestamp), 'YYYY-MM-DD HH24:MI')::timestamp as agg_date, domain_id, COUNT(DISTINCT a.visitor_ip) AS unique_visitors, COUNT(a.domain_id) as total_visitors
        FROM skitapi.analytics a
        WHERE a.response_code = ANY(array[200, 304]) AND a.request_timestamp >= current_date - interval '3 days'
        GROUP BY agg_date, a.domain_id
        ORDER BY agg_date DESC
    ON CONFLICT (aggregate_date, domain_id) DO UPDATE SET unique_visitors = EXCLUDED.unique_visitors, total_visitors = EXCLUDED.total_visitors;

    -- Insert the initial data into 404 hourly table

    INSERT INTO skitapi.analytics_visitors_agg_hourly_404 (aggregate_date, domain_id, unique_visitors, total_visitors)
        SELECT TO_CHAR(DATE_TRUNC('hour', a.request_timestamp), 'YYYY-MM-DD HH24:MI')::timestamp as agg_date, domain_id, COUNT(DISTINCT a.visitor_ip) AS unique_visitors, COUNT(a.domain_id) as total_visitors
        FROM skitapi.analytics a
        WHERE a.response_code = 404 AND a.request_timestamp >= current_date - interval '3 days'
        GROUP BY agg_date, a.domain_id
        ORDER BY agg_date DESC
    ON CONFLICT (aggregate_date, domain_id) DO UPDATE SET unique_visitors = EXCLUDED.unique_visitors, total_visitors = EXCLUDED.total_visitors;

    -- Insert the initial data for referrers table

    INSERT INTO skitapi.analytics_referrers (aggregate_date, referrer, request_path, domain_id, visit_count)
        SELECT DATE(a.request_timestamp) as req_date, regexp_replace(regexp_replace(COALESCE(a.referrer, ''), '^https?://(www\.)?', ''), '/$', '') as referrer_domain, a.request_path, a.domain_id, COUNT(a.domain_id) AS total_count
        FROM skitapi.analytics a
        WHERE a.response_code IN (200, 304) AND a.request_timestamp >= current_date - interval '30 days'
        GROUP BY req_date, referrer_domain, a.request_path, a.domain_id
    ON CONFLICT (aggregate_date, referrer, request_path, domain_id) DO UPDATE SET visit_count = EXCLUDED.visit_count;

    -- Insert the initial data for analytics table

    INSERT INTO skitapi.analytics_visitors_by_countries (aggregate_date, country_iso_code, domain_id, visit_count)
        SELECT DATE(a.request_timestamp) as agg_date, a.country_iso_code, a.domain_id, COUNT(DISTINCT a.visitor_ip) AS count
        FROM skitapi.analytics a
        WHERE a.response_code IN (200, 304) AND a.country_iso_code IS NOT NULL AND a.request_timestamp >= current_date - interval '31 days'
        GROUP BY a.country_iso_code, agg_date, domain_id
    ON CONFLICT (aggregate_date, country_iso_code, domain_id) DO UPDATE SET visit_count = EXCLUDED.visit_count;

  EXCEPTION
    WHEN duplicate_table THEN  -- postgres raises duplicate_table at surprising times. Ex.: for UNIQUE constraints.
    WHEN duplicate_object THEN
      RAISE NOTICE 'Table constraint already exists';
  END;
END $$;

DROP TABLE IF EXISTS skitapi.analytics_archive;

-- ================================
-- 0005_create_volumes_table.up.sql
-- ================================

CREATE TABLE IF NOT EXISTS skitapi.stormkit_config (
    config_id serial primary key NOT NULL,
    volumes_config jsonb NULL,
    updated_at timestamp without time zone NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.volumes (
    file_id bigserial primary key NOT NULL,
    file_name text NOT NULL,
    file_path text NOT NULL,
    file_size bigint NOT NULL,
    is_public boolean NOT NULL,
    env_id bigint NOT NULL,
    updated_at timestamp without time zone NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

DO $$
BEGIN
  BEGIN

    ALTER TABLE ONLY skitapi.volumes
        ADD CONSTRAINT volumes_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.volumes
        ADD CONSTRAINT volumes_file_name_env_id_key UNIQUE (file_name, env_id);

  EXCEPTION
    WHEN duplicate_table THEN  -- postgres raises duplicate_table at surprising times. Ex.: for UNIQUE constraints.
    WHEN duplicate_object THEN
      RAISE NOTICE 'Table constraint already exists';
  END;
END $$;

-- ===============================================
-- 0006_alter_outbound_webhooks_and_mailer_setup.up.sql
-- ===============================================

ALTER TABLE skitapi.app_outbound_webhooks ALTER COLUMN trigger_when TYPE TEXT USING trigger_when::TEXT;
UPDATE skitapi.app_outbound_webhooks SET trigger_when = 'on_deploy_success' WHERE trigger_when = 'on_deploy';

ALTER TABLE skitapi.apps_build_conf ADD COLUMN IF NOT EXISTS mailer_conf jsonb;

CREATE TABLE IF NOT EXISTS skitapi.mailer (
    email_id bigserial primary key NOT NULL,
    email_to text NOT NULL,
    email_from text NOT NULL,
    email_subject text NOT NULL,
    email_body text NOT NULL,
    env_id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

DO $$
BEGIN
  BEGIN

    ALTER TABLE ONLY skitapi.mailer
        ADD CONSTRAINT mailer_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;

  EXCEPTION
    WHEN duplicate_table THEN  -- postgres raises duplicate_table at surprising times. Ex.: for UNIQUE constraints.
    WHEN duplicate_object THEN
      RAISE NOTICE 'Table constraint already exists';
  END;
END $$;

-- =======================================================================
-- 0007_alter_deployments_add_webhook_event_and_auto_deploy_commits.up.sql
-- =======================================================================

ALTER TABLE skitapi.deployments ADD COLUMN IF NOT EXISTS webhook_event JSONB;
ALTER TABLE skitapi.apps_build_conf ADD COLUMN IF NOT EXISTS auto_deploy_commits TEXT;

-- =====================================================
-- 0008_alter_function_triggers_add_webhook_event.up.sql
-- =====================================================

CREATE TABLE IF NOT EXISTS skitapi.function_trigger_logs (
    ftl_id serial primary key NOT NULL,
    trigger_id bigint NOT NULL,
    request jsonb NOT NULL,  -- includes information such as payload, method, path etc...
    response jsonb NOT NULL, -- includes information such as status, response body etc...
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

ALTER TABLE skitapi.function_triggers DROP CONSTRAINT IF EXISTS unique_config;
ALTER TABLE skitapi.function_triggers DROP COLUMN IF EXISTS timezone;
ALTER TABLE skitapi.function_triggers DROP COLUMN IF EXISTS app_id;
ALTER TABLE skitapi.function_triggers ALTER COLUMN cron TYPE text;

DO
$$
    BEGIN
        ALTER TABLE skitapi.function_triggers RENAME COLUMN options TO trigger_options;
		ALTER TABLE skitapi.function_triggers RENAME COLUMN status TO trigger_status;
        ALTER TABLE skitapi.function_triggers RENAME COLUMN id TO trigger_id;
    EXCEPTION
        WHEN undefined_column THEN RAISE NOTICE 'column does not exist';
    END;
$$;


DO $$
BEGIN
  BEGIN

    ALTER TABLE ONLY skitapi.function_trigger_logs
        ADD CONSTRAINT function_trigger_logs_trigger_id_fkey FOREIGN KEY (trigger_id) REFERENCES skitapi.function_triggers(trigger_id) ON DELETE CASCADE;

  EXCEPTION
    WHEN duplicate_table THEN  -- postgres raises duplicate_table at surprising times. Ex.: for UNIQUE constraints.
    WHEN duplicate_object THEN
      RAISE NOTICE 'Table constraint already exists';
  END;
END $$;

-- ===============================
-- 0009_domain_ping_updates.up.sql
-- ===============================

ALTER TABLE skitapi.domains ADD COLUMN IF NOT EXISTS last_ping JSONB;
ALTER TABLE skitapi.stormkit_config ADD COLUMN IF NOT EXISTS config_data JSONB;

DO $$
BEGIN
    UPDATE skitapi.stormkit_config
        SET config_data = jsonb_build_object('volumes', volumes_config);
    EXCEPTION
        WHEN undefined_column THEN
            RAISE NOTICE 'Column does not exist: %', SQLERRM;
END $$;

ALTER TABLE skitapi.stormkit_config DROP COLUMN IF EXISTS volumes_config;

-- =============================
-- 0010_volumes_file_type.up.sql
-- =============================

ALTER TABLE skitapi.volumes ADD COLUMN IF NOT EXISTS file_metadata JSONB;

-- =====================================
-- 0011_snippet_unique_constraint.up.sql
-- =====================================

ALTER TABLE skitapi.snippets ADD COLUMN IF NOT EXISTS snippet_content_hash text NULL;
CREATE UNIQUE INDEX IF NOT EXISTS snippets_snippet_content_hash_key ON skitapi.snippets USING btree (env_id, snippet_content_hash);

-- =================================================
-- 0012_remove_unnecessary_columns_and_tables.up.sql
-- =================================================

DO $$
BEGIN
    ALTER TABLE skitapi.user_referrals DROP CONSTRAINT IF EXISTS idx_user_referrals_details;
    ALTER TABLE skitapi.user_referrals DROP CONSTRAINT IF EXISTS idx_user_referrals_invited_by;
    ALTER TABLE skitapi.user_referrals DROP CONSTRAINT IF EXISTS user_referrals_invited_by_users_id_fkey;
    ALTER TABLE skitapi.user_referrals DROP CONSTRAINT IF EXISTS user_referrals_display_name_provider_key;
    ALTER TABLE skitapi.apps_build_conf DROP CONSTRAINT IF EXISTS apps_build_conf_domain_unique_key;
    ALTER TABLE skitapi.apps_build_conf DROP CONSTRAINT IF EXISTS idx_apps_build_conf_domain;

    EXCEPTION
        WHEN undefined_table THEN
            RAISE NOTICE 'Table does not exist: %', SQLERRM;
END $$;

DROP TABLE IF EXISTS skitapi.user_referrals;

ALTER TABLE skitapi.deployments DROP COLUMN IF EXISTS is_v2;
ALTER TABLE skitapi.deployments DROP COLUMN IF EXISTS lambda_version;
ALTER TABLE skitapi.deployments DROP COLUMN IF EXISTS percentage;
ALTER TABLE skitapi.deployments DROP COLUMN IF EXISTS is_serverless;
ALTER TABLE skitapi.deployments DROP COLUMN IF EXISTS s3_key_prefix;
ALTER TABLE skitapi.deployments DROP COLUMN IF EXISTS s3_bucket_name;
ALTER TABLE skitapi.deployments DROP COLUMN IF EXISTS stormkit_config;
ALTER TABLE skitapi.apps DROP COLUMN IF EXISTS stormkit_config_hash;
ALTER TABLE skitapi.apps_build_conf DROP COLUMN IF EXISTS domain_name;
ALTER TABLE skitapi.apps_build_conf DROP COLUMN IF EXISTS domain_verified;
ALTER TABLE skitapi.apps_build_conf DROP COLUMN IF EXISTS domain_verified_at;
ALTER TABLE skitapi.apps_build_conf DROP COLUMN IF EXISTS domain_token;
ALTER TABLE skitapi.apps_build_conf DROP COLUMN IF EXISTS snippets;
ALTER TABLE skitapi.apps_build_conf DROP COLUMN IF EXISTS proxy_enabled;

-- ======================
-- 0013_last_login.up.sql
-- ======================

ALTER TABLE skitapi.users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMP WITHOUT TIME ZONE NULL;

-- =====================
-- 0014_auth_wall.up.sql
-- =====================

CREATE TABLE IF NOT EXISTS skitapi.auth_wall (
    login_id bigserial primary key NOT NULL,
    login_email text NOT NULL,
    login_password text NOT NULL,
    last_login_at timestamp without time zone,
    env_id bigint NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS auth_wall_env_id_login_email ON skitapi.auth_wall USING btree (env_id, login_email);

ALTER TABLE skitapi.apps_build_conf ADD COLUMN IF NOT EXISTS auth_wall_conf jsonb;

DO $$
BEGIN
  BEGIN

    ALTER TABLE ONLY skitapi.auth_wall
        ADD CONSTRAINT auth_wall_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON UPDATE CASCADE ON DELETE CASCADE;

  EXCEPTION
    WHEN duplicate_table THEN  -- postgres raises duplicate_table at surprising times. Ex.: for UNIQUE constraints.
    WHEN duplicate_object THEN
      RAISE NOTICE 'Table constraint already exists';
  END;
END $$;

DROP INDEX IF EXISTS skitapi.idx_apps_api_key;
DROP INDEX IF EXISTS skitapi.idx_apps_build_conf_api_key;
DROP INDEX IF EXISTS skitapi.idx_apps_api_key_unique;
DROP INDEX IF EXISTS skitapi.idx_apps_build_conf_api_key_unique;

ALTER TABLE skitapi.apps DROP COLUMN IF EXISTS api_key;
ALTER TABLE skitapi.apps_build_conf DROP COLUMN IF EXISTS api_key;

-- ======================
-- 0015_2025-03-31.up.sql
-- ======================

ALTER TABLE skitapi.apps ALTER COLUMN repo DROP NOT NULL;
ALTER TABLE skitapi.deployments ALTER COLUMN branch DROP NOT NULL;
ALTER TABLE skitapi.deployments ALTER COLUMN checkout_repo DROP NOT NULL;

-- ======================
-- 0016_2025-09-29.up.sql
-- ======================

-- Drop feature flags
DROP INDEX IF EXISTS skitapi.feature_flags_ff_name_unique_key;
DROP INDEX IF EXISTS skitapi.idx_feature_flags_app_id;

DO $$ 
BEGIN
	IF EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'skitapi' AND table_name = 'feature_flags') THEN
		ALTER TABLE ONLY skitapi.feature_flags DROP CONSTRAINT IF EXISTS feature_flags_app_id_fkey;
		ALTER TABLE ONLY skitapi.feature_flags DROP CONSTRAINT IF EXISTS feature_flags_env_id_fkey;
	END IF;
END $$;

DROP TABLE IF EXISTS skitapi.feature_flags;

-- Drop stripe client id from users and add metadata column
DROP INDEX IF EXISTS skitapi.idx_users_stripe_client_id;
ALTER TABLE ONLY skitapi.users DROP COLUMN IF EXISTS stripe_client_id;
ALTER TABLE ONLY skitapi.users ADD COLUMN IF NOT EXISTS metadata JSONB;

-- Drop user stats: our business model is no longer based on deployments
DROP INDEX IF EXISTS skitapi.idx_userid_user_stats;
DROP TABLE IF EXISTS skitapi.user_stats;

-- ======================
-- 0017_2025-10-21.up.sql
-- ======================

-- ==========================================================
-- create user_id column in teams table
-- rename column to client_package_size in deployments table
-- ==========================================================

ALTER TABLE skitapi.teams ADD COLUMN IF NOT EXISTS user_id BIGINT NULL;

DO $$
BEGIN
  BEGIN

    ALTER TABLE ONLY skitapi.teams
        ADD CONSTRAINT teams_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;

  EXCEPTION
    WHEN duplicate_table THEN  -- postgres raises duplicate_table at surprising times. Ex.: for UNIQUE constraints.
    WHEN duplicate_object THEN
      RAISE NOTICE 'Table constraint already exists';
  END;
END $$;

UPDATE skitapi.teams t SET user_id = (
    SELECT
        user_id
    FROM
        skitapi.team_members tm
    WHERE
        tm.team_id = t.team_id AND
        tm.member_role = 'owner'
);

ALTER TABLE skitapi.teams ALTER COLUMN user_id SET NOT NULL;

DO $$
BEGIN
  BEGIN

    ALTER TABLE skitapi.deployments RENAME COLUMN s3_total_size_in_bytes TO client_package_size;
    ALTER TABLE skitapi.deployments ALTER COLUMN client_package_size TYPE BIGINT;

  EXCEPTION
    WHEN undefined_column THEN
  END;
END $$;

-- ======================
-- 0018_2025-11-13.up.sql
-- ======================

ALTER TABLE skitapi.users ADD COLUMN IF NOT EXISTS is_approved BOOLEAN NULL DEFAULT TRUE;

-- ======================
-- 0019_2025-12-09.up.sql
-- ======================

ALTER TABLE skitapi.apps_build_conf ADD COLUMN IF NOT EXISTS schema_conf BYTEA;
ALTER TABLE skitapi.deployments ADD COLUMN IF NOT EXISTS migrations_folder TEXT;
ALTER TABLE skitapi.deployments ADD COLUMN IF NOT EXISTS upload_result jsonb;
ALTER TABLE skitapi.deployments DROP COLUMN IF EXISTS s3_number_of_files;

-- Migrate data from old columns to upload_result jsonb structure
DO $$
DECLARE
    has_api_package_size boolean;
    has_client_package_size boolean;
    has_server_package_size boolean;
    has_function_location boolean;
    has_storage_location boolean;
    has_api_location boolean;
    json_parts TEXT[];
    where_parts TEXT[];
BEGIN
    -- Check which columns exist
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'skitapi' 
        AND table_name = 'deployments' 
        AND column_name = 'api_package_size'
    ) INTO has_api_package_size;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'skitapi' 
        AND table_name = 'deployments' 
        AND column_name = 'client_package_size'
    ) INTO has_client_package_size;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'skitapi' 
        AND table_name = 'deployments' 
        AND column_name = 'server_package_size'
    ) INTO has_server_package_size;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'skitapi' 
        AND table_name = 'deployments' 
        AND column_name = 'function_location'
    ) INTO has_function_location;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'skitapi' 
        AND table_name = 'deployments' 
        AND column_name = 'storage_location'
    ) INTO has_storage_location;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'skitapi' 
        AND table_name = 'deployments' 
        AND column_name = 'api_location'
    ) INTO has_api_location;

    -- Only migrate if at least one old column exists
    IF has_api_package_size OR has_client_package_size OR has_server_package_size OR 
       has_function_location OR has_storage_location OR has_api_location THEN
        
        -- Build array of non-null JSON parts
        json_parts := ARRAY[]::TEXT[];
        where_parts := ARRAY[]::TEXT[];

        IF has_api_package_size THEN
            json_parts := array_append(json_parts, '''serverlessBytes'', api_package_size');
            where_parts := array_append(where_parts, 'api_package_size IS NOT NULL');
        END IF;

        IF has_client_package_size THEN
            json_parts := array_append(json_parts, '''clientBytes'', client_package_size');
            where_parts := array_append(where_parts, 'client_package_size IS NOT NULL');
        END IF;

        IF has_server_package_size THEN
            json_parts := array_append(json_parts, '''serverBytes'', server_package_size');
            where_parts := array_append(where_parts, 'server_package_size IS NOT NULL');
        END IF;

        IF has_function_location THEN
            json_parts := array_append(json_parts, '''serverLocation'', function_location');
            where_parts := array_append(where_parts, 'function_location IS NOT NULL');
        END IF;

        IF has_storage_location THEN
            json_parts := array_append(json_parts, '''clientLocation'', storage_location');
            where_parts := array_append(where_parts, 'storage_location IS NOT NULL');
        END IF;

        IF has_api_location THEN
            json_parts := array_append(json_parts, '''serverlessLocation'', api_location');
            where_parts := array_append(where_parts, 'api_location IS NOT NULL');
        END IF;

        -- Execute update with properly joined parts
        EXECUTE format($sql$
            UPDATE skitapi.deployments
            SET upload_result = jsonb_build_object(%s)
            WHERE upload_result IS NULL
            AND (%s)
        $sql$,
            array_to_string(json_parts, ', '),
            array_to_string(where_parts, ' OR ')
        );
        
        RAISE NOTICE 'Migrated deployment upload results from old columns to upload_result jsonb';
    ELSE
        RAISE NOTICE 'No old columns found, skipping migration';
    END IF;

    -- Drop old columns if they exist
    IF has_api_package_size THEN
        ALTER TABLE skitapi.deployments DROP COLUMN api_package_size;
        RAISE NOTICE 'Dropped column api_package_size';
    END IF;

    IF has_client_package_size THEN
        ALTER TABLE skitapi.deployments DROP COLUMN client_package_size;
        RAISE NOTICE 'Dropped column client_package_size';
    END IF;

    IF has_server_package_size THEN
        ALTER TABLE skitapi.deployments DROP COLUMN server_package_size;
        RAISE NOTICE 'Dropped column server_package_size';
    END IF;

    IF has_function_location THEN
        ALTER TABLE skitapi.deployments DROP COLUMN function_location;
        RAISE NOTICE 'Dropped column function_location';
    END IF;

    IF has_storage_location THEN
        ALTER TABLE skitapi.deployments DROP COLUMN storage_location;
        RAISE NOTICE 'Dropped column storage_location';
    END IF;

    IF has_api_location THEN
        ALTER TABLE skitapi.deployments DROP COLUMN api_location;
        RAISE NOTICE 'Dropped column api_location';
    END IF;
END $$;

-- ======================
-- 0020_2026-01-04.up.sql
-- ======================

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

-- ===================
-- 0021_v1-26.5.up.sql
-- ===================

ALTER TABLE skitapi.oauth_configs ADD COLUMN IF NOT EXISTS provider_data bytea NOT NULL;
ALTER TABLE skitapi.oauth_configs DROP COLUMN IF EXISTS provider_client_id;
ALTER TABLE skitapi.oauth_configs DROP COLUMN IF EXISTS provider_client_secret;
ALTER TABLE skitapi.oauth_configs DROP COLUMN IF EXISTS provider_redirect_url;
ALTER TABLE skitapi.oauth_configs DROP COLUMN IF EXISTS provider_scopes;


