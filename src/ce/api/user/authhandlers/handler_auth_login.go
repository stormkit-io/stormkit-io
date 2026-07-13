package authhandlers

import (
	"net/http"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/bitbucket"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/github"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/gitlab"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAuthLogin redirects the user to the desired provider.
func handlerAuthLogin(req *shttp.RequestContext) *shttp.Response {
	provider := req.Vars()["provider"]

	state, err := user.JWT(jwt.MapClaims{
		"provider": provider,
	})

	if err != nil {
		return shttp.UnexpectedError(err)
	}

	var url string

	cnf := admin.MustConfig()

	switch provider {
	case github.ProviderName:
		if !cnf.IsGithubEnabled() {
			return shttp.BadRequest(map[string]any{
				"error": "GitHub OAuth is not configured",
			})
		}

		url = github.AuthCodeURL(state)
	case bitbucket.ProviderName:
		if !cnf.IsBitbucketEnabled() {
			return shttp.BadRequest(map[string]any{
				"error": "Bitbucket OAuth is not configured",
			})
		}

		url = bitbucket.AuthCodeURL(state)
	case gitlab.ProviderName:
		if !cnf.IsGitlabEnabled() {
			return shttp.BadRequest(map[string]any{
				"error": "GitLab OAuth is not configured",
			})
		}

		url = gitlab.AuthCodeURL(state)
	default:
		return shttp.BadRequest()
	}

	req.Redirect(url, http.StatusTemporaryRedirect)
	return nil
}
