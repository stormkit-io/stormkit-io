package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"golang.org/x/oauth2"
)

// ProviderName represents the oauth provider name.
const ProviderName = "forgejo"

// Forgejo is a client for the REST API of the self-hosted Forgejo instance
// configured by the admin. Gitea exposes the same API, so it works too.
type Forgejo struct {
	baseURL string
	client  *http.Client
	user    *oauth.User
}

// NewClient returns a new client for the given user.
func NewClient(userID types.ID) (*Forgejo, error) {
	conf := oauth2Config()

	if conf == nil {
		return nil, nil
	}

	usr, err := oauth.NewStore().OAuthUser(userID, conf, ProviderName)

	if usr == nil || err != nil {
		return nil, oauth.ErrProviderNotConnected
	}

	return newClientWithToken(usr.Token), nil
}

// NewClientWithCode exchanges the code obtained at the end of the
// authorization flow for a token, and returns a client using it.
func NewClientWithCode(code string) (*Forgejo, error) {
	conf := oauth2Config()

	if conf == nil {
		return nil, nil
	}

	token, err := conf.Exchange(context.Background(), code)

	if err != nil {
		return nil, err
	}

	return newClientWithToken(token), nil
}

func newClientWithToken(token *oauth2.Token) *Forgejo {
	return &Forgejo{
		baseURL: admin.MustConfig().AuthConfig.Forgejo.BaseURL,
		client:  oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(token)),
		user:    &oauth.User{Token: token, ProviderName: ProviderName},
	}
}

func oauth2Config() *oauth2.Config {
	cnf := admin.MustConfig()

	if !cnf.IsForgejoEnabled() {
		slog.Info(
			"Forgejo client is not configured and is trying to be accessed. " +
				"Configure it from the admin panel or through the FORGEJO_* environment variables.",
		)

		return nil
	}

	fc := cnf.AuthConfig.Forgejo

	// Forgejo does not implement OAuth2 scopes yet, so none are requested.
	return &oauth2.Config{
		RedirectURL:  fc.RedirectURL,
		ClientID:     fc.ClientID,
		ClientSecret: fc.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  fc.BaseURL + "/login/oauth/authorize",
			TokenURL: fc.BaseURL + "/login/oauth/access_token",
		},
	}
}

// AuthCodeURL returns the url for the authentication.
func AuthCodeURL(state string) string {
	return oauth2Config().AuthCodeURL(state)
}

// CloneURL returns the HTTPS clone address of the given repository
// on the configured Forgejo instance.
func CloneURL(ownerSlug string) string {
	return fmt.Sprintf("%s/%s.git", admin.MustConfig().AuthConfig.Forgejo.BaseURL, ownerSlug)
}

// Token returns the access token for the user.
func (f *Forgejo) Token() string {
	return f.user.Token.AccessToken
}

// repoPath converts a Stormkit formatted repository (forgejo/owner/repo)
// into the escaped owner/repo path used by the API.
func (f *Forgejo) repoPath(repo string) string {
	owner, name := oauth.ParseRepo(repo)
	return fmt.Sprintf("%s/%s", url.PathEscape(owner), url.PathEscape(name))
}

type request struct {
	method string
	path   string
	query  url.Values
	body   any
	out    any
}

// do sends a request to the API. It returns the response so that callers can
// read headers, and closes its body.
func (f *Forgejo) do(r request) (*http.Response, error) {
	var body io.Reader

	if r.body != nil {
		data, err := json.Marshal(r.body)

		if err != nil {
			return nil, err
		}

		body = bytes.NewReader(data)
	}

	address := f.baseURL + "/api/v1" + r.path

	if len(r.query) > 0 {
		address = address + "?" + r.query.Encode()
	}

	req, err := http.NewRequest(r.method, address, body)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := f.client.Do(req)

	if err != nil {
		return nil, oauth.ErrProviderNotReachable
	}

	defer res.Body.Close()

	switch {
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		return res, oauth.ErrNotAuthorized
	case res.StatusCode == http.StatusNotFound:
		return res, oauth.ErrRepoNotFound
	case res.StatusCode >= 300:
		return res, fmt.Errorf("forgejo: %s %s returned %d", r.method, r.path, res.StatusCode)
	}

	if r.out == nil {
		return res, nil
	}

	return res, json.NewDecoder(res.Body).Decode(r.out)
}
