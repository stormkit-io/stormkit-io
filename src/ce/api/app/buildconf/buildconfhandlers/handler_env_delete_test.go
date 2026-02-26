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

type HandlerEnvDeleteSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerEnvDeleteSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerEnvDeleteSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerEnvDeleteSuite) Test_Success() {
	now := time.Now()
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"Name": "development",
	})

	// Required for audits
	admin.SetMockLicense()

	defer func() { admin.ResetMockLicense() }()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(buildconfhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		"/app/env",
		map[string]any{
			"appId": app.ID.String(),
			"env":   env.Name,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	conf, err := buildconf.NewStore().Environment(context.Background(), 2, "development")

	s.NoError(err)
	s.Nil(conf)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		AppID: app.ID,
	})

	s.Nil(err)
	s.Len(audits, 1)
	s.GreaterOrEqual(now.Unix()+1000, audits[0].Timestamp.Unix())
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Action:      "DELETE:ENV",
		AppID:       app.ID,
		UserID:      usr.ID,
		TeamID:      app.TeamID,
		UserDisplay: usr.Display(),
		Timestamp:   audits[0].Timestamp,
		Diff: &audit.Diff{
			Old: audit.DiffFields{
				EnvName: env.Name,
			},
		},
	}, audits[0])
}

func (s *HandlerEnvDeleteSuite) Test_Success_WithEnvID() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"Name": "my-other-domain",
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(buildconfhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		"/app/env",
		map[string]any{
			"appId": app.ID.String(),
			"envId": env.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	conf, err := buildconf.NewStore().Environment(context.Background(), 2, "my-other-domain")

	s.NoError(err)
	s.Nil(conf)
}

func TestHandlerEnvDelete(t *testing.T) {
	suite.Run(t, &HandlerEnvDeleteSuite{})
}
