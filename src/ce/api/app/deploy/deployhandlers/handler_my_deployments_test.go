package deployhandlers_test

import (
	"bytes"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy/deployhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type HandlerMyDeploymentsSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerMyDeploymentsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerMyDeploymentsSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerMyDeploymentsSuite) responseTemplate() (*template.Template, error) {
	fns := template.FuncMap{
		"last": func(x int, a any) bool {
			return x == reflect.ValueOf(a).Len()-1
		},
	}

	return template.New("response").Funcs(fns).Parse(`{
		"deployments": [
			{{ range $i, $record := .records }}
				{
					"id": "{{ index $record "deploymentId" }}",
					"apiPathPrefix": "",
					"appId": "{{ index $record "appId" }}",
					"branch": "main",
					"commit": {
						"author": "David Lorenzo",
						"message": "fix: deployment message",
						"sha": "16ab41e8"
					},
					"createdAt": "{{ index $record "createdAt" }}",
					"displayName": "{{ index $record "displayName" }}",
					"envId": "{{ index $record "envId" }}",
					"envName": "production",
					"error": "",
					"logs": {{ index $record "logs" }},
					"isAutoDeploy": false,
					"isAutoPublish": false,
					"repo": "github/svedova/react-minimal",
					"snapshot": {
						"env": "",
						"envId": "",
						"build": {
							"buildCmd": "npm run build",
							"distFolder": "build",
							"previewLinks": null,
							"statusChecks": [{
								"cmd": "npm run e2e",
								"name": "run e2e tests",
								"description": ""
							}],
							"vars": {
								"NODE_ENV": "production"
							}
						}
					},
					"detailsUrl": "/apps/{{ index $record "appId" }}/environments/{{ index $record "envId" }}/deployments/{{ index $record "deploymentId" }}",
					"previewUrl": "http://{{ index $record "displayName" }}--{{ index $record "deploymentId" }}.stormkit:8888",
					"status": "running",
					"stoppedAt": null,
					"stoppedManually": false,
					"uploadResult": null,
					"published": [],
					"statusChecksPassed": null,
					"statusChecks": null,
					"duration": 0
				}
				{{ if not (last $i $.records) }}, {{ end }}
			{{ end }}
		]
	}`)
}

func (s *HandlerMyDeploymentsSuite) Test_Success_Team() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	deps := s.MockDeployments(3, env, nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/my/deployments?teamId=%d", usr.DefaultTeamID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	tmpl, err := s.responseTemplate()

	s.NoError(err)

	data := []map[string]any{}

	for i := len(deps) - 1; i >= 0; i = i - 1 {
		data = append(data, map[string]any{
			"createdAt":    deps[i].CreatedAt.UnixStr(),
			"deploymentId": deps[i].ID.String(),
			"appId":        appl.ID.String(),
			"envId":        env.ID.String(),
			"displayName":  appl.DisplayName,
			"logs":         "null",
		})
	}

	var wr bytes.Buffer
	err = tmpl.Execute(&wr, map[string]any{
		"records": data,
	})

	s.NoError(err)
	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(wr.String(), response.String())
}

func (s *HandlerMyDeploymentsSuite) Test_Success_Env() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	env2 := s.MockEnv(appl, map[string]any{
		"Name": "development",
	})
	deps := s.MockDeployments(3, env, nil)
	_ = s.MockDeployments(3, env2, nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/my/deployments?envId=%s", env.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	tmpl, err := s.responseTemplate()

	s.NoError(err)

	data := []map[string]any{}

	for i := len(deps) - 1; i >= 0; i = i - 1 {
		data = append(data, map[string]any{
			"createdAt":    deps[i].CreatedAt.UnixStr(),
			"deploymentId": deps[i].ID.String(),
			"appId":        appl.ID.String(),
			"envId":        env.ID.String(),
			"displayName":  appl.DisplayName,
			"logs":         "null",
		})
	}

	var wr bytes.Buffer
	err = tmpl.Execute(&wr, map[string]any{
		"records": data,
	})

	s.NoError(err)
	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(wr.String(), response.String())
}

func (s *HandlerMyDeploymentsSuite) Test_Success_DeploymentID() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	logs := `[{"title": "hello", "message": "world"}]`
	deps := s.MockDeployments(3, env, map[string]any{
		"Logs": null.NewString(logs, true),
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/my/deployments?deploymentId=%s", deps[0].ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	tmpl, err := s.responseTemplate()

	s.NoError(err)

	data := []map[string]any{
		{
			"createdAt":    deps[0].CreatedAt.UnixStr(),
			"deploymentId": deps[0].ID.String(),
			"appId":        appl.ID.String(),
			"envId":        env.ID.String(),
			"displayName":  appl.DisplayName,
			"logs":         `[{"title": "hello", "duration": 0, "message": "world", "status": false, "payload": null}]`,
		},
	}

	var wr bytes.Buffer
	err = tmpl.Execute(&wr, map[string]any{
		"records": data,
	})

	s.NoError(err)
	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(wr.String(), response.String())
}

func TestHandlerMyDeployments(t *testing.T) {
	suite.Run(t, &HandlerMyDeploymentsSuite{})
}
