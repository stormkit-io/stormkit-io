package authwallhandlers_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall/authwallhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) Test_Services() {
	services := shttp.NewRouter().RegisterService(authwallhandlers.Services)

	handlers := []string{
		"DELETE:/team/auth-wall/logins",
		"GET:/auth-wall/config",
		"GET:/auth-wall/magic",
		"GET:/team/auth-wall/envs",
		"GET:/team/auth-wall/logins",
		"PATCH:/team/auth-wall/logins",
		"POST:/auth-wall/config",
		"POST:/auth-wall/login",
		"POST:/auth-wall/magic",
		"POST:/auth-wall/magic/verify",
		"POST:/team/auth-wall/logins",
		"PUT:/team/auth-wall/envs",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func (s *ServicesSuite) Test_EE() {
	services := shttp.NewRouter().RegisterService(authwallhandlers.Services)
	s.NotNil(services)

	// Most handlers are EE-only, except for the visitor login endpoints
	for k, fn := range services.HandlerFuncs() {
		res := fn(&shttp.RequestContext{
			Request: &http.Request{URL: &url.URL{}},
		})

		// Login is not EE-gated; without a valid form or link token it is rejected.
		if k == "POST:/auth-wall/login" || strings.HasPrefix(k, "POST:/auth-wall/magic") || k == "GET:/auth-wall/magic" {
			s.Equal(http.StatusBadRequest, res.Status, "handler %s should reject a missing token", k)
			continue
		}

		s.Equal(http.StatusPaymentRequired, res.Status, "handler %s should return 402", k)
	}
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
