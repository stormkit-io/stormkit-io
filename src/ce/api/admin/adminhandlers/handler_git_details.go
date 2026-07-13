package adminhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerGitDetails(req *user.RequestContext) *shttp.Response {
	cnf, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	auth := cnf.AuthConfig

	if auth == nil {
		return &shttp.Response{
			Status: http.StatusOK,
			Data: map[string]any{
				"github":    nil,
				"gitlab":    nil,
				"bitbucket": nil,
			},
		}
	}

	data := map[string]any{}

	if cnf.IsGithubEnabled() {
		data["github"] = map[string]any{
			"appId":           utils.Int64ToString(int64(auth.Github.AppID)),
			"account":         auth.Github.Account,
			"clientId":        auth.Github.ClientID,
			"runnerRepo":      auth.Github.RunnerRepo,
			"hasRunnerToken":  auth.Github.RunnerToken != "",
			"hasPrivateKey":   true,
			"hasClientSecret": true,
		}
	}

	if cnf.IsGitlabEnabled() {
		data["gitlab"] = map[string]any{
			"clientId":        auth.Gitlab.ClientID,
			"redirectUrl":     auth.Gitlab.RedirectURL,
			"hasClientSecret": true,
		}
	}

	if cnf.IsBitbucketEnabled() {
		data["bitbucket"] = map[string]any{
			"clientId":        auth.Bitbucket.ClientID,
			"hasDeployKey":    auth.Bitbucket.DeployKey != "",
			"hasClientSecret": true,
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   data,
	}
}
