package domainhandlers

import (
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type CertPutRequest struct {
	DomainID types.ID `json:"domainId"`
	Key      string   `json:"certKey"`
	Cert     string   `json:"certValue"`
}

func HandlerCertPut(req *app.RequestContext) *shttp.Response {
	data := CertPutRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.BadRequest().SetError(err)
	}

	if !strings.Contains(data.Key, "-----BEGIN PRIVATE KEY-----") {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Invalid private key provided.",
			},
		}
	}

	if !strings.Contains(data.Cert, "-----BEGIN CERTIFICATE-----") {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Invalid certificate provided.",
			},
		}
	}

	store := buildconf.DomainStore()
	domain, err := store.DomainByID(req.Context(), data.DomainID)

	if err != nil {
		return shttp.Error(err)
	}

	if domain == nil || domain.AppID != req.App.ID {
		return shttp.NotFound()
	}

	var oldCert string
	var oldKey string

	if domain.CustomCert != nil {
		oldCert = domain.CustomCert.Value
		oldKey = domain.CustomCert.Key
	}

	domain.CustomCert = &buildconf.CustomCert{
		Value: data.Cert,
		Key:   data.Key,
	}

	if err := store.UpdateDomainCert(req.Context(), domain); err != nil {
		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		err = audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeDomain).
			WithDiff(&audit.Diff{
				Old: audit.DiffFields{DomainName: domain.Name, DomainCertValue: oldCert, DomainCertKey: oldKey},
				New: audit.DiffFields{DomainName: domain.Name, DomainCertValue: data.Cert, DomainCertKey: data.Key},
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
