package authwallhandlers

import (
	"net/url"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
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
	referrer := req.Referer()

	token := user.ParseJWT(&user.ParseJWTArgs{
		Bearer:  req.FormValue("token"),
		MaxMins: 5,
	})

	if token == nil {
		return failedLoginResponse(referrer, "invalid_token")
	}

	if email == "" || password == "" || envID == 0 {
		return failedLoginResponse(referrer, "invalid_credentials")
	}

	aw := &authwall.AuthWall{
		LoginEmail:    email,
		LoginPassword: password,
		EnvID:         envID,
	}

	jwtToken, err := user.JWT(jwt.MapClaims{})

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
