package volumeshandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes/volumeshandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) TestServices() {
	services := shttp.NewRouter().RegisterService(volumeshandlers.Services)
	s.NotNil(services)

	handlers := []string{
		"DELETE:/volumes",
		"GET:/volumes",
		"GET:/volumes/config",
		"GET:/volumes/download",
		"GET:/volumes/download/url",
		"GET:/volumes/file/{hash}",
		"GET:/volumes/size",
		"POST:/volumes",
		"POST:/volumes/config",
		"POST:/volumes/visibility",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
