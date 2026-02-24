package publicapiv1

import (
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth/skauthhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func HandlerSession(req *shttp.RequestContext) *shttp.Response {
	bearer := user.ParseBearer(req.Header.Get("Authorization"))
	pieces := strings.SplitN(bearer, ":", 2)
	envID := utils.StringToID(pieces[0])

	if len(pieces) != 2 {
		return shttp.BadRequest(map[string]any{
			"error": "Invalid Bearer token",
		})
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), envID)

	if err != nil {
		return shttp.Error(err)
	}

	if env == nil || env.AuthConf == nil || env.SchemaConf == nil {
		return shttp.NotAllowed()
	}

	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer:  pieces[1],
		Secret:  env.AuthConf.Secret,
		MaxMins: int(env.AuthConf.TTL),
	})

	if claims == nil {
		return shttp.NotAllowed()
	}

	userID, ok := claims["uid"].(string)

	if !ok || utils.StringToID(userID) == 0 {
		return shttp.NotAllowed()
	}

	user, err := skauthhandlers.AuthUser(req.Context(), env, utils.StringToID(userID))

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Data: map[string]any{
			"user": user,
		},
	}
}
