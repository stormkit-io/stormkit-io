package jobs

import "fmt"

var (
	tableDeploys = "deployments"
	tableEnvs    = "apps_build_conf"
	tableLogs    = "app_logs"
)

type statement struct {
	markDeploymentsSoftDeleted      string
	markDeploymentArtifactsDeleted  string
	markStaleAppsAndEnvsSoftDeleted string
	deleteStaleEnvironments         string
	selectTimedOutDeployments       string
	selectOldOrDeletedDeployments   string
	removeOldLogs                   string
	removeOldAnalytics              string
	removeOldAnalyticsEvents        string
	syncAnalyticsVisitors           string
	syncAnalyticsReferrers          string
	syncAnalyticsCountries          string
	syncAnalyticsEvents             string
	syncAnalyticsPaths              string
	syncAnalyticsPathCountries      string
	syncAnalyticsCampaigns          string
	syncAnalyticsChannels           string
	selectUserIDsWithoutAPIKeys     string
	selectStaleSchemas              string
	selectStaleVolumeEnvs           string
	clearSchemaConf                 string
	selectAccessLogPartitions       string
	selectAuthEnabledEnvs           string
}

var stmt = &statement{
	markDeploymentsSoftDeleted: fmt.Sprintf(`
		UPDATE %s SET deleted_at = NOW()
		WHERE env_id IN (
			SELECT
				envs.env_id
			FROM
				%s envs
			WHERE
				envs.deleted_at IS NOT NULL
			LIMIT 50);
	`, tableDeploys, tableEnvs),
	removeOldLogs: fmt.Sprintf(`
       DELETE FROM
         %s
       WHERE
         to_timestamp(timestamp)::date < now() - interval '30 days'
	`, tableLogs),
	removeOldAnalytics: `
		DELETE FROM analytics
		WHERE ctid IN (
			SELECT ctid
			FROM analytics
			WHERE request_timestamp < now() - make_interval(days => $1)
			LIMIT $2
		)
	`,
	removeOldAnalyticsEvents: `
		DELETE FROM analytics_events
		WHERE ctid IN (
			SELECT ctid
			FROM analytics_events
			WHERE event_ts < now() - make_interval(days => $1)
			LIMIT $2
		)
	`,
	selectAccessLogPartitions: `
		SELECT c.relname
		FROM pg_inherits i
		JOIN pg_class c ON c.oid = i.inhrelid
		JOIN pg_class p ON p.oid = i.inhparent
		WHERE p.relname = 'request_logs';
	`,
	deleteStaleEnvironments: fmt.Sprintf(`
		DELETE FROM %s
		WHERE env_id IN (
			SELECT
				env.env_id
			FROM
				%s env
			JOIN %s d ON env.env_id = d.env_id
			WHERE
				env.deleted_at IS NOT NULL
				AND d.artifacts_deleted = TRUE
				AND d.deleted_at IS NOT NULL
			LIMIT 50
	);`, tableEnvs, tableEnvs, tableDeploys),

	selectTimedOutDeployments: `
		SELECT
			d.deployment_id
		FROM
			deployments d
		WHERE
			d.created_at < NOW() AT TIME ZONE 'UTC' - INTERVAL '15 minutes' AND
			d.exit_code IS NULL
		LIMIT 100;
	`,

	selectOldOrDeletedDeployments: `
		SELECT
			d.deployment_id, d.app_id, d.upload_result
		FROM
			deployments d
		LEFT JOIN
			deployments_published dp ON dp.deployment_id = d.deployment_id
		WHERE
			(
				d.created_at < NOW() AT TIME ZONE 'UTC' - INTERVAL '{{ .days }} days' OR 
				d.deleted_at IS NOT NULL
			)
			AND
			dp.deployment_id IS NULL AND
			d.artifacts_deleted != TRUE
		LIMIT $1;
	`,

	markDeploymentArtifactsDeleted: `
		UPDATE
			deployments
		SET
			deleted_at = COALESCE(deleted_at, NOW()),
			artifacts_deleted = TRUE
		WHERE
			deployment_id = ANY($1);
	`,

	markStaleAppsAndEnvsSoftDeleted: `
		WITH
			updated_apps AS (
				UPDATE
					apps
				SET
					deleted_at = NOW()
				FROM
					teams
				WHERE
					teams.deleted_at IS NOT NULL AND
					apps.team_id = teams.team_id AND
					apps.deleted_at IS NULL
				RETURNING apps.app_id
			)
		UPDATE
			apps_build_conf
		SET
			deleted_at = NOW()
		FROM
			updated_apps
		WHERE
			apps_build_conf.app_id = updated_apps.app_id AND
			deleted_at IS NULL;
	`,

	syncAnalyticsVisitors: `
		INSERT INTO {{ .tableName }}
			(aggregate_date, domain_id, unique_visitors, total_visitors)
			SELECT
				{{ .column }} as agg_date, domain_id,
				COUNT(DISTINCT a.visitor_ip) AS unique_visitors,
				COUNT(a.visitor_ip) as total_visitors
			FROM
				analytics a
			WHERE
				a.response_code = ANY($1) AND
				a.request_timestamp >= {{ .interval }}
			GROUP BY
				agg_date, a.domain_id
			ORDER BY
		 		agg_date DESC
		ON CONFLICT
			(aggregate_date, domain_id)
		DO UPDATE SET
			unique_visitors = EXCLUDED.unique_visitors,
    		total_visitors = EXCLUDED.total_visitors`,

	syncAnalyticsReferrers: `
		INSERT INTO analytics_referrers
			(aggregate_date, referrer, referrer_hash, request_path, request_path_hash, domain_id, visit_count)
            SELECT
                DATE(a.request_timestamp) as req_date,
				regexp_replace(
					regexp_replace(
						COALESCE(a.referrer, ''),
						'^https?://(www\.)?', ''
					),
					'/$', ''
				) as referrer_domain,
				decode(md5(
					regexp_replace(
						regexp_replace(
							COALESCE(a.referrer, ''),
							'^https?://(www\.)?', ''
						),
						'/$', ''
					)
				), 'hex') as referrer_hash,
				a.request_path,
				decode(md5(a.request_path), 'hex') as request_path_hash,
				a.domain_id,
				COUNT(a.domain_id) AS total_count
            FROM
				analytics a
            WHERE
				a.response_code IN (200, 304) AND
				a.request_timestamp >= current_date - interval '1 days'
            GROUP BY
				req_date, referrer_domain,
				a.request_path, a.domain_id
		ON CONFLICT
			(aggregate_date, referrer_hash, request_path_hash, domain_id)
		DO UPDATE SET
			visit_count = EXCLUDED.visit_count;
	`,

	syncAnalyticsCountries: `
		INSERT INTO analytics_visitors_by_countries
			(aggregate_date, country_iso_code, domain_id, visit_count)
			SELECT
				DATE(a.request_timestamp) as agg_date,
				a.country_iso_code,
				a.domain_id,
				COUNT(DISTINCT a.visitor_ip) AS count
			FROM analytics a
			WHERE
				a.response_code IN (200, 304) AND
				a.country_iso_code IS NOT NULL AND
				a.request_timestamp >= current_date - interval '1 days'
			GROUP BY
				a.country_iso_code, agg_date, domain_id
		ON CONFLICT
			(aggregate_date, country_iso_code, domain_id)
		DO UPDATE SET
			visit_count = EXCLUDED.visit_count;
	`,

	syncAnalyticsEvents: `
		INSERT INTO analytics_events_agg
			(aggregate_date, domain_id, event_name, total_count, unique_actors)
			SELECT
				DATE(e.event_ts) AS agg_date,
				e.domain_id,
				e.event_name,
				COUNT(*) AS total_count,
				COUNT(DISTINCT e.visitor_id) AS unique_actors
			FROM analytics_events e
			WHERE
				e.domain_id IS NOT NULL AND
				e.event_ts >= current_date - interval '1 days'
			GROUP BY
				agg_date, e.domain_id, e.event_name
		ON CONFLICT
			(aggregate_date, domain_id, event_name)
		DO UPDATE SET
			total_count = EXCLUDED.total_count,
			unique_actors = EXCLUDED.unique_actors;
	`,

	// The path rollups recompute whole days in [$1, $2), so a re-run replaces a
	// day's rows instead of adding to them.
	syncAnalyticsPaths: `
		INSERT INTO analytics_paths_agg
			(aggregate_date, domain_id, request_path, request_path_hash,
			 unique_visitors, page_views, server_views, client_views)
			SELECT
				DATE(a.request_timestamp) AS agg_date,
				a.domain_id,
				a.request_path,
				decode(md5(a.request_path), 'hex'),
				COUNT(DISTINCT a.visitor_ip),
				COUNT(*),
				COUNT(*) FILTER (WHERE COALESCE(a.source, 'server') = 'server'),
				COUNT(*) FILTER (WHERE a.source = 'client')
			FROM
				analytics a
			WHERE
				a.response_code IN (200, 304) AND
				a.domain_id IS NOT NULL AND
				a.request_timestamp >= $1::date AND
				a.request_timestamp < $2::date
			GROUP BY
				agg_date, a.domain_id, a.request_path
		ON CONFLICT
			(domain_id, aggregate_date, request_path_hash)
		DO UPDATE SET
			unique_visitors = EXCLUDED.unique_visitors,
			page_views = EXCLUDED.page_views,
			server_views = EXCLUDED.server_views,
			client_views = EXCLUDED.client_views;
	`,

	syncAnalyticsPathCountries: `
		INSERT INTO analytics_path_countries_agg
			(aggregate_date, domain_id, request_path, request_path_hash,
			 country_iso_code, unique_visitors)
			SELECT
				DATE(a.request_timestamp) AS agg_date,
				a.domain_id,
				a.request_path,
				decode(md5(a.request_path), 'hex'),
				a.country_iso_code,
				COUNT(DISTINCT a.visitor_ip)
			FROM
				analytics a
			WHERE
				a.response_code IN (200, 304) AND
				a.domain_id IS NOT NULL AND
				a.country_iso_code IS NOT NULL AND
				a.request_timestamp >= $1::date AND
				a.request_timestamp < $2::date
			GROUP BY
				agg_date, a.domain_id, a.request_path, a.country_iso_code
		ON CONFLICT
			(domain_id, aggregate_date, request_path_hash, country_iso_code)
		DO UPDATE SET
			unique_visitors = EXCLUDED.unique_visitors;
	`,

	syncAnalyticsCampaigns: `
		INSERT INTO analytics_campaigns_agg
			(aggregate_date, domain_id, utm_source, utm_medium, utm_campaign, ref,
			 campaign_hash, request_path, request_path_hash, unique_visitors, visits)
			SELECT
				agg_date, domain_id, utm_source, utm_medium, utm_campaign, ref,
				decode(md5(concat_ws(chr(31), utm_source, utm_medium, utm_campaign, ref)), 'hex'),
				request_path,
				decode(md5(request_path), 'hex'),
				COUNT(DISTINCT visitor_ip),
				COUNT(*)
			FROM (
				SELECT
					DATE(a.request_timestamp) AS agg_date,
					a.domain_id,
					COALESCE(a.utm_source, '') AS utm_source,
					COALESCE(a.utm_medium, '') AS utm_medium,
					COALESCE(a.utm_campaign, '') AS utm_campaign,
					COALESCE(a.ref, '') AS ref,
					a.request_path,
					a.visitor_ip
				FROM
					analytics a
				WHERE
					a.response_code IN (200, 304) AND
					a.domain_id IS NOT NULL AND
					a.request_timestamp >= $1::date AND
					a.request_timestamp < $2::date AND
					(a.utm_source IS NOT NULL OR a.utm_medium IS NOT NULL OR
					 a.utm_campaign IS NOT NULL OR a.ref IS NOT NULL)
			) c
			GROUP BY
				agg_date, domain_id, utm_source, utm_medium, utm_campaign, ref, request_path
		ON CONFLICT
			(domain_id, aggregate_date, campaign_hash, request_path_hash)
		DO UPDATE SET
			unique_visitors = EXCLUDED.unique_visitors,
			visits = EXCLUDED.visits;
	`,

	// syncAnalyticsChannels classifies each page view's referrer host once, at
	// rollup time. The host is the stored referrer without a scheme, path or
	// leading "www.". It is internal when it is one of the page's environment
	// domains or their subdomains; otherwise the pattern lists bound as $3-$6
	// (ai, email, search, social) are checked in that order.
	syncAnalyticsChannels: `
		INSERT INTO analytics_channels_agg
			(aggregate_date, domain_id, request_path, request_path_hash,
			 referrer, referrer_hash, channel, visits)
			SELECT
				g.agg_date,
				g.domain_id,
				g.request_path,
				decode(md5(g.request_path), 'hex'),
				g.host,
				decode(md5(g.host), 'hex'),
				CASE
					WHEN g.host = '' THEN 'direct'
					WHEN EXISTS (
						SELECT 1
						FROM domains d
						CROSS JOIN LATERAL (
							SELECT regexp_replace(lower(d.domain_name), '^www\.', '') AS apex
						) site
						WHERE
							d.env_id = g.env_id AND
							(g.host = site.apex OR right(g.host, length(site.apex) + 1) = '.' || site.apex)
					) THEN 'internal'
					WHEN g.host ~ ANY($3::text[]) THEN 'ai'
					WHEN g.host ~ ANY($4::text[]) THEN 'email'
					WHEN g.host ~ ANY($5::text[]) THEN 'search'
					WHEN g.host ~ ANY($6::text[]) THEN 'social'
					ELSE 'referral'
				END,
				g.visits
			FROM (
				SELECT
					DATE(a.request_timestamp) AS agg_date,
					a.domain_id,
					a.env_id,
					a.request_path,
					regexp_replace(
						split_part(regexp_replace(lower(COALESCE(a.referrer, '')), '^[a-z][a-z0-9+.-]*://', ''), '/', 1),
						'^www\.', ''
					) AS host,
					COUNT(*) AS visits
				FROM
					analytics a
				WHERE
					a.response_code IN (200, 304) AND
					a.domain_id IS NOT NULL AND
					a.request_timestamp >= $1::date AND
					a.request_timestamp < $2::date
				GROUP BY
					agg_date, a.domain_id, a.env_id, a.request_path, host
			) g
		ON CONFLICT
			(domain_id, aggregate_date, request_path_hash, referrer_hash)
		DO UPDATE SET
			channel = EXCLUDED.channel,
			visits = EXCLUDED.visits;
	`,

	selectStaleSchemas: `
		SELECT
			env_id, app_id, schema_conf
		FROM
			apps_build_conf
		WHERE
			deleted_at IS NOT NULL AND
			schema_conf IS NOT NULL
		ORDER BY
			deleted_at ASC, env_id ASC
		LIMIT 50;
	`,

	selectStaleVolumeEnvs: `
		SELECT DISTINCT
			v.env_id
		FROM
			volumes v
		JOIN
			apps_build_conf e ON e.env_id = v.env_id
		WHERE
			e.deleted_at IS NOT NULL
		LIMIT 50;
	`,

	clearSchemaConf: `
		UPDATE apps_build_conf SET schema_conf = NULL WHERE env_id = $1;
	`,

	// auth_conf is an opaque bytea blob, so its Status can't be filtered in SQL;
	// callers decode each row and skip envs where auth is disabled.
	selectAuthEnabledEnvs: `
		SELECT
			env_id, auth_conf, schema_conf
		FROM
			apps_build_conf
		WHERE
			deleted_at IS NULL AND
			auth_conf IS NOT NULL AND
			schema_conf IS NOT NULL
		ORDER BY
			env_id ASC;
	`,

	selectUserIDsWithoutAPIKeys: `
		SELECT
			u.user_id
		FROM
			users u
		LEFT JOIN
			api_keys ak ON u.user_id = ak.user_id AND ak.key_scope = 'user'
		WHERE
			u.deleted_at IS NULL AND
			ak.key_id IS NULL
		LIMIT
			100;
	`,
}
