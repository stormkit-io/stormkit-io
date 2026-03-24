package instancehandlers

import (
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services installs the user services.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/instance").
		Handler(shttp.MethodGet, "", handlerInstanceDetails)

	s.NewEndpoint("/changelog").
		Handler(shttp.MethodGet, "", handlerChangelog)

	return s
}
