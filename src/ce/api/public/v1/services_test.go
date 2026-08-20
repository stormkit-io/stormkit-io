package publicapiv1_test

import (
	"net/http"
	"testing"

	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) Test_Services_SelfHosted() {
	config.SetIsSelfHosted(true)
	services := shttp.NewRouter().RegisterService(publicapiv1.Services)
	s.NotNil(services)

	handlers := []string{
		"DELETE:/v1/deployments/{id:[0-9]+}",
		"DELETE:/v1/domains",
		"DELETE:/v1/domains/cert",
		"DELETE:/v1/env",
		"DELETE:/v1/schema",
		"DELETE:/v1/snippets",
		"DELETE:/v1/trigger",
		"GET:/v1/access-logs",
		"GET:/v1/app",
		"GET:/v1/app/config",
		"GET:/v1/app/{appId:[0-9]+}",
		"GET:/v1/apps",
		"GET:/v1/auth/config",
		"GET:/v1/auth/providers",
		"GET:/v1/deployments",
		"GET:/v1/deployments/{id:[0-9]+}",
		"GET:/v1/deployments/{id:[0-9]+}/poll",
		"GET:/v1/deployments/{id:[0-9]+}/runtime-logs",
		"GET:/v1/domains",
		"GET:/v1/env/pull",
		"GET:/v1/envs",
		"GET:/v1/mailer/config",
		"GET:/v1/mailer/emails",
		"GET:/v1/mcp",
		"GET:/v1/redirects",
		"GET:/v1/schema",
		"GET:/v1/snippets",
		"GET:/v1/teams",
		"GET:/v1/trigger/logs",
		"GET:/v1/triggers",
		"PATCH:/v1/trigger",
		"POST:/v1/app",
		"POST:/v1/auth/config",
		"POST:/v1/auth/providers",
		"POST:/v1/deploy",
		"POST:/v1/deployments/{id:[0-9]+}/prioritize",
		"POST:/v1/deployments/{id:[0-9]+}/publish",
		"POST:/v1/deployments/{id:[0-9]+}/restart",
		"POST:/v1/deployments/{id:[0-9]+}/stop",
		"POST:/v1/domains",
		"POST:/v1/env",
		"POST:/v1/mail",
		"POST:/v1/mailer/config",
		"POST:/v1/mcp",
		"POST:/v1/redirects",
		"POST:/v1/schema",
		"POST:/v1/schema/configure",
		"POST:/v1/snippets",
		"POST:/v1/teams",
		"POST:/v1/trigger",
		"POST:/v1/trigger/invoke",
		"POST:/v1/volumes",
		"PUT:/v1/domains",
		"PUT:/v1/domains/cert",
		"PUT:/v1/env",
		"PUT:/v1/snippets",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func (s *ServicesSuite) Test_Services_StormkitCloud() {
	config.SetIsStormkitCloud(true)
	services := shttp.NewRouter().RegisterService(publicapiv1.Services)
	s.NotNil(services)

	handlers := []string{
		"DELETE:/v1/deployments/{id:[0-9]+}",
		"DELETE:/v1/domains",
		"DELETE:/v1/domains/cert",
		"DELETE:/v1/env",
		"DELETE:/v1/snippets",
		"DELETE:/v1/trigger",
		"GET:/v1/access-logs",
		"GET:/v1/app",
		"GET:/v1/app/config",
		"GET:/v1/app/{appId:[0-9]+}",
		"GET:/v1/apps",
		"GET:/v1/deployments",
		"GET:/v1/deployments/{id:[0-9]+}",
		"GET:/v1/deployments/{id:[0-9]+}/poll",
		"GET:/v1/deployments/{id:[0-9]+}/runtime-logs",
		"GET:/v1/domains",
		"GET:/v1/env/pull",
		"GET:/v1/envs",
		"GET:/v1/license/check",
		"GET:/v1/mailer/config",
		"GET:/v1/mailer/emails",
		"GET:/v1/mcp",
		"GET:/v1/redirects",
		"GET:/v1/snippets",
		"GET:/v1/teams",
		"GET:/v1/trigger/logs",
		"GET:/v1/triggers",
		"PATCH:/v1/trigger",
		"POST:/v1/app",
		"POST:/v1/deploy",
		"POST:/v1/deployments/{id:[0-9]+}/prioritize",
		"POST:/v1/deployments/{id:[0-9]+}/publish",
		"POST:/v1/deployments/{id:[0-9]+}/restart",
		"POST:/v1/deployments/{id:[0-9]+}/stop",
		"POST:/v1/domains",
		"POST:/v1/env",
		"POST:/v1/license/check",
		"POST:/v1/mail",
		"POST:/v1/mailer/config",
		"POST:/v1/mcp",
		"POST:/v1/redirects",
		"POST:/v1/snippets",
		"POST:/v1/teams",
		"POST:/v1/trigger",
		"POST:/v1/trigger/invoke",
		"POST:/v1/volumes",
		"PUT:/v1/domains",
		"PUT:/v1/domains/cert",
		"PUT:/v1/env",
		"PUT:/v1/snippets",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func (s *ServicesSuite) Test_EE() {
	services := shttp.NewRouter().RegisterService(publicapiv1.Services)
	s.NotNil(services)

	statusMap := map[string]int{
		"GET:/v1/access-logs":                          http.StatusForbidden,
		"GET:/v1/apps":                                 http.StatusForbidden,
		"GET:/v1/app":                                  http.StatusForbidden,
		"GET:/v1/app/{appId:[0-9]+}":                   http.StatusForbidden,
		"POST:/v1/app":                                 http.StatusForbidden,
		"GET:/v1/app/config":                           http.StatusForbidden,
		"GET:/v1/deployments/{id:[0-9]+}":              http.StatusForbidden,
		"GET:/v1/deployments/{id:[0-9]+}/runtime-logs": http.StatusForbidden,
		"GET:/v1/deployments/{id:[0-9]+}/poll":         http.StatusForbidden,
		"GET:/v1/deployments":                          http.StatusForbidden,
		"POST:/v1/deployments/{id:[0-9]+}/prioritize":  http.StatusForbidden,
		"POST:/v1/deployments/{id:[0-9]+}/publish":     http.StatusForbidden,
		"POST:/v1/deployments/{id:[0-9]+}/restart":     http.StatusForbidden,
		"POST:/v1/deployments/{id:[0-9]+}/stop":        http.StatusForbidden,
		"DELETE:/v1/deployments/{id:[0-9]+}":           http.StatusForbidden,
		"POST:/v1/deploy":                              http.StatusForbidden,
		"POST:/v1/env":                                 http.StatusForbidden,
		"PUT:/v1/env":                                  http.StatusForbidden,
		"GET:/v1/env/pull":                             http.StatusForbidden,
		"GET:/v1/envs":                                 http.StatusForbidden,
		"GET:/v1/env":                                  http.StatusForbidden,
		"DELETE:/v1/env":                               http.StatusForbidden,
		"GET:/v1/domains":                              http.StatusForbidden,
		"POST:/v1/domains":                             http.StatusForbidden,
		"PUT:/v1/domains":                              http.StatusForbidden,
		"DELETE:/v1/domains":                           http.StatusForbidden,
		"PUT:/v1/domains/cert":                         http.StatusPaymentRequired,
		"DELETE:/v1/domains/cert":                      http.StatusPaymentRequired,
		"GET:/v1/redirects":                            http.StatusForbidden,
		"POST:/v1/redirects":                           http.StatusForbidden,
		"GET:/v1/license":                              http.StatusOK,
		"GET:/v1/license/check":                        http.StatusBadRequest,
		"POST:/v1/volumes":                             http.StatusForbidden,
		"GET:/v1/teams":                                http.StatusForbidden,
		"POST:/v1/teams":                               http.StatusForbidden,
		"POST:/v1/mcp":                                 http.StatusForbidden,
		"GET:/v1/mcp":                                  http.StatusForbidden,
		"GET:/v1/auth/config":                          http.StatusForbidden,
		"POST:/v1/auth/config":                         http.StatusForbidden,
		"GET:/v1/auth/providers":                       http.StatusForbidden,
		"POST:/v1/auth/providers":                      http.StatusForbidden,
		"POST:/v1/mail":                                http.StatusForbidden,
		"GET:/v1/mailer/config":                        http.StatusForbidden,
		"GET:/v1/mailer/emails":                        http.StatusForbidden,
		"POST:/v1/mailer/config":                       http.StatusForbidden,
		"GET:/v1/snippets":                             http.StatusForbidden,
		"POST:/v1/snippets":                            http.StatusForbidden,
		"PUT:/v1/snippets":                             http.StatusForbidden,
		"DELETE:/v1/snippets":                          http.StatusForbidden,
		"POST:/v1/trigger":                             http.StatusForbidden,
		"PATCH:/v1/trigger":                            http.StatusForbidden,
		"DELETE:/v1/trigger":                           http.StatusForbidden,
		"POST:/v1/trigger/invoke":                      http.StatusForbidden,
		"GET:/v1/trigger/logs":                         http.StatusForbidden,
		"GET:/v1/triggers":                             http.StatusForbidden,
	}

	for k, fn := range services.HandlerFuncs() {
		status := statusMap[k]

		// Default status
		if status == 0 {
			status = http.StatusUnauthorized
		}

		s.Equal(status, fn(shttp.NewRequestContext(nil)).Status, "handler %s should return %d", k, status)
	}
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
