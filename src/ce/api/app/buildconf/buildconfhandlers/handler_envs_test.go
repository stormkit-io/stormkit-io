package buildconfhandlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/buildconfhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"

	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerEnvsGetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerEnvsGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerEnvsGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerEnvsGetSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"AutoDeploy": false,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(buildconfhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/app/%d/envs", app.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	str := response.String()

	s.Equal(http.StatusOK, response.Code)

	expected := fmt.Sprintf(`{
		"envs": [
		  {
			"env": "production",
			"branch": "main",
			"build": {
			  "distFolder": "build",
			  "buildCmd": "npm run build",
			  "previewLinks": null,
			  "vars": {
				"NODE_ENV": "production"
			  }
			},
			"autoDeploy": false,
			"autoPublish": false,
			"autoDeployBranches": null,
			"autoDeployCommits": null,
			"id": "%s",
			"appId": "%s",
			"domain": {
			  "verified": false
			},
			"preview": "http://%s.stormkit:8888"
		  }
		]
	  }`,
		env.ID.String(),
		app.ID.String(),
		app.DisplayName,
	)

	s.JSONEq(expected, str)
}

func (s *HandlerEnvsGetSuite) Test_ShowsPublished() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(nil, map[string]any{
		"AutoDeploy": false,
	})

	d := s.MockDeployment(env, map[string]any{
		"StorageLocation":  null.NewString("aws:s3-bucket-name/s3-key-prefix", true),
		"FunctionLocation": null.NewString("aws:arn:aws:lambda:eu-central-1::function:my-lambda-name/35", true),
		"Published": deploy.PublishedInfo{
			{Percentage: 100, EnvID: env.ID},
		},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(buildconfhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/app/%d/envs", app.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	str := response.String()

	s.Equal(http.StatusOK, response.Code)
	var aq buildconfhandlers.HandlerEnvsResponse
	s.NoError(json.Unmarshal([]byte(str), &aq))

	s.Equal(aq.Envs[0].Published, []*buildconf.PublishedInfo{
		{
			Percentage:    100,
			DeploymentID:  d.ID,
			Branch:        d.Branch,
			CommitAuthor:  d.Commit.Author,
			CommitSha:     d.Commit.ID,
			CommitMessage: d.Commit.Message,
		},
	})
}

func TestHandlerEnvs(t *testing.T) {
	suite.Run(t, &HandlerEnvsGetSuite{})
}
