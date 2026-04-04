package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// deploymentListLimit is the maximum number of deployments returned per page.
const deploymentListLimit = 20

func handlerDeploymentList(req *RequestContext) *shttp.Response {
	q := req.Query()
	v := &Validators{}

	from, err := v.ToInt(q.Get("from"), "from")

	if err != nil {
		return shttp.BadRequest(map[string]any{"errors": []string{err.Error()}})
	}

	depls, err := deploy.NewStore().MyDeployments(req.Context(), &deploy.DeploymentsQueryFilters{
		EnvID:  req.Env.ID,
		From:   from,
		Limit:  deploymentListLimit + 1,
		Branch: q.Get("branch"),
	})

	if err != nil {
		return shttp.Error(err)
	}

	hasNextPage := len(depls) > deploymentListLimit

	if hasNextPage {
		depls = depls[:deploymentListLimit]
	}

	deployments := make([]map[string]any, 0, len(depls))

	for _, d := range depls {
		deployments = append(deployments, d.JSON(false))
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"deployments": deployments,
			"hasNextPage": hasNextPage,
		},
	}
}
