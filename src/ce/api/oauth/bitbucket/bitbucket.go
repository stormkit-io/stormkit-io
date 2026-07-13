package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"golang.org/x/oauth2"
	bb "golang.org/x/oauth2/bitbucket"
)

const (
	bitbucketAPIEndpoint = "https://api.bitbucket.org/2.0"

	// ProviderName represents the provider name.
	ProviderName = "bitbucket"
)

// List of permissions
const (
	PermissionWebhooks         = "webhook"
	PermissionTeam             = "team"
	PermissionAccount          = "account"
	PermissionPullRequestRead  = "pullrequest"
	PermissionRepositoryWrite  = "repository:write"  // This comes automatically when pull request is needed
	PermissionRepositoryAdmin  = "repository:admin"  // Required to add deploy keys
	PermissionPullRequestWrite = "pullrequest:write" // Required to leave comments on pull requests
)

var bitbucketPermissions = []string{
	PermissionWebhooks,
	PermissionRepositoryWrite,
	PermissionTeam,
	PermissionAccount,
	PermissionPullRequestRead,
}

// Bitbucket is a wrapper around the bitbucket client to provide
// access to additional information.
type Bitbucket struct {
	// User represents the interface to make user related calls.
	User *UserInterface

	// client represents an http client.
	client *http.Client

	// User represents the oauth2 user, that is fetched from the database.
	user *oauth.User

	// oauth2 represents the oauth2 config.
	oauth2 *oauth2.Config
}

// App represents an application using bitbucket.
type App struct {
	ID         types.ID
	Repo       string
	Secret     string
	PrivateKey *utils.PrivateKey
}

// LinkHref represents a link.href property.
type LinkHref struct {
	Href string `json:"href"`
}

// NewClient returns a new Bitbucket instance for oauth2
// operations. Unlike Github, Bitbucket uses always the oauth2 client
// to communicate. There is no such concept as App.
func NewClient(userID types.ID) (*Bitbucket, error) {
	conf := oauth2Config()

	if conf == nil {
		return nil, nil
	}

	usr, err := oauth.NewStore().OAuthUser(userID, conf, ProviderName)

	if usr == nil || err != nil {
		return nil, oauth.ErrProviderNotConnected
	}

	b := newClientWithToken(conf, usr.Token)
	b.user = usr
	return b, nil
}

// NewClientWithScope returns a new Bitbucket instance for oauth2
// operations, with only required permissions.
func NewClientWithScope(userID types.ID, permissions []string) (*Bitbucket, error) {
	conf := oauth2Config()

	if conf == nil {
		return nil, nil
	}

	conf.Scopes = permissions

	usr, err := oauth.NewStore().OAuthUser(userID, conf, ProviderName)

	if usr == nil || err != nil {
		return nil, oauth.ErrProviderNotConnected
	}

	b := newClientWithToken(conf, usr.Token)
	b.user = usr
	return b, nil
}

// NewClientWithCode returns a new bitbucket client for the given
// access code. The code is obtained after a two round shake with the provider.
func NewClientWithCode(code string) (*Bitbucket, error) {
	conf := oauth2Config()

	if conf == nil {
		return nil, nil
	}

	token, err := conf.Exchange(context.Background(), code, oauth2.AccessTypeOffline) // TypeOffline enables the refresh token

	if err != nil {
		return nil, err
	}

	return newClientWithToken(conf, token), nil
}

// AuthCodeURL returns the url for the authentication.
func AuthCodeURL(token string) string {
	return oauth2Config().AuthCodeURL(token)
}

// Token returns the access token.
func (b *Bitbucket) Token() string {
	return b.user.Token.AccessToken
}

// newClientWithToken returns a new client with given token.
func newClientWithToken(conf *oauth2.Config, token *oauth2.Token) *Bitbucket {
	b := &Bitbucket{
		user:   &oauth.User{ProviderName: ProviderName, Token: token},
		oauth2: conf,
		client: conf.Client(context.Background(), token),
	}

	b.User = &UserInterface{b}

	return b
}

// parse is a shortand function to parse bitbucket responses.
func (b *Bitbucket) parse(res *http.Response, into interface{}) error {
	err := json.NewDecoder(res.Body).Decode(into)

	if err != nil {
		slog.Errorf("bitbucket cannot parse payload: %v", err)
	}

	return err
}

// get is a shorthand function to perform get requests.
func (b *Bitbucket) get(url string) (*http.Response, error) {
	return b.request(http.MethodGet, url, nil)
}

// post is a shorthand function to perform post request.
func (b *Bitbucket) post(url string, body interface{}) (*http.Response, error) {
	return b.request(http.MethodPost, url, body)
}

// delete is a shorthand function to perform delete  request.
func (b *Bitbucket) delete(url string) (*http.Response, error) {
	return b.request(http.MethodDelete, url, nil)
}

// request performs a new request to the oauth provider.
func (b *Bitbucket) request(method, url string, body any) (response *http.Response, err error) {
	if method == http.MethodGet {
		response, err = b.client.Get(bitbucketAPIEndpoint + url)
	} else if method == http.MethodPost {
		payload, _ := json.Marshal(body)
		response, err = b.client.Post(bitbucketAPIEndpoint+url, "application/json", bytes.NewBuffer(payload))
	} else if method == http.MethodDelete {
		request, _ := http.NewRequest(http.MethodDelete, bitbucketAPIEndpoint+url, nil)
		response, err = b.client.Do(request)
	}

	if err != nil {
		slog.Errorf("error while performing request to: %s, err=%v", url, err)
		return nil, err
	}

	if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusUnauthorized {
		slog.Errorf("user is unauthorized: %s", url)
		return nil, oauth.ErrCredsInvalidPermissions
	}

	if response.StatusCode == http.StatusNotFound {
		return nil, oauth.ErrRepoNotFound
	}

	if response.Status == "" || string(response.Status[0]) != "2" {
		return response, fmt.Errorf("was expecting a 2xx response but received %s for request: %s", response.Status, url)
	}

	return response, err
}

// oauth2Config returns a new oauth2.Config instance.
func oauth2Config() *oauth2.Config {
	cnf := admin.MustConfig()

	if !cnf.IsBitbucketEnabled() {
		slog.Info(
			"Bitbucket client is not configured and is trying to be accessed. " +
				"Configure it through the BITBUCKET_* environment variables.",
		)

		return nil
	}

	permissions := bitbucketPermissions

	// We need the admin permission only if deploy key is not specified.
	// Self-Hosted instances can specify it through an environment variable.
	if cnf.AuthConfig.Bitbucket.DeployKey == "" {
		if !utils.InSliceString(bitbucketPermissions, PermissionRepositoryAdmin) {
			permissions = append(bitbucketPermissions, PermissionRepositoryAdmin)
		}
	}

	return &oauth2.Config{
		ClientID:     cnf.AuthConfig.Bitbucket.ClientID,
		ClientSecret: cnf.AuthConfig.Bitbucket.ClientSecret,
		Endpoint:     bb.Endpoint,
		Scopes:       permissions,
	}
}
