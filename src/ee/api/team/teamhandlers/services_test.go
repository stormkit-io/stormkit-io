package teamhandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ee/api/team/teamhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) Test_Services() {
	services := shttp.NewRouter().RegisterService(teamhandlers.Services)

	s.NotNil(s)

	handlers := []string{
		"DELETE:/team",
		"DELETE:/team/member",
		"GET:/team/members",
		"GET:/team/stats",
		"GET:/team/stats/domains",
		"PATCH:/team",
		"POST:/team/enroll",
		"POST:/team/invite",
		"POST:/team/migrate",
	}

	s.Equal(handlers, services.HandlerKeys())
}

func (s *ServicesSuite) Test_EE() {
	services := shttp.NewRouter().RegisterService(teamhandlers.Services)
	s.NotNil(services)

	// All handlers are EE only
	for k, fn := range services.HandlerFuncs() {
		s.Equal(
			http.StatusPaymentRequired,
			fn(shttp.NewRequestContext(nil)).Status,
			"handler %s should return 402", k,
		)
	}
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
