package audithandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ee/api/audit/audithandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (s *ServicesSuite) TestServices() {
	services := shttp.NewRouter().RegisterService(audithandlers.Services)

	handlers := []string{
		"GET:/audits",
	}

	s.Equal(handlers, services.Handlers())
}

func (s *ServicesSuite) Test_EE() {
	services := shttp.NewRouter().RegisterService(audithandlers.Services)
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
