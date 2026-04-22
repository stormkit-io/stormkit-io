package hosting

import (
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func WithTimeout(h http.Handler) http.Handler {
	timeout := config.Get().HTTPTimeouts.ReadTimeout
	regular := http.TimeoutHandler(h, timeout, "timeout")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bypass timeout for SSE streams — http.TimeoutHandler buffers the
		// entire response which is incompatible with streaming.
		if r.Method == http.MethodGet && r.URL.Path == "/v1/mcp" {
			h.ServeHTTP(w, r)
			return
		}

		// Bypass timeout for requests with a body — large uploads can take
		// longer than the configured read timeout.
		if r.Body != nil && r.Body != http.NoBody {
			h.ServeHTTP(w, r)
			return
		}

		regular.ServeHTTP(w, r)
	})
}

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
