package authhandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user/authhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) Test_Services() {
	services := shttp.NewRouter().RegisterService(authhandlers.Services)
	s.NotNil(services)

	handlers := []string{
		"GET:/auth/github/installation",
		"GET:/auth/providers",
		"GET:/auth/{provider:github|gitlab|bitbucket}",
		"GET:/auth/{provider:github|gitlab|bitbucket}/callback",
		"POST:/auth/admin/login",
		"POST:/auth/admin/register",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
