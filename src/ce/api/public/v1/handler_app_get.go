package publicapiv1

import (
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// handlerAppGet returns a single application by its ID.
// Authentication requires a an API key with at least team scope (user-scoped or team-scoped).
//
// Path parameters:
//
//	appId - Required. The ID of the application to retrieve.
//
// Response:
//
//	app - The application object.
func handlerAppGet(req *RequestContext) *shttp.Response {
	v := &Validators{}

	appIdInt, err := v.ToInt(req.Vars()["appId"], "appId")

	if err != nil {
		return shttp.BadRequest(map[string]any{
			"error": err.Error(),
		})
	}

	if appIdInt == 0 {
		return shttp.BadRequest(map[string]any{
			"error": "The 'appId' path parameter is required",
		})
	}

	appID := types.ID(appIdInt)

	myApp, err := app.NewStore().AppByID(req.Context(), appID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to fetch app %d: %s", appID, err.Error()))
	}

	if myApp == nil {
		return shttp.NotFound()
	}

	// Verify the authenticated user is a member of the app's team.
	var isMember bool

	if req.Token.TeamID != 0 {
		isMember = req.Token.TeamID == myApp.TeamID
	} else {
		isMember = team.NewStore().IsMember(req.Context(), req.Token.UserID, myApp.TeamID)
	}

	if !isMember {
		return shttp.Forbidden()
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"app": myApp.JSON(),
		},
	}
}
