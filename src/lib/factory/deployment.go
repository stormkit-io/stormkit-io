package factory

import (
	"encoding/json"
	"fmt"

	"gopkg.in/guregu/null.v3"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type MockDeployment struct {
	*deploy.Deployment
	*Factory
}

func (d MockDeployment) Insert(conn databasetest.TestDB) error {
	if !d.CreatedAt.Valid {
		d.CreatedAt = utils.NewUnix()
	}

	err := conn.PrepareOrPanic(`
		INSERT INTO deployments
			(app_id, config_snapshot, branch, env_name, env_id,
			is_auto_deploy, pull_request_number,
			commit_id, commit_author, commit_message, is_fork, auto_publish,
			checkout_repo, exit_code, github_run_id,
			created_at, deleted_at, build_manifest,
			function_location, storage_location, api_location, logs, is_immutable)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23
		)
		RETURNING deployment_id, created_at`,
	).QueryRow(
		d.AppID, d.ConfigCopy, d.Branch, d.Env, d.EnvID, d.IsAutoDeploy, d.PullRequestNumber,
		d.Commit.ID, d.Commit.Author, d.Commit.Message, d.IsFork, d.ShouldPublish, d.CheckoutRepo, d.ExitCode,
		d.GithubRunID, d.CreatedAt, d.DeletedAt, d.BuildManifest,
		d.FunctionLocation, d.StorageLocation, d.APILocation, d.Logs, d.IsImmutable,
	).Scan(&d.ID, &d.CreatedAt)

	if err != nil {
		return err
	}

	updatePublishedInfo := func(envID types.ID, percentage float64) error {
		_, err = conn.PrepareOrPanic(`
				INSERT INTO deployments_published
					(env_id, deployment_id, percentage_released)
				VALUES
					($1, $2, $3)
			`).Exec(envID, d.ID, percentage)

		return err
	}

	if len(d.PublishedV2) > 0 {
		for _, p := range d.PublishedV2 {
			if err := updatePublishedInfo(p.EnvID, p.Percentage); err != nil {
				return err
			}
		}
	}

	return nil
}

func (f *Factory) MockDeployment(env *MockEnv, overwrites ...map[string]any) *MockDeployment {
	if env == nil {
		env = f.GetEnv()
	}

	conf := deploy.ConfigSnapshot{
		BuildConfig: &buildconf.BuildConf{
			BuildCmd:   "npm run build",
			DistFolder: "build",
			StatusChecks: []buildconf.StatusCheck{
				{Cmd: "npm run e2e", Name: "run e2e tests"},
			},
			Vars: map[string]string{
				"NODE_ENV": "production",
			},
		},
	}

	snapshot, err := json.Marshal(conf)

	if err != nil {
		panic(fmt.Sprintf("error while marshaling env data: %v", err))
	}

	depl := &deploy.Deployment{
		AppID:         env.AppID,
		EnvID:         env.ID,
		Env:           env.Name,
		EnvBranchName: env.Branch,
		Branch:        "main",
		ConfigCopy:    snapshot,
		IsAutoDeploy:  false,
		CheckoutRepo:  f.GetApp().Repo,
		BuildConfig:   conf.BuildConfig,
		BuildManifest: &deploy.BuildManifest{
			CDNFiles: []deploy.CDNFile{
				{Name: "index", Headers: map[string]string{"Keep-Alive": "30"}},
				{Name: "about", Headers: map[string]string{"Accept-Encoding": "None"}},
			},
			Redirects:       []deploy.Redirect{},
			FunctionHandler: "",
			APIHandler:      "",
		},
		Commit: deploy.CommitInfo{
			ID:      null.NewString("16ab41e8", true),
			Author:  null.NewString("David Lorenzo", true),
			Message: null.NewString("fix: deployment message", true),
		},
	}

	for _, o := range overwrites {
		merge(depl, o)
	}

	mock := f.newObject(MockDeployment{
		Deployment: depl,
		Factory:    f,
	}).(MockDeployment)

	mock.Insert(f.conn)

	return &mock
}

func (f *Factory) MockDeployments(times int, env *MockEnv, overwrites ...map[string]any) []*MockDeployment {
	deployments := []*MockDeployment{}

	for i := 0; i < times; i = i + 1 {
		var overwrite map[string]any

		if len(overwrites) > i {
			overwrite = overwrites[i]
		}

		deployments = append(deployments, f.MockDeployment(env, overwrite))
	}

	return deployments
}
