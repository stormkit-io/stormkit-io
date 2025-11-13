package adminhandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) Test_Services_SelfHosted() {
	config.SetIsSelfHosted(true)
	services := shttp.NewRouter().RegisterService(adminhandlers.Services)

	s.NotNil(services)

	handlers := []string{
		"GET:/admin/domains",
		"GET:/admin/git/details",
		"GET:/admin/git/github/callback",
		"GET:/admin/system/mise",
		"GET:/admin/system/proxies",
		"GET:/admin/system/runtimes",
		"GET:/admin/users/sign-up-mode",
		"POST:/admin/domains",
		"POST:/admin/git/configure",
		"POST:/admin/git/github/manifest",
		"POST:/admin/jobs/remove-old-artifacts",
		"POST:/admin/jobs/sync-analytics",
		"POST:/admin/license",
		"POST:/admin/system/mise",
		"POST:/admin/system/runtimes",
		"POST:/admin/users/sign-up-mode",
		"PUT:/admin/system/proxies",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func (s *ServicesSuite) Test_Services_Cloud() {
	config.SetIsStormkitCloud(true)
	services := shttp.NewRouter().RegisterService(adminhandlers.Services)

	s.NotNil(services)

	handlers := []string{
		"DELETE:/admin/cloud/app",
		"GET:/admin/cloud/app",
		"GET:/admin/domains",
		"GET:/admin/git/details",
		"GET:/admin/git/github/callback",
		"GET:/admin/system/mise",
		"GET:/admin/system/proxies",
		"GET:/admin/system/runtimes",
		"GET:/admin/users/sign-up-mode",
		"POST:/admin/cloud/impersonate",
		"POST:/admin/cloud/license",
		"POST:/admin/domains",
		"POST:/admin/git/configure",
		"POST:/admin/git/github/manifest",
		"POST:/admin/jobs/remove-old-artifacts",
		"POST:/admin/jobs/sync-analytics",
		"POST:/admin/license",
		"POST:/admin/system/mise",
		"POST:/admin/system/runtimes",
		"POST:/admin/users/sign-up-mode",
		"PUT:/admin/system/proxies",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
