package hosting

import (
	"context"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// WithTimeout enforces per-read idle deadlines on request bodies via context
// cancellation, equivalent to nginx's client_body_timeout. Works for both
// HTTP/1.1 and HTTP/2. No handler-level timeout is applied; ReadHeaderTimeout
// on the server covers the header phase and ResponseHeaderTimeout on the proxy
// client covers upstream stalls.
func WithTimeout(h http.Handler) http.Handler {
	cfg := config.Get().HTTPTimeouts

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.ClientBodyTimeout > 0 && r.Body != nil && r.Body != http.NoBody {
			ctx, cancel := context.WithCancel(r.Context())
			r = r.WithContext(ctx)
			r.Body = shttp.NewIdleTimeoutReader(r.Body, cancel, cfg.ClientBodyTimeout)
		}

		h.ServeHTTP(w, r)
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
