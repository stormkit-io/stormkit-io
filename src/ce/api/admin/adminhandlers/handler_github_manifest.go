package adminhandlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// GitHubManifestRequest represents the request for generating a GitHub App manifest
type GitHubManifestRequest struct {
	AppName      string `json:"appName"`      // GitHub App name
	Organization string `json:"organization"` // GitHub organization (optional)
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

	// Generate a random state for security
	state, err := user.JWT(nil)

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
