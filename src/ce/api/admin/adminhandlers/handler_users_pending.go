package adminhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerUsersPending(req *user.RequestContext) *shttp.Response {
	pendingUsers, err := user.NewStore().PendingUsers(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	users := make([]map[string]any, 0, len(pendingUsers))

	for _, usr := range pendingUsers {
		users = append(users, usr.JSON())
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"users": users,
		},
	}
}
