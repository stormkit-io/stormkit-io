package hosting

import (
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	pieces := strings.Split(admin.MustConfig().DomainConfig.Dev, "//")
	devDomain := pieces[0]

	if len(pieces) > 1 {
		devDomain = pieces[1]
	}

	s := r.NewService()
	s.NewEndpoint("/").CatchAll(WithHost(HandlerForward), devDomain)

	return s
}

func WithTimeout(h http.Handler) http.Handler {
	timeout := config.Get().HTTPTimeouts.ReadTimeout
	regular := http.TimeoutHandler(h, timeout, "timeout")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http.TimeoutHandler buffers the entire response, which is incompatible
		// with SSE streams. Bypass it only for the known streaming endpoint to
		// prevent clients from opting out of timeouts via the Accept header alone.
		if r.Method == http.MethodGet && r.URL.Path == "/v1/mcp" {
			h.ServeHTTP(w, r)
			return
		}

		regular.ServeHTTP(w, r)
	})
}
