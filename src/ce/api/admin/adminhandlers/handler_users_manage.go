package adminhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type UserManageRequest struct {
	UserIDs []types.ID `json:"userIds"`
	Action  string     `json:"action"` // "approve" or "reject"
}

func handlerUsersManage(req *user.RequestContext) *shttp.Response {
	data := UserManageRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if data.Action != "approve" && data.Action != "reject" {
		return shttp.BadRequest(map[string]any{
			"error": "Invalid action provided. Must be either 'approve' or 'reject'",
		})
	}

	if len(data.UserIDs) == 0 {
		return shttp.BadRequest(map[string]any{
			"error": "No user IDs provided",
		})
	}

	err := user.NewStore().UpdateApprovalStatus(req.Context(), data.UserIDs, data.Action == "approve")

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
	}
}
