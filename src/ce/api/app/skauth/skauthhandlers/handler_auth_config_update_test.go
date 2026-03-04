package skauthhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth/skauthhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthConfigUpdateSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	usr  *factory.MockUser
	app  *factory.MockApp
	env  *factory.MockEnv
}

func (s *HandlerAuthConfigUpdateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	// Create test user, app, and environment
	s.usr = s.MockUser(nil)
	s.app = s.MockApp(s.usr, nil)
	s.env = s.MockEnv(s.app)
}

func (s *HandlerAuthConfigUpdateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAuthConfigUpdateSuite) Test_Update() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/skauth/config",
		map[string]any{
			"envId":      s.env.ID,
			"successUrl": "/success",
			"tokenTtl":   10,
			"status":     true,
		},
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	env, err := buildconf.NewStore().EnvironmentByID(context.Background(), s.env.ID)
	s.NoError(err)
	s.Equal("/success", env.AuthConf.SuccessURL)
	s.Equal(10, env.AuthConf.TTL)
	s.True(env.AuthConf.Status)
	s.Len(env.AuthConf.Secret, 128)
}

func (s *HandlerAuthConfigUpdateSuite) Test_Update_Existing() {
	authConf := &buildconf.SKAuthConf{
		Secret:     utils.RandomToken(128),
		SuccessURL: "/old-success",
		TTL:        5,
		Status:     true,
	}

	s.NoError(buildconf.NewStore().SaveAuthConf(context.Background(), s.env.ID, authConf))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/skauth/config",
		map[string]any{
			"envId":      s.env.ID,
			"successUrl": "/success",
			"tokenTtl":   10,
			"status":     false,
		},
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	env, err := buildconf.NewStore().EnvironmentByID(context.Background(), s.env.ID)
	s.NoError(err)
	s.Equal("/success", env.AuthConf.SuccessURL)
	s.Equal(10, env.AuthConf.TTL)
	s.Equal(env.AuthConf.Secret, authConf.Secret) // Secret should remain unchanged
	s.False(env.AuthConf.Status)
	s.Len(env.AuthConf.Secret, 128)
}

func (s *HandlerAuthConfigUpdateSuite) Test_InvalidURL() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/skauth/config",
		map[string]any{
			"envId":      s.env.ID,
			"successUrl": "$!4!(!%(!!@@))",
			"tokenTtl":   10,
		},
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	expected := `{
		"error": "Success URL format is not valid. Make sure to provide a relative URL.",
		"hint": "Provide a relative URL such as: /success"
	}`

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerAuthConfigUpdateSuite) Test_AbsoluteURL_ShouldFail() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/skauth/config",
		map[string]any{
			"envId":      s.env.ID,
			"successUrl": "https://example.org/success",
			"tokenTtl":   10,
		},
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	expected := `{
		"error": "Success URL is not a relative URL.",
		"hint": "Provide a relative URL such as: /success"
	}`

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerAuthConfigUpdate(t *testing.T) {
	suite.Run(t, &HandlerAuthConfigUpdateSuite{})
}
