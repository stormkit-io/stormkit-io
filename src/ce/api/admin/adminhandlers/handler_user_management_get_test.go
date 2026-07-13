package adminhandlers_test

import (
	"context"
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

type HandlerUserManagementGetSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerUserManagementGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerUserManagementGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerUserManagementGetSuite) Test_Success_Default() {
	usr := s.MockUser(map[string]any{
		"IsAdmin": true,
	})

	err := admin.Store().UpsertConfig(context.Background(), admin.InstanceConfig{})
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/users/sign-up-mode",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{ "signUpMode": "on", "whitelist": [] }`, response.String())
}

func (s *HandlerUserManagementGetSuite) Test_Success_Configured() {
	usr := s.MockUser(map[string]any{
		"IsAdmin": true,
	})

	config := admin.InstanceConfig{
		AuthConfig: &admin.AuthConfig{
			UserManagement: admin.UserManagement{
				Whitelist:  []string{"stormkit.io", "example.org"},
				SignUpMode: admin.SIGNUP_MODE_WAITLIST,
			},
		},
	}

	err := admin.Store().UpsertConfig(context.Background(), config)
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/users/sign-up-mode",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{
		"signUpMode": "waitlist",
		"whitelist": ["stormkit.io", "example.org"]
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerUserManagementGetSuite) Test_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/users/sign-up-mode",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerUserManagementGetMode(t *testing.T) {
	suite.Run(t, &HandlerUserManagementGetSuite{})
}
