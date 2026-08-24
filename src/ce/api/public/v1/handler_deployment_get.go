package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerDeploymentGet(req *RequestContext) *shttp.Response {
	id := utils.StringToID(req.Vars()["id"])

	if id == 0 {
		return shttp.NotFound()
	}

	withLogs := req.Query().Get("logs") == "true"

	fetch := func(includeLogs bool) (*deploy.Deployment, error) {
		return deploy.NewStore().MyDeployment(req.Context(), &deploy.DeploymentsQueryFilters{
			DeploymentID: id,
			EnvID:        req.Env.ID,
			IncludeLogs:  &includeLogs,
		})
	}

	depl, err := fetch(withLogs)

	if err != nil {
		return shttp.Error(err)
	}

	if depl == nil {
		return shttp.NotFound()
	}

	data := depl.JSON(withLogs)

	// A failed deployment carries its reason inline so a caller can triage
	// without a second call, plus a link to the full logs. The logs are only
	// loaded here -- not on the hot path of polling a running deployment -- and
	// only when they were not already fetched.
	if depl.Status() == "failed" {
		if !withLogs {
			if full, ferr := fetch(true); ferr == nil && full != nil {
				depl = full
			}
		}

		if summary := depl.FailureSummary(); summary != "" {
			data["failureSummary"] = summary
		}

		data["logsUrl"] = deploymentLogsURL(depl)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"deployment": data,
		},
	}
}

// deploymentLogsURL returns a fully-qualified link to the deployment's details
// page (where the full logs render), falling back to the relative path when no
// app domain is configured.
func deploymentLogsURL(depl *deploy.Deployment) string {
	path := depl.DetailsPath()

	if url := admin.MustConfig().AppURL(path); url != "" {
		return url
	}

	return path
}
