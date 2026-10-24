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

	domains, err := envDomains(req)

	if err != nil {
		return shttp.Error(err)
	}

	domainID := types.ID(0)

	if v := req.Query().Get("domainId"); v != "" {
		if domainID = utils.StringToID(v); domainID == 0 {
			return shttp.BadRequest(map[string]any{"error": "domainId must be a numeric domain id"})
		}
	}

	for _, d := range domains {
		if domainID == 0 || d.ID == domainID {
			params.DomainIDs = append(params.DomainIDs, d.ID)
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

// envDomains returns every domain of the request's environment.
func envDomains(req *RequestContext) ([]*buildconf.DomainModel, error) {
	all := []*buildconf.DomainModel{}
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

		all = append(all, page...)

		if !hasNext {
			return all, nil
		}

		filters.AfterID = page[len(page)-1].ID
	}
}
