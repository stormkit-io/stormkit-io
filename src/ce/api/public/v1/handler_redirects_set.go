package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type RedirectsSetRequest struct {
	Redirects []redirects.Redirect `json:"redirects"`
}

func handlerRedirectsSet(req *RequestContext) *shttp.Response {
	data := RedirectsSetRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if errs := redirects.Validate(data.Redirects); len(errs) > 0 {
		return shttp.BadRequest(map[string]any{"errors": errs})
	}

	store := buildconf.NewStore()
	req.Env.Data.Redirects = data.Redirects

	if err := store.Update(req.Context(), req.Env); err != nil {
		return shttp.Error(err)
	}

	if err := appcache.Service().Reset(req.Env.ID); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"redirects": req.Env.Data.Redirects,
		},
	}
}
