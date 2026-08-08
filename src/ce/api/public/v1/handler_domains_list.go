package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var DefaultDomainsLimit = 100
var MaxDomainsLimit = 250

func HandlerDomainsList(req *RequestContext) *shttp.Response {
	query := req.Query()
	filters := buildconf.DomainFilters{
		EnvID: req.Env.ID,
		Limit: DefaultDomainsLimit,
	}

	if p := query.Get("pageSize"); p != "" {
		size := utils.GetInt(utils.StringToInt(p), DefaultDomainsLimit)

		if size > MaxDomainsLimit {
			size = MaxDomainsLimit
		}

		filters.Limit = size
	}

	if p := query.Get("verified"); p != "" {
		verified := p == "true"
		filters.Verified = &verified
	}

	if p := query.Get("domainName"); p != "" {
		filters.DomainName = p
	}

	if afterId := query.Get("afterId"); afterId != "" {
		filters.AfterID = utils.StringToID(afterId)
	}

	domains, err := buildconf.DomainStore().Domains(req.Context(), filters)

	if err != nil {
		return shttp.Error(err)
	}

	domainsLen := len(domains)
	pagination := map[string]any{
		"hasNextPage": false,
	}

	if domainsLen > filters.Limit {
		pagination = map[string]any{
			"hasNextPage": true,
			"afterId":     domains[domainsLen-2].ID.String(),
		}

		domains = domains[:domainsLen-1]
	}

	response := []map[string]any{}

	for _, domain := range domains {
		response = append(response, map[string]any{
			"id":                domain.ID.String(),
			"domainName":        domain.Name,
			"verified":          domain.Verified,
			"token":             domain.Token.ValueOrZero(),
			"customCert":        domain.CustomCert,
			"lastPing":          domain.LastPing,
			"analyticsExcluded": domain.AnalyticsExcluded,
		})
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"domains":    response,
			"pagination": pagination,
		},
	}
}
