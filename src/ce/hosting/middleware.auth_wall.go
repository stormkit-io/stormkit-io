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
}

func WithAuthWall(req *RequestContext) (*shttp.Response, error) {
	authWall := req.Host.Config.AuthWall

	if authWall == "" {
		return nil, nil
	}

	if authWall == "dev" && !req.Host.IsStormkitSubdomain {
		return nil, nil
	}

	tokens := authwall.Token{EnvID: req.Host.Config.EnvID}

	// Already logged in for this environment
	if cookie, err := req.Cookie(SESSION_COOKIE_NAME); cookie != nil && err == nil && tokens.IsValidSession(cookie.Value) {
		return nil, nil
	}

	// User logged in successfully:
	// - Remove stormkit_success query parameter
	// - Create a server-side cookie to handle session management
	// - Redirect the user back to the original page
	if token := req.Query().Get("stormkit_success"); token != "" {
		if tokens.IsValidSession(token) {
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

	token, _ := tokens.Form()
	content := html.MustRender(html.RenderArgs{
		PageTitle:   "Stormkit - Password protected deployment",
		PageContent: html.Templates["login"],
		ContentData: map[string]any{
			"api_host": admin.MustConfig().ApiURL(""),
			"env_id":   req.Host.Config.EnvID.String(),
			"token":    token,
			"error":    stormkitErrors[req.Query().Get("stormkit_error")],
			"title":    "Password protected deployment",
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
