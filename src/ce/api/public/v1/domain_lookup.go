package publicapiv1

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// domainInEnv resolves a domain and asserts it belongs to the request's
// environment. Scoping to the environment rather than the app matters because an
// env-scoped API key would otherwise reach every domain of every sibling
// environment.
//
// A domain that does not exist and one that belongs elsewhere both answer 404,
// so the endpoint cannot be used to enumerate domain ids. Exactly one of the two
// return values is non-nil.
func (req *RequestContext) domainInEnv(domainID types.ID) (*buildconf.DomainModel, *shttp.Response) {
	domain, err := buildconf.DomainStore().DomainByID(req.Context(), domainID)

	if err != nil {
		return nil, shttp.Error(err)
	}

	if domain == nil || domain.EnvID != req.Env.ID {
		return nil, shttp.NotFound()
	}

	return domain, nil
}
