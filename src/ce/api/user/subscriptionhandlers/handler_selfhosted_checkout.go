package subscriptionhandlers

import (
	"net/http"
	"net/url"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

const (
	planPortal        = "portal"
	planPremium       = "premium"
	planUltimate      = "ultimate"
	editionSelfHosted = "self-hosted"
)

// HandleSelfHostedCheckout redirects users to the appropriate Stripe page.
// The edition parameter determines whether to use the self-hosted or cloud payment link.
func HandleSelfHostedCheckout(req *shttp.RequestContext) *shttp.Response {
	plan := req.Request.URL.Query().Get("plan")
	ref := req.Request.URL.Query().Get("ref")
	edition := req.Request.URL.Query().Get("edition")
	stripe := config.Get().Stripe

	if stripe == nil {
		return &shttp.Response{Status: http.StatusServiceUnavailable}
	}

	var redirectURL string

	switch plan {
	case planPortal:
		redirectURL = stripe.CustomerPortalLink
	case planPremium:
		if edition == editionSelfHosted {
			redirectURL = withStripeParams(stripe.PaymentLinkPremiumSH, ref)
		} else {
			redirectURL = withStripeParams(stripe.PaymentLinkPremiumCloud, ref)
		}
	case planUltimate:
		redirectURL = withStripeParams(stripe.PaymentLinkUltimateSH, ref)
	}

	if redirectURL == "" {
		return shttp.BadRequest()
	}

	req.Redirect(redirectURL, http.StatusFound)

	return nil
}

func withStripeParams(baseURL, ref string) string {
	if ref == "" {
		return baseURL
	}

	u, err := url.Parse(baseURL)

	if err != nil {
		return baseURL
	}

	q := u.Query()
	q.Set("client_reference_id", ref)
	q.Set("prefilled_email", ref)
	u.RawQuery = q.Encode()

	return u.String()
}
