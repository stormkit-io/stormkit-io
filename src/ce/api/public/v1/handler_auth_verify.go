package publicapiv1

import (
	"fmt"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// HandlerAuthVerify verifies an email address using the token sent during registration.
// On success it returns a session token.
// GET /v1/auth/verify?token=&envId=
func HandlerAuthVerify(req *shttp.RequestContext) *shttp.Response {
	token := req.Query().Get("token")
	envIDStr := req.Query().Get("envId")

	if token == "" {
		return shttp.BadRequest(map[string]any{"errors": []string{"token is required"}})
	}

	envID := utils.StringToID(envIDStr)

	if envID == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"envId is required"}})
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), envID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get environment by ID %d", envID))
	}

	if env == nil || env.AuthConf == nil || !env.AuthConf.Status || env.SchemaConf == nil {
		return shttp.NotFound()
	}

	prv, err := skauth.NewStore().Provider(req.Context(), envID, skauth.ProviderEmail)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get provider: %s", err.Error()))
	}

	if prv == nil || !prv.Status {
		return shttp.NotFound()
	}

	if claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: token, Secret: env.AuthConf.Secret}); claims == nil {
		return shttp.BadRequest(map[string]any{"errors": []string{"invalid or expired verification token"}})
	}

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get schema store: %s", err.Error()))
	}

	userID, err := store.VerifyEmailUser(req.Context(), token)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to verify email: %s", err.Error()))
	}

	if userID == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"invalid or expired verification token"}})
	}

	authUser, err := store.AuthUser(req.Context(), userID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to fetch user: %s", err.Error()))
	}

	if authUser == nil {
		return shttp.Error(fmt.Errorf("user %d not found after verification", userID), "internal error")
	}

	sessionToken, err := user.JWT(jwt.MapClaims{
		"uid": authUser.ID,
		"eid": fmt.Sprintf("%d", envID),
		"prv": skauth.ProviderEmail,
	}, env.AuthConf.Secret)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to generate session token: %s", err.Error()))
	}

	return &shttp.Response{
		Data: map[string]any{"token": sessionToken, "email": authUser.Email, "userId": authUser.ID},
	}
}
