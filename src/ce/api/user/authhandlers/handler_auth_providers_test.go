package authhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/authhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthProvidersSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerAuthProvidersSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.NoError(admin.Store().DeleteConfig(context.Background()))
}

func (s *HandlerAuthProvidersSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAuthProvidersSuite) Test_Providers() {
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{
		Github: admin.GithubConfig{
			ClientID:     "my-client-id",
			ClientSecret: "my-secret",
			Account:      "my-account",
			PrivateKey:   "my-private-key",
			AppID:        12345,
		},
		Gitlab: admin.GitlabConfig{
			ClientID:     "my-client-id",
			ClientSecret: "my-secret",
		},
		Bitbucket: admin.BitbucketConfig{
			ClientID:     "my-client-id",
			ClientSecret: "my-secret",
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/auth/providers",
		nil,
		nil,
	)

	expected := `{
		"github": true,
		"gitlab": true,
		"bitbucket": true
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerAuthProvidersSuite) Test_Provider_BasicAuthEnabled() {
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{}
	cnf.AdminUserConfig = &admin.AdminUserConfig{
		Email:    "test@admin.com",
		Password: "password",
	}

	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/auth/providers",
		nil,
		nil,
	)

	expected := `{
		"github": false,
		"gitlab": false,
		"bitbucket": false,
		"basicAuth": "enabled"
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerAuthProviders(t *testing.T) {
	suite.Run(t, &HandlerAuthProvidersSuite{})
}
