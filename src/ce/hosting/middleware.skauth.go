package hosting

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func WithSKAuth(req *RequestContext) (*shttp.Response, error) {
	// Not enabled
	if req.Host.Config.SKAuth == nil {
		return nil, nil
	}

	if !strings.HasPrefix(req.URL().Path, "/_stormkit/auth") {
		return nil, nil
	}

	var head, content string

	code := req.Query().Get("code")
	status := http.StatusOK

	if code == "" {
		status = http.StatusBadRequest
		content = "code is missing"
	}

	sessionToken := rediscache.Client().Get(req.Context(), code).Val()

	if sessionToken != "" {
		head = fmt.Sprintf(
			`<script>localStorage.setItem('skauth', JSON.stringify('%s'));window.location.href="%s";</script>`,
			sessionToken,
			req.Host.Config.SKAuth.SuccessURL,
		)
	} else {
		content = "invalid session"
	}

	return &shttp.Response{
		Status: status,
		Headers: shttp.HeadersFromMap(map[string]string{
			"Content-Type": "text/html",
		}),
		Data: html.MustRender(html.RenderArgs{
			PageTitle:   "Stormkit - Auth",
			PageContent: content,
			PageHead:    head,
		}),
	}, nil
}
