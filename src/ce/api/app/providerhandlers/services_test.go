package providerhandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/providerhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) TestServices() {
	services := shttp.NewRouter().RegisterService(providerhandlers.Services)

	s.NotNil(s)

	handlers := []string{
		"GET:/provider/{provider:github|gitlab|bitbucket}/accounts",
		"GET:/provider/{provider:github|gitlab|bitbucket}/repos",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
