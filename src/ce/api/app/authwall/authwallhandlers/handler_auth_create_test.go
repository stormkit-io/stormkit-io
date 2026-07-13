package authwallhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall/authwallhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthCreateSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerAuthCreateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerAuthCreateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerAuthCreateSuite) Test_AuthCreate_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth-wall",
		map[string]any{
			"envId":    env.ID.String(),
			"email":    "my-email@example.org",
			"password": "my-paSword",
		},
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	logins, err := authwall.Store().Logins(context.Background(), env.ID)
	s.NoError(err)
	s.Len(logins, 1)
	s.Equal("my-email@example.org", logins[0].LoginEmail)
	s.Equal("my-paSword", utils.DecryptToString(logins[0].LoginPassword))

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.NoError(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "CREATE:AUTHWALL",
		EnvName:     env.Name,
		EnvID:       env.ID,
		AppID:       app.ID,
		TeamID:      app.TeamID,
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		Diff: &audit.Diff{
			New: audit.DiffFields{
				AuthWallCreateLoginEmail: "my-email@example.org",
				AuthWallCreateLoginID:    logins[0].LoginID.String(),
			},
		},
	}, audits[0])
}

func (s *HandlerAuthCreateSuite) Test_AuthCreate_FailDuplicate() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	aw := &authwall.AuthWall{
		LoginEmail:    "my-email@example.org",
		LoginPassword: "123pass",
		EnvID:         env.ID,
	}

	s.NoError(authwall.Store().CreateLogin(context.Background(), aw))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth-wall",
		map[string]any{
			"envId":    env.ID.String(),
			"email":    "my-email@example.org",
			"password": "my-paSword",
		},
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusConflict, response.Code)
}

func TestHandlerAuthCreateSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthCreateSuite{})
}
