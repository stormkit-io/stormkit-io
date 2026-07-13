package app

import (
	"context"
	"database/sql"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/bitbucket"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
)

// appSetup sets up the artifacts for an app.
func appSetup(ctx context.Context, a *App, tx *sql.Tx) error {
	if err := setupHooks(ctx, a); err != nil {
		return err
	}

	if err := createBuildConf(ctx, a, tx); err != nil {
		return err
	}

	return nil
}

// createBuildConf creates the default build configuration.
func createBuildConf(ctx context.Context, a *App, tx *sql.Tx) error {
	cnf := buildconf.DefaultConfig(a.ID)

	if defaultBranch := a.DefaultBranch(); defaultBranch != "" {
		cnf.Branch = defaultBranch
	}

	if a.IsDefault {
		cnf.Data.DistFolder = "build"
		cnf.Data.BuildCmd = "npm run build"
	}

	return buildconf.NewStore().Insert(ctx, cnf, tx)
}

// setupHooks installs the webhooks & deploy keys for the repository.
func setupHooks(ctx context.Context, a *App) error {
	// This is only necessary for bitbucket.
	if !strings.HasPrefix(a.Repo, "bitbucket/") {
		return nil
	}

	cnf := admin.MustConfig()

	// No need to install webhooks / deploy keys when the DeployKey is
	// specified manually. Users can configure the steps manually:
	// They'll have to create:
	// 1. Deploy Key from <project>/admin/access-keys
	// 2. Install webhooks themselves <project>/admin/webhooks
	if cnf.IsBitbucketEnabled() && cnf.AuthConfig.Bitbucket.DeployKey != "" {
		return nil
	}

	// TODO: Once the bitbucket client is mocked through mockery
	// write a test to cover this case.
	if config.IsTest() {
		return nil
	}

	// Create a secret from the app id.
	secretStr := a.Secret()

	if secretStr == "" {
		return ErrInvalidAppSecret
	}

	client, err := bitbucket.NewClient(a.UserID)

	if err != nil {
		return err
	}

	return client.InstallWebhooks(&bitbucket.App{
		ID:         a.ID,
		Repo:       a.Repo,
		Secret:     a.Secret(),
		PrivateKey: a.PrivateKey(ctx),
	})
}
