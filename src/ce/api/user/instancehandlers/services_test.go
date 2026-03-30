package instancehandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user/instancehandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func (ss *ServicesSuite) Test_Services() {
	r := shttp.NewRouter()
	s := r.RegisterService(instancehandlers.Services)

	ss.NotNil(s)

	handlers := []string{
		"GET:/changelog",
		"GET:/instance",
	}

	ss.Equal(handlers, s.HandlerKeys())
}

func TestServices(t *testing.T) {
	suite.Run(t, &ServicesSuite{})
}
