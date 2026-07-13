package authhandlers_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/authhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthLoginSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerAuthLoginSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{}
	s.NoError(admin.Store().UpsertConfig(context.TODO(), cnf))
}

func (s *HandlerAuthLoginSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAuthLoginSuite) Test_SuccessBitbucket() {
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{
		Bitbucket: admin.BitbucketConfig{
			ClientID:     "my-id",
			ClientSecret: "my-secret",
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.TODO(), cnf))

	response := shttptest.Request(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/auth/bitbucket",
		nil,
	)

	s.Equal(http.StatusTemporaryRedirect, response.Code)

	loc := response.Header().Get("Location")
	s.True(strings.HasPrefix(loc, "https://bitbucket.org/site/oauth2/authorize?"),
		"Invalid redirect URL: %s", loc)
}

func (s *HandlerAuthLoginSuite) Test_Github_Success() {
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{
		Github: admin.GithubConfig{
			ClientID:     "my-id",
			ClientSecret: "my-secret",
			PrivateKey:   "my-key",
			Account:      "my-account",
			AppID:        12345,
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.TODO(), cnf))

	response := shttptest.Request(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/auth/github",
		nil,
	)

	s.Equal(http.StatusTemporaryRedirect, response.Code)
	s.True(strings.HasPrefix(response.Header().Get("Location"), "https://github.com/login/oauth/authorize?client_id=my-id"))
}

func (s *HandlerAuthLoginSuite) Test_FailProvider() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/auth/something-else",
		nil,
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func (s *HandlerAuthLoginSuite) Test_BitbucketMissingConfig() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/auth/bitbucket",
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func (s *HandlerAuthLoginSuite) Test_BitbucketRedirectContainsParameters() {
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{
		Bitbucket: admin.BitbucketConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.TODO(), cnf))

	response := shttptest.Request(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/auth/bitbucket",
		nil,
	)

	s.Equal(http.StatusTemporaryRedirect, response.Code)

	loc := response.Header().Get("Location")
	s.Contains(loc, "client_id=test-client-id")
	s.Contains(loc, "response_type=code")
}

func TestHandlerAuthLoginSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthLoginSuite{})
}
