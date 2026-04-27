package subscriptionhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

const (
	planPortal   = "portal"
	planPremium  = "premium"
	planUltimate = "ultimate"
)

// HandleSelfHostedCheckout redirects self-hosted users to the appropriate Stripe page.
func HandleSelfHostedCheckout(req *shttp.RequestContext) *shttp.Response {
	plan := req.Request.URL.Query().Get("plan")
	stripe := config.Get().Stripe

	if stripe == nil {
		return &shttp.Response{Status: http.StatusServiceUnavailable}
	}

	var redirectURL string

	switch plan {
	case planPortal:
		redirectURL = stripe.CustomerPortalLink
	case planPremium:
		redirectURL = stripe.PaymentLinkPremium
	case planUltimate:
		redirectURL = stripe.PaymentLinkUltimate
	}

	if redirectURL == "" {
		return shttp.BadRequest()
	}

	req.Redirect(redirectURL, http.StatusFound)

	return nil
}
