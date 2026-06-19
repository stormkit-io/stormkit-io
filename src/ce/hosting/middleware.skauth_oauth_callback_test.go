package hosting_test

import (
	"context"
	"fmt"
	"net/http"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"
)

// stateToken mints the OAuth2 state JWT the provider hands back on the callback.
// The hosting callback only reads "prv" and "ref" — the environment is resolved
// from the request Host, not from the state — so those are the only claims set.
func (s *WithSKAuthOAuthSuite) stateToken(provider, ref string) string {
	token, err := user.JWT(jwt.MapClaims{
		"prv": provider,
		"ref": ref,
	})

	s.Require().NoError(err)

	return token
}

// setupCallbackEnv provisions an environment wired for the full callback path:
// auth enabled with a signing secret, a real schema store with the auth table
// created, and an enabled Google provider.
func (s *WithSKAuthOAuthSuite) setupCallbackEnv(providerStatus bool) *hosting.Host {
	env := s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.SKAuthConf{
			Secret: "test-secret-padded-to-32-chars!!",
			Status: true,
		},
		"SchemaConf": &buildconf.SchemaConf{
			SchemaName:        s.conn.Cfg.Schema,
			DBName:            s.conn.Cfg.DBName,
			Port:              s.conn.Cfg.Port,
			Host:              s.conn.Cfg.Host,
			MigrationUserName: s.conn.Cfg.User,
			MigrationPassword: s.conn.Cfg.Password,
			AppUserName:       s.conn.Cfg.User,
			AppPassword:       s.conn.Cfg.Password,
		},
	})

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeMigrations)
	s.Require().NoError(err)
	s.Require().NoError(store.CreateAuthTable(context.Background()))

	s.Require().NoError(skauth.NewStore().SaveProvider(context.Background(), skauth.SaveProviderArgs{
		EnvID: env.ID,
		AppID: s.app.ID,
		Provider: &skauth.Provider{
			Name:   skauth.ProviderGoogle,
			Status: providerStatus,
		},
	}))

	return s.hostFor(env.ID)
}

func (s *WithSKAuthOAuthSuite) Test_Callback_Success() {
	host := s.setupCallbackEnv(true)

	token := &oauth2.Token{}

	s.mockClient.On("Exchange", mock.Anything, mock.MatchedBy(func(req *shttp.RequestContext) bool {
		return req.FormValue("code") == "test-code"
	})).Return(token, nil).Once()

	s.mockClient.On("UserInfo", mock.Anything, token).Return(&skauth.UserInfo{
		AccountID: "test-account-id",
		Email:     "jane@stormkit.io",
		FirstName: "Jane",
		LastName:  "Doe",
		Avatar:    "link-to-avatar",
	}, nil).Once()

	state := s.stateToken(skauth.ProviderGoogle, "https://app.example.com/login")

	res, err := hosting.WithSKAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&code=test-code", state),
		nil,
	))

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Contains(*res.Redirect, "https://app.example.com/_stormkit/auth?code=")
}

func (s *WithSKAuthOAuthSuite) Test_Callback_InvalidState() {
	host := s.setupCallbackEnv(true)

	res, err := hosting.WithSKAuth(s.oauthRequest(
		host,
		"/_stormkit/auth/callback?state=not-a-jwt&code=test-code",
		nil,
	))

	s.Require().NoError(err)
	s.Equal(http.StatusBadRequest, res.Status)
}

func (s *WithSKAuthOAuthSuite) Test_Callback_ProviderDisabled_NotFound() {
	host := s.setupCallbackEnv(false)

	state := s.stateToken(skauth.ProviderGoogle, "https://app.example.com/login")

	res, err := hosting.WithSKAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&code=test-code", state),
		nil,
	))

	s.Require().NoError(err)
	s.Equal(http.StatusNotFound, res.Status)
}
