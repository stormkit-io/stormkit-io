package schemahandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/schemahandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) Test_Services() {
	services := shttp.NewRouter().RegisterService(schemahandlers.Services)
	s.NotNil(services)

	handlers := []string{
		"GET:/schema",
		"POST:/schema",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
