package subscriptionhandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user/subscriptionhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) Test_Services_SelfHosted() {
	services := shttp.NewRouter().RegisterService(subscriptionhandlers.Services)

	s.NotNil(services)

	handlers := []string{
		"GET:/billing/checkout",
		"POST:/user/subscription/update",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func TestServicesSuite(t *testing.T) {
	suite.Run(t, new(ServicesSuite))
}
