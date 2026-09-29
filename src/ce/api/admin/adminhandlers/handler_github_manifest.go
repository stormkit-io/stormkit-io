package adminhandlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// GitHubManifestRequest represents the request for generating a GitHub App manifest
type GitHubManifestRequest struct {
	AppName      string `json:"appName"`      // GitHub App name
	Organization string `json:"organization"` // GitHub organization (optional)
}

// githubManifestStatePrefix namespaces manifest flow states in Redis.
const githubManifestStatePrefix = "github-manifest-state:"

// githubManifestStateTTL is how long an admin has to complete the flow.
const githubManifestStateTTL = 30 * time.Minute

// githubManifestState issues and redeems the state of the GitHub App manifest
// flow. The state is a random, single-use code bound to the admin who started
// the flow, so it cannot be forged, replayed or used as a session token, and
// it carries nothing readable in the URL sent to GitHub.
type githubManifestState struct {
	ctx context.Context
}

// issue returns a new state for the given admin.
func (s githubManifestState) issue(adminID types.ID) (string, error) {
	code, err := utils.SecureRandomToken(48)

	if err != nil {
		return "", err
	}

	if err := rediscache.Client().Set(s.ctx, githubManifestStatePrefix+code, adminID.String(), githubManifestStateTTL).Err(); err != nil {
		return "", err
	}

	return code, nil
}

// redeem consumes the state and returns the admin it was issued to. It
// returns zero when the state is unknown, expired or already used.
func (s githubManifestState) redeem(code string) (types.ID, error) {
	adminID, err := rediscache.Client().GetDel(s.ctx, githubManifestStatePrefix+code).Result()

	if errors.Is(err, redis.Nil) {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	return utils.StringToID(adminID), nil
}

// handlerGitHubGenerateManifest generates a GitHub App manifest and returns the GitHub creation URL
func handlerGitHubGenerateManifest(req *user.RequestContext) *shttp.Response {
	data := GitHubManifestRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if data.AppName == "" {
		return shttp.BadRequest(map[string]any{
			"error": "App name is required",
		})
	}

	state, err := githubManifestState{ctx: req.Context()}.issue(req.User.ID)

	if err != nil {
		return shttp.Error(err)
	}

	// Create the GitHub manifest creation URL
	var githubURL string

	if data.Organization != "" {
		githubURL = fmt.Sprintf("https://github.com/organizations/%s/settings/apps/new", url.QueryEscape(data.Organization))
	} else {
		githubURL = "https://github.com/settings/apps/new"
	}

	// Get the base URL for callbacks (you might need to configure this)
	baseURL := strings.TrimRight(admin.MustConfig().ApiURL(""), "/")
	manifest := map[string]any{
		"name": data.AppName,
		"url":  baseURL,
		"hook_attributes": map[string]any{
			"url":    fmt.Sprintf("%s/app/webhooks/github/deploy", baseURL),
			"active": true,
		},
		"redirect_url": fmt.Sprintf("%s/admin/git/github/callback", baseURL),
		"setup_url":    fmt.Sprintf("%s/auth/github/installation", baseURL),
		"callback_urls": []string{
			fmt.Sprintf("%s/auth/github/callback", baseURL),
		},
		"public": false,
		"default_permissions": map[string]string{
			"administration":   "write",
			"checks":           "write",
			"statuses":         "write",
			"contents":         "read",
			"pull_requests":    "write",
			"repository_hooks": "read",
			"emails":           "read",
		},
		"default_events": []string{
			"push",
			"pull_request",
		},
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"url":      githubURL + "?state=" + state,
			"manifest": manifest,
		},
	}
}
