package deploy_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type DeploymentModelSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *DeploymentModelSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *DeploymentModelSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *DeploymentModelSuite) Test_InsertingBuildManifest() {
	manifest := &deploy.BuildManifest{
		CDNFiles: []deploy.CDNFile{
			{
				Name:    "index.html",
				Headers: map[string]string{"ETag": "12345"},
			},
		},
	}

	mockDeploy := s.MockDeployment(nil, map[string]interface{}{
		"BuildManifest": manifest,
	})

	d, err := deploy.NewStore().DeploymentByID(context.Background(), mockDeploy.ID)

	s.Equal(manifest, d.BuildManifest)
	s.NoError(err)
}

func (s *DeploymentModelSuite) Test_EmptyBuildManifest() {
	mockDeploy := s.MockDeployment(nil, map[string]any{
		"BuildManifest": &deploy.BuildManifest{},
	})

	d, err := deploy.NewStore().DeploymentByID(context.Background(), mockDeploy.ID)

	s.NoError(err)
	s.Nil(d.BuildManifest)
}

func (s *DeploymentModelSuite) Test_RepoCloneURL() {
	d := &deploy.Deployment{}

	d.CheckoutRepo = "gitlab/stormkit-js/test"
	s.Equal("https://gitlab.com/stormkit-js/test.git", d.RepoCloneURL())

	d.CheckoutRepo = "gitlab/stormkit-js/another-scope/test"
	s.Equal("https://gitlab.com/stormkit-js/another-scope/test.git", d.RepoCloneURL())

	d.CheckoutRepo = "bitbucket/stormkit-js/another-scope/test"
	s.Equal("git@bitbucket.org:stormkit-js/another-scope/test.git", d.RepoCloneURL())

	d.CheckoutRepo = "github/stormkit-js/sample-project"
	s.Equal("https://github.com/stormkit-js/sample-project.git", d.RepoCloneURL())

	d.CheckoutRepo = "github/stormkit-js/test.github.io"
	s.Equal("https://github.com/stormkit-js/test.github.io.git", d.RepoCloneURL())
}

func (s *DeploymentModelSuite) Test_DeploymentLogs_StillRunningButLogsFinished() {
	d := &deploy.Deployment{
		Logs: null.NewString(
			"[sk-step] Clone\n"+
				"Success\n"+
				"[sk-step] [system] building finished",
			true),
	}

	expected := `[
		{
			"title": "Clone",
			"duration": 0,
			"message": "Success\n",
			"status": true,
			"payload": null
		},
		{
			"title": "deploy",
			"duration": 0,
			"message": "Deploying your application... This may take a while...",
			"status": true,
			"payload": null
		}
	]`

	b, err := json.Marshal(d.PrepareLogs(d.Logs.ValueOrZero(), false))
	s.Nil(err)
	s.JSONEq(expected, string(b))
}

func (s *DeploymentModelSuite) Test_DeploymentLogs_FinishedWithError() {
	d := &deploy.Deployment{
		Error: null.NewString("We could not detect an index.html", true),
		Logs: null.NewString(
			"[sk-step] Clone\n"+
				"Success\n"+
				"[sk-step] [system] building finished",
			true),
	}

	b, err := json.Marshal(d.PrepareLogs(d.Logs.ValueOrZero(), false))
	s.Nil(err)
	s.JSONEq(`[{"title":"Clone","duration":0,"message":"Success\n","status":true,"payload":null},{"title":"deploy","duration":0,"message":"We could not detect an index.html","status":false,"payload":null}]`, string(b))
}

func (s *DeploymentModelSuite) Test_DeploymentLogs_Result() {
	d := &deploy.Deployment{
		UploadResult: &deploy.UploadResult{
			ClientBytes:     5919,
			ServerBytes:     591919,
			ServerlessBytes: 8192,
		},
		Logs: null.NewString(
			"[sk-step] Clone\n"+
				"Success\n"+
				"[sk-step] [system] building finished",
			true),
	}

	expected := `[
		{
			"title": "Clone",
			"duration": 0,
			"message": "Success\n",
			"status": true,
			"payload": null
		},
		{
			"title": "deploy",
			"duration": 0,
			"message": "\nSuccessfully deployed client side.\nTotal bytes uploaded: 5.9kB\n\n\nSuccessfully deployed server side.\nPackage size: 591.9kB\n\n\nSuccessfully deployed api.\nPackage size: 8.2kB",
			"status": true,
			"payload": null
		}
	]`

	b, err := json.Marshal(d.PrepareLogs(d.Logs.ValueOrZero(), false))
	s.Nil(err)
	s.JSONEq(expected, string(b))
}

func (s *DeploymentModelSuite) Test_DeploymentLogs_WithMultipleSteps() {
	stoppedAt := utils.Unix{Time: time.Unix(1726054991, 0), Valid: true}

	d := &deploy.Deployment{
		UploadResult: &deploy.UploadResult{
			ClientBytes: 5919,
			ServerBytes: 591919,
		},
		ExitCode:  null.IntFrom(0),
		StoppedAt: stoppedAt,
		Logs: null.NewString(
			"[sk-step] clone [ts:1726053541]\n"+
				"Success\n"+
				"[sk-step] version [ts:1726053641]\n"+
				"v1.6.16\n"+
				"[sk-step] [system] building finished [ts:1726053751]\n"+
				"[sk-step] [system] deployment finished [ts:1726053991]",
			true),
	}

	b, err := json.Marshal(d.PrepareLogs(d.Logs.ValueOrZero(), false))
	s.Nil(err)
	s.JSONEq(`[
		{
			"title": "clone",
			"duration": 100,
			"message": "Success\n",
			"status": true,
			"payload": null
		},
		{
			"title": "version",
			"duration": 110,
			"message": "v1.6.16\n",
			"status": true,
			"payload": null
		},
		{
			"title": "deploy",
			"duration": 1240,
			"message": "\nSuccessfully deployed client side.\nTotal bytes uploaded: 5.9kB\n\n\nSuccessfully deployed server side.\nPackage size: 591.9kB\n\n",
			"status":true,
			"payload":null
		}
	]`, string(b))
}

func TestDeploymentModel(t *testing.T) {
	suite.Run(t, &DeploymentModelSuite{})
}
