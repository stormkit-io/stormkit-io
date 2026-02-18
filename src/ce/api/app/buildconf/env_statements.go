package buildconf

type statement struct {
	selectByEnvID       string
	selectByAppID       string
	selectByAppIDAndEnv string
	markAsDeleted       string
	insertConfig        string
	updateConfig        string
	isMember            string
	saveSchemaConf      string
}

var stmt = &statement{
	selectByEnvID: `
		SELECT
			e.env_id, e.env_name, e.app_id, e.build_conf,
			e.auto_publish, e.branch, e.auto_deploy, e.auto_deploy_branches,
			e.auto_deploy_commits, e.updated_at, e.schema_conf, e.auth_conf,
			e.mailer_conf
		FROM
			apps_build_conf e
		WHERE
			env_id = $1 AND deleted_at IS NULL;
	`,

	selectByAppID: `
		SELECT
			e.env_id, e.env_name, e.app_id, e.build_conf, e.auto_publish,
			e.branch, e.auto_deploy, e.auto_deploy_branches, e.auto_deploy_commits,
			d.created_at, d.deployment_id, d.exit_code, (
				SELECT json_agg(
					json_build_object(
						'percentage', dp.percentage_released,
						'deploymentId', d2.deployment_id::text,
						'branch', d2.branch,
						'commitSha', d2.commit_id,
						'commitMessage', d2.commit_message,
						'commitAuthor', d2.commit_author 
					)
				) published
				FROM deployments_published dp
				LEFT JOIN deployments d2 ON d2.deployment_id = dp.deployment_id
				WHERE dp.env_id = e.env_id
			) published
		FROM apps_build_conf e
		LEFT JOIN LATERAL (
			SELECT
				d.created_at, d.deployment_id, d.exit_code
			FROM deployments d
			WHERE
				d.env_id = e.env_id AND
				d.deleted_at IS NULL
			ORDER BY
				d.created_at DESC
			LIMIT 1
		) d on true
		WHERE
			e.app_id = $1 AND
			e.deleted_at IS NULL
		ORDER BY e.env_id ASC
		LIMIT 50;
	`,

	selectByAppIDAndEnv: `
		SELECT
			e.env_id, e.env_name, e.app_id, e.build_conf, e.auto_publish,
			e.branch, e.auto_deploy, e.auto_deploy_branches, e.auto_deploy_commits,
			e.updated_at, e.schema_conf, e.auth_conf, e.mailer_conf
		FROM apps_build_conf e
		WHERE
			app_id = $1 AND deleted_at IS NULL AND LOWER(env_name) = LOWER($2);
	`,

	markAsDeleted: `
		WITH update_domains AS (
			UPDATE {{ .domainsTableName }} SET domain_verified = FALSE
			WHERE env_id = $1
		)
		UPDATE
			{{ .tableName }}
		SET
			deleted_at = NOW()
		WHERE
			env_id = $1;
	`,

	insertConfig: `
		INSERT INTO apps_build_conf
			(app_id, env_name, branch, build_conf, auto_publish, auto_deploy, auto_deploy_branches, auto_deploy_commits)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING env_id;
	`,

	updateConfig: `
		UPDATE
			apps_build_conf
		SET
			env_name = $1,
			branch = $2,
			build_conf = $3,
			auto_publish = $4,
			auto_deploy = $5,
			auto_deploy_branches = $6,
			auto_deploy_commits = $7
		WHERE
			env_id = $8;
	`,

	isMember: `
		SELECT COUNT(*) FROM apps_build_conf e
		LEFT JOIN apps a ON a.app_id = e.app_id
		LEFT JOIN team_members tm ON tm.team_id = a.team_id
		WHERE
			e.env_id = $1 AND
			tm.user_id = $2 AND
			tm.membership_status IS TRUE
		LIMIT 1;
	`,

	saveSchemaConf: `
		UPDATE
			apps_build_conf
		SET
			schema_conf = $1
		WHERE
			env_id = $2;
	`,
}
