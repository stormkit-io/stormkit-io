package subscriptionhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services installs the user services.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/user/subscription").
		Handler(shttp.MethodPost, "/update", shttp.WithRateLimit(handlerSubscriptionUpdate))

	s.NewEndpoint("/billing").
		Handler(shttp.MethodGet, "/checkout", HandleSelfHostedCheckout)

	return s
}
