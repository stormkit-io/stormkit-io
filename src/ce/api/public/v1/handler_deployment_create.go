package publicapiv1

import (
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type DeploymentCreateRequest struct {
	// Branch is the git branch to deploy. Defaults to the app's default branch.
	Branch string `json:"branch"`
	// Publish indicates whether the deployment should be published (made live) after a successful build.
	Publish bool `json:"publish"`
}

func handlerDeploymentCreate(req *RequestContext) *shttp.Response {
	data := &DeploymentCreateRequest{}
	err := req.Post(data)

	if err != nil {
		return shttp.Error(err)
	}

	if req.App == nil {
		req.App, err = app.NewStore().AppByID(req.Context(), req.Env.AppID)

		if err != nil {
			return shttp.Error(err, fmt.Sprintf("error while fetching app for deploy api: %v", err.Error()))
		}

		if req.App == nil {
			return &shttp.Response{
				Status: http.StatusNotFound,
				Data: map[string]string{
					"error": "App not found or deleted.",
				},
			}
		}
	}

	depl := deploy.New(req.App)
	depl.PopulateFromEnv(req.Env)
	depl.Branch = utils.GetString(data.Branch, req.Env.Branch)
	depl.ShouldPublish = data.Publish

	if err = deployservice.New().Deploy(req.Context(), req.App, depl); err != nil {
		if err == oauth.ErrRepoNotFound || err == oauth.ErrCredsInvalidPermissions {
			return &shttp.Response{
				Status: http.StatusNotFound,
				Data: map[string]string{
					"error": "Repository is not found or is inaccessible.",
				},
			}
		}

		if err == deployservice.ErrBuildMinutesExceeded {
			return &shttp.Response{
				Status: http.StatusPaymentRequired,
				Data: map[string]string{
					"error": "You have exceeded your build minutes limit. Please upgrade your plan to continue building your projects.",
				},
			}
		}

		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusCreated,
		Data:   depl.JSON(false),
	}
}
