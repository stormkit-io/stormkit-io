package hosting

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/maintenance"
	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// WithMaintenance blocks public traffic with a maintenance page while the
// environment's maintenance mode is turned on. It returns a 503 so search
// engines treat the downtime as temporary.
func WithMaintenance(req *RequestContext) (*shttp.Response, error) {
	if req.Host.Config.Maintenance != maintenance.StatusOn {
		return nil, nil
	}

	content := html.MustRender(html.RenderArgs{
		PageTitle:   "Under maintenance",
		PageContent: html.Templates["maintenance"],
	})

	return &shttp.Response{
		Status: http.StatusServiceUnavailable,
		Data:   content,
		Headers: http.Header{
			"Content-Type": []string{"text/html; charset=utf-8"},
			"Retry-After":  []string{"300"},
		},
	}, nil
}
