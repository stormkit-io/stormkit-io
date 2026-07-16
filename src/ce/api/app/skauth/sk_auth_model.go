package skauth

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"golang.org/x/oauth2"
)

// oauthStateTTL bounds how long an authorization-request state token stays
// valid — long enough to complete a provider consent screen, short enough to
// limit replay of a captured state.
const oauthStateTTL = 15 * time.Minute

const ProviderGoogle = "google"
const ProviderX = "x"
const ProviderEmail = "email"
const ProviderMagicLink = "magiclink"

var Providers = []string{
	ProviderGoogle,
	ProviderX,
	ProviderEmail,
	ProviderMagicLink,
}

type OAuthToken struct {
	*oauth2.Token

	AppID    types.ID
	Provider string
}

// OAuthConfig holds your OAuth2 configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

type UserInfo struct {
	ID        string `json:"id,omitempty"`
	AccountID string `json:"accountId,omitempty"`
	Email     string `json:"email,omitempty"`
	// EmailVerified reports whether the provider has verified ownership of
	// Email. Auto-linking a provider onto an existing email-keyed account is
	// only safe when this is true; otherwise a provider that returns an
	// attacker-controlled, unverified address could take over the account.
	EmailVerified bool   `json:"emailVerified,omitempty"`
	Avatar        string `json:"avatar,omitempty"`
	FirstName     string `json:"firstName,omitempty"`
	LastName      string `json:"lastName,omitempty"`
	// Provider profile extras (handle, profile link). Embedded so a client can
	// hand its whole UserMetadata to the stored User in one assignment.
	UserMetadata
}

// UserMetadata holds provider-specific profile extras that not every provider
// supplies (currently the handle and a link to the public profile). It is
// stored as a JSONB column so new fields can be added without a schema change.
type UserMetadata struct {
	Username   string `json:"username,omitempty"`   // provider handle, e.g. an X @handle (empty when the provider has none)
	ProfileURL string `json:"profileUrl,omitempty"` // link to the public profile, e.g. https://x.com/<handle>
}

// Value serialises the metadata as plain JSON for the JSONB column. Unlike
// utils.ByteaValue it does not encrypt: these are public profile fields and
// benefit from staying queryable as JSON in Postgres.
func (m UserMetadata) Value() (driver.Value, error) {
	b, err := json.Marshal(m)

	if err != nil {
		return nil, err
	}

	return string(b), nil
}

// Scan reads the JSONB column back into the struct. A NULL or empty value
// leaves the zero struct.
func (m *UserMetadata) Scan(value any) error {
	if value == nil {
		return nil
	}

	var b []byte

	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("failed to scan UserMetadata: unexpected type %T", value)
	}

	if len(b) == 0 {
		return nil
	}

	return json.Unmarshal(b, m)
}

type ProviderData struct {
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	RedirectURL  string   `json:"redirectURL"`
	Scopes       []string `json:"scopes"`
	FromAddress  string   `json:"fromAddress,omitempty"`
}

type Provider struct {
	ID     types.ID
	Name   string
	Data   ProviderData
	Status bool
}

var DefaultClient Client

// Supported reports whether the provider name is a recognized auth provider.
// Use it to validate a provider before persisting it, instead of building a
// throwaway client.
func (p *Provider) Supported() bool {
	switch p.Name {
	case ProviderGoogle, ProviderX, ProviderEmail, ProviderMagicLink:
		return true
	default:
		return false
	}
}

// ClientParams configures the provider client returned by Provider.Client.
type ClientParams struct {
	// RedirectURL is the absolute OAuth2 callback the provider redirects back
	// to; it must match between the authorization request and the token
	// exchange, and is ignored by non-OAuth providers.
	RedirectURL string
	// Secret is the environment's auth secret used to sign and verify the OAuth
	// state JWT. X needs it on the token exchange to recover the encrypted PKCE
	// verifier carried in the state; other providers ignore it.
	Secret string
}

