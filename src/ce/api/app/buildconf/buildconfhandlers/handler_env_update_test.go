package buildconfhandlers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/buildconfhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerEnvUpdateSuite struct {
	suite.Suite
	*factory.Factory

	conn             databasetest.TestDB
	mockCacheService *mocks.CacheInterface
}

func (s *HandlerEnvUpdateSuite) SetupSuite() {
	s.mockCacheService = &mocks.CacheInterface{}
}

func (s *HandlerEnvUpdateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	appcache.DefaultCacheService = s.mockCacheService
}

func (s *HandlerEnvUpdateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	appcache.DefaultCacheService = nil
}

func (s *HandlerEnvUpdateSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{"AutoDeploy": false})

	s.mockCacheService.On("Reset", env.ID).Return(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(buildconfhandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/app/env",
		map[string]any{
			"appId":       app.ID.String(),
			"id":          env.ID.String(),
			"branch":      "live",
			"name":        "production",
			"autoPublish": false,
			"autoDeploy":  true,
			"build": map[string]any{
				"previewLinks":  false,
				"installCmd":    "pnpm install",
				"serverCmd":     "node index.js",
				"serverFolder":  "/dist",
				"errorFile":     "/index.html",
				"headers":       "/*\nCache-Control:no-cache",
				"apiFolder":     "/my/api/path",
				"apiPathPrefix": "/api/",
				"statusChecks": []map[string]string{
					{"name": "Header rules", "cmd": "npm run test:headers", "description": "Test header rules in deployment."},
					{"name": "E2E", "cmd": "npm run test:e2e", "description": "Test e2e changes."},
				},
				"redirects": []map[string]string{
					{"from": "/path", "to": "/new-path"},
				},
				"vars": map[string]string{
					"NODE_ENV": "production",
				},
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	e, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)

	s.NoError(err)
	s.Equal(env.ID, e.ID)
	s.Equal(env.Name, e.Name)
	s.Equal("production", e.Env)
	s.False(e.AutoPublish)
	s.True(e.AutoDeploy)
	s.False(e.Data.PreviewLinks.ValueOrZero())
	s.True(e.Data.PreviewLinks.Valid)
	s.Len(e.Data.Redirects, 1)
	s.Equal("pnpm install", e.Data.InstallCmd)
	s.Equal("/path", e.Data.Redirects[0].From)
	s.Equal("/new-path", e.Data.Redirects[0].To)
	s.Equal("live", e.Branch)
	s.Equal("/dist", e.Data.ServerFolder)
	s.Equal("node index.js", e.Data.ServerCmd)
	s.Equal("/index.html", e.Data.ErrorFile)
	s.Equal("/*\nCache-Control:no-cache", e.Data.Headers)
	s.Equal("/my/api/path", e.Data.APIFolder)
	s.Equal("/api", e.Data.APIPathPrefix)
	s.Equal([]buildconf.StatusCheck{
		{Name: "Header rules", Cmd: "npm run test:headers", Description: "Test header rules in deployment."},
		{Name: "E2E", Cmd: "npm run test:e2e", Description: "Test e2e changes."},
	}, e.Data.StatusChecks)
}

func (s *HandlerEnvUpdateSuite) Test_Success_Alternative() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{"AutoDeploy": false})
	now := time.Now()

	s.mockCacheService.On("Reset", env.ID).Return(nil)

	// Required for audits
	admin.SetMockLicense()

	defer func() { admin.ResetMockLicense() }()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(buildconfhandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/app/env",
		map[string]any{
			"appId":       app.ID.String(),
			"id":          env.ID.String(),
			"branch":      "live",
			"env":         "production",
			"autoPublish": false,
			"autoDeploy":  true,
			"build": map[string]any{
				"previewLinks": true,
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	e, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)

	s.NoError(err)
	s.Equal(env.ID, e.ID)
	s.Equal(env.Name, e.Name)
	s.Equal("production", e.Env)
	s.Equal("live", e.Branch)
	s.False(e.AutoPublish)
	s.True(e.AutoDeploy)
	s.True(e.Data.PreviewLinks.ValueOrZero())
	s.True(e.Data.PreviewLinks.Valid)
	s.Nil(e.Data.Redirects)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.Nil(err)
	s.Len(audits, 1)
	s.GreaterOrEqual(now.Unix()+1000, audits[0].Timestamp.Unix())
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "UPDATE:ENV",
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		AppID:       app.ID,
		EnvID:       env.ID,
		EnvName:     env.Name,
		TeamID:      app.TeamID,
		Diff: &audit.Diff{
			Old: audit.DiffFields{
				EnvName:               env.Name,
				EnvBranch:             env.Branch,
				EnvAutoPublish:        &env.AutoPublish,
				EnvAutoDeploy:         &env.AutoDeploy,
				EnvAutoDeployBranches: env.AutoDeployBranches.ValueOrZero(),
				EnvBuildConfig:        env.Data,
			},
			New: audit.DiffFields{
				EnvName:               e.Env,
				EnvBranch:             e.Branch,
				EnvAutoPublish:        audit.Bool(false),
				EnvAutoDeploy:         audit.Bool(true),
				EnvAutoDeployBranches: e.AutoDeployBranches.ValueOrZero(),
				EnvBuildConfig:        e.Data,
			},
		},
	}, audits[0])
}

func (s *HandlerEnvUpdateSuite) TestFail_EnvIDInvalid() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	s.MockEnv(appl)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(buildconfhandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/app/env",
		map[string]any{
			"appId":       appl.ID.String(),
			"id":          "14141",
			"branch":      "live",
			"env":         "development",
			"autoPublish": false,
			"build": &buildconf.BuildConf{
				Vars: map[string]string{
					"NODE_ENV": "production",
				},
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func (s *HandlerEnvUpdateSuite) TestFail_DoubleHyphens() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"Name": "development",
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(buildconfhandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/app/env",
		map[string]any{
			"appId":  app.ID.String(),
			"id":     env.ID.String(),
			"branch": "my-branch",
			"env":    "development--renamed",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	a := assert.New(s.T())
	s.Equal(http.StatusBadRequest, response.Code)
	a.Contains(response.String(), "Double hyphens (--) are not allowed as they are reserved for Stormkit")
}

func (s *HandlerEnvUpdateSuite) TestFail_AutoDeployBranchesInvalid() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"Name": "development",
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(buildconfhandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/app/env",
		map[string]any{
			"appId":              app.ID.String(),
			"id":                 env.ID.String(),
			"branch":             "my-branch",
			"env":                "production",
			"autoDeployBranches": "(invalid-regex",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "parsing regexp: missing closing ) in `(invalid-regex`\"")
}

func TestHandlerEnvUpdate(t *testing.T) {
	suite.Run(t, &HandlerEnvUpdateSuite{})
}
