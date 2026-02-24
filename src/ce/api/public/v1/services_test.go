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
		"DELETE:/v1/domains",
		"DELETE:/v1/domains/cert",
		"DELETE:/v1/env",
		"DELETE:/v1/snippets",
		"GET:/v1/app/config",
		"GET:/v1/auth",
		"GET:/v1/auth/callback",
		"GET:/v1/auth/session",
		"GET:/v1/domains",
		"GET:/v1/env/pull",
		"GET:/v1/redirects",
		"GET:/v1/snippets",
		"POST:/v1/domains",
		"POST:/v1/env",
		"POST:/v1/mail",
		"POST:/v1/redirects",
		"POST:/v1/snippets",
		"PUT:/v1/domains/cert",
		"PUT:/v1/snippets",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func (s *ServicesSuite) Test_Services_StormkitCloud() {
	config.SetIsStormkitCloud(true)
	services := shttp.NewRouter().RegisterService(publicapiv1.Services)
	s.NotNil(services)

	handlers := []string{
		"DELETE:/v1/domains",
		"DELETE:/v1/domains/cert",
		"DELETE:/v1/env",
		"DELETE:/v1/snippets",
		"GET:/v1/app/config",
		"GET:/v1/domains",
		"GET:/v1/env/pull",
		"GET:/v1/license",
		"GET:/v1/license/check",
		"GET:/v1/redirects",
		"GET:/v1/snippets",
		"POST:/v1/domains",
		"POST:/v1/env",
		"POST:/v1/mail",
		"POST:/v1/redirects",
		"POST:/v1/snippets",
		"PUT:/v1/domains/cert",
		"PUT:/v1/snippets",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func (s *ServicesSuite) Test_EE() {
	services := shttp.NewRouter().RegisterService(publicapiv1.Services)
	s.NotNil(services)

	statusMap := map[string]int{
		"PUT:/v1/domains/cert":    http.StatusPaymentRequired,
		"DELETE:/v1/domains/cert": http.StatusPaymentRequired,
		"GET:/v1/license":         http.StatusOK,
		"GET:/v1/license/check":   http.StatusBadRequest,
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
