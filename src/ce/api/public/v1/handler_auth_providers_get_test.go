package publicapiv1_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth/skauthhandlers"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthProvidersGetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
	app  *factory.MockApp
	env  *factory.MockEnv
	key  *factory.MockAPIKey
}

func (s *HandlerAuthProvidersGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.app = s.MockApp(nil)
	s.env = s.MockEnv(s.app)

	s.key = s.MockAPIKey(nil, s.env, map[string]any{
		"Scope": apikey.SCOPE_ENV,
		"EnvID": s.env.ID,
	})
}

func (s *HandlerAuthProvidersGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAuthProvidersGetSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

// Test_Get_MasksClientSecret is the guarantee that keeps OAuth credentials out
// of the public API: the provider is stored with a real secret and the
// response may only ever carry the placeholder.
func (s *HandlerAuthProvidersGetSuite) Test_Get_MasksClientSecret() {
	err := skauth.NewStore().SaveProvider(context.Background(), skauth.SaveProviderArgs{
		EnvID: s.env.ID,
		AppID: s.app.ID,
		Provider: &skauth.Provider{
			Name:   skauth.ProviderGoogle,
			Status: true,
			Data: skauth.ProviderData{
				ClientID:     "client-id",
				ClientSecret: "super-secret",
			},
		},
	})

	s.Require().NoError(err)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/auth/providers?envId="+s.env.ID.String(),
		nil,
		map[string]string{"Authorization": s.key.Value},
	)

	body := response.String()

	s.Equal(http.StatusOK, response.Code)
	s.NotContains(body, "super-secret")
	s.Contains(body, skauthhandlers.ClientSecretPlaceholder)
	s.Contains(body, "client-id")
	s.Contains(body, "google")
}

func (s *HandlerAuthProvidersGetSuite) Test_Get_EmptyWhenUnset() {
	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/auth/providers?envId="+s.env.ID.String(),
		nil,
		map[string]string{"Authorization": s.key.Value},
	)

	s.Equal(http.StatusOK, response.Code)
	s.Contains(response.String(), `"providers":{}`)
}

func TestHandlerAuthProvidersGetSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthProvidersGetSuite{})
}
