package skauth

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"golang.org/x/oauth2"
)

const ProviderGoogle = "google"

var Providers = []string{
	ProviderGoogle,
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

type Provider struct {
	ID           types.ID
	Name         string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	Status       bool
}

type Client interface {
	UserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error)
	Config() *oauth2.Config
	Name() string
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
	LastLoginAt utils.Unix `json:"deletedAt,omitempty"`
}
