package buildconfhandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/buildconfhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) Test_Services() {
	services := shttp.NewRouter().RegisterService(buildconfhandlers.Services)

	handlers := []string{
		"DELETE:/app/env",
		"GET:/app/{did:[0-9]+}/envs/{env:[0-9a-zA-Z-]+}",
		"POST:/app/env",
		"PUT:/app/env",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
