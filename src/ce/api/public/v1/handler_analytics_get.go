package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// analyticsDomainsPageSize is how many domains are read per query while
// collecting an environment's domains.
const analyticsDomainsPageSize = 250

func handlerAnalyticsGet(req *RequestContext) *shttp.Response {
	if resp := req.RequireEnterprise(); resp != nil {
		return resp
	}

	params, errs := analytics.ReportParamsFromQuery(req.Query())

	if len(errs) > 0 {
		return shttp.BadRequest(map[string]any{"errors": errs})
	}

	domains, err := envDomainIDs(req)

	if err != nil {
		return shttp.Error(err)
	}

	domainID := types.ID(0)

	if v := req.Query().Get("domainId"); v != "" {
		if domainID = utils.StringToID(v); domainID == 0 {
			return shttp.BadRequest(map[string]any{"error": "domainId must be a numeric domain id"})
		}
	}

	for _, id := range domains {
		if domainID == 0 || id == domainID {
			params.DomainIDs = append(params.DomainIDs, id)
		}
	}

	// A domain of another environment is reported as missing, not empty.
	if domainID != 0 && len(params.DomainIDs) == 0 {
		return shttp.NotFound()
	}

	report, err := analytics.NewStore().Report(req.Context(), params)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   report,
	}
}

// envDomainIDs returns the ids of every domain of the request's environment.
func envDomainIDs(req *RequestContext) ([]types.ID, error) {
	ids := []types.ID{}
	filters := buildconf.DomainFilters{EnvID: req.Env.ID, Limit: analyticsDomainsPageSize}

	for {
		page, err := buildconf.DomainStore().Domains(req.Context(), filters)

		if err != nil {
			return nil, err
		}

		// The store returns one row past the limit when there is a next page.
		hasNext := len(page) > analyticsDomainsPageSize

		if hasNext {
			page = page[:analyticsDomainsPageSize]
		}

		for _, d := range page {
			ids = append(ids, d.ID)
		}

		if !hasNext {
			return ids, nil
		}

		filters.AfterID = page[len(page)-1].ID
	}
}
