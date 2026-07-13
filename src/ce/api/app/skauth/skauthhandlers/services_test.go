package skauthhandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth/skauthhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) Test_Services() {
	services := shttp.NewRouter().RegisterService(skauthhandlers.Services)

	handlers := []string{
		"DELETE:/skauth/users/{id}",
		"GET:/skauth/providers",
		"GET:/skauth/users",
		"POST:/skauth",
		"POST:/skauth/config",
		"PUT:/skauth/users/{id}",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
