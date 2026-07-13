package gitlab

import (
	"context"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
	gl "golang.org/x/oauth2/gitlab"
)

// ProviderName represents the oauth provider name.
const ProviderName = "gitlab"

// CreateMergeRequestDiscussionOptions is a shorthand export for
// gitlab.CreateMergeRequestDiscussionOptions.
type CreateMergeRequestDiscussionOptions = gitlab.CreateMergeRequestDiscussionOptions

// ResolveMergeRequestDiscussionOptions is a shorthand export for
// gitlab.ResolveMergeRequestDiscussionOptions.
type ResolveMergeRequestDiscussionOptions = gitlab.ResolveMergeRequestDiscussionOptions

// Gitlab is a wrapper around the github client to provide
// access to additional information.
type Gitlab struct {
	*gitlab.Client

	Owner string
	Repo  string

	// User represents the oauth2 user, that is fetched from the database.
	user *oauth.User
}

// NewClient returns a new client for the given user.
func NewClient(userID types.ID) (*Gitlab, error) {
	conf := oauth2Config()

	if conf == nil {
		return nil, nil
	}

	usr, err := oauth.
		NewStore().
		OAuthUser(userID, conf, ProviderName)

	if usr == nil || err != nil {
		return nil, oauth.ErrProviderNotConnected
	}

	return newClientWithToken(usr.Token), nil
}

// NewClientWithCode returns a new bitbucket client for the given
// access code. The code is obtained after a two round shake with the provider.
func NewClientWithCode(code string) (*Gitlab, error) {
	conf := oauth2Config()

	if conf == nil {
		return nil, nil
	}

	token, err := conf.Exchange(context.Background(), code, oauth2.AccessTypeOffline) // TypeOffline enables the refresh token

	if err != nil {
		return nil, err
	}

	return newClientWithToken(token), nil
}

// newClientWithToken returns a new client with given token.
func newClientWithToken(token *oauth2.Token) *Gitlab {
	client, err := gitlab.NewOAuthClient(token.AccessToken)

	if err != nil {
		slog.Errorf("failed creating gitLab client: %v", err)
	}

	return &Gitlab{
		user:   &oauth.User{Token: token, ProviderName: ProviderName},
		Client: client,
	}
}

// oauth2Config returns the configuration for gitbub.
func oauth2Config() *oauth2.Config {
	cnf := admin.MustConfig()

	if !cnf.IsGitlabEnabled() {
		slog.Info(
			"GitLab client is not configured and is trying to be accessed. " +
				"Configure it through the GITLAB_* environment variables.",
		)

		return nil
	}

	return &oauth2.Config{
		RedirectURL:  cnf.AuthConfig.Gitlab.RedirectURL,
		ClientID:     cnf.AuthConfig.Gitlab.ClientID,
		ClientSecret: cnf.AuthConfig.Gitlab.ClientSecret,
		Endpoint:     gl.Endpoint,
		Scopes: []string{
			"read_user",
			"read_repository",
			"write_repository",
		},
	}
}

// AuthCodeURL returns the url for the authentication.
func AuthCodeURL(token string) string {
	return oauth2Config().AuthCodeURL(token)
}

// Token returns the access token for the user.
func (g *Gitlab) Token() string {
	return g.user.Token.AccessToken
}

// SanitizeRepo sanitizes a repository string that is in Stormkit repo format
// and returns a repo name that is equivalent to path_with_namespace in Gitlab API.
func (g *Gitlab) SanitizeRepo(repo string) string {
	return strings.Replace(repo, "gitlab/", "", 1)
}
