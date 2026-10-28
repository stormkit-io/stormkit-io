package skauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var GitHubAPIBase = "https://api.github.com"

// Step 1: https://github.com/settings/developers (or the organization's settings)
// Step 2: Create a new OAuth App
// Step 3: Authorization callback URL: http://sample.stormkit:8888/_stormkit/auth/callback
// Step 4: Generate a client secret
// Step 5: Obtain client ID and client secret
type GitHubClient struct {
	oauth2Config *oauth2.Config
}

func NewGitHubClient(clientID, secretKey, redirectURL string) Client {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: secretKey,
		RedirectURL:  redirectURL,
		Endpoint:     github.Endpoint,
		Scopes: []string{
			"read:user",
			"user:email", // Required to list the user's emails, including private ones
		},
	}

	return &GitHubClient{
		oauth2Config: config,
	}
}

type GitHubUserInfo struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (g *GitHubClient) UserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := oauth2.NewClient(ctx, g.oauth2Config.TokenSource(ctx, token))

	var userInfo GitHubUserInfo

	if err := g.get(githubGetParams{ctx: ctx, client: client, path: "/user", out: &userInfo}); err != nil {
		return nil, err
	}

	// The profile's email is only the public one, and may be empty or
	// unverified, so the address comes from the emails endpoint instead.
	var emails []GitHubEmail

	if err := g.get(githubGetParams{ctx: ctx, client: client, path: "/user/emails", out: &emails}); err != nil {
		return nil, err
	}

	email := g.verifiedEmail(emails)
	name := userInfo.Name

	if name == "" {
		name = userInfo.Login
	}

	return &UserInfo{
		AccountID:     fmt.Sprintf("%d", userInfo.ID),
		Email:         email,
		EmailVerified: email != "",
		Avatar:        userInfo.AvatarURL,
		FirstName:     name,
		UserMetadata: UserMetadata{
			Username:   userInfo.Login,
			ProfileURL: userInfo.HTMLURL,
		},
	}, nil
}

// verifiedEmail returns the primary email when it is verified, otherwise the
// first verified one. GitHub's relay addresses (<id>+<login>@users.noreply.github.com)
// cannot receive mail and are skipped.
func (g *GitHubClient) verifiedEmail(emails []GitHubEmail) string {
	candidates := []GitHubEmail{}

	for _, email := range emails {
		if email.Verified && !strings.HasSuffix(strings.ToLower(email.Email), "@users.noreply.github.com") {
			candidates = append(candidates, email)
		}
	}

	for _, email := range candidates {
		if email.Primary {
			return email.Email
		}
	}

	if len(candidates) > 0 {
		return candidates[0].Email
	}

	return ""
}

type githubGetParams struct {
	ctx    context.Context
	client *http.Client
	path   string
	out    any
}

func (g *GitHubClient) get(p githubGetParams) error {
	req, err := http.NewRequestWithContext(p.ctx, http.MethodGet, GitHubAPIBase+p.path, nil)

	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github: GET %s returned %d", p.path, resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(p.out)
}

func (g *GitHubClient) Exchange(ctx context.Context, req *shttp.RequestContext) (*oauth2.Token, error) {
	return g.oauth2Config.Exchange(ctx, req.FormValue("code"))
}

func (g *GitHubClient) AuthCodeURL(params AuthCodeURLParams) (string, error) {
	state, err := user.JWT(user.JWTParams{Purpose: user.PurposeSkAuthOAuthState, Claims: params.Claims(), Secret: params.Secret})

	if err != nil {
		return "", err
	}

	return g.oauth2Config.AuthCodeURL(state), nil
}
