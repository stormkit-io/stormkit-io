package deployhooks

type statement struct {
	appDetailsForHooks string
}

var stmt = &statement{
	appDetailsForHooks: `
		SELECT
			d.app_id, COALESCE(d.is_auto_deploy, false),
			COALESCE(d.pull_request_number, 0),
			COALESCE(app.repo, ''), app.display_name,
			app.user_id, COALESCE(d.auto_publish, FALSE)
		FROM deployments d
		INNER JOIN apps app ON app.app_id = d.app_id
		WHERE d.deployment_id = $1;
	`,
}
