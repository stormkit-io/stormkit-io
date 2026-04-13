package hosting

import (
	"fmt"
	"net/http"
	"strings"

	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func WithSKAuth(req *RequestContext) (*shttp.Response, error) {
	// Not enabled
	if req.Host.Config.SKAuth == nil {
		return nil, nil
	}

	path := req.URL().Path

	if !strings.HasPrefix(path, "/_stormkit/auth") {
		return nil, nil
	}

	if path == "/_stormkit/auth/register" {
		if req.Method != http.MethodPost {
			return &shttp.Response{
				Status: http.StatusMethodNotAllowed,
				Data:   map[string]any{"errors": []string{"method not allowed"}},
			}, nil
		}

		// Inject the environment ID from the host config so the handler can
		// look up the environment without requiring it in the request body.
		reqURL := req.URL()
		q := reqURL.Query()
		q.Set("envId", req.Host.Config.EnvID.String())
		reqURL.RawQuery = q.Encode()
		req.ResetQuery()

		return publicapiv1.HandlerAuthEmailRegister(req.RequestContext), nil
	}

	var head, content string

	code := req.Query().Get("code")
	status := http.StatusOK

	if code == "" {
		status = http.StatusBadRequest
		content = "code is missing"
	} else if sessionToken := rediscache.Client().Get(req.Context(), code).Val(); sessionToken != "" {
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
