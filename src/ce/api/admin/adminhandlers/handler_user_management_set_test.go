package adminhandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerUserManagementSetSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerUserManagementSetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerUserManagementSetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerUserManagementSetSuite) Test_Success() {
	usr := s.MockUser(map[string]any{
		"IsAdmin": true,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/users/sign-up-mode",
		map[string]any{
			"signUpMode": admin.SIGNUP_MODE_WAITLIST,
			"whitelist":  []string{"example.org"},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	cfg := admin.MustConfig().AuthConfig

	s.Equal(http.StatusOK, response.Code)
	s.NotNil(cfg)
	s.Equal(admin.SIGNUP_MODE_WAITLIST, cfg.UserManagement.SignUpMode)
	s.Equal([]string{"example.org"}, cfg.UserManagement.Whitelist)
}

func (s *HandlerUserManagementSetSuite) Test_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/users/sign-up-mode",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerUserManagementSetMode(t *testing.T) {
	suite.Run(t, &HandlerUserManagementSetSuite{})
}
