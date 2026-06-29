package analyticshandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// authorizedDomainID resolves the `domainId` query param and verifies it belongs
// to the request's authorized environment. The analytics queries filter on
// domain_id alone, so without this check any env member could read another
// tenant's analytics (including raw event metadata) by passing an arbitrary,
// enumerable domainId. On failure it returns a response the handler must return.
func authorizedDomainID(req *app.RequestContext) (types.ID, *shttp.Response) {
	domainID := utils.StringToID(req.Query().Get("domainId"))

	if domainID == 0 {
		return 0, shttp.BadRequest()
	}

	domain, err := buildconf.DomainStore().DomainByID(req.Context(), domainID)

	if err != nil {
		return 0, shttp.Error(err)
	}

	if domain == nil || domain.EnvID != req.EnvID {
		return 0, shttp.Forbidden()
	}

	return domainID, nil
}
