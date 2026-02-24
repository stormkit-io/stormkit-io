package skauth

import (
	"context"
	"encoding/json"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Step 1: https://console.cloud.google.com/auth/overview
// Step 2: Create oAuth client (application type: web application)
// Step 3: Authorized Javascript Origins: http://sample.stormkit:8888
// Step 4: Authorized redirect URIs: http://sample.stormkit:8888/api/auth/callback/google
// Step 5: Obtain client ID and client secret
type GoogleClient struct {
	oauth2Config *oauth2.Config
}

func NewGoogleClient(clientID, secretKey string) Client {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: secretKey,
		RedirectURL:  RedirectURL(),
		Endpoint:     google.Endpoint,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}

	return &GoogleClient{
		oauth2Config: config,
	}
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	FamilyName    string `json:"family_name"` // Doe
	GivenName     string `json:"given_name"`  // Joe
	Name          string `json:"name"`        // Joe Doe
	Picture       string `json:"picture"`     // URL to avatar
	VerifiedEmail bool   `json:"verified_email"`
}

func (g *GoogleClient) UserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := oauth2.NewClient(ctx, g.oauth2Config.TokenSource(ctx, token))

	// Get user info
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var userInfo GoogleUserInfo

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &UserInfo{
		AccountID: userInfo.ID,
		Email:     userInfo.Email,
		Avatar:    userInfo.Picture,
		FirstName: userInfo.GivenName,
		LastName:  userInfo.FamilyName,
	}, nil
}

func (g *GoogleClient) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.oauth2Config.Exchange(ctx, code)
}

func (g *GoogleClient) AuthCodeURL(state string) string {
	return g.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}
