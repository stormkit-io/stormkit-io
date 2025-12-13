package app

import (
	"fmt"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var (
	tableApps             = "apps"
	tableMembers          = "app_members"
	tableEnvs             = "apps_build_conf"
	tableDomains          = "domains"
	tableOutboundWebhooks = "app_outbound_webhooks"
)

type statement struct {
	selectApp              string
	selectApps             string
	selectAppPrivateKey    string
	selectDeployCandidates string
	selectAppSettings      string
	insertApp              string
	updateApp              string
	updatePrivateKey       string
	deletedApps            string
	isMember               string
	markAsDeleted          string
	markArtifactsAsDeleted string
	membersCount           string
	removeDeployTrigger    string
	updateDeployTrigger    string
	selectOutboundWebhook  string
	selectOutboundWebhooks string
	insertOutboundWebhook  string
	updateOutboundWebhook  string
	deleteOutboundWebhook  string
}

var stmt = &statement{
	removeDeployTrigger: utils.QUpdate(tableApps, `app_id = $2`, "deploy_trigger"),

	selectApps: fmt.Sprintf(`
		SELECT
			a.app_id, COALESCE(a.repo, ''), a.created_at, a.user_id,
			a.display_name, a.auto_deploy, a.default_env_name, a.team_id, runtime
		FROM %s a
		WHERE
			a.team_id = $1
			AND a.deleted_at IS NULL
			:filter
		ORDER BY a.app_id ASC
		LIMIT $2 OFFSET $3;
	`, tableApps),

	selectApp: `
		SELECT
			a.app_id, COALESCE(a.repo, ''), a.user_id,
			a.created_at, a.client_id,
			a.client_secret, a.display_name, a.auto_deploy,
			a.default_env_name, a.team_id, a.runtime
		FROM
			apps a
		{{ .join }}
		WHERE {{ .where }};
	`,

	selectDeployCandidates: `
		SELECT
			a.app_id, COALESCE(a.repo, ''),
			a.created_at, a.display_name,
			a.default_env_name,
			a.auto_deploy, COALESCE(a.runtime, ''),
			a.user_id, a.team_id,
			envs.env_name, envs.env_id, envs.auto_publish,
			envs.build_conf, envs.branch,
			envs.auto_deploy_branches, envs.auto_deploy_commits,
			envs.auto_deploy, envs.schema_conf
		FROM
			apps a
		LEFT JOIN
			apps_build_conf envs ON envs.app_id = a.app_id
		WHERE
			{{ .where }} AND
			a.deleted_at IS NULL AND
			envs.auto_deploy IS TRUE AND
			envs.deleted_at IS NULL
		ORDER BY a.app_id DESC
		LIMIT 50;
	`,

	selectAppPrivateKey: fmt.Sprintf(`
		SELECT private_key FROM %s
		WHERE
			app_id = $1 AND deleted_at IS NULL;
	`, tableApps),

	insertApp: fmt.Sprintf(`
		INSERT INTO %s (
			repo, user_id, client_id, client_secret,
			private_key, display_name, is_sample_project,
			runtime, auto_deploy, team_id
		)
		VALUES (
			$1, $2, $3, $4, $5, 
			$6, $7, $8, $9, $10
		)
		RETURNING app_id, created_at;
	`, tableApps),

	updateApp: fmt.Sprintf(`
		UPDATE %s SET
			repo = $1,
			display_name = $2,
			auto_deploy = $3,
			default_env_name = $4,
			runtime = $5
		WHERE app_id = $6;
	`, tableApps),

	updatePrivateKey: fmt.Sprintf(`
		UPDATE %s SET private_key = $1 WHERE app_id = $2 AND deleted_at IS NULL;
	`, tableApps),

	markArtifactsAsDeleted: fmt.Sprintf(`
		UPDATE %s SET artifacts_deleted = TRUE WHERE app_id = $1;
	`, tableApps),

	deletedApps: fmt.Sprintf(`
		SELECT
			app_id
		FROM %s
		WHERE
			deleted_at <= NOW() - '$0 day'::INTERVAL AND
			artifacts_deleted = FALSE
		LIMIT $1;
	`, tableApps),

	isMember: fmt.Sprintf(`
		SELECT a.app_id FROM %s a
		WHERE a.app_id = $1 AND a.user_id = $2
		UNION
		SELECT m.app_id FROM %s m
		WHERE m.app_id = $3 AND m.user_id = $4
	`, tableApps, tableMembers),

	markAsDeleted: `
		WITH update_domains AS (
			UPDATE {{ .domainsTableName }} SET domain_verified = FALSE
			WHERE app_id = $1
		), mark_envs_as_deleted AS (
			UPDATE {{ .envsTableName }} SET deleted_at = NOW()
			WHERE app_id = $1
		)
		UPDATE
			{{ .tableName }}
		SET
			deleted_at = NOW()
		WHERE
			app_id = $1;
	`,

	membersCount: fmt.Sprintf(`
		SELECT COUNT(*) FROM %s
		WHERE app_id = $1
	`, tableMembers),

	selectAppSettings: fmt.Sprintf(`
		SELECT
			COALESCE(a.deploy_trigger, ''),
			array(SELECT env_name FROM %s WHERE app_id = $1) envs,
			a.runtime
		FROM %s a
		WHERE a.app_id = $2;
	`, tableEnvs, tableApps),

	updateDeployTrigger: fmt.Sprintf(`
		UPDATE %s SET deploy_trigger = $1 WHERE app_id = $2;
	`, tableApps),

	selectOutboundWebhook: fmt.Sprintf(`
		SELECT
			wh.request_headers,
			wh.request_body,
			wh.request_url,
			wh.request_method,
			wh.trigger_when,
			wh.wh_id
		FROM %s wh
		WHERE wh.app_id = $1 AND wh.wh_id = $2;
	`, tableOutboundWebhooks),

	selectOutboundWebhooks: fmt.Sprintf(`
		SELECT
			wh.request_headers,
			wh.request_body,
			wh.request_url,
			wh.request_method,
			wh.trigger_when,
			wh.wh_id
		FROM %s wh
		WHERE wh.app_id = $1
		LIMIT 10 OFFSET 0;
	`, tableOutboundWebhooks),

	insertOutboundWebhook: fmt.Sprintf(`
		INSERT INTO %s (
			app_id,
			request_headers,
			request_body,
			request_url,
			request_method,
			trigger_when
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, tableOutboundWebhooks),

	updateOutboundWebhook: fmt.Sprintf(`
		UPDATE %s SET
			request_headers = $1,
			request_body = $2,
			request_url = $3,
			request_method = $4,
			trigger_when = $5
		WHERE
			app_id = $6 AND
			wh_id = $7;
	`, tableOutboundWebhooks),

	deleteOutboundWebhook: fmt.Sprintf(`
		DELETE FROM %s WHERE app_id = $1 AND wh_id = $2;
	`, tableOutboundWebhooks),
}
