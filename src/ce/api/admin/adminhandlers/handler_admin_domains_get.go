package adminhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerAdminDomainsGet(req *user.RequestContext) *shttp.Response {
	cnf := admin.MustConfig()

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
