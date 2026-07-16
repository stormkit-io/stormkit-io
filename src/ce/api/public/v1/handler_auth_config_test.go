package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthConfigSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerAuthConfigSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAuthConfigSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAuthConfigSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerAuthConfigSuite) envKey() (*factory.MockEnv, *factory.MockAPIKey) {
	app := s.MockApp(nil)
	env := s.MockEnv(app)
	key := s.MockAPIKey(nil, env, map[string]any{
		"Scope": apikey.SCOPE_ENV,
		"EnvID": env.ID,
	})

	return env, key
}

func (s *HandlerAuthConfigSuite) Test_Forbidden_NoAPIKey() {
	for _, method := range []string{shttp.MethodGet, shttp.MethodPost} {
		response := shttptest.RequestWithHeaders(
			s.handler(),
			method,
			"/v1/auth/config",
			nil,
			map[string]string{},
		)

		s.Equal(http.StatusForbidden, response.Code)
	}
}

func (s *HandlerAuthConfigSuite) Test_Get_DefaultsWhenUnset() {
	env, key := s.envKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/auth/config?envId=%s", env.ID),
		nil,
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{
		"status": false,
		"successUrl": "",
		"tokenTtl": 0,
		"allowedOrigins": [],
		"oauthServerEnabled": false,
		"oauthResourcePath": "",
		"oauthAllowLoopback": false,
		"cookieDomain": "",
		"loginUrl": ""
	}`, response.String())
}

func (s *HandlerAuthConfigSuite) Test_Set_CreatesConfigAndReturnsIt() {
	env, key := s.envKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		fmt.Sprintf("/v1/auth/config?envId=%s", env.ID),
		map[string]any{
			"status":     true,
			"successUrl": "/auth/success",
			"tokenTtl":   120,
		},
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusOK, response.Code)

	stored, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)

	s.NoError(err)
	s.Require().NotNil(stored.AuthConf)
	s.True(stored.AuthConf.Status)
	s.Equal("/auth/success", stored.AuthConf.SuccessURL)
	s.Equal(120, stored.AuthConf.TTL)
	// A signing secret is generated for a brand-new config.
	s.Len(stored.AuthConf.Secret, 128)
}

// Test_Set_IsPatch verifies that a follow-up request touching only the OAuth
// server fields leaves the previously stored session settings intact.
func (s *HandlerAuthConfigSuite) Test_Set_IsPatch() {
	env, key := s.envKey()

	s.NoError(buildconf.NewStore().SaveAuthConf(context.Background(), env.ID, &buildconf.SKAuthConf{
		Secret:     "existing-secret",
		Status:     true,
		SuccessURL: "/dashboard",
		TTL:        60,
	}))

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		fmt.Sprintf("/v1/auth/config?envId=%s", env.ID),
		map[string]any{
			"oauthServerEnabled": true,
			"oauthResourcePath":  "mcp",
			"oauthAllowLoopback": true,
		},
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusOK, response.Code)

	stored, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)

	s.NoError(err)
	s.Require().NotNil(stored.AuthConf)
	// Untouched session fields survive the patch.
	s.True(stored.AuthConf.Status)
	s.Equal("/dashboard", stored.AuthConf.SuccessURL)
	s.Equal(60, stored.AuthConf.TTL)
	s.Equal("existing-secret", stored.AuthConf.Secret)
	// OAuth server fields applied and the path normalized to a leading slash.
	s.Require().NotNil(stored.AuthConf.OAuthServer)
	s.True(stored.AuthConf.OAuthServer.Enabled)
	s.Equal("/mcp", stored.AuthConf.OAuthServer.ResourcePath)
	s.True(stored.AuthConf.OAuthServer.AllowLoopback)
}

func (s *HandlerAuthConfigSuite) Test_Set_BadRequest_AbsoluteSuccessURL() {
	env, key := s.envKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		fmt.Sprintf("/v1/auth/config?envId=%s", env.ID),
		map[string]any{
			"successUrl": "https://evil.example.com/success",
		},
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "Success URL is not a relative URL.")
}

func (s *HandlerAuthConfigSuite) Test_Set_BadRequest_ResourcePathWithQuery() {
	env, key := s.envKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		fmt.Sprintf("/v1/auth/config?envId=%s", env.ID),
		map[string]any{
			"oauthServerEnabled": true,
			"oauthResourcePath":  "/mcp?x=1",
		},
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func TestHandlerAuthConfig(t *testing.T) {
	suite.Run(t, &HandlerAuthConfigSuite{})
}
