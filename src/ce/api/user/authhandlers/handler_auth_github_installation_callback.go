package authhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAuthGithubInstallationCallback is responsible for handling the provider registration/login flow.
// It returns an html response. This will post a message to the parent window with the json bytes.
func handlerAuthGithubInstallationCallback(req *shttp.RequestContext) *shttp.Response {
	return cbResponse(http.StatusOK).
		html("Installation complete").
		json(jsonMsg{"success": true}).
		send()
}
