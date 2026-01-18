package skauthhandlers

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerAuthCallback(req *shttp.RequestContext) *shttp.Response {
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
		return shttp.Error(err)
	}

	if env == nil || env.SchemaConf == nil {
		return shttp.NotFound()
	}

	config, err := skauth.NewStore().Config(req.Context(), skauth.ConfigArgs{
		EnvID:        env.ID,
		ProviderName: provider,
	})

	if err != nil {
		return shttp.Error(err)
	}

	// Exchange authorization code for token
	token, err := Exchange(req.Context(), config, req.FormValue("code"))

	if err != nil {
		return shttp.Error(err)
	}

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return shttp.Error(err)
	}

	info, err := GetProviderClient(provider, config.ClientID, config.ClientSecret).UserInfo(req.Context(), token)

	if err != nil {
		return shttp.Error(err)
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
		return shttp.Error(err)
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
		return shttp.Error(err)
	}

	return &shttp.Response{
		Data: map[string]any{
			"token": fmt.Sprintf("%s:%s", env.ID, sessionToken),
		},
	}
}
