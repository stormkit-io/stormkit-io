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

type HandlerEnvUpdateSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerEnvUpdateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerEnvUpdateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerEnvUpdateSuite) Test_Forbidden_NoAPIKey() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPut,
		"/v1/env",
		nil,
		map[string]string{},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func (s *HandlerEnvUpdateSuite) Test_Success() {
	now := time.Now()
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"Name":   "production",
		"Branch": "main",
	})
	key := s.MockAPIKey(nil, env, map[string]any{
		"Scope": apikey.SCOPE_ENV,
		"EnvID": env.ID,
	})

	admin.SetMockLicense()

	defer func() { admin.ResetMockLicense() }()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPut,
		fmt.Sprintf("/v1/env?envId=%s", env.ID),
		map[string]any{
			"branch":       "release",
			"buildCmd":     "npm run build:prod",
			"distFolder":   "dist",
			"autoPublish":  true,
			"previewLinks": true,
			"envVars": map[string]string{
				"NODE_ENV": "staging",
			},
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	updated, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)

	s.NoError(err)
	s.NotNil(updated)
	s.Equal("production", updated.Name)
	s.Equal("release", updated.Branch)
	s.Equal("npm run build:prod", updated.Data.BuildCmd)
	s.Equal("/dist", updated.Data.DistFolder)
	s.True(updated.AutoPublish)
	s.True(updated.Data.PreviewLinks.ValueOrZero())
	s.Equal("staging", updated.Data.Vars["NODE_ENV"])

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		AppID: app.ID,
	})

	s.Nil(err)
	s.Len(audits, 1)
	s.GreaterOrEqual(now.Unix()+1000, audits[0].Timestamp.Unix())
	s.Equal("UPDATE:ENV", audits[0].Action)
	s.Equal(env.ID, audits[0].EnvID)
	s.Equal(app.ID, audits[0].AppID)
	s.Equal(key.Name, audits[0].TokenName)
}

func (s *HandlerEnvUpdateSuite) Test_Success_PartialUpdate_OnlyBuildCmd() {
	app := s.MockApp(nil)
	env := s.MockEnv(app, map[string]any{
		"Name":   "staging",
		"Branch": "develop",
	})
	key := s.MockAPIKey(nil, env, map[string]any{
		"Scope": apikey.SCOPE_ENV,
		"EnvID": env.ID,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPut,
		fmt.Sprintf("/v1/env?envId=%s", env.ID),
		map[string]any{
			"buildCmd": "npm run build:staging",
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	updated, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)

	s.NoError(err)
	s.NotNil(updated)
	// Only buildCmd should change; branch and name stay untouched.
	s.Equal("staging", updated.Name)
	s.Equal("develop", updated.Branch)
	s.Equal("npm run build:staging", updated.Data.BuildCmd)
}

func (s *HandlerEnvUpdateSuite) Test_BadRequest_InvalidHeaders() {
	app := s.MockApp(nil)
	env := s.MockEnv(app)
	key := s.MockAPIKey(nil, env, map[string]any{
		"Scope": apikey.SCOPE_ENV,
		"EnvID": env.ID,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPut,
		fmt.Sprintf("/v1/env?envId=%s", env.ID),
		map[string]any{
			"headers": "not-valid-header-format",
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func (s *HandlerEnvUpdateSuite) Test_Success_AppScopedKey() {
	app := s.MockApp(nil)
	env := s.MockEnv(app, map[string]any{
		"Name":   "production",
		"Branch": "main",
	})
	key := s.MockAPIKey(app, nil, map[string]any{
		"Scope": apikey.SCOPE_APP,
		"EnvID": types.ID(0),
		"AppID": app.ID,
	})

	admin.SetMockLicense()

	defer func() { admin.ResetMockLicense() }()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPut,
		"/v1/env",
		map[string]any{
			"envId":    env.ID.String(),
			"buildCmd": "npm run build:prod",
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	updated, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)

	s.NoError(err)
	s.NotNil(updated)
	s.Equal("npm run build:prod", updated.Data.BuildCmd)
}

// Test_BadRequest_InvalidRedirects verifies that sending invalid redirect rules in a PUT /v1/env request returns 400.
func (s *HandlerEnvUpdateSuite) Test_BadRequest_InvalidRedirects() {
	app := s.MockApp(nil)
	env := s.MockEnv(app)
	key := s.MockAPIKey(nil, env, map[string]any{
		"Scope": apikey.SCOPE_ENV,
		"EnvID": env.ID,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPut,
		fmt.Sprintf("/v1/env?envId=%s", env.ID),
		map[string]any{
			"redirects": []map[string]any{
				{"from": "", "to": "/new-path"},
			},
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "'from' is required")
}

func (s *HandlerEnvUpdateSuite) Test_BadRequest_DoubleHyphens() {
	app := s.MockApp(nil)
	env := s.MockEnv(app, map[string]any{
		"Name": "development",
	})
	key := s.MockAPIKey(nil, env, map[string]any{
		"Scope": apikey.SCOPE_ENV,
		"EnvID": env.ID,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPut,
		fmt.Sprintf("/v1/env?envId=%s", env.ID),
		map[string]any{
			"name": "development--renamed",
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "Double hyphens (--) are not allowed as they are reserved for Stormkit")
}

func (s *HandlerEnvUpdateSuite) Test_BadRequest_InvalidAutoDeployBranches() {
	app := s.MockApp(nil)
	env := s.MockEnv(app, map[string]any{
		"Name": "development",
	})
	key := s.MockAPIKey(nil, env, map[string]any{
		"Scope": apikey.SCOPE_ENV,
		"EnvID": env.ID,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPut,
		fmt.Sprintf("/v1/env?envId=%s", env.ID),
		map[string]any{
			"autoDeployBranches": "(invalid-regex",
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "parsing regexp: missing closing ) in `(invalid-regex`")
}

// Test_Success_DisableAutoDeploy verifies that sending autoDeploy=false with empty
// autoDeployBranches and autoDeployCommits (as the frontend does when clearing) correctly
// disables auto deploy without the empty-string branches re-enabling it.
func (s *HandlerEnvUpdateSuite) Test_Success_DisableAutoDeploy() {
	app := s.MockApp(nil)
	env := s.MockEnv(app, map[string]any{
		"AutoDeploy": true,
	})
	key := s.MockAPIKey(nil, env, map[string]any{
		"Scope": apikey.SCOPE_ENV,
		"EnvID": env.ID,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPut,
		fmt.Sprintf("/v1/env?envId=%s", env.ID),
		map[string]any{
			"autoDeploy":         false,
			"autoDeployBranches": "",
			"autoDeployCommits":  "",
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	updated, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)

	s.NoError(err)
	s.NotNil(updated)
	s.False(updated.AutoDeploy)
}

func TestHandlerEnvUpdate(t *testing.T) {
	suite.Run(t, &HandlerEnvUpdateSuite{})
}
