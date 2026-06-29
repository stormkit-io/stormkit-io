package analyticshandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerEvents(req *app.RequestContext) *shttp.Response {
	span := req.Query().Get("ts")

	if span == "" {
		span = analytics.SPAN_30D
	}

	domainID, errResp := authorizedDomainID(req)

	if errResp != nil {
		return errResp
	}

	events, err := analytics.NewStore().Events(req.Context(), analytics.EventsArgs{
		Span:     span,
		DomainID: domainID,
	})

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   events,
	}
}

func handlerEventProperties(req *app.RequestContext) *shttp.Response {
	span := req.Query().Get("ts")

	if span == "" {
		span = analytics.SPAN_30D
	}

	domainID, errResp := authorizedDomainID(req)

	if errResp != nil {
		return errResp
	}

	keys, err := analytics.NewStore().EventPropertyKeys(req.Context(), analytics.EventBreakdownArgs{
		DomainID:  domainID,
		EventName: req.Query().Get("event"),
		Span:      span,
	})

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   keys,
	}
}

func handlerEventBreakdown(req *app.RequestContext) *shttp.Response {
	span := req.Query().Get("ts")

	if span == "" {
		span = analytics.SPAN_30D
	}

	domainID, errResp := authorizedDomainID(req)

	if errResp != nil {
		return errResp
	}

	breakdown, err := analytics.NewStore().EventBreakdown(req.Context(), analytics.EventBreakdownArgs{
		DomainID:  domainID,
		EventName: req.Query().Get("event"),
		Property:  req.Query().Get("property"),
		Span:      span,
	})

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   breakdown,
	}
}
