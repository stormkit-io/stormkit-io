package buildconfhandlers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/buildconfhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerEnvInsertSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerEnvInsertSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerEnvInsertSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerEnvInsertSuite) Test_BadRequestEnvMissing() {
	app := s.MockApp(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(buildconfhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/env",
		map[string]string{
			"appId": app.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(app.UserID),
		},
	)

	expected := `{
		"errors": [
			"Branch is a required field",
			"Name is a required field"
		]
	}`

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerEnvInsertSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	now := time.Now()

	// Required for audits
	admin.SetMockLicense()

	defer func() { admin.ResetMockLicense() }()

	conf := &buildconf.Env{
		Env:    "dev",
		Branch: "dev-branch",
		Data: &buildconf.BuildConf{
			Vars: map[string]string{
				"NODE_ENV": "production",
			},
		},
	}

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(buildconfhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/env",
		map[string]any{
			"name":        conf.Env,
			"branch":      conf.Branch,
			"autoPublish": false,
			"appId":       app.ID.String(),
			"build":       conf.Data,
		},
		map[string]string{
			"Authorization": usertest.Authorization(app.UserID),
		},
	)

	conf, _ = buildconf.NewStore().Environment(context.Background(), 1, "dev")
	s.Equal("dev-branch", conf.Branch)
	s.Equal(false, conf.AutoPublish)
	s.Equal(http.StatusCreated, response.Code)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		AppID: app.ID,
	})

	s.Nil(err)
	s.Len(audits, 1)
	s.GreaterOrEqual(now.Unix()+1000, audits[0].Timestamp.Unix())
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "CREATE:ENV",
		AppID:       app.ID,
		TeamID:      app.TeamID,
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		Diff: &audit.Diff{
			New: audit.DiffFields{
				EnvID:   conf.ID.String(),
				EnvName: conf.Name,
			},
		},
	}, audits[0])
}

func (s *HandlerEnvInsertSuite) Test_SuccessWithSameBranch() {
	app := s.MockApp(nil)

	conf := &buildconf.Env{
		Env:    "dev",
		Branch: "master",
		Data: &buildconf.BuildConf{
			Vars: map[string]string{
				"NODE_ENV": "production",
			},
		},
	}

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(buildconfhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/env",
		map[string]any{
			"name":        conf.Env,
			"branch":      conf.Branch,
			"autoPublish": false,
			"appId":       app.ID.String(),
			"build":       conf.Data,
		},
		map[string]string{
			"Authorization": usertest.Authorization(app.UserID),
		},
	)

	conf, _ = buildconf.NewStore().Environment(context.Background(), 1, "dev")
	s.Equal("master", conf.Branch)
	s.Equal(false, conf.AutoPublish)
	s.Equal(http.StatusCreated, response.Code)
}

func TestHandlerEnvInsert(t *testing.T) {
	suite.Run(t, &HandlerEnvInsertSuite{})
}
