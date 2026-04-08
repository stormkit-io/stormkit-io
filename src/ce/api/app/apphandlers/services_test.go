package apphandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) Test_Services() {
	services := shttp.NewRouter().RegisterService(apphandlers.Services)
	s.NotNil(services)

	handlers := []string{
		"DELETE:/app",
		"DELETE:/app/outbound-webhooks",
		"DELETE:/app/{did:[0-9]+}/deploy-trigger",
		"GET:/app/{did:[0-9]+}/outbound-webhooks",
		"GET:/app/{did:[0-9]+}/outbound-webhooks/{wid:[0-9]+}/trigger",
		"GET:/app/{did:[0-9]+}/settings",
		"GET:/apps",
		"GET:/deploy",
		"GET:/hooks/app/{did:[0-9]+}/deploy/{hash}/{env}",
		"POST:/app",
		"POST:/app/outbound-webhooks",
		"POST:/app/proxy",
		"POST:/app/webhooks/{provider:github|gitlab|bitbucket}",
		"POST:/app/webhooks/{provider:github|gitlab|bitbucket}/{secret-id}",
		"POST:/hooks/app/{did:[0-9]+}/deploy/{hash}/{env}",
		"PUT:/app",
		"PUT:/app/deploy-trigger",
		"PUT:/app/outbound-webhooks",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
