package skauth

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"golang.org/x/oauth2"
)

const ProviderGoogle = "google"
const ProviderX = "x"

var Providers = []string{
	ProviderGoogle,
	ProviderX,
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
	Avatar    string `json:"avatar,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
}

type ProviderData struct {
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	RedirectURL  string   `json:"redirectURL"`
	Scopes       []string `json:"scopes"`
}

type Provider struct {
	ID           types.ID
	Name         string
	Data         ProviderData
	Status       bool
	cachedClient Client
}

var DefaultClient Client

// Client returns the OAuth client for the provider.
func (p *Provider) Client() Client {
	if DefaultClient != nil {
		return DefaultClient
	}

	if p.cachedClient != nil {
		return p.cachedClient
	}

	switch p.Name {
	case ProviderGoogle:
		p.cachedClient = NewGoogleClient(p.Data.ClientID, p.Data.ClientSecret)
	case ProviderX:
		p.cachedClient = NewXClient(p.Data.ClientID, p.Data.ClientSecret)
	}

	return p.cachedClient
}

type Client interface {
	UserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error)
	Config() *oauth2.Config
}

type OAuth struct {
	ID           types.ID
	AccountID    string
	ProviderName string
	CreatedAt    utils.Unix
	AccessToken  string
	RefreshToken string
	TokenType    string
	Expiry       utils.Unix
}

type User struct {
	ID          types.ID   `json:"id,string"`
	FirstName   string     `json:"firstName"`
	LastName    string     `json:"lastName"`
	Email       string     `json:"email"`
	Avatar      string     `json:"avatar,omitempty"`
	CreatedAt   utils.Unix `json:"createdAt"`
	LastLoginAt utils.Unix `json:"lastLoginAt,omitempty"`
}

// RedirectURL returns the OAuth2 redirect URL.
func RedirectURL() string {
	return admin.MustConfig().ApiURL("/v1/auth/callback")
}

// AuthURL returns the URL where the users can start the flow.
func AuthURL() string {
	return admin.MustConfig().ApiURL("/v1/auth")
}
