package adminhandlers

import (
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type UserManagementSetRequest struct {
	SignUpMode string   `json:"signUpMode"`
	Whitelist  []string `json:"whitelist"`
}

func handlerUserManagementSet(req *user.RequestContext) *shttp.Response {
	data := UserManagementSetRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	allowed := []string{
		admin.SIGNUP_MODE_OFF,
		admin.SIGNUP_MODE_ON,
		admin.SIGNUP_MODE_WAITLIST,
	}

	if !utils.InSliceString(allowed, data.SignUpMode) {
		return shttp.BadRequest(map[string]any{
			"error": fmt.Sprintf("Invalid sign up mode provided. Must be one of: %s", strings.Join(allowed, ", ")),
		})
	}

	vc := admin.MustConfig()

	if vc.AuthConfig == nil {
		vc.AuthConfig = &admin.AuthConfig{}
	}

	vc.AuthConfig.UserManagement = admin.UserManagement{
		Whitelist:  data.Whitelist,
		SignUpMode: data.SignUpMode,
	}

	if err := admin.Store().UpsertConfig(req.Context(), vc); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
