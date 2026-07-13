package adminhandlers

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type ImpersonateRequest struct {
	UserID types.ID `json:"userId"`
}

func handlerImpersonate(req *user.RequestContext) *shttp.Response {
	data := ImpersonateRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if data.UserID == 0 {
		return shttp.BadRequest(map[string]any{
			"error": "userId is required",
		})
	}

	usr, err := user.NewStore().UserByID(data.UserID)

	if err != nil {
		return shttp.Error(err)
	}

	if usr == nil {
		return shttp.BadRequest(map[string]any{
			"error": "User not found",
		})
	}

	if usr.IsAdmin {
		return shttp.NotAllowed()
	}

	jwt, err := user.JWT(jwt.MapClaims{
		"uid": data.UserID.String(),
	})

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: 200,
		Data: map[string]any{
			"token": jwt,
		},
	}
}
