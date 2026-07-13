package authwallhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall/authwallhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthConfigSetSuite struct {
	suite.Suite
	*factory.Factory
	conn             databasetest.TestDB
	mockCacheService *mocks.CacheInterface
}

func (s *HandlerAuthConfigSetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockCacheService = &mocks.CacheInterface{}
	appcache.DefaultCacheService = s.mockCacheService
	admin.SetMockLicense()
}

func (s *HandlerAuthConfigSetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	appcache.DefaultCacheService = nil
	admin.ResetMockLicense()
}

func (s *HandlerAuthConfigSetSuite) Test_AuthConfigSet_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	s.mockCacheService.On("Reset", types.ID(env.ID)).Return(nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth-wall/config",
		map[string]any{
			"envId":    env.ID.String(),
			"authwall": "all",
		},
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	config, err := authwall.Store().AuthWallConfig(context.Background(), env.ID)
	s.NoError(err)
	s.NotNil(config)
	s.Equal("all", config.Status)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.NoError(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "UPDATE:AUTHWALL",
		EnvName:     env.Name,
		EnvID:       env.ID,
		AppID:       app.ID,
		TeamID:      app.TeamID,
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		Diff: &audit.Diff{
			Old: audit.DiffFields{
				AuthWallStatus: "off",
			},
			New: audit.DiffFields{
				AuthWallStatus: authwall.StatusAll,
			},
		},
	}, audits[0])
}

func (s *HandlerAuthConfigSetSuite) Test_AuthConfigSet_ResetSuccess() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	s.NoError(authwall.Store().SetAuthWallConfig(context.Background(), env.ID, &authwall.Config{
		Status: "all",
	}))

	s.mockCacheService.On("Reset", types.ID(env.ID)).Return(nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth-wall/config",
		map[string]any{
			"envId":    env.ID.String(),
			"authwall": "",
		},
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	config, err := authwall.Store().AuthWallConfig(context.Background(), env.ID)
	s.NoError(err)
	s.NotNil(config)
	s.Equal("", config.Status)
}

func (s *HandlerAuthConfigSetSuite) Test_AuthConfigSet_InvalidOption() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth-wall/config",
		map[string]any{
			"envId":    env.ID.String(),
			"authwall": "not-allowed",
		},
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{ "error": "Invalid authwall status. Available options are: all | dev | ''" }`

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerAuthConfigSetSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthConfigSetSuite{})
}
