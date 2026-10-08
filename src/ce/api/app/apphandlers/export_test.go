package apphandlers

// SetRunWebhookDeploys replaces how webhook deployments are scheduled. It
// returns a function that restores the default.
func SetRunWebhookDeploys(run func(fn func())) func() {
	original := runWebhookDeploys
	runWebhookDeploys = run

	return func() { runWebhookDeploys = original }
}

// RunWebhookDeploysSync makes webhooks create their deployments before the
// response is returned. It returns a function that restores the default.
func RunWebhookDeploysSync() func() {
	return SetRunWebhookDeploys(func(fn func()) { fn() })
}

// SetCreateGithubStatus replaces how GitHub commit statuses are posted. It
// returns a function that restores the default.
func SetCreateGithubStatus(fn func(repo, branch, url, status string) error) func() {
	original := createGithubStatus
	createGithubStatus = fn

	return func() { createGithubStatus = original }
}
