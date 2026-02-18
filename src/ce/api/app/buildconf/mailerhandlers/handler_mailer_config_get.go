package mailerhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func HandlerMailerConfigGet(req *app.RequestContext) *shttp.Response {
	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	response := map[string]any{
		"config": nil,
	}

	if env.MailerConf != nil {
		response["config"] = map[string]any{
			"host":     env.MailerConf.Host,
			"port":     env.MailerConf.Port,
			"username": env.MailerConf.Username,
			"password": env.MailerConf.Password,
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   response,
	}
}
