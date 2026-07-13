package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerEnvPull returns the environment's variable values in plaintext. Values
// are masked in every other response, so this is the dedicated reveal endpoint.
//
// Any team member with access to the environment may reveal the values — this
// matches what they can already do through an API key, and (unlike the masked
// list) editing env vars requires revealing first. Membership is enforced by the
// route's scope middleware before this handler runs.
//
// Revealing plaintext secrets is audited: the audit attributes the caller to the
// team member (JWT) or the API key's token name (SK_ key).
func handlerEnvPull(req *RequestContext) *shttp.Response {
	if req.License().IsEnterprise() {
		err := audit.FromRequestContext(req).
			WithAction(audit.RevealAction, audit.TypeEnv).
			WithEnvID(req.Env.ID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   req.Env.Data.Vars,
	}
}
