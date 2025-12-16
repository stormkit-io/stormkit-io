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
	selectOldOrDeletedDeployments   string
	removeOldLogs                   string
	syncAnalyticsVisitors           string
	syncAnalyticsReferrers          string
	syncAnalyticsCountries          string
	selectUserIDsWithoutAPIKeys     string
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

	selectOldOrDeletedDeployments: `
		SELECT
			d.deployment_id, d.app_id, d.upload_result
		FROM
			deployments d
		LEFT JOIN
			deployments_published dp ON dp.deployment_id = d.deployment_id
		WHERE
			(
				d.created_at < NOW() - INTERVAL '{{ .days }} days' OR 
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
			(aggregate_date, referrer, request_path, domain_id, visit_count)
            SELECT
                DATE(a.request_timestamp) as req_date,
				regexp_replace(
					regexp_replace(
						COALESCE(a.referrer, ''),
						'^https?://(www\.)?', ''
					),
					'/$', ''
				) as referrer_domain,
				a.request_path,
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
			(aggregate_date, referrer, request_path, domain_id)
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
