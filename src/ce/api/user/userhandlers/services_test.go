package userhandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user/userhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) TestServices_SelfHosted() {
	config.SetIsSelfHosted(true)
	defer config.SetIsSelfHosted(false)

	services := shttp.NewRouter().RegisterService(userhandlers.Services)

	s.NotNil(services)

	handlers := []string{
		"DELETE:/user",
		"GET:/user",
		"GET:/user/emails",
		"PUT:/user/access-token",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func (s *ServicesSuite) TestServices_StormkitCloud() {
	config.SetIsStormkitCloud(true)
	defer config.SetIsStormkitCloud(false)

	services := shttp.NewRouter().RegisterService(userhandlers.Services)

	s.NotNil(s)

	handlers := []string{
		"DELETE:/user",
		"GET:/user",
		"GET:/user/emails",
		"PUT:/user/access-token",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
