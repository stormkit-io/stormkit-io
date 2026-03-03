package publicapiv1

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func HandlerAuthCallback(req *shttp.RequestContext) *shttp.Response {
	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer: req.FormValue("state"),
	})

	provider, ok := claims["prv"].(string)
	envID, envOK := claims["eid"].(string)
	refer, refOK := claims["ref"].(string)

	if !refOK || !envOK || !ok || !utils.InSliceString(skauth.Providers, provider) {
		return shttp.BadRequest(map[string]any{
			"error": "invalid state parameter",
		})
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), utils.StringToID(envID))

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get environment by ID %s", envID))
	}

	if env == nil {
		return shttp.NotFound()
	}

	if env.SchemaConf == nil || env.AuthConf == nil || !env.AuthConf.Status {
		return shttp.BadRequest(map[string]any{
			"error": "Stormkit Auth is not enabled for this environment",
		})
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

	token, err := client.Exchange(req.Context(), req)

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
		"eid": envID,
		"prv": provider,
	}, secret)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to generate JWT: %s", err.Error()))
	}

	code := utils.RandomToken(64)

	if err := rediscache.Client().Set(req.Context(), code, sessionToken, time.Minute*2).Err(); err != nil {
		return shttp.Error(err, fmt.Sprintf("failed saving random token: %s", err.Error()))
	}

	parsed, err := url.ParseRequestURI(refer)

	if err != nil {
		return shttp.BadRequest(map[string]any{
			"error": "Referrer URL is not a valid format",
		})
	}

	req.Redirect(fmt.Sprintf("%s://%s/_stormkit/auth", utils.GetString(parsed.Scheme, "https"), parsed.Host), http.StatusFound)
	return nil
}
