package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerMailerConfigGet returns the SMTP configuration for the environment.
// The password is never included; a placeholder marks that one is stored.
func handlerMailerConfigGet(req *RequestContext) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"config": req.Env.MailerConf.JSON()},
	}
}
