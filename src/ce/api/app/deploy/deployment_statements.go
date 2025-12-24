package deploy

import (
	"fmt"
)

var (
	tableDeploys = "deployments"
)

type statement struct {
	selectDeploymentsV2      string
	selectBuildManifest      string
	insertDeployment         string
	restartDeployment        string
	updateExitCode           string
	updateCommitInfo         string
	updateLogs               string
	updateStatusChecks       string
	lockDeployment           string
	markDeploymentsAsDeleted string
	isDeploymentAlreadyBuilt string
	stopDeployment           string
	stopStatusChecks         string
	selectExitCode           string
	updateGithubRunID        string
	updateDeploymentResult   string
	markArtifactsAsDeleted   string
	publish                  string
	updateUserMetrics        string
}

var stmt = &statement{
	selectDeploymentsV2: `
		SELECT
			d.deployment_id, d.app_id,
			d.env_id, d.env_name, COALESCE(d.branch, ''),
			d.created_at deploymentDate, d.stopped_at,
			d.exit_code, d.config_snapshot,
			d.commit_id, d.commit_author, d.commit_message, d.github_run_id,
			d.error, d.is_auto_deploy,
			d.auto_publish, d.pull_request_number,
			d.build_manifest, d.api_path_prefix, d.is_immutable, d.upload_result,
			d.migrations_folder, d.status_checks_passed,
			{{ if .logs }} d.status_checks, d.logs {{ else }} '', '' {{ end }},
			a.display_name, COALESCE(a.repo, ''),
			(SELECT json_agg(
				jsonb_build_object(
					'envId', dp.env_id,
					'percentage', dp.percentage_released)) as published
			 FROM deployments_published dp
			 WHERE dp.percentage_released > 0 AND dp.deployment_id = d.deployment_id) as published
		FROM deployments d
		LEFT JOIN apps a ON a.app_id = d.app_id
		{{ .joins }}
		WHERE
			d.deleted_at IS NULL AND 
			a.deleted_at IS NULL
			{{ if .where }} AND {{ end }}
			{{ .where }}
		ORDER BY d.deployment_id DESC
		LIMIT {{ or .limit 25 }} OFFSET {{ or .offset 0 }};
	`,

	selectBuildManifest: fmt.Sprintf(`
		SELECT
		    d.build_manifest
		FROM %s d
		WHERE
			d.deployment_id = $1 AND
			d.app_id = $2
	`, tableDeploys),

	insertDeployment: `
		INSERT INTO deployments (
			app_id, config_snapshot, branch, env_name, env_id,
			is_auto_deploy, pull_request_number,
			commit_id, is_fork, auto_publish, checkout_repo,
			api_path_prefix, webhook_event, commit_author, migrations_folder
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15
		)
		RETURNING
			deployment_id,
			created_at;
	`,

	restartDeployment: `
		UPDATE deployments SET
			created_at = NOW() AT TIME ZONE 'UTC',
			stopped_at = NULL,
			config_snapshot = NULL,
			exit_code = NULL,
			logs = NULL,
			status_checks = NULL,
			status_checks_passed = NULL,
			is_immutable = false
		WHERE
			deployment_id = $1;
	`,

	updateExitCode: `
		UPDATE deployments SET
			exit_code = $1,
			stopped_at = NOW() AT TIME ZONE 'UTC'
		WHERE
			deployment_id = $2 AND
			exit_code IS NULL AND
			is_immutable IS NOT TRUE;
	`,

	updateCommitInfo: fmt.Sprintf(`
		UPDATE %s SET
			commit_id = $1,
			commit_author = $2,
			commit_message = $3
		WHERE
			deployment_id = $4 AND
			exit_code IS NULL AND 
			is_immutable IS NOT TRUE;
	`, tableDeploys),

	updateLogs: `
		UPDATE deployments SET logs = $1 WHERE deployment_id = $2 AND is_immutable IS NOT TRUE;
	`,

	updateStatusChecks: `
		UPDATE deployments SET status_checks = $1 WHERE deployment_id = $2 AND exit_code = 0 AND is_immutable IS NOT TRUE;
	`,

	lockDeployment: `
		UPDATE deployments SET
			is_immutable = TRUE,
			status_checks_passed = $1,
			stopped_at = NOW() AT TIME ZONE 'UTC'
		WHERE
			deployment_id = $2 AND
			is_immutable IS NOT TRUE;
	`,

	markDeploymentsAsDeleted: fmt.Sprintf(`
		UPDATE %s
		SET
			deleted_at = NOW() AT TIME ZONE 'UTC',
			exit_code = COALESCE(exit_code, -1)
		WHERE
			deployment_id = ANY($1) AND
			deleted_at IS NULL;
	`, tableDeploys),

	isDeploymentAlreadyBuilt: `
		SELECT COUNT(*) FROM deployments d WHERE d.commit_id = $1;
	`,

	stopDeployment: `
		UPDATE deployments
		SET
			stopped_at = NOW() AT TIME ZONE 'UTC',
			exit_code = -1
		WHERE
			deployment_id = $1 AND
			exit_code IS NULL;
	`,

	stopStatusChecks: `
		UPDATE deployments
		SET
			status_checks_passed = FALSE
		WHERE
			deployment_id = $1 AND
			status_checks_passed IS NULL;
	`,

	selectExitCode: fmt.Sprintf(`
		SELECT exit_code FROM %s WHERE deployment_id = $1;
	`, tableDeploys),

	updateGithubRunID: fmt.Sprintf(`
		UPDATE %s SET github_run_id = $1 WHERE deployment_id = $2 AND github_run_id IS NULL;
	`, tableDeploys),

	updateDeploymentResult: `
		UPDATE deployments
		SET
			upload_result = $1,
			error = $2,
			exit_code = $3,
			build_manifest = $4,
			logs = COALESCE(logs, '') || $5,
			stopped_at = NOW() AT TIME ZONE 'UTC'
		WHERE
			deployment_id = $6
		RETURNING
			stopped_at;
	`,

	markArtifactsAsDeleted: fmt.Sprintf(`
		UPDATE %s
		SET
			deleted_at = COALESCE(deleted_at, NOW() AT TIME ZONE 'UTC'),
			artifacts_deleted = TRUE,
			logs = NULL,
			commit_author = NULL,
			commit_message = NULL,
			config_snapshot = NULL,
			build_manifest = NULL
		WHERE deployment_id = ANY($1);
	`, tableDeploys),

	publish: `
		WITH delete_published AS (
			DELETE FROM deployments_published WHERE env_id = ANY({{ .envIDsParam }})
		),
		update_ts AS (
			UPDATE apps_build_conf e SET updated_at = NOW()
			WHERE e.env_id = ANY({{ .envIDsParam }})
		)
		INSERT INTO deployments_published
			(env_id, deployment_id, percentage_released)
		VALUES
			{{ generateValues 3 (len .records) }};
	`,

	updateUserMetrics: `
		WITH owner_user AS (
			SELECT
				t.user_id,
				extract(YEAR FROM CURRENT_TIMESTAMP AT TIME ZONE 'UTC')::INTEGER AS year,
				extract(MONTH FROM CURRENT_TIMESTAMP AT TIME ZONE 'UTC')::INTEGER AS month,
				ceil(ABS(extract(epoch FROM (d.stopped_at - d.created_at)) / 60))::INTEGER AS build_minutes,
				coalesce(upload_result->>'serverlessBytes', '0')::bigint + coalesce(upload_result->>'serverBytes', '0')::bigint + coalesce(upload_result->>'clientBytes', '0')::bigint as storage_used
			FROM deployments d
			JOIN apps a ON a.app_id = d.app_id
			JOIN teams t ON t.team_id = a.team_id
			WHERE d.deployment_id = $1
		)
		INSERT INTO user_metrics (user_id, year, month, build_minutes, storage_bytes)
		SELECT 
			ou.user_id, 
			ou.year, 
			ou.month, 
			ou.build_minutes,
			ou.storage_used
		FROM owner_user ou
		ON CONFLICT (user_id, year, month) 
		DO UPDATE SET 
			build_minutes = user_metrics.build_minutes + EXCLUDED.build_minutes,
			storage_bytes = user_metrics.storage_bytes + EXCLUDED.storage_bytes;
	`,
}
