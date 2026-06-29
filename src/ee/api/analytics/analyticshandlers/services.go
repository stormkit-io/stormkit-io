package analyticshandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	opts := &app.Opts{Env: true}

	// Endpoints with environment id.
	// Try to move previous endpoints to this endpoint
	s.NewEndpoint("/analytics").
		Middleware(user.WithEE).
		Handler(shttp.MethodGet, "/visitors", app.WithApp(handlerVisitors, opts)).
		Handler(shttp.MethodGet, "/referrers", app.WithApp(handlerTopReferrers, opts)).
		Handler(shttp.MethodGet, "/paths", app.WithApp(handlerTopPaths, opts)).
		Handler(shttp.MethodGet, "/countries", app.WithApp(handlerCountries, opts)).
		Handler(shttp.MethodGet, "/events", app.WithApp(handlerEvents, opts)).
		Handler(shttp.MethodGet, "/events/properties", app.WithApp(handlerEventProperties, opts)).
		Handler(shttp.MethodGet, "/events/breakdown", app.WithApp(handlerEventBreakdown, opts))

	return s
}
