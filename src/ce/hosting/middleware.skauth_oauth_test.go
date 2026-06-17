package hosting_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type WithSKAuthOAuthSuite struct {
	suite.Suite
	*factory.Factory
	conn       databasetest.TestDB
	app        *factory.MockApp
	mockClient *mocks.Client
}

func (s *WithSKAuthOAuthSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.app = s.MockApp(s.MockUser(nil), nil)
	s.mockClient = &mocks.Client{}
	skauth.DefaultClient = s.mockClient
}

func (s *WithSKAuthOAuthSuite) AfterTest(_, _ string) {
	skauth.DefaultClient = nil
	s.conn.CloseTx()
}

func (s *WithSKAuthOAuthSuite) setupEnv(allowed []string, providerStatus bool) *factory.MockEnv {
	env := s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.SKAuthConf{
			Secret:         "test-secret-padded-to-32-chars!!",
			Status:         true,
			AllowedOrigins: allowed,
		},
	})

	s.Require().NoError(skauth.NewStore().SaveProvider(context.Background(), skauth.SaveProviderArgs{
		EnvID: env.ID,
		AppID: s.app.ID,
		Provider: &skauth.Provider{
			Name:   skauth.ProviderGoogle,
			Status: providerStatus,
		},
	}))

	return env
}

func (s *WithSKAuthOAuthSuite) hostFor(envID types.ID) *hosting.Host {
	return &hosting.Host{
		Name: "api.example.com",
		Config: &appconf.Config{
			EnvID: envID,
			SKAuth: &buildconf.SKAuthConf{
				Secret:     "test-secret-padded-to-32-chars!!",
				SuccessURL: "/dashboard",
			},
		},
	}
}

func (s *WithSKAuthOAuthSuite) oauthRequest(host *hosting.Host, target string, headers map[string]string) *hosting.RequestContext {
	pieces := strings.SplitN(target, "?", 2)
	rawPath := pieces[0]
	query := ""

	if len(pieces) > 1 {
		query = pieces[1]
	}

	header := make(http.Header)

	for k, v := range headers {
		header.Set(k, v)
	}

	rq := &hosting.RequestContext{
		Host: host,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Method: http.MethodGet,
			Header: header,
			URL: &url.URL{
				Scheme:   "https",
				Host:     host.Name,
				Path:     rawPath,
				RawQuery: query,
				RawPath:  rawPath,
			},
		}),
	}

	rq.OriginalPath = rawPath

	return rq
}

func (s *WithSKAuthOAuthSuite) Test_Initiate_AllowedRedirect_Redirects() {
	env := s.setupEnv([]string{"https://app.example.com"}, true)

	s.mockClient.On("AuthCodeURL", skauth.AuthCodeURLParams{
		EnvID:        env.ID,
		ProviderName: skauth.ProviderGoogle,
		Referrer:     "https://app.example.com",
	}).Return("https://accounts.google.com/o/oauth2/auth", nil)

	res, err := hosting.WithSKAuth(s.oauthRequest(
		s.hostFor(env.ID),
		"/_stormkit/auth/google?redirect=https://app.example.com",
		nil,
	))

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Equal("https://accounts.google.com/o/oauth2/auth", *res.Redirect)
}

func (s *WithSKAuthOAuthSuite) Test_Initiate_DisallowedRedirect_Returns403() {
	env := s.setupEnv([]string{"https://app.example.com"}, true)

	res, err := hosting.WithSKAuth(s.oauthRequest(
		s.hostFor(env.ID),
		"/_stormkit/auth/google?redirect=https://evil.example.com",
		nil,
	))

	s.Require().NoError(err)
	s.Equal(http.StatusForbidden, res.Status)
}

func (s *WithSKAuthOAuthSuite) Test_Initiate_NoAllowList_FallsBackToOwnOrigin() {
	env := s.setupEnv(nil, true)

	// Without an allow-list, the supplied cross-origin redirect is ignored and
	// the flow returns the user to the app's own origin.
	s.mockClient.On("AuthCodeURL", skauth.AuthCodeURLParams{
		EnvID:        env.ID,
		ProviderName: skauth.ProviderGoogle,
		Referrer:     "https://api.example.com",
	}).Return("https://accounts.google.com/o/oauth2/auth", nil)

	res, err := hosting.WithSKAuth(s.oauthRequest(
		s.hostFor(env.ID),
		"/_stormkit/auth/google?redirect=https://other.example.com",
		nil,
	))

	s.Require().NoError(err)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
}

func (s *WithSKAuthOAuthSuite) Test_Initiate_FallsBackToRefererHeader() {
	env := s.setupEnv([]string{"https://app.example.com"}, true)

	s.mockClient.On("AuthCodeURL", skauth.AuthCodeURLParams{
		EnvID:        env.ID,
		ProviderName: skauth.ProviderGoogle,
		Referrer:     "https://app.example.com",
	}).Return("https://accounts.google.com/o/oauth2/auth", nil)

	res, err := hosting.WithSKAuth(s.oauthRequest(
		s.hostFor(env.ID),
		"/_stormkit/auth/google",
		map[string]string{"Referer": "https://app.example.com/login"},
	))

	s.Require().NoError(err)
	s.Equal(http.StatusFound, res.Status)
}

func (s *WithSKAuthOAuthSuite) Test_Initiate_ProviderDisabled_NotFound() {
	env := s.setupEnv([]string{"https://app.example.com"}, false)

	res, err := hosting.WithSKAuth(s.oauthRequest(
		s.hostFor(env.ID),
		"/_stormkit/auth/google?redirect=https://app.example.com",
		nil,
	))

	s.Require().NoError(err)
	s.Equal(http.StatusNotFound, res.Status)
}

func TestWithSKAuthOAuth(t *testing.T) {
	suite.Run(t, &WithSKAuthOAuthSuite{})
}
