package analyticshandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics/analyticshandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) Test_Handlers() {
	services := shttp.NewRouter().RegisterService(analyticshandlers.Services)
	s.NotNil(services)

	s.Equal([]string{
		"GET:/analytics/countries",
		"GET:/analytics/events",
		"GET:/analytics/events/breakdown",
		"GET:/analytics/events/properties",
		"GET:/analytics/paths",
		"GET:/analytics/referrers",
		"GET:/analytics/visitors",
	}, services.HandlerKeys())
}

func (s *ServicesSuite) Test_EE() {
	services := shttp.NewRouter().RegisterService(analyticshandlers.Services)
	s.NotNil(services)

	// All handlers are EE only
	for k, fn := range services.HandlerFuncs() {
		s.Equal(
			http.StatusPaymentRequired,
			fn(&shttp.RequestContext{}).Status,
			"handler %s should return 402", k,
		)
	}
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
