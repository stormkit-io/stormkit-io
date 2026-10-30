package hosting

import (
	"net/http"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var stormkitErrors = map[string]string{
	"invalid_credentials":     "Credentials are invalid. Contact your administrator for help.",
	"invalid_token":           "Token expired. Please submit the form again.",
	"token_generation_failed": "Token generation failed. Please try again.",
	"invalid_email":           "Enter a valid email to get a sign-in link.",
	"magic_link_unavailable":  "Sign-in links are not available. Log in with your password.",
}

// Headers that carry the Auth Wall login to the app.
const (
	HeaderAuthWallEmail    = "X-Auth-Wall-Email"
	HeaderAuthWallMetadata = "X-Auth-Wall-Metadata"

	// HeaderAuthWallTrigger carries the ID of the periodic trigger that sent
	// the request, when the request came through the wall as one.
	HeaderAuthWallTrigger = "X-Auth-Wall-Trigger"
)

// WithAuthWall shows the login form to visitors of a protected environment
// until they log in, and passes the login's email and metadata to the app.
func WithAuthWall(req *RequestContext) (*shttp.Response, error) {
	// Only the wall sets these, so a visitor cannot pose as a login.
	req.Header.Del(HeaderAuthWallEmail)
	req.Header.Del(HeaderAuthWallMetadata)
	req.Header.Del(HeaderAuthWallTrigger)

	// The trigger's token never reaches the app, which could otherwise replay
	// it while it is still valid.
	triggerToken := req.Header.Get(authwall.HeaderTriggerToken)
	req.Header.Del(authwall.HeaderTriggerToken)

	if !req.Host.IsAuthWallProtected() {
		return nil, nil
	}

	tokens := authwall.Token{EnvID: req.Host.Config.EnvID}

	if triggerID := tokens.ParseTrigger(triggerToken); triggerID != 0 {
		req.Header.Set(HeaderAuthWallTrigger, triggerID.String())
		return nil, nil
	}

	// Already logged in for this environment
	if cookie, err := req.Cookie(SESSION_COOKIE_NAME); cookie != nil && err == nil {
		if session := tokens.ParseSession(cookie.Value); session != nil {
			if session.Email != "" {
				req.Header.Set(HeaderAuthWallEmail, session.Email)
			}

			if session.Metadata != "" {
				req.Header.Set(HeaderAuthWallMetadata, session.Metadata)
			}

			return nil, nil
		}
	}

	// User logged in successfully:
	// - Remove stormkit_success query parameter
	// - Create a server-side cookie to handle session management
	// - Redirect the user back to the original page
	if token := req.Query().Get("stormkit_success"); token != "" {
		if tokens.ParseSession(token) != nil {
			url := req.URL()
			query := url.Query()
			query.Del("stormkit_success")
			url.RawQuery = query.Encode()
			redirectURL := url.String()

			return &shttp.Response{
				Cookies: []http.Cookie{{
					Name:     SESSION_COOKIE_NAME,
					Value:    token,
					Path:     "/",
					Expires:  utils.NewUnix().Add(time.Hour * 24),
					HttpOnly: true,
					Secure:   true,
					SameSite: http.SameSiteStrictMode,
				}},
				Redirect: &redirectURL,
				Status:   http.StatusFound,
			}, nil
		}
	}

	// The page URL, without the parameters of a previous login attempt, is
	// where the visitor returns after submitting the form. The query is only
	// rewritten when needed, so the page's own parameters keep their order.
	returnTo := *req.URL()
	query := returnTo.Query()

	if query.Has("stormkit_error") || query.Has("stormkit_success") {
		query.Del("stormkit_error")
		query.Del("stormkit_success")
		returnTo.RawQuery = query.Encode()
	}

	token, _ := tokens.Form(returnTo.String())
	content := html.MustRender(html.RenderArgs{
		PageTitle:   "Stormkit - Password protected deployment",
		PageContent: html.Templates["login"],
		ContentData: map[string]any{
			"api_host":   admin.MustConfig().ApiURL(""),
			"env_id":     req.Host.Config.EnvID.String(),
			"token":      token,
			"error":      stormkitErrors[req.Query().Get("stormkit_error")],
			"title":      "Password protected deployment",
			"magic_link": authwall.MagicLinkEmail{}.Available(),
		},
	})

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   content,
		Headers: http.Header{
			"Content-Type": []string{"text/html; charset=utf-8"},
		},
	}, nil
}
