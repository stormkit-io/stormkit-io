package apphandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type AppDeleteSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *AppDeleteSuite) SetupSuite() {
}

func (s *AppDeleteSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *AppDeleteSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *AppDeleteSuite) TestHandlerAppDelete_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		"/app",
		map[string]string{
			"appId": appl.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(appl.UserID),
		},
	)

	myApp, err := app.NewStore().AppByID(context.Background(), appl.ID)

	s.NoError(err)
	s.Nil(myApp)
	s.Equal(http.StatusOK, response.Code)

	// Make sure environments are also deleted
	myEnv, err := buildconf.NewStore().EnvironmentByID(context.Background(), 1)

	s.NoError(err)
	s.Nil(myEnv)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		TeamID: appl.TeamID,
	})

	s.NoError(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "DELETE:APP",
		UserDisplay: usr.Display(),
		UserID:      appl.UserID,
		TeamID:      appl.TeamID,
		Diff: &audit.Diff{
			Old: audit.DiffFields{
				AppName: appl.DisplayName,
				AppRepo: appl.Repo,
			},
		},
	}, audits[0])
}

func TestHandlerAppDelete(t *testing.T) {
	suite.Run(t, &AppDeleteSuite{})
}
