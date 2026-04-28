package subscriptionhandlers

import (
	"fmt"
	"net/http"
	"net/url"

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
	ref := req.Request.URL.Query().Get("ref")
	stripe := config.Get().Stripe

	if stripe == nil {
		return &shttp.Response{Status: http.StatusServiceUnavailable}
	}

	var redirectURL string

	switch plan {
	case planPortal:
		redirectURL = stripe.CustomerPortalLink
	case planPremium:
		redirectURL = withClientReferenceID(stripe.PaymentLinkPremium, ref)
	case planUltimate:
		redirectURL = withClientReferenceID(stripe.PaymentLinkUltimate, ref)
	}

	if redirectURL == "" {
		return shttp.BadRequest()
	}

	req.Redirect(redirectURL, http.StatusFound)

	return nil
}

func withClientReferenceID(baseURL, ref string) string {
	if ref == "" {
		return baseURL
	}

	u, err := url.Parse(baseURL)

	if err != nil {
		return baseURL
	}

	q := u.Query()
	q.Set("client_reference_id", fmt.Sprintf("selfhosted:%s", ref))
	q.Set("prefilled_email", ref)
	u.RawQuery = q.Encode()

	return u.String()
}
