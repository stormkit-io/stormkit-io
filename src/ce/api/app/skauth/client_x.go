package skauth

import (
	"context"
	"encoding/json"

	"golang.org/x/oauth2"
)

// Step 1: https://developer.x.com/en/portal/dashboard
// Step 2: Create a new project and app with Elevated access (required for email)
// Step 3: Enable OAuth 2.0 and set the callback URL
// Step 4: In User authentication settings, enable "Request email address from users"
// Step 5: Callback URL: http://sample.stormkit:8888/api/auth/callback/x
// Step 6: Obtain client ID and client secret
type XClient struct {
	oauth2Config *oauth2.Config
}

func NewXClient(clientID, secretKey string) Client {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: secretKey,
		RedirectURL:  RedirectURL(),
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://twitter.com/i/oauth2/authorize",
			TokenURL: "https://api.twitter.com/2/oauth2/token",
		},
		Scopes: []string{
			"tweet.read",
			"users.read",
			"offline.access", // Required for refresh tokens
		},
	}

	return &XClient{
		oauth2Config: config,
	}
}

// Name returns the name of the provider.
func (x *XClient) Name() string {
	return ProviderX
}

// Data returns the provider data.
func (x *XClient) Data() ProviderData {
	return ProviderData{
		ClientID:     x.oauth2Config.ClientID,
		ClientSecret: x.oauth2Config.ClientSecret,
		RedirectURL:  x.oauth2Config.RedirectURL,
		Scopes:       x.oauth2Config.Scopes,
	}
}

type XUserInfo struct {
	Data struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		Username        string `json:"username"`
		ProfileImageURL string `json:"profile_image_url"`
		Email           string `json:"email"` // Only available with elevated access
	} `json:"data"`
}

func (x *XClient) UserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := oauth2.NewClient(ctx, x.oauth2Config.TokenSource(ctx, token))
	resp, err := client.Get("https://api.twitter.com/2/users/me?user.fields=profile_image_url,email")

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var userInfo XUserInfo

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &UserInfo{
		AccountID: userInfo.Data.ID,
		Email:     userInfo.Data.Email, // Requires elevated access and user consent
		Avatar:    userInfo.Data.ProfileImageURL,
		FirstName: userInfo.Data.Name,
		LastName:  "",
	}, nil
}

func (x *XClient) Config() *oauth2.Config {
	return x.oauth2Config
}
