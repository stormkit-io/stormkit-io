package adminhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// GitHubCallbackRequest represents the callback from GitHub after app creation
type GitHubCallbackRequest struct {
	Code  string `json:"code"`  // Temporary code from GitHub
	State string `json:"state"` // State for security verification
}

// GitHubAppCredentials represents the app credentials returned by GitHub
type GitHubAppCredentials struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	WebhookSecret string `json:"webhook_secret"`
	PEM           string `json:"pem"`
}

// handlerGitHubManifestCallback handles the callback from GitHub after app creation
func handlerGitHubManifestCallback(req *shttp.RequestContext) *shttp.Response {
	code := req.Query().Get("code")
	state := req.Query().Get("state")

	if code == "" {
		return shttp.BadRequest(map[string]any{
			"error": "Missing code parameter",
		})
	}

	if state == "" {
		return shttp.BadRequest(map[string]any{
			"error": "Missing state parameter",
		})
	}

	// GitHub has created the app by now, so an invalid state sends the admin
	// back to the admin panel with an explanation rather than a bare error.
	invalidState := &shttp.Response{
		Status:   http.StatusFound,
		Redirect: utils.Ptr(admin.MustConfig().AppURL("/admin/git?error=github_app_state_invalid")),
	}

	adminID, err := githubManifestState{ctx: req.Context()}.redeem(state)

	if err != nil {
		return shttp.Error(err)
	}

	if adminID == 0 {
		return invalidState
	}

	adminUser, err := user.NewStore().UserByID(adminID)

	if err != nil {
		return shttp.Error(err)
	}

	if adminUser == nil || !adminUser.IsAdmin {
		return invalidState
	}

	// Exchange the code for app credentials
	credentials, err := exchangeCodeForCredentials(code)

	if err != nil {
		return shttp.Error(err)
	}

	// Store the credentials in the configuration
	cnf, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	cnf = cnf.Clone()

	if cnf.AuthConfig == nil {
		cnf.AuthConfig = &admin.AuthConfig{}
	}

	// Update the GitHub configuration with the new app credentials
	cnf.AuthConfig.Github = admin.GithubConfig{
		ClientID:     credentials.ClientID,
		ClientSecret: credentials.ClientSecret,
		PrivateKey:   credentials.PEM,
		AppID:        int(credentials.ID),
		Account:      credentials.Name,

		WebhookSecret: credentials.WebhookSecret,
	}

	if err := admin.Store().UpsertConfig(context.Background(), cnf); err != nil {
		return shttp.Error(err)
	}

	// Redirect to the admin panel with success
	return &shttp.Response{
		Status:   http.StatusFound,
		Redirect: utils.Ptr(admin.MustConfig().AppURL("/admin/git?success=github_app_created")),
	}
}

// exchangeCodeForCredentials exchanges the temporary code for app credentials
func exchangeCodeForCredentials(code string) (*GitHubAppCredentials, error) {
	// Make a POST request to GitHub's API to exchange the code
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	url := fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code)
	res, err := shttp.NewRequestV2(http.MethodPost, url).Headers(headers).Do()

	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("GitHub API returned status %d", res.StatusCode)
	}

	var credentials GitHubAppCredentials

	if err := json.NewDecoder(res.Body).Decode(&credentials); err != nil {
		return nil, fmt.Errorf("failed to decode credentials: %w", err)
	}

	return &credentials, nil
}
