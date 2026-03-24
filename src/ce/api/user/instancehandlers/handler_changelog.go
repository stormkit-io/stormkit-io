package instancehandlers

import (
	"net/http"
	"strings"

	skcontent "github.com/stormkit-io/stormkit-io/content"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerChangelog returns the What's New changelog as raw markdown.
func handlerChangelog(req *shttp.RequestContext) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"markdown": string(stripFrontmatter(skcontent.WhatsNew))},
	}
}

// stripFrontmatter removes YAML front matter delimited by "---" from the start of the content.
func stripFrontmatter(content []byte) []byte {
	s := string(content)

	if !strings.HasPrefix(s, "---") {
		return content
	}

	// Find the closing "---"
	rest := s[3:]
	idx := strings.Index(rest, "\n---")

	if idx == -1 {
		return content
	}

	return []byte(strings.TrimSpace(rest[idx+4:]))
}
