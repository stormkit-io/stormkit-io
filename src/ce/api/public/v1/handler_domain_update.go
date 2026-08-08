package publicapiv1

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type DomainUpdateRequest struct {
	DomainID types.ID `json:"domainId"`

	// AnalyticsExcluded is a pointer so an omitted field leaves the current
	// value alone rather than resetting it to false.
	AnalyticsExcluded *bool `json:"analyticsExcluded"`
}

// HandlerDomainUpdate updates the per-domain settings. The domain name itself is
// immutable — delete and re-add to change it.
func HandlerDomainUpdate(req *RequestContext) *shttp.Response {
	data := DomainUpdateRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.BadRequest().SetError(err)
	}

	if data.DomainID == 0 {
		return shttp.BadRequest()
	}

	domain, res := req.domainInEnv(data.DomainID)

	if res != nil {
		return res
	}

	// Nothing to do when the field is omitted or already holds the requested
	// value — writing anyway would purge the host config cache on every node and
	// add an audit entry whose old and new values are identical.
	if data.AnalyticsExcluded == nil || *data.AnalyticsExcluded == domain.AnalyticsExcluded {
		return shttp.OK()
	}

	oldExcluded := domain.AnalyticsExcluded
	domain.AnalyticsExcluded = *data.AnalyticsExcluded

	if err := buildconf.DomainStore().UpdateDomainFlags(req.Context(), domain); err != nil {
		return shttp.Error(err)
	}

	// The hosting layer reads the flag from the cached host config, so the cache
	// has to be dropped for the change to take effect on the next request. This
	// runs before the audit insert so that a failing audit cannot leave the nodes
	// serving the stale flag.
	if err := appcache.Service().Reset(0, domain.Name); err != nil {
		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		err := audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeDomain).
			WithDiff(&audit.Diff{
				Old: audit.DiffFields{DomainName: domain.Name, DomainAnalyticsExcluded: &oldExcluded},
				New: audit.DiffFields{DomainName: domain.Name, DomainAnalyticsExcluded: &domain.AnalyticsExcluded},
			}).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}
