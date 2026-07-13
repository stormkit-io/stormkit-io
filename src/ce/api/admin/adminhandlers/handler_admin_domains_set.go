package adminhandlers

import (
	"net/url"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type AdminDomainsSetRequest struct {
	Dev string `json:"dev"` // https://example.org
	App string `json:"app"` // https://stormkit.example.org
	API string `json:"api"` // https://api.example.org
}

func handlerAdminDomainsSet(req *user.RequestContext) *shttp.Response {
	cnf := admin.MustConfig().Clone()
	data := AdminDomainsSetRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if !isValidDomain(data.Dev) {
		return shttp.BadRequest(map[string]any{
			"error": "Dev domain is invalid",
		})
	}

	if !isValidDomain(data.App) {
		return shttp.BadRequest(map[string]any{
			"error": "App domain is invalid",
		})
	}

	if !isValidDomain(data.API) {
		return shttp.BadRequest(map[string]any{
			"error": "API domain is invalid",
		})
	}

	cnf.DomainConfig.Dev = data.Dev
	cnf.DomainConfig.App = data.App
	cnf.DomainConfig.API = data.API

	if err := admin.Store().UpsertConfig(req.Context(), cnf); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Data: map[string]any{
			"domains": map[string]any{
				"dev": cnf.DomainConfig.Dev,
				"app": cnf.DomainConfig.App,
				"api": cnf.DomainConfig.API,
			},
		},
	}
}

func isValidDomain(domain string) bool {
	parsed, err := url.ParseRequestURI(domain)

	if err != nil || parsed == nil {
		return false
	}

	return true
}