// Client returns the provider client (OAuth or non-OAuth) for the provider.
func (p *Provider) Client(params ClientParams) Client {
	if DefaultClient != nil {
		return DefaultClient
	}

	switch p.Name {
	case ProviderGoogle:
		return NewGoogleClient(p.Data.ClientID, p.Data.ClientSecret, params.RedirectURL)
	case ProviderX:
		return NewXClient(p.Data.ClientID, p.Data.ClientSecret, params.RedirectURL, params.Secret)
	case ProviderEmail:
		return NewEmailClient()
	case ProviderMagicLink:
		return NewMagicLinkClient()
	}

	return nil
}

type AuthCodeURLParams struct {
	EnvID        types.ID
	ProviderName string
	Referrer     string
	// Secret signs the state token. It is the environment's auth secret so the
	// state is scoped to this tenant (a state minted for one environment is not
	// accepted by another) and shares the session token's signing key. An empty
	// value falls back to the global app secret in JWT().
	Secret string
}

// Claims returns the JWT claims for the authorization request, including
// environment ID, provider name and a short expiry that bounds state replay.
func (a AuthCodeURLParams) Claims() jwt.MapClaims {
	return jwt.MapClaims{
		"eid": a.EnvID,
		"prv": a.ProviderName,
		"ref": a.Referrer,
		"exp": time.Now().Add(oauthStateTTL).Unix(),
	}
}

type Client interface {
	UserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error)
	AuthCodeURL(data AuthCodeURLParams) (string, error)
	Exchange(ctx context.Context, req *shttp.RequestContext) (*oauth2.Token, error)
}

type OAuth struct {
	ID                types.ID
	AccountID         string
	ProviderName      string
	CreatedAt         utils.Unix
	AccessToken       string
	RefreshToken      string
	TokenType         string
	Expiry            utils.Unix
	VerificationToken string
}

type User struct {
	ID           types.ID     `json:"-"`  // Internal numeric primary key. Used for joins; not exposed externally.
	UUID         string       `json:"id"` // External identifier surfaced via the X-User-Id header and API responses.
	FirstName    string       `json:"firstName"`
	LastName     string       `json:"lastName"`
	Email        string       `json:"email"`
	Avatar       string       `json:"avatar,omitempty"`
	CreatedAt    utils.Unix   `json:"createdAt"`
	LastLoginAt  utils.Unix   `json:"lastLoginAt,omitempty"`
	VerifiedAt   utils.Unix   `json:"verifiedAt,omitempty"`
	PasswordHash string       `json:"-"`
	Metadata     UserMetadata `json:"-"` // provider extras; surfaced flattened by JSON(), stored as the metadata JSONB column
}

// JSON returns a map representation of the user for API responses.
func (u *User) JSON() map[string]any {
	return map[string]any{
		"id":          u.UUID,
		"firstName":   u.FirstName,
		"lastName":    u.LastName,
		"email":       u.Email,
		"avatar":      u.Avatar,
		"username":    u.Metadata.Username,
		"profileUrl":  u.Metadata.ProfileURL,
		"createdAt":   u.CreatedAt,
		"lastLoginAt": u.LastLoginAt,
	}
}

// CallbackPath is the OAuth2 callback path, relative to the app's own hosting
// domain. The provider redirects the browser back here after authorization,
// and the same absolute URL (https://<app-domain>/_stormkit/auth/callback) is
// used in both the authorization request and the token exchange.
const CallbackPath = "/_stormkit/auth/callback"

// RedirectURL returns the OAuth2 callback path, relative to the app's own
// hosting domain. The host is intentionally omitted: an app can be served from
// several domains, so the UI prepends each domain to build the full redirect
// URL the customer registers in the provider portal, and the server derives the
// host from the request at flow time.
func RedirectURL() string {
	return CallbackPath
}

// AuthURL returns the path, relative to the app's own hosting domain, where
// users start the OAuth2 flow. The provider segment is appended by the caller
// (e.g. /_stormkit/auth/google).
func AuthURL() string {
	return "/_stormkit/auth"
}
