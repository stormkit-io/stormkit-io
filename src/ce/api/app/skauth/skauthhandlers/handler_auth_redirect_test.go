package skauthhandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth/skauthhandlers"
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
}

func (s *HandlerAuthRedirectSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAuthRedirectSuite) Test_InvalidProvider() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/auth/v1?provider=invalid&envId=1",
		nil,
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"invalid query parameter: missing or invalid provider"}`, response.String())
}

func (s *HandlerAuthRedirectSuite) Test_InvalidEnvID() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/auth/v1?provider=google",
		nil,
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"invalid query parameter: missing or invalid envId"}`, response.String())
}

func (s *HandlerAuthRedirectSuite) Test_Success() {
	s.NoError(skauth.NewStore().SaveProvider(context.Background(), skauth.SaveProviderArgs{
		Client: skauth.NewGoogleClient("abc", "def"),
		EnvID:  s.env.ID,
		AppID:  s.env.AppID,
		Status: true,
	}))

	target := fmt.Sprintf("/auth/v1?provider=google&envId=%d", s.env.ID)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
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

func TestHandlerAuthRedirect(t *testing.T) {
	suite.Run(t, &HandlerAuthRedirectSuite{})
}
