package analyticshandlers

import (
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerTopReferrers(req *app.RequestContext) *shttp.Response {
	query := req.Query()

	domainID, errResp := authorizedDomainID(req)

	if errResp != nil {
		return errResp
	}

	visitors, err := analytics.NewStore().TopReferrers(req.Context(), analytics.TopReferrersArgs{
		EnvID:       req.EnvID,
		RequestPath: strings.ToLower(strings.TrimSpace(query.Get("requestPath"))),
		DomainID:    domainID,
	})

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   visitors,
	}
}
