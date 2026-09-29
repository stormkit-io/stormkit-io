package publicapiv1_test

import (
	"bytes"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"net/http"
	"testing"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
)

type HandlerAppConfSuite struct {
	suite.Suite
	*factory.Factory

	conn     databasetest.TestDB
	mockMise *mocks.MiseInterface
}

func (s *HandlerAppConfSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockMise = &mocks.MiseInterface{}
	mise.DefaultMise = s.mockMise
}

func (s *HandlerAppConfSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	mise.DefaultMise = nil
}

func (s *HandlerAppConfSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr, map[string]any{"DisplayName": "sample-project"})
	env := s.MockEnv(app)
	key := s.MockAPIKey(app, env, map[string]any{
		"UserID": usr.ID,
	})
	dep := s.MockDeployment(env, map[string]any{
		"Published": deploy.PublishedInfo{
			{EnvID: env.ID},
		},
	})

	s.mockMise.On("BinPaths", mock.Anything).Return(map[string]string{"MISE_GO_PATH": "my-path"}, nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/app/config?hostName=sample-project.stormkit:8888",
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	expectedTemplate := `{
		"configs": [{
			"domains": null,
			"apiPathPrefix": "/api",
			"isPublished": true,
			"updatedAt": null,
			"staticFiles": {
				"/about": { "fileName": "about", "headers": { "accept-encoding": "None", "content-type": "text/html; charset=utf-8" }},
				"/index": { "fileName": "index", "headers": { "keep-alive": "30", "content-type": "text/html; charset=utf-8" }}
			},
			"deploymentId": "{{.DeploymentID}}",
			"appId": "{{ .AppID }}",
			"envId": "{{ .EnvID }}",
			"isEnterprise": true,
			"billingUserId": "1",
			"envVariables": {
				"NODE_ENV": "production",
				"MISE_GO_PATH": "my-path",
				"SK_APP_ID": "{{ .AppID }}",
				"SK_DEPLOYMENT_ID": "{{ .DeploymentID }}",
				"SK_DEPLOYMENT_URL": "http://sample-project--{{ .DeploymentID }}.stormkit:8888",
				"SK_ENV": "production",
				"SK_ENV_ID": "{{ .EnvID }}",
				"SK_ENV_URL": "http://sample-project.stormkit:8888",
				"STORMKIT": "true"
			}
		}]
	}`

	tmpl := template.Must(template.New("expected").Parse(expectedTemplate))
	var buf bytes.Buffer

	err := tmpl.Execute(&buf, map[string]string{
		"DeploymentID": dep.ID.String(),
		"AppID":        app.ID.String(),
		"EnvID":        env.ID.String(),
	})

	s.Require().NoError(err)
	expected := buf.String()

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerAppConfSuite) Test_NoContent() {
	usr := s.MockUser()
	app := s.MockApp(usr, map[string]any{"DisplayName": "sample-project"})
	key := s.MockAPIKey(app, nil, map[string]any{
		"UserID": usr.ID,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/app/config?hostName=sample-project.stormkit:8888",
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	expected := `{ "error": "Config is not found. Did you publish your deployment?" }`

	s.Equal(http.StatusNoContent, response.Code)
	s.JSONEq(expected, response.String())
}

// Test_OtherAppsHost verifies that a key of one app cannot read the config,
// including the environment variables, of another app's host name.
func (s *HandlerAppConfSuite) Test_OtherAppsHost() {
	victim := s.MockApp(s.MockUser(), map[string]any{"DisplayName": "sample-project"})
	victimEnv := s.MockEnv(victim)
	s.MockDeployment(victimEnv, map[string]any{
		"Published": deploy.PublishedInfo{
			{EnvID: victimEnv.ID},
		},
	})

	attacker := s.MockUser()
	attackerApp := s.MockApp(attacker, map[string]any{"DisplayName": "attacker-project"})
	key := s.MockAPIKey(attackerApp, nil, map[string]any{
		"UserID": attacker.ID,
		"EnvID":  types.ID(0),
		"Scope":  apikey.SCOPE_APP,
	})

	s.mockMise.On("BinPaths", mock.Anything).Return(map[string]string{}, nil).Maybe()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/app/config?hostName=sample-project.stormkit:8888",
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusNoContent, response.Code)
	s.NotContains(response.String(), "envVariables")
}

func TestHandlerAppConf(t *testing.T) {
	suite.Run(t, &HandlerAppConfSuite{})
}
