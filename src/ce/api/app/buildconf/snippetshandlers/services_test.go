package snippetshandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/snippetshandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) Test_Services() {
	services := shttp.NewRouter().RegisterService(snippetshandlers.Services)
	s.NotNil(services)

	handlers := []string{
		"DELETE:/snippets",
		"DELETE:/snippets/analytics",
		"GET:/snippets",
		"POST:/snippets",
		"POST:/snippets/analytics",
		"PUT:/snippets",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
