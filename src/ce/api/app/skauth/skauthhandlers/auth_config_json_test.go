package skauthhandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth/skauthhandlers"
	"github.com/stretchr/testify/suite"
)

type AuthConfigJSONSuite struct {
	suite.Suite
}

// Test_OAuthServerEnabled_RequiresAuthStatus pins the effective-value contract:
// the OAuth server is inert while Stormkit Auth is off, so the rendered flag
// must not claim otherwise.
func (s *AuthConfigJSONSuite) Test_OAuthServerEnabled_RequiresAuthStatus() {
	conf := &buildconf.SKAuthConf{
		Status:      false,
		OAuthServer: &buildconf.OAuthServerConf{Enabled: true},
	}

	s.Equal(false, skauthhandlers.AuthConfigJSON(conf)["oauthServerEnabled"])

	conf.Status = true
	s.Equal(true, skauthhandlers.AuthConfigJSON(conf)["oauthServerEnabled"])
}

func (s *AuthConfigJSONSuite) Test_NilConf() {
	out := skauthhandlers.AuthConfigJSON(nil)

	s.Equal(false, out["status"])
	s.Equal(false, out["oauthServerEnabled"])
}

func TestAuthConfigJSONSuite(t *testing.T) {
	suite.Run(t, &AuthConfigJSONSuite{})
}
