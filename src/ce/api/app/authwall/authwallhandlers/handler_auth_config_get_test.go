package authwallhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall/authwallhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthConfigGetSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerAuthConfigGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerAuthConfigGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerAuthConfigGetSuite) Test_AuthConfigGet_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	s.NoError(authwall.Store().SetAuthWallConfig(context.Background(), env.ID, &authwall.Config{
		Status: "all",
	}))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/auth-wall/config?envId="+env.ID.String(),
		nil,
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{ "authwall": "all" }`, response.String())
}

func (s *HandlerAuthConfigGetSuite) Test_AuthConfigGet_SuccessEmptyState() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/auth-wall/config?envId="+env.ID.String(),
		nil,
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{ "authwall": "" }`, response.String())
}

func TestHandlerAuthConfigGetSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthConfigGetSuite{})
}
