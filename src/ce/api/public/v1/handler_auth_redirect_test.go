package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthRedirectSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	app  *factory.MockApp
	env  *factory.MockEnv
}

func (s *HandlerAuthRedirectSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	// Create test app and environment
	s.app = s.MockApp(s.MockUser(nil), nil)
	s.env = s.MockEnv(s.app)

	config.SetIsSelfHosted(true)
}

func (s *HandlerAuthRedirectSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	config.SetIsSelfHosted(false)
}

func (s *HandlerAuthRedirectSuite) Test_InvalidProvider() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/auth?provider=invalid&envId=1",
		nil,
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"invalid query parameter: missing or invalid provider"}`, response.String())
}

func (s *HandlerAuthRedirectSuite) Test_InvalidEnvID() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/auth?provider=google",
		nil,
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"invalid query parameter: missing or invalid envId"}`, response.String())
}

func (s *HandlerAuthRedirectSuite) Test_Success() {
	s.NoError(skauth.NewStore().SaveProvider(context.Background(), skauth.SaveProviderArgs{
		Provider: &skauth.Provider{
			Status: true,
			Name:   skauth.ProviderGoogle,
			Data: skauth.ProviderData{
				ClientID:     "abc",
				ClientSecret: "def",
			},
		},
		EnvID: s.env.ID,
		AppID: s.env.AppID,
	}))

	target := fmt.Sprintf("/v1/auth?provider=google&envId=%d", s.env.ID)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		target,
		nil,
		nil,
	)

	s.Equal(http.StatusFound, response.Code)
	loc := response.Header().Get("Location")
	s.Contains(loc, "state=")
	s.Contains(loc, "access_type=offline")
}

func (s *HandlerAuthRedirectSuite) Test_Fail_DisabledProvider() {
	s.NoError(skauth.NewStore().SaveProvider(context.Background(), skauth.SaveProviderArgs{
		Provider: &skauth.Provider{
			Status: false,
			Name:   skauth.ProviderGoogle,
			Data: skauth.ProviderData{
				ClientID:     "abc",
				ClientSecret: "def",
			},
		},
		EnvID: s.env.ID,
		AppID: s.env.AppID,
	}))

	target := fmt.Sprintf("/v1/auth?provider=google&envId=%d", s.env.ID)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		target,
		nil,
		nil,
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func TestHandlerAuthRedirect(t *testing.T) {
	suite.Run(t, &HandlerAuthRedirectSuite{})
}
