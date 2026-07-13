package userhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type updatePersonalAccessTokenRequest struct {
	Token string `json:"token"`
}

// handlerUpdatePersonalAccessToken is an endpoint to update the personal access token.
// When this token is provided, it is going to be used as a primary token for oauth2.
func handlerUpdatePersonalAccessToken(req *user.RequestContext) *shttp.Response {
	data := &updatePersonalAccessTokenRequest{}

	if err := req.Post(data); err != nil {
		return shttp.ValidationError(err)
	}

	if err := user.NewStore().UpdatePersonalAccessToken(req.User.ID, data.Token); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
