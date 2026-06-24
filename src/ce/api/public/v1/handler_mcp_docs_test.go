package publicapiv1_test

import (
	"net/http"
	"strings"
	"testing"

	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerMCPDocsSuite struct {
	suite.Suite
}

func (s *HandlerMCPDocsSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerMCPDocsSuite) get() shttptest.Response {
	return shttptest.RequestWithHeaders(s.handler(), shttp.MethodGet, "/mcp", nil, map[string]string{})
}

// The docs page is public — no API key required.
func (s *HandlerMCPDocsSuite) Test_Public_HTML() {
	resp := s.get()

	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Header().Get("Content-Type"), "text/html")

	body := resp.String()
	s.Contains(body, "Stormkit MCP Server")
	s.Contains(body, "/v1/mcp")
	s.Contains(body, "Authorization: Bearer")
}

// Every tool in the manifest is rendered on the page.
func (s *HandlerMCPDocsSuite) Test_Lists_All_Tools() {
	resp := s.get()
	body := resp.String()

	for _, tool := range []string{
		"deploy", "get_deployment", "list_deployments", "create_app",
		"list_apps", "create_environment", "list_domains", "create_team",
	} {
		s.Contains(body, "<code>"+tool+"</code>", "expected tool %q on the page", tool)
	}
}

// Database tools are gated to self-hosted/development builds, mirroring the
// JSON-RPC tools/list manifest.
func (s *HandlerMCPDocsSuite) Test_Database_Tools_Gated() {
	config.SetIsStormkitCloud(true)
	defer config.SetIsStormkitCloud(false)

	resp := s.get()
	body := resp.String()
	s.False(strings.Contains(body, "enable_database_integration"),
		"database tools must not appear on Stormkit Cloud")
}

func TestHandlerMCPDocsSuite(t *testing.T) {
	suite.Run(t, new(HandlerMCPDocsSuite))
}
