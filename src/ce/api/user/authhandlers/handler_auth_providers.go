package authhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerAuthProviders(req *shttp.RequestContext) *shttp.Response {
	cfg, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	data := map[string]any{
		"github":    cfg.IsGithubEnabled(),
		"gitlab":    cfg.IsGitlabEnabled(),
		"bitbucket": cfg.IsBitbucketEnabled(),
		"forgejo":   cfg.IsForgejoEnabled(),
	}

	// The dashboard builds repository and commit links from the instance URL.
	if cfg.IsForgejoEnabled() {
		data["forgejoUrl"] = cfg.AuthConfig.Forgejo.BaseURL
	}

	if cfg.AdminUserConfig != nil && cfg.AdminUserConfig.Email != "" {
		data["basicAuth"] = "enabled"
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   data,
	}
}
