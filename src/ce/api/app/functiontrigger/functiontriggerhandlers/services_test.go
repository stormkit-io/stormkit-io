package functiontriggerhandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger/functiontriggerhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) Test_Services() {
	services := shttp.NewRouter().RegisterService(functiontriggerhandlers.Services)
	s.NotNil(services)

	handlers := []string{
		"DELETE:/apps/trigger",
		"GET:/apps/trigger/logs",
		"GET:/apps/triggers",
		"PATCH:/apps/trigger",
		"POST:/apps/trigger",
		"POST:/apps/trigger/invoke",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
