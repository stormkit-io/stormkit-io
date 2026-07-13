package apikeyhandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey/apikeyhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) Test_Services() {
	services := shttp.NewRouter().RegisterService(apikeyhandlers.Services)
	s.NotNil(services)

	handlers := []string{
		"DELETE:/api-keys",
		"GET:/api-keys",
		"POST:/api-keys",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
