package adminhandlers

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type GitConfigureRequest struct {
	Provider     string `json:"provider"`     // github, gitlab, or bitbucket
	AppID        string `json:"appId"`        // Github application id
	Account      string `json:"account"`      // GitHub account name
	Organization string `json:"organization"` // GitHub organization (optional)
	PrivateKey   string `json:"privateKey"`   // GitHub App Private Key
	ClientID     string `json:"clientId"`     // OAuth Client ID
	ClientSecret string `json:"clientSecret"` // OAuth Client Secret
	RedirectURL  string `json:"redirectUrl"`  // GitLab/Bitbucket Redirect URL
	DeployKey    string `json:"deployKey"`    // Bitbucket Deploy Key
	RunnerRepo   string `json:"runnerRepo"`   // GitHub Runner Repository
	RunnerToken  string `json:"runnerToken"`  // GitHub Runner Token
}

func handlerGitConfigure(req *user.RequestContext) *shttp.Response {
	data := GitConfigureRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if data.Provider == "" {
		return shttp.BadRequest(map[string]any{
			"error": "Provider is required",
		})
	}

	cnf, err := admin.Store().Config(req.Context())
	if err != nil {
		return shttp.Error(err)
	}

	cnf = cnf.Clone()

	if cnf.AuthConfig == nil {
		cnf.AuthConfig = &admin.AuthConfig{}
	}

	switch data.Provider {
	case "github":
		cnf.AuthConfig.Github.Account = data.Account
		cnf.AuthConfig.Github.ClientID = data.ClientID
		cnf.AuthConfig.Github.ClientSecret = data.ClientSecret
		cnf.AuthConfig.Github.PrivateKey = data.PrivateKey
		cnf.AuthConfig.Github.AppID = utils.StringToInt(data.AppID)

		if !cnf.IsGithubEnabled() {
			return shttp.BadRequest(map[string]any{
				"error": "GitHub App is not properly configured. Please provide all required fields.",
			})
		}
	case "gitlab":
		cnf.AuthConfig.Gitlab = admin.GitlabConfig{
			ClientID:     data.ClientID,
			ClientSecret: data.ClientSecret,
			RedirectURL: utils.GetString(
				data.RedirectURL,
				cnf.AuthConfig.Gitlab.RedirectURL,
				admin.MustConfig().ApiURL("/auth/gitlab/callback"),
			),
		}

		if !cnf.IsGitlabEnabled() {
			return shttp.BadRequest(map[string]any{
				"error": "GitLab is not properly configured. Please provide all required fields.",
			})
		}
	case "bitbucket":
		cnf.AuthConfig.Bitbucket = admin.BitbucketConfig{
			ClientID:     data.ClientID,
			ClientSecret: data.ClientSecret,
			DeployKey:    data.DeployKey,
		}

		if !cnf.IsBitbucketEnabled() {
			return shttp.BadRequest(map[string]any{
				"error": "Bitbucket is not properly configured. Please provide all required fields.",
			})
		}
	default:
		return shttp.BadRequest(map[string]any{
			"error": "Invalid provider. Must be one of: github, gitlab, bitbucket",
		})
	}

	if err := admin.Store().UpsertConfig(context.Background(), cnf); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
