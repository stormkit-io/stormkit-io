package authhandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/bitbucket"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/github"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/authhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"

	"golang.org/x/oauth2"
)

const githubJWTToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3N1ZWQiOjE1NjkxNjY2MDMsInByb3ZpZGVyIjoiZ2l0aHViIn0.c6hYTMRTz8rSE5D14aVrG2JqIH6ZQBM3GYdFHq_UnAE"

type HandlerAuthCallbackSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerAuthCallbackSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAuthCallbackSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerAuthCallbackSuite) Test_403() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/auth/github/callback?code=some-random-code&state=%s", githubJWTToken),
		nil,
	)

	s.Equal(http.StatusForbidden, response.Code)
	s.Equal("text/html; charset=utf-8", response.Header().Get("Content-Type"))
	s.Contains(response.String(), `{"auth":false,"error":"token-mismatch","success":false}`)
}

func (s *HandlerAuthCallbackSuite) Test_GithubNewUser() {
	token := "vxz9414sf9zx93"
	email := "test-123@stormkit.io"
	now := time.Now()

	response := authhandlers.Login(context.Background(), &oauth.User{
		AccountURI:   "https://github.com/test-123",
		Emails:       []oauth.Email{{Address: email, IsPrimary: true, IsVerified: true}},
		ProviderName: github.ProviderName,
		DisplayName:  "test-123",
		FullName:     "FirstName LastName",
		AvatarURI:    "https://githubavatar.com/test-123",
		Token:        &oauth2.Token{AccessToken: token},
	})

	u, _ := user.NewStore().UserByEmail(context.Background(), []string{email})

	s.Equal(response.Status, http.StatusOK)
	s.Equal(response.Headers.Get("Content-Type"), "text/html; charset=utf-8")
	s.Equal(u.FirstName.ValueOrZero(), "FirstName")
	s.Equal(u.LastName.ValueOrZero(), "LastName")
	s.Equal(u.Metadata.PackageName, config.PackageFree)
	s.GreaterOrEqual(u.LastLogin.Unix(), now.Unix())
	s.NotEqual(u.LastLogin.Unix(), 0)
}

func (s *HandlerAuthCallbackSuite) Test_BitbucketNewUser() {
	token := "vxz9414sf9zx93"
	email := "test-124@stormkit.io"

	response := authhandlers.Login(context.Background(), &oauth.User{
		AccountURI:   "https://bitbucket.org/test-123",
		Emails:       []oauth.Email{{Address: email, IsPrimary: true, IsVerified: true}},
		ProviderName: bitbucket.ProviderName,
		DisplayName:  "test-124",
		FullName:     "FirstName LastName",
		AvatarURI:    "https://bitbucketavatar.com/test-123",
		Token:        &oauth2.Token{AccessToken: token},
	})

	u, _ := user.NewStore().UserByEmail(context.Background(), []string{email})

	s.Equal(response.Status, http.StatusOK)
	s.Equal(response.Headers.Get("Content-Type"), "text/html; charset=utf-8")
	s.Equal(u.FirstName.ValueOrZero(), "FirstName")
	s.Equal(u.LastName.ValueOrZero(), "LastName")
}

func (s *HandlerAuthCallbackSuite) Test_ErrorWrongClaims() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/auth/bitbucket/callback?state=%s&code=%s", "", "xvVafxAfa"),
		nil,
	)

	s.Equal(http.StatusForbidden, response.Code)
	s.Equal("text/html; charset=utf-8", response.Header().Get("Content-Type"))
	s.Contains(response.String(), `{"auth":false,"error":"token-mismatch","success":false}`)
}

func (s *HandlerAuthCallbackSuite) Test_SeatsExhausted_EE_Version() {
	config.SetIsSelfHosted(true)
	defer config.SetIsSelfHosted(false)

	admin.CachedLicense = &admin.License{
		Seats:   0,
		Premium: true,
	}

	token := "vxz9414sf9zx93"
	email := "test-5435@stormkit.io"

	response := authhandlers.Login(context.Background(), &oauth.User{
		AccountURI:   "https://github.com/test-123",
		Emails:       []oauth.Email{{Address: email, IsPrimary: true, IsVerified: true}},
		ProviderName: github.ProviderName,
		DisplayName:  "test-123",
		FullName:     "FirstName LastName",
		AvatarURI:    "https://githubavatar.com/test-123",
		Token:        &oauth2.Token{AccessToken: token},
	})

	count, err := user.NewStore().SelectTotalUsers(context.Background())

	s.NoError(err)
	s.Equal(int64(0), count)
	s.Equal(http.StatusBadRequest, response.Status)
	s.Equal(response.Headers.Get("Content-Type"), "text/html; charset=utf-8")
	s.Contains(response.String(), "seats-full")
}

func TestHandlerAuthCallbackSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthCallbackSuite{})
}
