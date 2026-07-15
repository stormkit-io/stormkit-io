package hosting_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	hosting "github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type OAuthChallengeSuite struct {
	suite.Suite
}

func TestOAuthChallengeSuite(t *testing.T) {
	suite.Run(t, new(OAuthChallengeSuite))
}

func (s *OAuthChallengeSuite) req(oauthEnabled bool) *hosting.RequestContext {
	return s.reqWithPath(oauthEnabled, "")
}

func (s *OAuthChallengeSuite) reqWithPath(oauthEnabled bool, resourcePath string) *hosting.RequestContext {
	return &hosting.RequestContext{
		Host: &hosting.Host{
			Name: "app.example.com",
			Config: &appconf.Config{
				SKAuth: &buildconf.SKAuthConf{
					Status: true,
					OAuthServer: &buildconf.OAuthServerConf{
						Enabled:      oauthEnabled,
						ResourcePath: resourcePath,
					},
				},
			},
		},
	}
}

const expectedChallenge = `Bearer resource_metadata="https://app.example.com/.well-known/oauth-protected-resource", scope="openid email profile offline_access"`

func (s *OAuthChallengeSuite) Test_AddsChallenge_On401_WhenEnabled() {
	res := hosting.InjectOAuthChallenge(s.req(true), &shttp.Response{Status: http.StatusUnauthorized})

	s.Equal(expectedChallenge, res.Headers.Get("WWW-Authenticate"))
}

// With an MCP path configured, the challenge must point at the RFC 9728
// path-aware metadata document so Claude probes the right resource.
func (s *OAuthChallengeSuite) Test_Challenge_IncludesResourcePath() {
	res := hosting.InjectOAuthChallenge(s.reqWithPath(true, "/mcp"), &shttp.Response{Status: http.StatusUnauthorized})

	s.Equal(
		`Bearer resource_metadata="https://app.example.com/.well-known/oauth-protected-resource/mcp", scope="openid email profile offline_access"`,
		res.Headers.Get("WWW-Authenticate"),
	)
}

func (s *OAuthChallengeSuite) Test_NoChallenge_WhenOAuthDisabled() {
	res := hosting.InjectOAuthChallenge(s.req(false), &shttp.Response{Status: http.StatusUnauthorized})

	s.Empty(res.Headers.Get("WWW-Authenticate"))
}

func (s *OAuthChallengeSuite) Test_NoChallenge_OnNon401() {
	res := hosting.InjectOAuthChallenge(s.req(true), &shttp.Response{Status: http.StatusOK})

	s.Empty(res.Headers.Get("WWW-Authenticate"))
}

// An app that speaks OAuth itself and sets its own challenge must not be
// overridden by the edge default.
func (s *OAuthChallengeSuite) Test_DoesNotOverrideExistingChallenge() {
	res := &shttp.Response{
		Status:  http.StatusUnauthorized,
		Headers: shttp.HeadersFromMap(map[string]string{"WWW-Authenticate": "Bearer realm=\"custom\""}),
	}

	out := hosting.InjectOAuthChallenge(s.req(true), res)

	s.Equal(`Bearer realm="custom"`, out.Headers.Get("WWW-Authenticate"))
}

func (s *OAuthChallengeSuite) Test_NilResponse_NoPanic() {
	s.Nil(hosting.InjectOAuthChallenge(s.req(true), nil))
}
