package analytics

import (
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

// Event is a custom analytics event (e.g. a product creation) emitted from the
// client tracking script or a server-side integration. RequestID links the
// event back to the originating server-side analytics hit when available.
type Event struct {
	AppID       types.ID    `json:"appId"`
	EnvID       types.ID    `json:"envId"`
	DomainID    types.ID    `json:"domainId"`
	VisitorID   null.String `json:"visitorId"`
	EventName   string      `json:"eventName"`
	RequestPath null.String `json:"requestPath"`
	RequestID   null.String `json:"requestId"`
	EventTS     utils.Unix  `json:"eventTs"`
	Metadata    null.String `json:"metadata"`
}
