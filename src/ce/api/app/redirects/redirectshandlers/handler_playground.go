package redirectshandlers

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type handlerPlaygroundRequest struct {
	Address   string               `json:"address"`
	Redirects []redirects.Redirect `json:"redirects"`
}

func handlerPlayground(req *app.RequestContext) *shttp.Response {
	data := handlerPlaygroundRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	addr := strings.TrimSpace(data.Address)
	u, err := url.Parse(addr)

	if err != nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Missing or invalid `addr` query parameter.",
			},
		}
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	configs, err := appconf.NewStore().Configs(context.TODO(), appconf.ConfigFilters{
		EnvName:     env.Name,
		DisplayName: req.App.DisplayName,
	})

	if err != nil {
		return shttp.Error(err)
	}

	if configs == nil {
		return shttp.NotFound()
	}

	match := redirects.Match(redirects.MatchArgs{
		URL:           u,
		HostName:      u.Host,
		APIPathPrefix: configs[0].APIPathPrefix,
		APILocation:   configs[0].APILocation,
		Redirects:     data.Redirects,
	})

	if match == nil {
		match = &redirects.MatchReturn{}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"against":  u.String(),
			"pattern":  match.Pattern,
			"match":    match.Redirect != "" || match.Proxy || match.Rewrite != "",
			"redirect": match.Redirect,
			"rewrite":  match.Rewrite,
			"proxy":    match.Proxy,
			"status":   match.Status,
		},
	}
}
