package authwallhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerAuthLogins(req *app.RequestContext) *shttp.Response {
	users, err := authwall.Store().Logins(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	data := []map[string]any{}

	for _, user := range users {
		data = append(data, map[string]any{
			"id":        user.LoginID,
			"email":     user.LoginEmail,
			"lastLogin": user.LastLogin.Unix(),
		})
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"logins": data,
		},
	}
}
