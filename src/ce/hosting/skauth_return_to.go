package hosting

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// oauthAuthorizePath is the only place a sign-in started from the app's OAuth
// page may return to.
const oauthAuthorizePath = "/_stormkit/oauth/authorize"

// oauthSignInParam tells the OAuth page how a sign-in started from it went. It
// carries one of the oauthSignIn* codes rather than a message, so a crafted
// link cannot put arbitrary text on the page.
const oauthSignInParam = "sk_sign_in"

const (
	oauthSignInLinkSent     = "link_sent"
	oauthSignInInvalidEmail = "invalid_email"
	oauthSignInExpired      = "expired"
	oauthSignInFailed       = "failed"
)

// oauthReturnTo validates where a sign-in started from the app's OAuth page
// returns to. Only this host's authorize endpoint is accepted, and it is
// returned as a relative path and query so the redirect can never leave the
// host. It returns "" for anything else, including when the OAuth server is
// disabled.
//
// The value rides the magic-link token and the Google/X state, so the pending
// authorization survives the round trip through an inbox or a provider without
// depending on the browser that started it.
func (m *skAuthMiddleware) oauthReturnTo(raw string) string {
	cnf := m.req.Host.Config

	if raw == "" || cnf == nil || !cnf.SKAuth.OAuthServerEnabled() {
		return ""
	}

	u, err := url.Parse(raw)

	if err != nil || u.Opaque != "" || u.User != nil || u.Path != oauthAuthorizePath {
		return ""
	}

	if u.Scheme != "" || u.Host != "" {
		if u.Scheme != "https" || !strings.EqualFold(u.Host, m.req.Host.Name) {
			return ""
		}
	}

	q := u.Query()
	q.Del(oauthSignInParam)

	return (&url.URL{Path: oauthAuthorizePath, RawQuery: q.Encode()}).String()
}

// backToOAuth sends the visitor to the OAuth page with the outcome of the
// sign-in they started there.
func (m *skAuthMiddleware) backToOAuth(p backToOAuthParams) *shttp.Response {
	target := appendQuery(p.ReturnTo, map[string]string{oauthSignInParam: p.Outcome})

	return &shttp.Response{
		Status:   http.StatusSeeOther,
		Redirect: &target,
		Headers: shttp.HeadersFromMap(map[string]string{
			"Cache-Control":   "no-store",
			"Referrer-Policy": "no-referrer",
		}),
	}
}

type backToOAuthParams struct {
	// ReturnTo is a value already validated by oauthReturnTo.
	ReturnTo string

	// Outcome is one of the oauthSignIn* codes.
	Outcome string
}

// isFormPost reports whether the request is an HTML form submission rather than
// a fetch/XHR call with a JSON body.
func isFormPost(r *http.Request) bool {
	mediaType := strings.TrimSpace(strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0])

	return strings.EqualFold(mediaType, "application/x-www-form-urlencoded") ||
		strings.EqualFold(mediaType, "multipart/form-data")
}
