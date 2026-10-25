package analyticshandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerVisitors(req *app.RequestContext) *shttp.Response {
	span := req.Query().Get("ts")

	if span == "" {
		span = analytics.SPAN_24h
	}

	if !analytics.IsVisitorsSpan(span) {
		return shttp.BadRequest(map[string]any{
			"error": "ts must be one of: 24h, 7d, 30d",
		})
	}

	domainID, errResp := authorizedDomainID(req)

	if errResp != nil {
		return errResp
	}

	visitors, err := analytics.NewStore().Visitors(req.Context(), analytics.VisitorsArgs{
		Span:       span,
		EnvID:      req.EnvID,
		DomainID:   domainID,
		StatusCode: http.StatusOK,
	})

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   visitors,
	}
}
