package mailerhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// HandlerMailerConfigSet patches the SMTP configuration for the environment.
// Only the fields present in the request body are changed; the rest keep their
// stored values. The response never echoes the password back.
func HandlerMailerConfigSet(req *app.RequestContext) *shttp.Response {
	data := ConfigUpdateRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	// WithApp only asserts that an envId was provided, not that it exists, and
	// the store returns (nil, nil) for an unknown id.
	if env == nil {
		return shttp.NotFound()
	}

	config := env.MailerConf

	if config == nil {
		config = &buildconf.MailerConf{}
	}

	config.EnvID = req.EnvID

	if err := ApplyConfigUpdate(config, data); err != nil {
		return shttp.Error(err)
	}

	if err := buildconf.MailerStore().UpsertConfig(req.Context(), config); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"config": config.JSON()},
	}
}
