package authwallhandlers

import (
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func addQueryParamToURL(referrer, param, value string) *string {
	if referrer != "" {
		if u, err := url.Parse(referrer); err == nil && u != nil {
			query := u.Query()
			query.Add(param, value)
			u.RawQuery = query.Encode()
			referrer = u.String()
		}
	}

	return &referrer
}

func failedLoginResponse(referrer, errCode string) *shttp.Response {
	return &shttp.Response{
		Redirect: addQueryParamToURL(referrer, "stormkit_error", errCode),
	}
}

func handlerAuth(req *shttp.RequestContext) *shttp.Response {
	email := req.FormValue("email")
	password := req.FormValue("password")
	envID := utils.StringToID(req.FormValue("envId"))
	tokens := authwall.Token{EnvID: envID}
	form := tokens.ParseForm(req.FormValue("token"))

	if form == nil {
		return shttp.BadRequest(map[string]any{
			"error": "The login form has expired. Go back, reload the page and try again.",
		})
	}

	// The form token carries the protected page's URL. The Referer header
	// only upgrades it to https when a proxy in front terminated TLS; it is
	// never used as the target, since a form on another site could otherwise
	// have the visitor, and on success their session, sent there.
	referrer := loginRedirect{returnTo: form.ReturnTo, referer: req.Referer()}.target()

	if form.Expired {
		return failedLoginResponse(referrer, "invalid_token")
	}

	if email == "" || password == "" {
		return failedLoginResponse(referrer, "invalid_credentials")
	}

	aw := &authwall.AuthWall{
		LoginEmail:    email,
		LoginPassword: password,
		EnvID:         envID,
	}

	jwtToken, err := tokens.Session()

	if err != nil || jwtToken == "" {
		return failedLoginResponse(referrer, "token_generation_failed")
	}

	store := authwall.Store()

	if valid, err := store.Login(req.Context(), aw); err != nil || !valid {
		return failedLoginResponse(referrer, "invalid_credentials")
	}

	if err := store.UpdateLastLogin(req.Context(), aw.LoginID); err != nil {
		slog.Errorf("error while updating last login: %s", err.Error())
	}

	return &shttp.Response{
		Redirect: addQueryParamToURL(referrer, "stormkit_success", jwtToken),
	}
}

// loginRedirect decides where a visitor goes after submitting the login form.
type loginRedirect struct {
	returnTo string
	referer  string
}

// target returns the return URL, upgraded to https when the Referer names the
// same host over https.
func (r loginRedirect) target() string {
	target, err := url.Parse(r.returnTo)

	if err != nil {
		return r.returnTo
	}

	if referer, err := url.Parse(r.referer); err == nil && referer.Scheme == "https" && strings.EqualFold(referer.Host, target.Host) {
		target.Scheme = "https"
	}

	return target.String()
}
