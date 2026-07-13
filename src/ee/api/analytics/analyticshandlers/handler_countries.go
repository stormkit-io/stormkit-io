package analyticshandlers

import (
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerCountries(req *app.RequestContext) *shttp.Response {
	domainID, errResp := authorizedDomainID(req)

	if errResp != nil {
		return errResp
	}

	countries, err := analytics.NewStore().ByCountries(req.Context(), analytics.ByCountriesArgs{
		DomainID: domainID,
	})

	if err != nil {
		fmt.Println(err.Error())
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   countries,
	}
}
