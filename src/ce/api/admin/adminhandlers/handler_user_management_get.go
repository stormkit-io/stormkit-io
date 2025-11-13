package adminhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerUserManagementGet(req *user.RequestContext) *shttp.Response {
	cfg := admin.MustConfig()
	whitelist := []string{}

	if cfg.AuthConfig != nil && len(cfg.AuthConfig.UserManagement.Whitelist) > 0 {
		whitelist = cfg.AuthConfig.UserManagement.Whitelist
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"signUpMode": cfg.SignUpMode(),
			"whitelist":  whitelist,
		},
	}
}
