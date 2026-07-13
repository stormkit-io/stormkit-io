package audithandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var DefaultLimit = 100

func handlerAudits(req *user.RequestContext) *shttp.Response {
	query := req.Query()
	envID := utils.StringToID(query.Get("envId"))
	appID := utils.StringToID(query.Get("appId"))
	teamID := utils.StringToID(query.Get("teamId"))
	beforeID := utils.StringToID(query.Get("beforeId"))

	if envID == 0 && appID == 0 && teamID == 0 {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "At least one of the following properties are required: envId, appId, teamId.",
			},
		}
	}

	audits, err := audit.NewStore().SelectAudits(req.Context(), audit.AuditFilters{
		EnvID:    envID,
		AppID:    appID,
		TeamID:   teamID,
		BeforeID: beforeID,
		Limit:    DefaultLimit,
	})

	if err != nil {
		return shttp.Error(err)
	}

	response := []map[string]any{}
	auditsLen := len(audits)
	pagination := map[string]any{
		"hasNextPage": false,
	}

	if auditsLen > DefaultLimit {
		pagination["hasNextPage"] = true
		pagination["beforeId"] = audits[auditsLen-2].ID.String()
		audits = audits[:auditsLen-1]
	}

	for _, audit := range audits {
		response = append(response, audit.ToMap())
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"audits":     response,
			"pagination": pagination,
		},
	}
}
