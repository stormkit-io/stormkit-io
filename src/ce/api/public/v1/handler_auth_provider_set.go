package publicapiv1

import (
	"errors"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth/skauthhandlers"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAuthProviderSet enables or updates a single sign-in provider for the
// environment. Client secrets are write-only.
func handlerAuthProviderSet(req *RequestContext) *shttp.Response {
	data := skauthhandlers.AuthUpsertRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	err := skauthhandlers.UpsertProvider(req.Context(), skauthhandlers.UpsertProviderParams{
		Env:   req.Env,
		AppID: req.App.ID,
		Data:  data,
	})

	if err != nil {
		var verr *skauthhandlers.ProviderValidationError

		if errors.As(err, &verr) {
			return shttp.BadRequest(map[string]any{"error": verr.Message})
		}

		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		// The client secret and the provider's stored credentials stay out of
		// the diff; the from address and enabled flag are what redirect or
		// disable end-user sign-in, so those are worth recording.
		diff := &audit.Diff{
			New: audit.DiffFields{
				AuthProviderName:        data.ProviderName,
				AuthProviderStatus:      data.Status,
				AuthProviderFromAddress: data.FromAddress,
				AuthProviderClientID:    data.ClientID,
			},
		}

		if err := audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeAuthProvider).
			WithDiff(diff).
			WithEnvID(req.Env.ID).
			Insert(); err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}
