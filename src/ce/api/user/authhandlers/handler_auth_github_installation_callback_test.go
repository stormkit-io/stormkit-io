package authhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/authhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthGithubInstallationCallbackSuite struct {
	suite.Suite
}

func (s *HandlerAuthGithubInstallationCallbackSuite) AfterTest(_, _ string) {
	admin.ResetCache(context.Background())
}

func (s *HandlerAuthGithubInstallationCallbackSuite) installationCallback() string {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/auth/github/installation",
		nil,
	)

	s.Equal(http.StatusOK, response.Code)
	s.Equal("text/html; charset=utf-8", response.Header().Get("Content-Type"))

	return response.String()
}

// Test_PostsToAppOrigin verifies that the result is posted only to the
// Stormkit app, never to whichever site opened the window.
func (s *HandlerAuthGithubInstallationCallbackSuite) Test_PostsToAppOrigin() {
	admin.MustConfig().SetURL("http://stormkit:8888")

	body := s.installationCallback()

	s.Contains(body, `window.opener.postMessage({"success":true}, "http://stormkit.stormkit:8888")`)
	s.NotContains(body, `"*"`)
}

// Test_NoAppURL verifies that nothing is posted when the app URL is unknown.
func (s *HandlerAuthGithubInstallationCallbackSuite) Test_NoAppURL() {
	cnf := admin.MustConfig()
	cnf.DomainConfig = &admin.DomainConfig{}
	admin.SetConfig(&cnf)

	s.NotContains(s.installationCallback(), "postMessage")
}

func TestHandlerAuthGithubInstallationCallbackSuite(t *testing.T) {
	suite.Run(t, new(HandlerAuthGithubInstallationCallbackSuite))
}
