package redirectshandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects/redirectshandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (ss *ServicesSuite) Test_Services() {
	r := shttp.NewRouter()
	s := r.RegisterService(redirectshandlers.Services)

	ss.NotNil(s)

	handlers := []string{
		"POST:/redirects/playground",
	}

	ss.Equal(handlers, s.Handlers())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
