package apphandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerDeployRemovedSuite struct {
	suite.Suite
}

// Test_Gone verifies that old "Deploy with Stormkit" links explain that the
// button was removed instead of acting on the template parameter.
func (s *HandlerDeployRemovedSuite) Test_Gone() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/deploy?template=https%3A%2F%2Fgithub.com%2Fstormkit-io%2Fmonorepo-template-react",
		nil,
	)

	body := response.String()

	s.Equal(http.StatusGone, response.Code)
	s.Equal("text/html; charset=utf-8", response.Header().Get("Content-Type"))
	s.Contains(body, "Deploy button removed")
	s.Contains(body, "Stormkit MCP server")
	s.NotContains(body, "monorepo-template-react")
}

func TestHandlerDeployRemoved(t *testing.T) {
	suite.Run(t, new(HandlerDeployRemovedSuite))
}
