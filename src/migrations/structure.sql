--
-- PostgreSQL database dump
--

-- Dumped from database version 17.7 (Debian 17.7-3.pgdg13+1)
-- Dumped by pg_dump version 17.7 (Debian 17.7-3.pgdg13+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: skitapi; Type: SCHEMA; Schema: -; Owner: skitadmin
--

CREATE SCHEMA skitapi;


ALTER SCHEMA skitapi OWNER TO skitadmin;

--
-- Name: access_token_type; Type: TYPE; Schema: skitapi; Owner: skitadmin
--

CREATE TYPE skitapi.access_token_type AS ENUM (
    'github',
    'bitbucket',
    'gitlab'
);


ALTER TYPE skitapi.access_token_type OWNER TO skitadmin;

--
-- Name: auto_deploy_type; Type: TYPE; Schema: skitapi; Owner: skitadmin
--

CREATE TYPE skitapi.auto_deploy_type AS ENUM (
    'commit',
    'pull_request'
);


ALTER TYPE skitapi.auto_deploy_type OWNER TO skitadmin;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: migrations; Type: TABLE; Schema: public; Owner: skitadmin
--

CREATE TABLE public.migrations (
    migration_version integer NOT NULL,
    seed_version integer,
    dirty boolean NOT NULL
);


ALTER TABLE public.migrations OWNER TO skitadmin;

--
-- Name: access_log_stats; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.access_log_stats (
    host_name text NOT NULL,
    median_duration integer NOT NULL,
    response_status integer NOT NULL,
    number_of_responses bigint NOT NULL,
    logs_date date NOT NULL
);


ALTER TABLE skitapi.access_log_stats OWNER TO skitadmin;

--
-- Name: access_logs; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.access_logs (
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


ALTER TABLE skitapi.access_logs OWNER TO skitadmin;

--
-- Name: analytics; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.analytics (
    analytics_id bigint NOT NULL,
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


ALTER TABLE skitapi.analytics OWNER TO skitadmin;

--
-- Name: analytics_analytics_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.analytics_analytics_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.analytics_analytics_id_seq OWNER TO skitadmin;

--
-- Name: analytics_analytics_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.analytics_analytics_id_seq OWNED BY skitapi.analytics.analytics_id;


--
-- Name: analytics_referrers; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.analytics_referrers (
    aggregate_date date NOT NULL,
    domain_id bigint NOT NULL,
    referrer text NOT NULL,
    request_path text NOT NULL,
    visit_count bigint NOT NULL,
    referrer_hash bytea NOT NULL,
    request_path_hash bytea NOT NULL
);


ALTER TABLE skitapi.analytics_referrers OWNER TO skitadmin;

--
-- Name: analytics_visitors_agg_200; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.analytics_visitors_agg_200 (
    aggregate_date date NOT NULL,
    domain_id bigint NOT NULL,
    unique_visitors bigint NOT NULL,
    total_visitors bigint NOT NULL
);


ALTER TABLE skitapi.analytics_visitors_agg_200 OWNER TO skitadmin;

--
-- Name: analytics_visitors_agg_404; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.analytics_visitors_agg_404 (
    aggregate_date date NOT NULL,
    domain_id bigint NOT NULL,
    unique_visitors bigint NOT NULL,
    total_visitors bigint NOT NULL
);


ALTER TABLE skitapi.analytics_visitors_agg_404 OWNER TO skitadmin;

--
-- Name: analytics_visitors_agg_hourly_200; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.analytics_visitors_agg_hourly_200 (
    aggregate_date timestamp without time zone NOT NULL,
    domain_id bigint NOT NULL,
    unique_visitors bigint NOT NULL,
    total_visitors bigint NOT NULL
);


ALTER TABLE skitapi.analytics_visitors_agg_hourly_200 OWNER TO skitadmin;

--
-- Name: analytics_visitors_agg_hourly_404; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.analytics_visitors_agg_hourly_404 (
    aggregate_date timestamp without time zone NOT NULL,
    domain_id bigint NOT NULL,
    unique_visitors bigint NOT NULL,
    total_visitors bigint NOT NULL
);


ALTER TABLE skitapi.analytics_visitors_agg_hourly_404 OWNER TO skitadmin;

--
-- Name: analytics_visitors_by_countries; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.analytics_visitors_by_countries (
    aggregate_date date NOT NULL,
    domain_id bigint NOT NULL,
    country_iso_code text NOT NULL,
    visit_count bigint NOT NULL
);


ALTER TABLE skitapi.analytics_visitors_by_countries OWNER TO skitadmin;

--
-- Name: api_keys; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.api_keys (
    key_id integer NOT NULL,
    app_id bigint,
    env_id bigint,
    key_name text NOT NULL,
    key_value text NOT NULL,
    key_scope text NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    user_id bigint,
    team_id bigint
);


ALTER TABLE skitapi.api_keys OWNER TO skitadmin;

--
-- Name: api_keys_key_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.api_keys_key_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.api_keys_key_id_seq OWNER TO skitadmin;

--
-- Name: api_keys_key_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.api_keys_key_id_seq OWNED BY skitapi.api_keys.key_id;


--
-- Name: app_logs; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.app_logs (
    app_id bigint,
    host_name text,
    "timestamp" bigint NOT NULL,
    request_id text,
    log_label text,
    log_data text NOT NULL,
    env_id bigint,
    deployment_id bigint,
    id bigint NOT NULL
);


ALTER TABLE skitapi.app_logs OWNER TO skitadmin;

--
-- Name: app_logs_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.app_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.app_logs_id_seq OWNER TO skitadmin;

--
-- Name: app_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.app_logs_id_seq OWNED BY skitapi.app_logs.id;


--
-- Name: app_outbound_webhooks; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.app_outbound_webhooks (
    app_id bigint,
    request_headers jsonb,
    request_body text,
    request_url text NOT NULL,
    request_method text DEFAULT 'GET'::text NOT NULL,
    trigger_when text NOT NULL,
    wh_id integer NOT NULL
);


ALTER TABLE skitapi.app_outbound_webhooks OWNER TO skitadmin;

--
-- Name: app_outbound_webhooks_wh_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.app_outbound_webhooks_wh_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.app_outbound_webhooks_wh_id_seq OWNER TO skitadmin;

--
-- Name: app_outbound_webhooks_wh_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.app_outbound_webhooks_wh_id_seq OWNED BY skitapi.app_outbound_webhooks.wh_id;


--
-- Name: apps; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.apps (
    app_id integer NOT NULL,
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


ALTER TABLE skitapi.apps OWNER TO skitadmin;

--
-- Name: apps_app_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.apps_app_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.apps_app_id_seq OWNER TO skitadmin;

--
-- Name: apps_app_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.apps_app_id_seq OWNED BY skitapi.apps.app_id;


--
-- Name: apps_build_conf; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.apps_build_conf (
    app_id bigint,
    env_id integer NOT NULL,
    env_name text NOT NULL,
    build_conf jsonb,
    branch text NOT NULL,
    auto_publish boolean DEFAULT true,
    deleted_at timestamp without time zone,
    updated_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    auto_deploy_commits text,
    auto_deploy_branches text,
    auto_deploy boolean DEFAULT false,
    mailer_conf jsonb,
    auth_wall_conf jsonb,
    auth_conf bytea,
    schema_conf bytea
);


ALTER TABLE skitapi.apps_build_conf OWNER TO skitadmin;

--
-- Name: apps_build_conf_env_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.apps_build_conf_env_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.apps_build_conf_env_id_seq OWNER TO skitadmin;

--
-- Name: apps_build_conf_env_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.apps_build_conf_env_id_seq OWNED BY skitapi.apps_build_conf.env_id;


--
-- Name: audit_logs; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.audit_logs (
    audit_id bigint NOT NULL,
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


ALTER TABLE skitapi.audit_logs OWNER TO skitadmin;

--
-- Name: audit_logs_audit_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.audit_logs_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.audit_logs_audit_id_seq OWNER TO skitadmin;

--
-- Name: audit_logs_audit_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.audit_logs_audit_id_seq OWNED BY skitapi.audit_logs.audit_id;


--
-- Name: auth_wall; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.auth_wall (
    login_id bigint NOT NULL,
    login_email text NOT NULL,
    login_password text NOT NULL,
    last_login_at timestamp without time zone,
    env_id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE skitapi.auth_wall OWNER TO skitadmin;

--
-- Name: auth_wall_login_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.auth_wall_login_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.auth_wall_login_id_seq OWNER TO skitadmin;

--
-- Name: auth_wall_login_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.auth_wall_login_id_seq OWNED BY skitapi.auth_wall.login_id;


--
-- Name: deployments; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.deployments (
    deployment_id integer NOT NULL,
    app_id bigint NOT NULL,
    env_id bigint,
    github_run_id bigint,
    pull_request_number integer,
    exit_code integer,
    artifacts_deleted boolean DEFAULT false,
    auto_publish boolean,
    is_auto_deploy boolean DEFAULT false NOT NULL,
    is_fork boolean,
    is_immutable boolean,
    status_checks_passed boolean,
    api_path_prefix text,
    branch text,
    checkout_repo text,
    commit_author text,
    commit_id text,
    commit_message text,
    config_snapshot text,
    env_name text,
    error text,
    logs text,
    migrations_folder text,
    status_checks text,
    upload_result jsonb,
    build_manifest jsonb,
    webhook_event jsonb,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    deleted_at timestamp without time zone,
    stopped_at timestamp without time zone
);


ALTER TABLE skitapi.deployments OWNER TO skitadmin;

--
-- Name: deployments_deployment_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.deployments_deployment_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.deployments_deployment_id_seq OWNER TO skitadmin;

--
-- Name: deployments_deployment_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.deployments_deployment_id_seq OWNED BY skitapi.deployments.deployment_id;


--
-- Name: deployments_published; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.deployments_published (
    deployment_id bigint NOT NULL,
    env_id bigint NOT NULL,
    percentage_released numeric(4,1) DEFAULT 0 NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);


ALTER TABLE skitapi.deployments_published OWNER TO skitadmin;

--
-- Name: domains; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.domains (
    domain_id integer NOT NULL,
    app_id bigint NOT NULL,
    env_id bigint NOT NULL,
    domain_name text NOT NULL,
    domain_token text,
    domain_verified boolean DEFAULT false NOT NULL,
    domain_verified_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text),
    custom_cert_value text,
    custom_cert_key text,
    last_ping jsonb,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);


ALTER TABLE skitapi.domains OWNER TO skitadmin;

--
-- Name: domains_domain_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.domains_domain_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.domains_domain_id_seq OWNER TO skitadmin;

--
-- Name: domains_domain_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.domains_domain_id_seq OWNED BY skitapi.domains.domain_id;


--
-- Name: function_trigger_logs; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.function_trigger_logs (
    ftl_id integer NOT NULL,
    trigger_id bigint NOT NULL,
    request jsonb NOT NULL,
    response jsonb NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);


ALTER TABLE skitapi.function_trigger_logs OWNER TO skitadmin;

--
-- Name: function_trigger_logs_ftl_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.function_trigger_logs_ftl_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.function_trigger_logs_ftl_id_seq OWNER TO skitadmin;

--
-- Name: function_trigger_logs_ftl_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.function_trigger_logs_ftl_id_seq OWNED BY skitapi.function_trigger_logs.ftl_id;


--
-- Name: function_triggers; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.function_triggers (
    trigger_id integer NOT NULL,
    trigger_options jsonb NOT NULL,
    trigger_status boolean DEFAULT true NOT NULL,
    env_id bigint NOT NULL,
    cron text NOT NULL,
    next_run_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    updated_at timestamp without time zone
);


ALTER TABLE skitapi.function_triggers OWNER TO skitadmin;

--
-- Name: function_triggers_trigger_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.function_triggers_trigger_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.function_triggers_trigger_id_seq OWNER TO skitadmin;

--
-- Name: function_triggers_trigger_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.function_triggers_trigger_id_seq OWNED BY skitapi.function_triggers.trigger_id;


--
-- Name: geo_countries; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.geo_countries (
    geoname_id integer NOT NULL,
    locale_code text,
    continent_code text,
    continent_name text,
    country_iso_code text,
    country_name text,
    is_in_european_union boolean
);


ALTER TABLE skitapi.geo_countries OWNER TO skitadmin;

--
-- Name: geo_countries_geoname_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.geo_countries_geoname_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.geo_countries_geoname_id_seq OWNER TO skitadmin;

--
-- Name: geo_countries_geoname_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.geo_countries_geoname_id_seq OWNED BY skitapi.geo_countries.geoname_id;


--
-- Name: geo_ips; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.geo_ips (
    id integer NOT NULL,
    network inet,
    geoname_id integer,
    registered_country_geoname_id integer,
    represented_country_geoname_id integer,
    is_anonymous_proxy boolean,
    is_satellite_provider boolean
);


ALTER TABLE skitapi.geo_ips OWNER TO skitadmin;

--
-- Name: geo_ips_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.geo_ips_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.geo_ips_id_seq OWNER TO skitadmin;

--
-- Name: geo_ips_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.geo_ips_id_seq OWNED BY skitapi.geo_ips.id;


--
-- Name: mailer; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.mailer (
    email_id bigint NOT NULL,
    email_to text NOT NULL,
    email_from text NOT NULL,
    email_subject text NOT NULL,
    email_body text NOT NULL,
    env_id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);


ALTER TABLE skitapi.mailer OWNER TO skitadmin;

--
-- Name: mailer_email_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.mailer_email_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.mailer_email_id_seq OWNER TO skitadmin;

--
-- Name: mailer_email_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.mailer_email_id_seq OWNED BY skitapi.mailer.email_id;


--
-- Name: oauth_configs; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.oauth_configs (
    provider_id integer NOT NULL,
    provider_name text NOT NULL,
    provider_data bytea NOT NULL,
    provider_status boolean DEFAULT false NOT NULL,
    env_id bigint NOT NULL,
    app_id bigint NOT NULL
);


ALTER TABLE skitapi.oauth_configs OWNER TO skitadmin;

--
-- Name: oauth_configs_provider_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.oauth_configs_provider_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.oauth_configs_provider_id_seq OWNER TO skitadmin;

--
-- Name: oauth_configs_provider_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.oauth_configs_provider_id_seq OWNED BY skitapi.oauth_configs.provider_id;


--
-- Name: snippets; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.snippets (
    snippet_id integer NOT NULL,
    app_id bigint NOT NULL,
    env_id bigint NOT NULL,
    snippet_title text NOT NULL,
    snippet_content text NOT NULL,
    snippet_content_hash text,
    snippet_location text DEFAULT 'head'::text NOT NULL,
    snippet_rules jsonb,
    should_prepend boolean DEFAULT false NOT NULL,
    is_enabled boolean DEFAULT false NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);


ALTER TABLE skitapi.snippets OWNER TO skitadmin;

--
-- Name: snippets_snippet_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.snippets_snippet_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.snippets_snippet_id_seq OWNER TO skitadmin;

--
-- Name: snippets_snippet_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.snippets_snippet_id_seq OWNED BY skitapi.snippets.snippet_id;


--
-- Name: stormkit_config; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.stormkit_config (
    config_id integer NOT NULL,
    config_data jsonb,
    updated_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);


ALTER TABLE skitapi.stormkit_config OWNER TO skitadmin;

--
-- Name: stormkit_config_config_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.stormkit_config_config_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.stormkit_config_config_id_seq OWNER TO skitadmin;

--
-- Name: stormkit_config_config_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.stormkit_config_config_id_seq OWNED BY skitapi.stormkit_config.config_id;


--
-- Name: team_members; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.team_members (
    member_id integer NOT NULL,
    team_id bigint NOT NULL,
    user_id bigint NOT NULL,
    inviter_id bigint,
    member_role text DEFAULT 'developer'::text NOT NULL,
    membership_status boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);


ALTER TABLE skitapi.team_members OWNER TO skitadmin;

--
-- Name: team_members_member_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.team_members_member_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.team_members_member_id_seq OWNER TO skitadmin;

--
-- Name: team_members_member_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.team_members_member_id_seq OWNED BY skitapi.team_members.member_id;


--
-- Name: teams; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.teams (
    team_id integer NOT NULL,
    team_name text NOT NULL,
    team_slug text NOT NULL,
    user_id bigint NOT NULL,
    is_default boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    deleted_at timestamp without time zone
);


ALTER TABLE skitapi.teams OWNER TO skitadmin;

--
-- Name: teams_team_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.teams_team_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.teams_team_id_seq OWNER TO skitadmin;

--
-- Name: teams_team_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.teams_team_id_seq OWNED BY skitapi.teams.team_id;


--
-- Name: user_access_tokens; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.user_access_tokens (
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


ALTER TABLE skitapi.user_access_tokens OWNER TO skitadmin;

--
-- Name: user_emails; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.user_emails (
    email_id integer NOT NULL,
    user_id bigint NOT NULL,
    email text NOT NULL,
    is_primary boolean DEFAULT false NOT NULL,
    is_verified boolean DEFAULT false NOT NULL
);


ALTER TABLE skitapi.user_emails OWNER TO skitadmin;

--
-- Name: user_emails_email_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.user_emails_email_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.user_emails_email_id_seq OWNER TO skitadmin;

--
-- Name: user_emails_email_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.user_emails_email_id_seq OWNED BY skitapi.user_emails.email_id;


--
-- Name: users; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.users (
    user_id integer NOT NULL,
    first_name text,
    last_name text,
    display_name text NOT NULL,
    avatar_uri text,
    is_admin boolean DEFAULT false NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    updated_at timestamp without time zone,
    last_login_at timestamp without time zone,
    deleted_at timestamp without time zone,
    is_approved boolean,
    metadata jsonb
);


ALTER TABLE skitapi.users OWNER TO skitadmin;

--
-- Name: users_user_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.users_user_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.users_user_id_seq OWNER TO skitadmin;

--
-- Name: users_user_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.users_user_id_seq OWNED BY skitapi.users.user_id;


--
-- Name: volumes; Type: TABLE; Schema: skitapi; Owner: skitadmin
--

CREATE TABLE skitapi.volumes (
    file_id bigint NOT NULL,
    file_name text NOT NULL,
    file_path text NOT NULL,
    file_size bigint NOT NULL,
    file_metadata jsonb,
    is_public boolean NOT NULL,
    env_id bigint NOT NULL,
    updated_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);


ALTER TABLE skitapi.volumes OWNER TO skitadmin;

--
-- Name: volumes_file_id_seq; Type: SEQUENCE; Schema: skitapi; Owner: skitadmin
--

CREATE SEQUENCE skitapi.volumes_file_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE skitapi.volumes_file_id_seq OWNER TO skitadmin;

--
-- Name: volumes_file_id_seq; Type: SEQUENCE OWNED BY; Schema: skitapi; Owner: skitadmin
--

ALTER SEQUENCE skitapi.volumes_file_id_seq OWNED BY skitapi.volumes.file_id;


--
-- Name: analytics analytics_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.analytics ALTER COLUMN analytics_id SET DEFAULT nextval('skitapi.analytics_analytics_id_seq'::regclass);


--
-- Name: api_keys key_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.api_keys ALTER COLUMN key_id SET DEFAULT nextval('skitapi.api_keys_key_id_seq'::regclass);


--
-- Name: app_logs id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.app_logs ALTER COLUMN id SET DEFAULT nextval('skitapi.app_logs_id_seq'::regclass);


--
-- Name: app_outbound_webhooks wh_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.app_outbound_webhooks ALTER COLUMN wh_id SET DEFAULT nextval('skitapi.app_outbound_webhooks_wh_id_seq'::regclass);


--
-- Name: apps app_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.apps ALTER COLUMN app_id SET DEFAULT nextval('skitapi.apps_app_id_seq'::regclass);


--
-- Name: apps_build_conf env_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.apps_build_conf ALTER COLUMN env_id SET DEFAULT nextval('skitapi.apps_build_conf_env_id_seq'::regclass);


--
-- Name: audit_logs audit_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.audit_logs ALTER COLUMN audit_id SET DEFAULT nextval('skitapi.audit_logs_audit_id_seq'::regclass);


--
-- Name: auth_wall login_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.auth_wall ALTER COLUMN login_id SET DEFAULT nextval('skitapi.auth_wall_login_id_seq'::regclass);


--
-- Name: deployments deployment_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.deployments ALTER COLUMN deployment_id SET DEFAULT nextval('skitapi.deployments_deployment_id_seq'::regclass);


--
-- Name: domains domain_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.domains ALTER COLUMN domain_id SET DEFAULT nextval('skitapi.domains_domain_id_seq'::regclass);


--
-- Name: function_trigger_logs ftl_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.function_trigger_logs ALTER COLUMN ftl_id SET DEFAULT nextval('skitapi.function_trigger_logs_ftl_id_seq'::regclass);


--
-- Name: function_triggers trigger_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.function_triggers ALTER COLUMN trigger_id SET DEFAULT nextval('skitapi.function_triggers_trigger_id_seq'::regclass);


--
-- Name: geo_countries geoname_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.geo_countries ALTER COLUMN geoname_id SET DEFAULT nextval('skitapi.geo_countries_geoname_id_seq'::regclass);


--
-- Name: geo_ips id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.geo_ips ALTER COLUMN id SET DEFAULT nextval('skitapi.geo_ips_id_seq'::regclass);


--
-- Name: mailer email_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.mailer ALTER COLUMN email_id SET DEFAULT nextval('skitapi.mailer_email_id_seq'::regclass);


--
-- Name: oauth_configs provider_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.oauth_configs ALTER COLUMN provider_id SET DEFAULT nextval('skitapi.oauth_configs_provider_id_seq'::regclass);


--
-- Name: snippets snippet_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.snippets ALTER COLUMN snippet_id SET DEFAULT nextval('skitapi.snippets_snippet_id_seq'::regclass);


--
-- Name: stormkit_config config_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.stormkit_config ALTER COLUMN config_id SET DEFAULT nextval('skitapi.stormkit_config_config_id_seq'::regclass);


--
-- Name: team_members member_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.team_members ALTER COLUMN member_id SET DEFAULT nextval('skitapi.team_members_member_id_seq'::regclass);


--
-- Name: teams team_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.teams ALTER COLUMN team_id SET DEFAULT nextval('skitapi.teams_team_id_seq'::regclass);


--
-- Name: user_emails email_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.user_emails ALTER COLUMN email_id SET DEFAULT nextval('skitapi.user_emails_email_id_seq'::regclass);


--
-- Name: users user_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.users ALTER COLUMN user_id SET DEFAULT nextval('skitapi.users_user_id_seq'::regclass);


--
-- Name: volumes file_id; Type: DEFAULT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.volumes ALTER COLUMN file_id SET DEFAULT nextval('skitapi.volumes_file_id_seq'::regclass);


--
-- Name: access_log_stats access_log_stats_host_name_response_status_logs_date_key; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.access_log_stats
    ADD CONSTRAINT access_log_stats_host_name_response_status_logs_date_key UNIQUE (host_name, response_status, logs_date);


--
-- Name: analytics analytics_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.analytics
    ADD CONSTRAINT analytics_pkey PRIMARY KEY (analytics_id);


--
-- Name: analytics_referrers analytics_referrers_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.analytics_referrers
    ADD CONSTRAINT analytics_referrers_pkey PRIMARY KEY (aggregate_date, referrer_hash, request_path_hash, domain_id);


--
-- Name: analytics_visitors_agg_200 analytics_visitors_agg_200_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.analytics_visitors_agg_200
    ADD CONSTRAINT analytics_visitors_agg_200_pkey PRIMARY KEY (aggregate_date, domain_id);


--
-- Name: analytics_visitors_agg_404 analytics_visitors_agg_404_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.analytics_visitors_agg_404
    ADD CONSTRAINT analytics_visitors_agg_404_pkey PRIMARY KEY (aggregate_date, domain_id);


--
-- Name: analytics_visitors_agg_hourly_200 analytics_visitors_agg_hourly_200_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.analytics_visitors_agg_hourly_200
    ADD CONSTRAINT analytics_visitors_agg_hourly_200_pkey PRIMARY KEY (aggregate_date, domain_id);


--
-- Name: analytics_visitors_agg_hourly_404 analytics_visitors_agg_hourly_404_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.analytics_visitors_agg_hourly_404
    ADD CONSTRAINT analytics_visitors_agg_hourly_404_pkey PRIMARY KEY (aggregate_date, domain_id);


--
-- Name: analytics_visitors_by_countries analytics_visitors_by_countries_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.analytics_visitors_by_countries
    ADD CONSTRAINT analytics_visitors_by_countries_pkey PRIMARY KEY (aggregate_date, country_iso_code, domain_id);


--
-- Name: api_keys api_keys_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.api_keys
    ADD CONSTRAINT api_keys_pkey PRIMARY KEY (key_id);


--
-- Name: app_logs app_logs_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.app_logs
    ADD CONSTRAINT app_logs_pkey PRIMARY KEY (id);


--
-- Name: app_outbound_webhooks app_outbound_webhooks_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.app_outbound_webhooks
    ADD CONSTRAINT app_outbound_webhooks_pkey PRIMARY KEY (wh_id);


--
-- Name: apps_build_conf apps_build_conf_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.apps_build_conf
    ADD CONSTRAINT apps_build_conf_pkey PRIMARY KEY (env_id);


--
-- Name: apps apps_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.apps
    ADD CONSTRAINT apps_pkey PRIMARY KEY (app_id);


--
-- Name: audit_logs audit_logs_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.audit_logs
    ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (audit_id);


--
-- Name: auth_wall auth_wall_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.auth_wall
    ADD CONSTRAINT auth_wall_pkey PRIMARY KEY (login_id);


--
-- Name: deployments deployments_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.deployments
    ADD CONSTRAINT deployments_pkey PRIMARY KEY (deployment_id);


--
-- Name: domains domains_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.domains
    ADD CONSTRAINT domains_pkey PRIMARY KEY (domain_id);


--
-- Name: function_trigger_logs function_trigger_logs_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.function_trigger_logs
    ADD CONSTRAINT function_trigger_logs_pkey PRIMARY KEY (ftl_id);


--
-- Name: function_triggers function_triggers_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.function_triggers
    ADD CONSTRAINT function_triggers_pkey PRIMARY KEY (trigger_id);


--
-- Name: geo_countries geo_countries_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.geo_countries
    ADD CONSTRAINT geo_countries_pkey PRIMARY KEY (geoname_id);


--
-- Name: geo_ips geo_ips_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.geo_ips
    ADD CONSTRAINT geo_ips_pkey PRIMARY KEY (id);


--
-- Name: mailer mailer_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.mailer
    ADD CONSTRAINT mailer_pkey PRIMARY KEY (email_id);


--
-- Name: oauth_configs oauth_configs_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.oauth_configs
    ADD CONSTRAINT oauth_configs_pkey PRIMARY KEY (provider_id);


--
-- Name: snippets snippets_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.snippets
    ADD CONSTRAINT snippets_pkey PRIMARY KEY (snippet_id);


--
-- Name: stormkit_config stormkit_config_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.stormkit_config
    ADD CONSTRAINT stormkit_config_pkey PRIMARY KEY (config_id);


--
-- Name: team_members team_members_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.team_members
    ADD CONSTRAINT team_members_pkey PRIMARY KEY (member_id);


--
-- Name: teams teams_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.teams
    ADD CONSTRAINT teams_pkey PRIMARY KEY (team_id);


--
-- Name: user_access_tokens user_access_tokens_user_id_provider_key; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.user_access_tokens
    ADD CONSTRAINT user_access_tokens_user_id_provider_key UNIQUE (user_id, provider);


--
-- Name: user_emails user_emails_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.user_emails
    ADD CONSTRAINT user_emails_pkey PRIMARY KEY (email_id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (user_id);


--
-- Name: volumes volumes_file_name_env_id_key; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.volumes
    ADD CONSTRAINT volumes_file_name_env_id_key UNIQUE (file_name, env_id);


--
-- Name: volumes volumes_pkey; Type: CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.volumes
    ADD CONSTRAINT volumes_pkey PRIMARY KEY (file_id);


--
-- Name: api_keys_value_unique_key; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE UNIQUE INDEX api_keys_value_unique_key ON skitapi.api_keys USING btree (key_value);


--
-- Name: apps_build_conf_env_name_unique_key; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE UNIQUE INDEX apps_build_conf_env_name_unique_key ON skitapi.apps_build_conf USING btree (app_id, env_name) WHERE (deleted_at IS NULL);


--
-- Name: apps_display_name_unique_key; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE UNIQUE INDEX apps_display_name_unique_key ON skitapi.apps USING btree (display_name) WHERE (deleted_at IS NULL);


--
-- Name: auth_wall_env_id_login_email; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE UNIQUE INDEX auth_wall_env_id_login_email ON skitapi.auth_wall USING btree (env_id, login_email);


--
-- Name: domains_domain_name_unique_key; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE UNIQUE INDEX domains_domain_name_unique_key ON skitapi.domains USING btree (domain_name) WHERE (domain_verified IS TRUE);


--
-- Name: idx_access_log_stats_host_name; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_access_log_stats_host_name ON skitapi.access_log_stats USING btree (host_name);


--
-- Name: idx_access_logs_host_name; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_access_logs_host_name ON skitapi.access_logs USING btree (host_name);


--
-- Name: idx_analytics_referrers_hash; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_analytics_referrers_hash ON skitapi.analytics_referrers USING btree (aggregate_date, referrer_hash, request_path_hash, domain_id);


--
-- Name: idx_app_deleted_at; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_app_deleted_at ON skitapi.apps USING btree (deleted_at);


--
-- Name: idx_app_display_name; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_app_display_name ON skitapi.apps USING btree (display_name);


--
-- Name: idx_app_logs_app_id_host_name; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_app_logs_app_id_host_name ON skitapi.app_logs USING btree (app_id, host_name);


--
-- Name: idx_app_logs_label; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_app_logs_label ON skitapi.app_logs USING btree (log_label);


--
-- Name: idx_app_logs_request_id; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_app_logs_request_id ON skitapi.app_logs USING btree (request_id);


--
-- Name: idx_app_repo; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_app_repo ON skitapi.apps USING btree (lower(repo));


--
-- Name: idx_app_user_id; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_app_user_id ON skitapi.apps USING btree (user_id);


--
-- Name: idx_apps_build_conf_branch; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_apps_build_conf_branch ON skitapi.apps_build_conf USING btree (branch);


--
-- Name: idx_deployments_app_id; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_deployments_app_id ON skitapi.deployments USING btree (app_id);


--
-- Name: idx_deployments_branch_name; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_deployments_branch_name ON skitapi.deployments USING btree (branch);


--
-- Name: idx_deployments_created_at; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_deployments_created_at ON skitapi.deployments USING btree (((created_at)::date));


--
-- Name: idx_deployments_env_name; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_deployments_env_name ON skitapi.deployments USING btree (env_name);


--
-- Name: idx_deployments_published_deployment_id; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_deployments_published_deployment_id ON skitapi.deployments_published USING btree (deployment_id);


--
-- Name: idx_deployments_published_env_id; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_deployments_published_env_id ON skitapi.deployments_published USING btree (env_id);


--
-- Name: idx_geo_ips_network; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_geo_ips_network ON skitapi.geo_ips USING gist (network inet_ops);


--
-- Name: idx_next_run_at; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_next_run_at ON skitapi.function_triggers USING btree (next_run_at);


--
-- Name: idx_user_access_tokens_user_id; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE INDEX idx_user_access_tokens_user_id ON skitapi.user_access_tokens USING btree (user_id);


--
-- Name: oauth_configs_env_provider_unique_key; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE UNIQUE INDEX oauth_configs_env_provider_unique_key ON skitapi.oauth_configs USING btree (env_id, provider_name);


--
-- Name: snippets_snippet_content_hash_key; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE UNIQUE INDEX snippets_snippet_content_hash_key ON skitapi.snippets USING btree (env_id, snippet_content_hash);


--
-- Name: team_members_team_id_user_id; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE UNIQUE INDEX team_members_team_id_user_id ON skitapi.team_members USING btree (team_id, user_id);


--
-- Name: user_emails_email_unique_key; Type: INDEX; Schema: skitapi; Owner: skitadmin
--

CREATE UNIQUE INDEX user_emails_email_unique_key ON skitapi.user_emails USING btree (email) WHERE (is_verified IS TRUE);


--
-- Name: access_logs access_logs_app_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.access_logs
    ADD CONSTRAINT access_logs_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: analytics analytics_app_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.analytics
    ADD CONSTRAINT analytics_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;


--
-- Name: analytics analytics_env_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.analytics
    ADD CONSTRAINT analytics_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;


--
-- Name: api_keys api_keys_app_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.api_keys
    ADD CONSTRAINT api_keys_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;


--
-- Name: api_keys api_keys_env_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.api_keys
    ADD CONSTRAINT api_keys_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;


--
-- Name: api_keys api_keys_team_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.api_keys
    ADD CONSTRAINT api_keys_team_id_fkey FOREIGN KEY (team_id) REFERENCES skitapi.teams(team_id) ON DELETE CASCADE;


--
-- Name: api_keys api_keys_user_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.api_keys
    ADD CONSTRAINT api_keys_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;


--
-- Name: app_logs app_logs_app_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.app_logs
    ADD CONSTRAINT app_logs_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: app_outbound_webhooks app_outbound_webhooks_app_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.app_outbound_webhooks
    ADD CONSTRAINT app_outbound_webhooks_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;


--
-- Name: apps_build_conf apps_build_conf_app_id_app_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.apps_build_conf
    ADD CONSTRAINT apps_build_conf_app_id_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- Name: apps apps_user_id_user_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.apps
    ADD CONSTRAINT apps_user_id_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;


--
-- Name: audit_logs audit_logs_app_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.audit_logs
    ADD CONSTRAINT audit_logs_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;


--
-- Name: audit_logs audit_logs_env_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.audit_logs
    ADD CONSTRAINT audit_logs_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;


--
-- Name: audit_logs audit_logs_team_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.audit_logs
    ADD CONSTRAINT audit_logs_team_id_fkey FOREIGN KEY (team_id) REFERENCES skitapi.teams(team_id) ON DELETE CASCADE;


--
-- Name: audit_logs audit_logs_user_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.audit_logs
    ADD CONSTRAINT audit_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id);


--
-- Name: auth_wall auth_wall_env_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.auth_wall
    ADD CONSTRAINT auth_wall_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: deployments deployments_app_id_app_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.deployments
    ADD CONSTRAINT deployments_app_id_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;


--
-- Name: deployments deployments_env_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.deployments
    ADD CONSTRAINT deployments_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;


--
-- Name: deployments_published deployments_published_deployment_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.deployments_published
    ADD CONSTRAINT deployments_published_deployment_id_fkey FOREIGN KEY (deployment_id) REFERENCES skitapi.deployments(deployment_id) ON DELETE CASCADE;


--
-- Name: deployments_published deployments_published_env_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.deployments_published
    ADD CONSTRAINT deployments_published_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;


--
-- Name: domains domains_app_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.domains
    ADD CONSTRAINT domains_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;


--
-- Name: domains domains_env_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.domains
    ADD CONSTRAINT domains_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;


--
-- Name: app_logs fk_deployment_id; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.app_logs
    ADD CONSTRAINT fk_deployment_id FOREIGN KEY (deployment_id) REFERENCES skitapi.deployments(deployment_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: app_logs fk_env_id; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.app_logs
    ADD CONSTRAINT fk_env_id FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: function_trigger_logs function_trigger_logs_trigger_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.function_trigger_logs
    ADD CONSTRAINT function_trigger_logs_trigger_id_fkey FOREIGN KEY (trigger_id) REFERENCES skitapi.function_triggers(trigger_id) ON DELETE CASCADE;


--
-- Name: function_triggers function_triggers_env_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.function_triggers
    ADD CONSTRAINT function_triggers_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;


--
-- Name: geo_ips geo_ips_geoname_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.geo_ips
    ADD CONSTRAINT geo_ips_geoname_id_fkey FOREIGN KEY (geoname_id) REFERENCES skitapi.geo_countries(geoname_id);


--
-- Name: mailer mailer_env_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.mailer
    ADD CONSTRAINT mailer_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;


--
-- Name: oauth_configs oauth_configs_app_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.oauth_configs
    ADD CONSTRAINT oauth_configs_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;


--
-- Name: oauth_configs oauth_configs_env_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.oauth_configs
    ADD CONSTRAINT oauth_configs_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;


--
-- Name: snippets snippets_app_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.snippets
    ADD CONSTRAINT snippets_app_id_fkey FOREIGN KEY (app_id) REFERENCES skitapi.apps(app_id) ON DELETE CASCADE;


--
-- Name: snippets snippets_env_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.snippets
    ADD CONSTRAINT snippets_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;


--
-- Name: team_members team_members_inviter_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.team_members
    ADD CONSTRAINT team_members_inviter_id_fkey FOREIGN KEY (inviter_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;


--
-- Name: team_members team_members_team_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.team_members
    ADD CONSTRAINT team_members_team_id_fkey FOREIGN KEY (team_id) REFERENCES skitapi.teams(team_id) ON DELETE CASCADE;


--
-- Name: team_members team_members_user_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.team_members
    ADD CONSTRAINT team_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;


--
-- Name: teams teams_user_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.teams
    ADD CONSTRAINT teams_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;


--
-- Name: user_access_tokens user_access_tokens_user_id_users_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.user_access_tokens
    ADD CONSTRAINT user_access_tokens_user_id_users_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;


--
-- Name: user_emails user_emails_user_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.user_emails
    ADD CONSTRAINT user_emails_user_id_fkey FOREIGN KEY (user_id) REFERENCES skitapi.users(user_id) ON DELETE CASCADE;


--
-- Name: volumes volumes_env_id_fkey; Type: FK CONSTRAINT; Schema: skitapi; Owner: skitadmin
--

ALTER TABLE ONLY skitapi.volumes
    ADD CONSTRAINT volumes_env_id_fkey FOREIGN KEY (env_id) REFERENCES skitapi.apps_build_conf(env_id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--
