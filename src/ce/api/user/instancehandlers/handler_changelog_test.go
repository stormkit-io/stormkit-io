package instancehandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user/instancehandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerChangelogSuite struct {
	suite.Suite
}

// Test_Success verifies that the handler returns 200 with raw markdown
// sourced from the embedded content/blog/whats-new.md file.
func (s *HandlerChangelogSuite) Test_Success() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(instancehandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/changelog",
		nil,
	)

	s.Equal(http.StatusOK, response.Code)

	data := response.Map()
	markdown, ok := data["markdown"].(string)
	s.True(ok)
	s.NotEmpty(markdown)
	// Frontmatter should be stripped
	s.NotContains(markdown, "---")
}

func TestHandlerChangelog(t *testing.T) {
	suite.Run(t, &HandlerChangelogSuite{})
}
