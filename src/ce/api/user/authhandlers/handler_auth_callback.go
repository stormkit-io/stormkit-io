package authhandlers

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/bitbucket"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/github"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/gitlab"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"go.uber.org/zap"
)

var responseTmpl = template.Must(template.New("oauthResponse").Parse(`
<!DOCTYPE html>
<html>
	<head>
		<title>Stormkit.io | Auth</title>
		<style>
			html, body {
				line-height: 1;
				font-family: 'Merriweather Sans',sans-serif;
				font-size: 0.9rem;
				color: #4f4f4f;
				background: #0f092b;
				color: #262525;
			}

			.wrapper {
				position: absolute;
				top: 50%;
				left: 50%;
				transform: translate(-50%, -50%);
				padding: 5rem 2rem;
				background-color: white;
				border-radius: 5px;
			}

			h1 {
				text-align: center;
			}
		</style>
	</head>
	<body>
		<div class="wrapper">
			<h1>{{.message}}</h1>
		</div>
		<script>
			window.opener && window.opener.postMessage({{.json}}, "*");
		</script>
	</body>
</html>
`))

// errorResponse is a helper function which sends an unknown error message
func errorResponse(err error, status int) *shttp.Response {
	if status == 0 {
		status = http.StatusInternalServerError
	}

	return cbResponse(status).
		json(jsonMsg{"error": err.Error()}).
		setError(err).
		send()
}

// handlerAuthCallback is responsible for handling the provider registration/login flow.
// It returns an html response. This will post a message to the parent window with the json bytes.
func handlerAuthCallback(req *shttp.RequestContext) *shttp.Response {
	query := req.Query()
	state := query.Get("state")
	code := query.Get("code")
	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: state})
	providerName := req.Vars()["provider"]

	if claims == nil || claims["provider"] != providerName {
		return cbResponse(http.StatusForbidden).json(jsonMsg{"auth": false, "error": "token-mismatch"}).send()
	}

	authUser, err := authUser(providerName, code)

	// Remove no-reply emails
	emails := []oauth.Email{}

	for _, email := range authUser.Emails {
		if strings.Contains(email.Address, "no-reply") || strings.Contains(email.Address, "noreply") {
			continue
		}

		emails = append(emails, email)
	}

	authUser.Emails = emails

	if err != nil {
		msg := jsonMsg{"auth": false, "error": err.Error()}

		if authUser != nil && len(authUser.Emails) == 0 {
			msg = jsonMsg{"email": false}
		}

		return cbResponse(http.StatusForbidden).json(msg).send()
	}

	return Login(req.Context(), authUser)
}

// Login logs in the authenticated user.
func Login(ctx context.Context, authUser *oauth.User) *shttp.Response {
	slog.Debug(slog.LogOpts{
		Msg:   "logging in user",
		Level: slog.DL2,
		Payload: []zap.Field{
			zap.String("provider", authUser.ProviderName),
			zap.String("displayName", authUser.DisplayName),
		},
	})

	store := user.NewStore()
	usr, err := store.MustUser(authUser)

	if usr != nil {
		slog.Debug(slog.LogOpts{
			Msg:   "user logged in",
			Level: slog.DL2,
			Payload: []zap.Field{
				zap.String("provider", authUser.ProviderName),
				zap.String("displayName", authUser.DisplayName),
				zap.String("userID", usr.ID.String()),
				zap.Bool("isNew", usr.IsNew),
			},
		})

		if !usr.IsAuthorizedToLogin() {
			return errorResponse(errors.New("pending-approval-or-rejected"), http.StatusUnauthorized)
		}
	}

	// Error while inserting
	if err != nil {
		return errorResponse(err, http.StatusBadRequest)
	}

	// Update the access token
	if err := oauth.NewStore().UpsertToken(usr.ID, authUser); err != nil {
		return errorResponse(err, 0)
	}

	if err := store.UpdateLastLogin(ctx, usr.ID); err != nil {
		return errorResponse(err, 0)
	}

	jwt, err := user.JWT(jwt.MapClaims{
		"uid": usr.ID.String(),
	})

	// Creating new token failed
	if err != nil {
		return errorResponse(err, 0)
	}

	accessToken := authUser.Token.AccessToken

	if authUser.ProviderName == github.ProviderName {
		accessToken = ""
	}

	return cbResponse(http.StatusOK).json(jsonMsg{
		"success":      true,
		"user":         usr,
		"sessionToken": jwt,
		"accessToken":  accessToken,
	}).send()
}

// authUser retrieves the authenticated user.
func authUser(providerName, code string) (*oauth.User, error) {
	switch providerName {
	case bitbucket.ProviderName:
		client, err := bitbucket.NewClientWithCode(code)
		if err != nil {
			return nil, err
		}

		return client.User.Profile()
	case github.ProviderName:
		client, err := github.NewClientWithCode(code)

		if err != nil {
			return nil, err
		}

		// check if the account is too new that is generally indicative of a bot or fraud
		profile, err := client.UserProfile()

		if err != nil {
			if !strings.Contains(err.Error(), "403") {
				slog.Errorf("error while accessing github repo: %v", err)
			}

			return nil, err
		}

		user, _, err := client.GetUser(context.Background(), profile.DisplayName)

		if user == nil || err != nil {
			slog.Errorf("error while getting github user=%v, err=%v", user, err)
			return nil, err
		}

		// don't allow accounts less than 5 days old
		fiveDaysAgo := time.Now().AddDate(0, 0, -5)
		if user.GetCreatedAt().After(fiveDaysAgo) {
			return nil, errors.New("account-too-new")
		}

		return profile, nil
	case gitlab.ProviderName:
		client, err := gitlab.NewClientWithCode(code)

		if err != nil {
			return nil, err
		}

		return client.UserProfile()
	default:
		return nil, user.ErrInvalidProvider
	}
}
