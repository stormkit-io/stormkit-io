package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// HandlerDomainDelete deletes the associated domain for the given environment.
func HandlerDomainDelete(req *RequestContext) *shttp.Response {
	query := req.Query()
	id := query.Get("domainId")

	if id == "" {
		id = query.Get("id")
	}

	domainID := utils.StringToID(id)

	if domainID == 0 {
		return shttp.BadRequest()
	}

	domain, res := req.domainInEnv(domainID)

	// Deleting is idempotent: a domain that is already gone — or that belongs to
	// another environment, which the caller must not be able to tell apart —
	// reports success without touching anything.
	if res != nil {
		if res.Status == http.StatusNotFound {
			return shttp.NoContent()
		}

		return res
	}

	// Reset the cache first because the Reset function checks the Database
	// for the domain names.
	if err := appcache.Service().Reset(0, domain.Name); err != nil {
		return shttp.Error(err)
	}

	err := buildconf.DomainStore().DeleteDomain(req.Context(), buildconf.DeleteDomainArgs{
		DomainID: domainID,
	})

	if err != nil {
		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		err = audit.FromRequestContext(req).
			WithAction(audit.DeleteAction, audit.TypeDomain).
			WithDiff(&audit.Diff{Old: audit.DiffFields{DomainName: domain.Name}}).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}
