package skauthhandlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth/skauthhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthsSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	usr  *factory.MockUser
	app  *factory.MockApp
	env  *factory.MockEnv
}

func (s *HandlerAuthsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	// Create test user, app, and environment
	s.usr = s.MockUser(nil)
	s.app = s.MockApp(s.usr)
	s.env = s.MockEnv(s.app)
}

func (s *HandlerAuthsSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAuthsSuite) Test_NoProviders() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/skauth/providers?envId=%d", s.env.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	res := struct {
		Providers   map[string]any `json:"providers"`
		RedirectURL string         `json:"redirectUrl"`
	}{}

	s.NoError(json.Unmarshal(response.Byte(), &res))
	s.Len(res.Providers, 0)
	s.Equal("http://api.stormkit:8888/v1/auth/callback", res.RedirectURL)
}

func (s *HandlerAuthsSuite) Test_ReturnsProviders() {
	// Enable a provider via the enable endpoint
	err := skauth.NewStore().SaveProvider(context.Background(), skauth.SaveProviderArgs{
		EnvID:  s.env.ID,
		AppID:  s.app.ID,
		Status: true,
		Client: skauth.NewGoogleClient("my-client-id", "my-client-secret"),
	})

	s.NoError(err)

	// Fetch providers
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/skauth/providers?envId=%d", s.env.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	res := struct {
		RedirectURL string `json:"redirectUrl"`
		Providers   map[string]struct {
			Status   bool
			ClientID string
		} `json:"providers"`
	}{}

	s.NoError(json.Unmarshal(response.Byte(), &res))
	s.Len(res.Providers, 1)
	s.Equal("http://api.stormkit:8888/v1/auth/callback", res.RedirectURL)

	google := res.Providers["google"]
	s.Equal("my-client-id", google.ClientID)
	s.True(google.Status)
}

func TestHandlerAuths(t *testing.T) {
	suite.Run(t, &HandlerAuthsSuite{})
}
