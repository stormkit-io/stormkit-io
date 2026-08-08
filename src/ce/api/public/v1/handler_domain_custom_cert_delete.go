package publicapiv1

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func HandlerCertDelete(req *RequestContext) *shttp.Response {
	id := req.Query().Get("domainId")

	if id == "" {
		id = req.Query().Get("id")
	}

	domainID := utils.StringToID(id)

	domain, res := req.domainInEnv(domainID)

	if res != nil {
		return res
	}

	store := buildconf.DomainStore()

	var oldCert string
	var oldKey string

	if domain.CustomCert != nil {
		oldCert = domain.CustomCert.Value
		oldKey = domain.CustomCert.Key
	}

	domain.CustomCert = nil

	if err := store.UpdateDomainCert(req.Context(), domain); err != nil {
		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		err := audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeDomain).
			WithDiff(&audit.Diff{
				Old: audit.DiffFields{DomainName: domain.Name, DomainCertValue: oldCert, DomainCertKey: oldKey},
				New: audit.DiffFields{DomainName: domain.Name, DomainCertValue: "", DomainCertKey: ""},
			}).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	if err := appcache.Service().Reset(0, domain.Name); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
