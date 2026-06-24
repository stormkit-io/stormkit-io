package functiontriggerhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/apps").
		Handler(shttp.MethodDelete, "/trigger", app.WithApp(handlerFunctionTriggerDelete)).
		Handler(shttp.MethodPatch, "/trigger", app.WithApp(handlerFunctionTriggerUpdate)).
		Handler(shttp.MethodPost, "/trigger", app.WithApp(handlerFunctionTriggerCreate)).
		Handler(shttp.MethodPost, "/trigger/invoke", app.WithApp(handlerFunctionTriggerInvoke)).
		Handler(shttp.MethodGet, "/triggers", app.WithApp(handlerFunctionTriggersGet)).
		Handler(shttp.MethodGet, "/trigger/logs", app.WithApp(handleTriggerLogsGet))

	return s
}
