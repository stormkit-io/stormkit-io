package github

import (
	"context"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v71/github"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"golang.org/x/oauth2"
	gh "golang.org/x/oauth2/github"
)

const ProviderName = "github"

var DefaultGithubClient GithubClient

type ListOptions = github.ListOptions
type ListRepositories = github.ListRepositories
type Repository = github.Repository
type Installation = github.Installation
type TemplateRepoRequest = github.TemplateRepoRequest
type Response = github.Response
type User = github.User

type GithubClient interface {
	// Github methods
	ListRepos(context.Context, *ListOptions) (*ListRepositories, *Response, error)
	ListUserInstallations(context.Context, *ListOptions) ([]*Installation, *Response, error)
	CreateFromTemplate(context.Context, string, string, *TemplateRepoRequest) (*Repository, *Response, error)
	GetUser(context.Context, string) (*User, *Response, error)

	// Custom methods
	UserProfile() (*oauth.User, error)
	ListEmails() ([]oauth.Email, error)
}

type githubClient struct {
	*github.Client
	user *oauth.User
}

func (gc *githubClient) ListRepos(ctx context.Context, opts *github.ListOptions) (*github.ListRepositories, *github.Response, error) {
	return gc.Apps.ListRepos(ctx, opts)
}

func (gc *githubClient) ListUserInstallations(ctx context.Context, opts *github.ListOptions) ([]*github.Installation, *github.Response, error) {
	return gc.Apps.ListUserInstallations(ctx, opts)
}

func (gc *githubClient) CreateFromTemplate(ctx context.Context, owner, repo string, template *TemplateRepoRequest) (*Repository, *Response, error) {
	return gc.Repositories.CreateFromTemplate(ctx, owner, repo, template)
}

func (gc *githubClient) GetUser(ctx context.Context, login string) (*User, *Response, error) {
	return gc.Users.Get(ctx, login)
}

// ListEmails lists the emails of a given user.
func (g *githubClient) ListEmails() ([]oauth.Email, error) {
	githubEmails, _, err := g.Users.ListEmails(context.Background(), nil)

	if err != nil || githubEmails == nil {
		return nil, err
	}

	emails := []oauth.Email{}

	for _, email := range githubEmails {
		emails = append(emails, oauth.Email{
			Address:    email.GetEmail(),
			IsPrimary:  email.GetPrimary(),
			IsVerified: email.GetVerified(),
		})
	}

	return emails, nil
}

// UserProfile returns information on the user.
func (g *githubClient) UserProfile() (*oauth.User, error) {
	login, _, err := g.Users.Get(context.Background(), "")

	if err != nil {
		return nil, err
	}

	if login.URL != nil {
		g.user.AccountURI = *login.URL
	}

	if login.AvatarURL != nil {
		g.user.AvatarURI = *login.AvatarURL
	}

	if login.Login != nil {
		g.user.DisplayName = *login.Login
	}

	if login.Name != nil {
		g.user.FullName = *login.Name
	}

	g.user.Emails, err = g.ListEmails()

	if err != nil {
		return g.user, err
	}

	return g.user, err
}

// NewClient returns a new client for the given user.
func NewClient(userID types.ID) (GithubClient, error) {
	if DefaultGithubClient != nil {
		return DefaultGithubClient, nil
	}

	usr, err := oauth.
		NewStore().
		OAuthUser(userID, oauth2Config(), ProviderName)

	if usr == nil || err != nil {
		if err != nil {
			slog.Errorf("Failed to get user from OAuth store: %v", err)
		}

		return nil, oauth.ErrProviderNotConnected
	}

	return newClientWithToken(usr.Token), nil
}

// NewAppClient returns a new client for the given user and installation ID.
func NewAppClient(installationID int64) (GithubClient, error) {
	if DefaultGithubClient != nil {
		return DefaultGithubClient, nil
	}

	cnf := admin.MustConfig().AuthConfig.Github
	itr, err := ghinstallation.New(http.DefaultTransport, int64(cnf.AppID), installationID, []byte(cnf.PrivateKey))

	if err != nil {
		return nil, err
	}

	return &githubClient{
		Client: github.NewClient(&http.Client{Transport: itr}),
	}, nil
}

// NewClientWithCode returns a new client. The code represents the
// authorization code that is returned by the provider callback.
func NewClientWithCode(code string) (GithubClient, error) {
	token, err := oauth2Config().Exchange(context.Background(), code, oauth2.AccessTypeOffline) // TypeOffline enables the refresh token

	if err != nil {
		return nil, err
	}

	return newClientWithToken(token), nil
}

// AuthCodeURL returns the url for the authentication.
func AuthCodeURL(token string) string {
	return oauth2Config().AuthCodeURL(token)
}

// newClientWithToken returns a new client that is initialized
// with the static token source.
func newClientWithToken(tkn *oauth2.Token) GithubClient {
	ts := oauth2.StaticTokenSource(tkn)

	return &githubClient{
		user:   &oauth.User{Token: tkn, ProviderName: ProviderName},
		Client: github.NewClient(oauth2.NewClient(context.Background(), ts)),
	}
}

// oauth2Config returns the configuration for github.
func oauth2Config() *oauth2.Config {
	conf := admin.MustConfig().AuthConfig.Github

	return &oauth2.Config{
		ClientID:     conf.ClientID,
		ClientSecret: conf.ClientSecret,
		Endpoint:     gh.Endpoint,
	}
}
