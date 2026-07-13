package deployhandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy/deployhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) TestServices() {
	services := shttp.NewRouter().RegisterService(deployhandlers.Services)

	handlers := []string{
		"DELETE:/app/deploy",
		"GET:/app/{did:[0-9]+}/manifest/{deploymentId:[0-9]+}",
		"GET:/my/deployments",
		"POST:/app/deploy",
		"POST:/app/deploy/callback",
		"POST:/app/deployments/publish",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
