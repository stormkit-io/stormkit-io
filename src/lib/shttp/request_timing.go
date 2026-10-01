package shttp

import (
	"context"
	"net/http"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/tracking"
)

type requestTimingKey struct{}

// requestTiming is shared between the outermost middleware, which owns the
// clock, and CatchAll, which decides whether the request is one the metrics
// cover and knows its status.
type requestTiming struct {
	start   time.Time
	tracked bool
	status  int
}

// requestTimingHandler times the whole request from outside every other
// middleware, so the body transfer is included whichever path it took: through
// the gzip middleware, which flushes after the handler returns, or straight to
// the connection.
func requestTimingHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timing := &requestTiming{start: time.Now()}

		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestTimingKey{}, timing)))

		if timing.tracked {
			tracking.RecordRequestTotalTime(r, timing.status, time.Since(timing.start))
		}
	})
}

// WithRequestTiming records the end-to-end time of the requests CatchAll
// tracks. It must be registered last, so it wraps the compression middleware.
func (r *Router) WithRequestTiming() *Router {
	r.RegisterMiddleware(requestTimingHandler)
	return r
}

// trackRequestTotal marks the request as one whose end-to-end time is to be
// recorded once the outermost middleware sees it finish.
func trackRequestTotal(ctx context.Context, status int) {
	if timing, ok := ctx.Value(requestTimingKey{}).(*requestTiming); ok {
		timing.tracked = true
		timing.status = status
	}
}
