package authwallhandlers

import (
	"net/http"
	"net/url"

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

	// The form token carries the protected page's URL. The Referer header is
	// ignored: a form on another site could otherwise have the visitor, and
	// on success their session, sent there.
	referrer, ok := tokens.FormReturnTo(req.FormValue("token"))

	if !ok {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data:   "The login form has expired. Go back, reload the page and try again.",
		}
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
