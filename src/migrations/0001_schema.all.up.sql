--
-- PostgreSQL database dump
--

-- Dumped from database version 16.3 (Debian 16.3-1.pgdg120+1)
-- Dumped by pg_dump version 16.3

-- Started on 2024-08-09 16:28:31 +03

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', 'skitapi', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

CREATE SCHEMA IF NOT EXISTS public;
CREATE SCHEMA IF NOT EXISTS skitapi;

COMMENT ON SCHEMA public IS 'standard public schema';


DO $$ BEGIN
    CREATE TYPE skitapi.access_token_type AS ENUM (
        'github',
        'bitbucket',
        'gitlab'
    );

    CREATE TYPE skitapi.auto_deploy_type AS ENUM (
        'commit',
        'pull_request'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

SET default_tablespace = '';

SET default_table_access_method = heap;

CREATE TABLE IF NOT EXISTS public.migrations (
    migration_version integer NOT NULL,
    seed_version integer NULL,
    dirty boolean NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.stormkit_config (
    config_id serial primary key NOT NULL,
    config_data jsonb NULL,
    updated_at timestamp without time zone NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.access_log_stats (
    host_name text NOT NULL,
    median_duration integer NOT NULL,
    response_status integer NOT NULL,
    number_of_responses bigint NOT NULL,
    logs_date date NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.access_logs (
    request_method text NOT NULL,
    request_path text NOT NULL,
    remote_addr text NOT NULL,
    user_agent text NOT NULL,
    scheme text NOT NULL,
    host_name text NOT NULL,
    "timestamp" bigint NOT NULL,
    duration numeric(8,2) NOT NULL,
    response_status integer NOT NULL,
    max_memory_used integer,
    memory_size integer,
    billed_duration integer,
    app_id bigint
);

CREATE TABLE IF NOT EXISTS skitapi.analytics (
    analytics_id bigserial primary key NOT NULL,
    app_id bigint NOT NULL,
    env_id bigint NOT NULL,
    domain_id bigint,
    visitor_ip text,
    request_timestamp timestamp without time zone,
    request_path text NOT NULL,
    response_code integer NOT NULL,
    user_agent text,
    referrer text,
    country_iso_code text
);

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
    aggregate_date timestamp without time zone NOT NULL ,
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
    domain_id bigint NOT NULL,
    referrer text NOT NULL,
    request_path text NOT NULL,
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

CREATE TABLE IF NOT EXISTS skitapi.api_keys (
    key_id serial primary key NOT NULL,
    app_id bigint,
    env_id bigint,
    key_name text NOT NULL,
    key_value text NOT NULL,
    key_scope text NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    user_id bigint,
    team_id bigint
);

CREATE TABLE IF NOT EXISTS skitapi.app_logs (
    app_id bigint,
    host_name text,
    "timestamp" bigint NOT NULL,
    request_id text,
    log_label text,
    log_data text NOT NULL,
    env_id bigint,
    deployment_id bigint,
    id bigserial primary key NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.app_members (
    app_id bigint,
    user_id bigint,
    invited_by bigint NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.app_outbound_webhooks (
    app_id bigint,
    request_headers jsonb,
    request_body text,
    request_url text NOT NULL,
    request_method text DEFAULT 'GET'::text NOT NULL,
    trigger_when text NOT NULL,
    wh_id serial primary key NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.apps (
    app_id serial primary key NOT NULL,
    private_key bytea NOT NULL,
    repo text,
    display_name text,
    user_id bigint NOT NULL,
    client_id text NOT NULL,
    client_secret bytea NOT NULL,
    auto_deploy skitapi.auto_deploy_type DEFAULT 'commit'::skitapi.auto_deploy_type,
    auto_deploy_commit_prefix text,
    is_sample_project boolean DEFAULT false,
    deploy_trigger text,
    runtime text,
    default_env_name text,
    proxy text,
    deleted_at timestamp without time zone,
    artifacts_deleted boolean DEFAULT false NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    is_lambda_migrated_to_v2 boolean DEFAULT false,
    team_id bigint
);

CREATE TABLE IF NOT EXISTS skitapi.apps_build_conf (
    app_id bigint,
    env_name text NOT NULL,
    build_conf jsonb,
    branch text NOT NULL,
    auto_publish boolean DEFAULT true,
    deleted_at timestamp without time zone,
    updated_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    env_id serial primary key NOT NULL,
    auto_deploy_commits text,
    auto_deploy_branches text,
    auto_deploy boolean DEFAULT false,
    mailer_conf jsonb,
    auth_wall_conf jsonb
);

CREATE TABLE IF NOT EXISTS skitapi.audit_logs (
    audit_id bigserial primary key NOT NULL,
    audit_action text NOT NULL,
    audit_diff jsonb,
    token_name text,
    team_id bigint,
    app_id bigint,
    env_id bigint,
    user_id bigint,
    user_display text,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.deployments (
    deployment_id serial primary key NOT NULL,
    branch text,
    env_name text,
    app_id bigint NOT NULL,
    env_id bigint,
    config_snapshot text,
    s3_number_of_files integer,
    exit_code integer,
    pull_request_number integer,
    logs text,
    status_checks text,
    status_checks_passed boolean,
    is_auto_deploy boolean DEFAULT false NOT NULL,
    commit_id text,
    stopped_at timestamp without time zone,
    deleted_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    is_fork boolean,
    auto_publish boolean,
    commit_message text,
    commit_author text,
    checkout_repo text,
    github_run_id bigint,
    server_package_size bigint,
    client_package_size bigint,
    api_package_size bigint,
    storage_location text,
    function_location text,
    api_location text,
    api_path_prefix text,
    error text,
    build_manifest jsonb,
    artifacts_deleted boolean DEFAULT false,
    webhook_event jsonb,
    is_immutable boolean
);

CREATE TABLE IF NOT EXISTS skitapi.deployments_published (
    deployment_id bigint NOT NULL,
    env_id bigint NOT NULL,
    percentage_released numeric(4,1) DEFAULT 0 NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.domains (
    domain_id serial primary key NOT NULL,
    app_id bigint NOT NULL,
    env_id bigint NOT NULL,
    domain_name text NOT NULL,
    domain_token text,
    domain_verified boolean DEFAULT false NOT NULL,
    domain_verified_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text),
    custom_cert_value text,
    custom_cert_key text,
    last_ping JSONB,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.function_triggers (
    trigger_id serial primary key NOT NULL,
    trigger_options jsonb NOT NULL,
    trigger_status boolean DEFAULT true NOT NULL,
    env_id bigint NOT NULL,
    cron text NOT NULL,
    next_run_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    updated_at timestamp without time zone
);

CREATE TABLE IF NOT EXISTS skitapi.function_trigger_logs (
    ftl_id serial primary key NOT NULL,
    trigger_id bigint NOT NULL,
    request jsonb NOT NULL,  -- includes information such as payload, method, path etc...
    response jsonb NOT NULL, -- includes information such as status, response body etc...
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.geo_countries (
    geoname_id serial primary key NOT NULL,
    locale_code text,
    continent_code text,
    continent_name text,
    country_iso_code text,
    country_name text,
    is_in_european_union boolean
);

CREATE TABLE IF NOT EXISTS skitapi.geo_ips (
    id serial primary key NOT NULL,
    network inet,
    geoname_id integer,
    registered_country_geoname_id integer,
    represented_country_geoname_id integer,
    is_anonymous_proxy boolean,
    is_satellite_provider boolean
);

CREATE TABLE IF NOT EXISTS skitapi.snippets (
    snippet_id serial primary key NOT NULL,
    app_id bigint NOT NULL,
    env_id bigint NOT NULL,
    snippet_title text NOT NULL,
    snippet_content text NOT NULL,
    snippet_content_hash text NULL,
    snippet_location text DEFAULT 'head'::text NOT NULL,
    snippet_rules jsonb,
    should_prepend boolean DEFAULT false NOT NULL,
    is_enabled boolean DEFAULT false NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.team_members (
    member_id serial primary key NOT NULL,
    team_id bigint NOT NULL,
    user_id bigint NOT NULL,
    inviter_id bigint,
    member_role text DEFAULT 'developer'::text NOT NULL,
    membership_status boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.teams (
    team_id serial primary key NOT NULL,
    team_name text NOT NULL,
    team_slug text NOT NULL,
    user_id bigint NOT NULL, -- This is a shorthand for the team owner for billing purposes
    is_default boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    deleted_at timestamp without time zone
);

CREATE TABLE IF NOT EXISTS skitapi.user_access_tokens (
    user_id bigint NOT NULL,
    display_name text NOT NULL,
    account_uri text NOT NULL,
    provider skitapi.access_token_type NOT NULL,
    token_type text NOT NULL,
    token_value text NOT NULL,
    token_refresh text,
    expire_at timestamp without time zone NOT NULL,
    personal_access_token bytea
);

CREATE TABLE IF NOT EXISTS skitapi.user_emails (
    email_id serial primary key NOT NULL,
    user_id bigint NOT NULL,
    email text NOT NULL,
    is_primary boolean DEFAULT false NOT NULL,
    is_verified boolean DEFAULT false NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.users (
    user_id serial primary key NOT NULL,
    first_name text,
    last_name text,
    display_name text NOT NULL,
    avatar_uri text,
    is_admin boolean DEFAULT false NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    updated_at timestamp without time zone,
    last_login_at timestamp without time zone,
    deleted_at timestamp without time zone,
    is_approved boolean, -- Null means pending, false means not approved, true means approved
    metadata JSONB
);

CREATE TABLE IF NOT EXISTS skitapi.volumes (
    file_id bigserial primary key NOT NULL,
    file_name text NOT NULL,
    file_path text NOT NULL,
    file_size bigint NOT NULL,
    file_metadata JSONB,
    is_public boolean NOT NULL,
    env_id bigint NOT NULL,
    updated_at timestamp without time zone NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.mailer (
    email_id bigserial primary key NOT NULL,
    email_to text NOT NULL,
    email_from text NOT NULL,
    email_subject text NOT NULL,
    email_body text NOT NULL,
    env_id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE TABLE IF NOT EXISTS skitapi.auth_wall (
    login_id bigserial primary key NOT NULL,
    login_email text NOT NULL,
    login_password text NOT NULL,
    last_login_at timestamp without time zone,
    env_id bigint NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS api_keys_value_unique_key ON skitapi.api_keys USING btree (key_value);

CREATE UNIQUE INDEX IF NOT EXISTS apps_build_conf_env_name_unique_key ON skitapi.apps_build_conf USING btree (app_id, env_name) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX IF NOT EXISTS apps_display_name_unique_key ON skitapi.apps USING btree (display_name) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX IF NOT EXISTS domains_domain_name_unique_key ON skitapi.domains USING btree (domain_name) WHERE (domain_verified IS TRUE);

CREATE UNIQUE INDEX IF NOT EXISTS snippets_snippet_content_hash_key ON skitapi.snippets USING btree (env_id, snippet_content_hash);

CREATE UNIQUE INDEX IF NOT EXISTS auth_wall_env_id_login_email ON skitapi.auth_wall USING btree (env_id, login_email);

CREATE INDEX IF NOT EXISTS idx_access_log_stats_host_name ON skitapi.access_log_stats USING btree (host_name);

CREATE INDEX IF NOT EXISTS idx_access_logs_host_name ON skitapi.access_logs USING btree (host_name);

CREATE INDEX IF NOT EXISTS idx_app_deleted_at ON skitapi.apps USING btree (deleted_at);

CREATE INDEX IF NOT EXISTS idx_app_display_name ON skitapi.apps USING btree (display_name);

CREATE INDEX IF NOT EXISTS idx_app_logs_app_id_host_name ON skitapi.app_logs USING btree (app_id, host_name);

CREATE INDEX IF NOT EXISTS idx_app_logs_label ON skitapi.app_logs USING btree (log_label);

CREATE INDEX IF NOT EXISTS idx_app_logs_request_id ON skitapi.app_logs USING btree (request_id);

CREATE INDEX IF NOT EXISTS idx_app_member_invited_by ON skitapi.app_members USING btree (invited_by);

CREATE INDEX IF NOT EXISTS idx_app_member_user_id ON skitapi.app_members USING btree (user_id);

CREATE INDEX IF NOT EXISTS idx_app_repo ON skitapi.apps USING btree (lower(repo));

CREATE INDEX IF NOT EXISTS idx_app_user_id ON skitapi.apps USING btree (user_id);

CREATE INDEX IF NOT EXISTS idx_geo_ips_network ON skitapi.geo_ips USING gist (network inet_ops);

CREATE INDEX IF NOT EXISTS idx_apps_build_conf_branch ON skitapi.apps_build_conf USING btree (branch);

CREATE INDEX IF NOT EXISTS idx_deployments_app_id ON skitapi.deployments USING btree (app_id);

CREATE INDEX IF NOT EXISTS idx_deployments_branch_name ON skitapi.deployments USING btree (branch);

CREATE INDEX IF NOT EXISTS idx_deployments_created_at ON skitapi.deployments USING btree (((created_at)::date));

CREATE INDEX IF NOT EXISTS idx_deployments_env_name ON skitapi.deployments USING btree (env_name);

CREATE INDEX IF NOT EXISTS idx_deployments_published_deployment_id ON skitapi.deployments_published USING btree (deployment_id);

CREATE INDEX IF NOT EXISTS idx_deployments_published_env_id ON skitapi.deployments_published USING btree (env_id);

CREATE INDEX IF NOT EXISTS idx_next_run_at ON skitapi.function_triggers USING btree (next_run_at);

CREATE INDEX IF NOT EXISTS idx_user_access_tokens_user_id ON skitapi.user_access_tokens USING btree (user_id);

CREATE UNIQUE INDEX IF NOT EXISTS team_members_team_id_user_id ON skitapi.team_members USING btree (team_id, user_id);

CREATE UNIQUE INDEX IF NOT EXISTS user_emails_email_unique_key ON skitapi.user_emails USING btree (email) WHERE (is_verified IS TRUE);

DO $$
BEGIN
  BEGIN
    ALTER TABLE ONLY skitapi.access_logs
        ADD CONSTRAINT access_logs_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON UPDATE CASCADE ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.analytics
        ADD CONSTRAINT analytics_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.analytics
        ADD CONSTRAINT analytics_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.api_keys
        ADD CONSTRAINT api_keys_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.api_keys
        ADD CONSTRAINT api_keys_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.api_keys
        ADD CONSTRAINT api_keys_team_id_fkey FOREIGN KEY (team_id) REFERENCES skitapi.teams(team_id) ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.api_keys
        ADD CONSTRAINT api_keys_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.app_logs
        ADD CONSTRAINT app_logs_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON UPDATE CASCADE ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.app_members
        ADD CONSTRAINT app_members_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.app_members
        ADD CONSTRAINT app_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.app_members
        ADD CONSTRAINT app_members_app_id_user_id_key UNIQUE (app_id, user_id);
  
    ALTER TABLE ONLY skitapi.app_outbound_webhooks
        ADD CONSTRAINT app_outbound_webhooks_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.apps_build_conf
        ADD CONSTRAINT apps_build_conf_app_id_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
  
    ALTER TABLE ONLY skitapi.apps
        ADD CONSTRAINT apps_user_id_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.audit_logs
        ADD CONSTRAINT audit_logs_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.audit_logs
        ADD CONSTRAINT audit_logs_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.audit_logs
        ADD CONSTRAINT audit_logs_team_id_fkey FOREIGN KEY (team_id) REFERENCES skitapi.teams(team_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.audit_logs
        ADD CONSTRAINT audit_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id);
  
    ALTER TABLE ONLY skitapi.deployments
        ADD CONSTRAINT deployments_app_id_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.deployments
        ADD CONSTRAINT deployments_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.deployments_published
        ADD CONSTRAINT deployments_published_deployment_id_fkey FOREIGN KEY (deployment_id) REFERENCES skitapi.deployments(deployment_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.deployments_published
        ADD CONSTRAINT deployments_published_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.domains
        ADD CONSTRAINT domains_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.domains
        ADD CONSTRAINT domains_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.app_logs
        ADD CONSTRAINT fk_deployment_id FOREIGN KEY (deployment_id) REFERENCES skitapi.deployments(deployment_id) ON UPDATE CASCADE ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.app_logs
        ADD CONSTRAINT fk_env_id FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON UPDATE CASCADE ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.function_triggers
        ADD CONSTRAINT function_triggers_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.function_trigger_logs
        ADD CONSTRAINT function_trigger_logs_trigger_id_fkey FOREIGN KEY (trigger_id) REFERENCES skitapi.function_triggers(trigger_id) ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.geo_ips
        ADD CONSTRAINT geo_ips_geoname_id_fkey FOREIGN KEY (geoname_id) REFERENCES skitapi.geo_countries(geoname_id);

    ALTER TABLE ONLY skitapi.snippets
        ADD CONSTRAINT snippets_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.snippets
        ADD CONSTRAINT snippets_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.teams
        ADD CONSTRAINT teams_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.team_members
        ADD CONSTRAINT team_members_inviter_id_fkey FOREIGN KEY (inviter_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.team_members
        ADD CONSTRAINT team_members_team_id_fkey FOREIGN KEY (team_id) REFERENCES skitapi.teams(team_id) ON DELETE CASCADE;
  
    ALTER TABLE ONLY skitapi.team_members
        ADD CONSTRAINT team_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.user_access_tokens
        ADD CONSTRAINT user_access_tokens_user_id_users_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;
      
    ALTER TABLE ONLY skitapi.user_emails
        ADD CONSTRAINT user_emails_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;
      
    ALTER TABLE ONLY skitapi.volumes
        ADD CONSTRAINT volumes_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.mailer
        ADD CONSTRAINT mailer_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.auth_wall
        ADD CONSTRAINT auth_wall_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON UPDATE CASCADE ON DELETE CASCADE;

    ALTER TABLE ONLY skitapi.user_access_tokens
        ADD CONSTRAINT user_access_tokens_user_id_provider_key UNIQUE (user_id, provider);

    ALTER TABLE ONLY skitapi.access_log_stats
        ADD CONSTRAINT access_log_stats_host_name_response_status_logs_date_key UNIQUE (host_name, response_status, logs_date);

    ALTER TABLE ONLY skitapi.volumes
        ADD CONSTRAINT volumes_file_name_env_id_key UNIQUE (file_name, env_id);
  EXCEPTION
    WHEN duplicate_table THEN  -- postgres raises duplicate_table at surprising times. Ex.: for UNIQUE constraints.
    WHEN duplicate_object THEN
      RAISE NOTICE 'Table constraint already exists';
  END;
END $$;

-- Completed on 2024-08-09 16:28:31 +03

--
-- PostgreSQL database dump complete
--

