package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type HandlerEnvAddSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerEnvAddSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerEnvAddSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerEnvAddSuite) Test_BadRequest() {
	app := s.MockApp(nil)
	key := s.MockAPIKey(app, nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/env",
		map[string]string{
			"appId":  app.ID.String(),
			"branch": "*?&",
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	expected := `{
		"errors": [
			"Branch name can only contain following characters: alphanumeric, -, +, /, ., and =",
			"Name is a required field"
		]
	}`

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerEnvAddSuite) Test_Success() {
	app := s.MockApp(nil)
	now := time.Now()
	key := s.MockAPIKey(app, nil, map[string]any{
		"Scope": apikey.SCOPE_APP,
		"EnvID": types.ID(0),
		"AppID": app.ID,
	})

	// Required for audits
	admin.SetMockLicense()

	defer func() { admin.ResetMockLicense() }()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/env",
		map[string]any{
			"branch":             "my-branch",
			"name":               "development",
			"apiFolder":          "/functions",
			"autoDeployBranches": "deploy-*",
			"buildCmd":           "npm run build:prod",
			"serverCmd":          "npm run start",
			"distFolder":         "./output/apps/my-app",
			"errorFile":          "error.html",
			"headersFile":        "headers.json",
			"previewLinks":       true,
			"envVars": map[string]string{
				"NODE_ENV": "production",
				"API_URL":  "https://api.my-app.com",
			},
			"statusChecks": []map[string]string{
				{"name": "Test", "cmd": "npm run test", "description": "Run tests"},
			},
			"redirects": []map[string]any{
				{"from": "/old", "to": "/new", "status": 301},
			},
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	env, err := buildconf.NewStore().Environment(context.Background(), app.ID, "development")

	s.NoError(err)
	s.NotNil(env)

	expected := fmt.Sprintf(`{ "envId": "%s" }`, env.ID.String())

	s.Equal(http.StatusCreated, response.Code)
	s.JSONEq(expected, response.String())

	s.Equal("my-branch", env.Branch)
	s.Equal("development", env.Name)
	s.Equal("deploy-*", env.AutoDeployBranches.ValueOrZero())
	s.Equal("", env.AutoDeployCommits.ValueOrZero())
	s.True(env.AutoDeploy)
	s.True(env.Data.PreviewLinks.ValueOrZero())
	s.Equal("/functions", env.Data.APIFolder)
	s.Equal("npm run build:prod", env.Data.BuildCmd)
	s.Equal("npm run start", env.Data.ServerCmd)
	s.Equal("./output/apps/my-app", env.Data.DistFolder)
	s.Equal("error.html", env.Data.ErrorFile)
	s.Equal("headers.json", env.Data.HeadersFile)
	s.Equal("production", env.Data.Vars["NODE_ENV"])

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		AppID: app.ID,
	})

	s.Nil(err)
	s.Len(audits, 1)
	s.GreaterOrEqual(now.Unix()+1000, audits[0].Timestamp.Unix())
	s.Equal(audit.Audit{
		ID:        audits[0].ID,
		Timestamp: audits[0].Timestamp,
		Action:    "CREATE:ENV",
		AppID:     app.ID,
		TeamID:    app.TeamID,
		TokenName: key.Name,
		Diff: &audit.Diff{
			New: audit.DiffFields{
				EnvID:   env.ID.String(),
				EnvName: env.Name,
			},
		},
	}, audits[0])
}

func TestHandlerEnvInsert(t *testing.T) {
	suite.Run(t, &HandlerEnvAddSuite{})
}
