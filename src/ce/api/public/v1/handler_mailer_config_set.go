package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/mailerhandlers"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerMailerConfigSet patches the SMTP configuration for the environment.
// Only the fields present in the request body are changed; the password is
// write-only and is never echoed back.
func handlerMailerConfigSet(req *RequestContext) *shttp.Response {
	data := mailerhandlers.ConfigUpdateRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	config := req.Env.MailerConf
	old := buildconf.MailerConf{}

	if config == nil {
		config = &buildconf.MailerConf{}
	} else {
		old = *config
	}

	config.EnvID = req.Env.ID

	// ApplyConfigUpdate returns a *shttperr.ValidationError, which shttp.Error
	// renders as a 400 with the field errors.
	if err := mailerhandlers.ApplyConfigUpdate(config, data); err != nil {
		return shttp.Error(err)
	}

	if err := buildconf.MailerStore().UpsertConfig(req.Context(), config); err != nil {
		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		// Record who repointed the mail relay, never the credential itself:
		// changing the host is enough to redirect an app's sign-in email.
		diff := &audit.Diff{
			Old: audit.DiffFields{
				MailerHost:     old.Host,
				MailerPort:     old.Port,
				MailerUsername: old.Username,
			},
			New: audit.DiffFields{
				MailerHost:     config.Host,
				MailerPort:     config.Port,
				MailerUsername: config.Username,
			},
		}

		// A rotation that touches nothing else leaves Old and New equal, and
		// Insert() drops a diff that has not changed - so the one write that
		// replaces the sending credential would go unrecorded. Flag it
		// instead of storing the secret.
		if old.Password != config.Password {
			changed := true
			diff.New.MailerPasswordChanged = &changed
		}

		err := audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeMailer).
			WithDiff(diff).
			WithEnvID(req.Env.ID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"config": config.JSON()},
	}
}
