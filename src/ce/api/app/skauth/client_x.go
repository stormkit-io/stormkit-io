package skauth

import (
	"context"
	"encoding/json"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"golang.org/x/oauth2"
)

var TwitterAPIBase = "https://api.twitter.com/2"
var TwitterAuthBase = "https://twitter.com/i/oauth2/authorize"

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
			AuthURL:  TwitterAuthBase,
			TokenURL: TwitterAPIBase + "/oauth2/token",
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
	resp, err := client.Get(TwitterAPIBase + "/users/me?user.fields=profile_image_url,email")

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

// Exchange exchanges the authorization code for an access token.
// This method implements PKCE (Proof Key for Code Exchange) by:
// 1. Extracting the encrypted PKCE verifier from the JWT state parameter
// 2. Decrypting the verifier using AES-GCM encryption
// 3. Passing the verifier to Twitter/X to complete the PKCE flow
//
// The PKCE verifier was originally generated during the authorization request,
// encrypted and stored in the JWT state to protect it from being read by
// potential interceptors while maintaining a stateless flow.
//
// If no verifier is found, falls back to a standard OAuth2 exchange (non-PKCE).
func (x *XClient) Exchange(ctx context.Context, req *shttp.RequestContext) (*oauth2.Token, error) {
	code := req.FormValue("code")

	// Parse JWT claims from state parameter to extract encrypted PKCE verifier
	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer: req.FormValue("state"),
	})

	// Extract and decrypt the PKCE verifier if present
	if encryptedVerifier, ok := claims["pkce"].(string); ok && encryptedVerifier != "" {
		verifier := utils.DecryptToString(encryptedVerifier)

		if verifier != "" {
			// Use the decrypted verifier for PKCE
			return x.oauth2Config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
		}
	}

	slog.Infof("pkce missing - falling back to exchange without it")

	// Fallback to exchange without PKCE if verifier not found
	return x.oauth2Config.Exchange(ctx, code)
}

// AuthCodeURL implements PKCE (Proof Key for Code Exchange) verifier and
// challenge handling:
//
// 1. Generates a random plaintext PKCE verifier.
// 2. Encrypts the verifier and stores it in the JWT state claims under "pkce".
// 3. Generates a SHA256 hash challenge: BASE64URL(SHA256(verifier)).
// 4. Sends the challenge to Twitter/X with method "S256".
//
// By encrypting and embedding the verifier in the JWT state parameter here,
// the verifier remains secret even though the authorization URL with the
// public challenge is exposed to the user agent and intermediaries.
func (x *XClient) AuthCodeURL(params AuthCodeURLParams) (string, error) {
	token, err := utils.SecureRandomToken(64)

	if err != nil {
		slog.Errorf("failed to generate random state token (falling back to RandomToken): %v", err)
		token = utils.RandomToken(64)
	}

	claims := params.Claims()
	claims["pkce"] = utils.EncryptToString(token)

	state, err := user.JWT(claims)

	if err != nil {
		return "", err
	}

	return x.oauth2Config.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", utils.SHA256Hash([]byte(token))),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	), nil
}
