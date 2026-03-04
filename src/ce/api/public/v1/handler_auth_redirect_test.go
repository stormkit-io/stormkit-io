package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type HandlerAuthRedirectSuite struct {
	suite.Suite
	*factory.Factory
	conn       databasetest.TestDB
	app        *factory.MockApp
	env        *factory.MockEnv
	mockClient *mocks.Client
}

func (s *HandlerAuthRedirectSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	// Create test app and environment
	s.app = s.MockApp(s.MockUser(nil), nil)
	s.env = s.MockEnv(s.app, map[string]any{
		"Name": "development",
		"AuthConf": &buildconf.SKAuthConf{
			Status: true,
		},
	})

	s.mockClient = &mocks.Client{}
	skauth.DefaultClient = s.mockClient

	config.SetIsSelfHosted(true)
	admin.MustConfig().SetURL("localhost")
}

func (s *HandlerAuthRedirectSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	config.SetIsSelfHosted(false)
	skauth.DefaultClient = nil
}

func (s *HandlerAuthRedirectSuite) saveProvider(status bool) {
	s.NoError(skauth.NewStore().SaveProvider(context.Background(), skauth.SaveProviderArgs{
		Provider: &skauth.Provider{
			Status: status,
			Name:   skauth.ProviderGoogle,
			Data: skauth.ProviderData{
				ClientID:     "abc",
				ClientSecret: "def",
			},
		},
		EnvID: s.env.ID,
		AppID: s.env.AppID,
	}))
}

func (s *HandlerAuthRedirectSuite) mockAuthCodeURL(referrer ...string) {
	ref := fmt.Sprintf("https://%s--%s.localhost", s.app.DisplayName, s.env.Name)

	if len(referrer) > 0 {
		ref = referrer[0]
	}

	s.mockClient.On("AuthCodeURL", skauth.AuthCodeURLParams{
		ProviderName: skauth.ProviderGoogle,
		EnvID:        s.env.ID,
		Referrer:     ref,
	}).Return("http://auth.url", nil)
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
	s.saveProvider(true)
	s.mockAuthCodeURL()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/auth?provider=google&envId=%d", s.env.ID),
		nil,
		nil,
	)

	s.Equal(http.StatusFound, response.Code)
	s.Equal("http://auth.url", response.Header().Get("Location"))
}

func (s *HandlerAuthRedirectSuite) Test_Disabled_Auth() {
	s.env = s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.SKAuthConf{
			Status: false,
		},
	})

	s.saveProvider(true)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/auth?provider=google&envId=%d", s.env.ID),
		nil,
		nil,
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func (s *HandlerAuthRedirectSuite) Test_Fail_DisabledProvider() {
	s.saveProvider(false)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/auth?provider=google&envId=%d", s.env.ID),
		nil,
		nil,
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func (s *HandlerAuthRedirectSuite) Test_Fail_InvalidReferrer() {
	s.saveProvider(true)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/auth?provider=google&envId=%d&referrer=invalid-url", s.env.ID),
		nil,
		nil,
	)

	expected := `{
		"error": "Referer is not a valid URL",
		"hint": "Make sure the URL is an absolute URL with a valid format, e.g., https://myapp.com"
	}`

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerAuthRedirectSuite) Test_Fail_ReferrerDoesNotBelong() {
	s.saveProvider(true)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/auth?provider=google&envId=%d&referrer=https://example.com", s.env.ID),
		nil,
		nil,
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerAuthRedirectSuite) Test_Success_WithReferrer() {
	domain := &buildconf.DomainModel{
		AppID:    s.app.ID,
		EnvID:    s.env.ID,
		Name:     "my.example.org",
		Token:    null.NewString("my-token", true),
		Verified: true,
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain))

	s.saveProvider(true)
	s.mockAuthCodeURL("https://" + domain.Name)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/auth?provider=google&envId=%d&referrer=https://my.example.org", s.env.ID),
		nil,
		nil,
	)

	s.Equal(http.StatusFound, response.Code)
}

func TestHandlerAuthRedirect(t *testing.T) {
	suite.Run(t, &HandlerAuthRedirectSuite{})
}
