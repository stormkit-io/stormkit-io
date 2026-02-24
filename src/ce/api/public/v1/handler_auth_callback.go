package publicapiv1

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func HandlerAuthCallback(req *shttp.RequestContext) *shttp.Response {
	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer: req.FormValue("state"),
	})

	provider, ok := claims["prv"].(string)
	envID, envOK := claims["eid"].(string)

	if !envOK || !ok || !utils.InSliceString(skauth.Providers, provider) {
		return shttp.BadRequest(map[string]any{
			"error": "invalid state parameter",
		})
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), utils.StringToID(envID))

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get environment by ID %s", envID))
	}

	if env == nil || env.SchemaConf == nil {
		return shttp.NotFound()
	}

	prv, err := skauth.NewStore().Provider(req.Context(), env.ID, provider)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get provider for env %d and provider %s", env.ID, provider))
	}

	if prv == nil || !prv.Status {
		return shttp.NotFound()
	}

	// Exchange authorization code for token
	client := prv.Client()

	if client == nil {
		return shttp.BadRequest(map[string]any{
			"error": "Provider is not an OAuth2 provider",
		})
	}

	token, err := client.Exchange(req.Context(), req.FormValue("code"))

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to exchange authorization code for token: %s", err.Error()))
	}

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get store for environment: %s", err.Error()))
	}

	info, err := client.UserInfo(req.Context(), token)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get user info: %s", err.Error()))
	}

	if info == nil {
		return shttp.NotFound()
	}

	oauth := skauth.OAuth{
		AccountID:    info.AccountID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       utils.UnixFrom(token.Expiry),
		ProviderName: provider,
	}

	usr := skauth.User{
		Email:     info.Email,
		Avatar:    info.Avatar,
		FirstName: info.FirstName,
		LastName:  info.LastName,
	}

	if err := store.InsertAuthUser(req.Context(), &oauth, &usr); err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to insert auth user: %s", err.Error()))
	}

	var secret string

	if env.AuthConf != nil {
		secret = env.AuthConf.Secret
	}

	sessionToken, err := user.JWT(jwt.MapClaims{
		"uid": usr.ID,
		"prv": provider,
	}, secret)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to generate JWT: %s", err.Error()))
	}

	return &shttp.Response{
		Data: map[string]any{
			"token": fmt.Sprintf("%s:%s", env.ID, sessionToken),
		},
	}
}
