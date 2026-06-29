package analyticshandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerTopPaths(req *app.RequestContext) *shttp.Response {
	domainID, errResp := authorizedDomainID(req)

	if errResp != nil {
		return errResp
	}

	visitors, err := analytics.NewStore().TopPaths(req.Context(), analytics.TopPathsArgs{
		EnvID:    req.EnvID,
		DomainID: domainID,
	})

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   visitors,
	}
}
