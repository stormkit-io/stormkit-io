package subscriptionhandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user/subscriptionhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerSelfHostedCheckoutSuite struct {
	suite.Suite
}

func (f *HandlerSelfHostedCheckoutSuite) AfterTest(_, _ string) {
	config.Get().Stripe = nil
}

func (s *HandlerSelfHostedCheckoutSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(subscriptionhandlers.Services).Router().Handler()
}

func (s *HandlerSelfHostedCheckoutSuite) Test_RedirectsToPortal() {
	config.Get().Stripe = &config.StripeConfig{
		CustomerPortalLink: "https://billing.stripe.com/p/login/test",
	}

	response := shttptest.Request(s.handler(), shttp.MethodGet, "/billing/checkout?plan=portal", nil)

	s.Equal(http.StatusFound, response.Code)
	s.Equal("https://billing.stripe.com/p/login/test", response.Header().Get("Location"))
}

func (s *HandlerSelfHostedCheckoutSuite) Test_RedirectsToPremium() {
	config.Get().Stripe = &config.StripeConfig{
		PaymentLinkPremium: "https://buy.stripe.com/premium_test",
	}

	response := shttptest.Request(s.handler(), shttp.MethodGet, "/billing/checkout?plan=premium", nil)

	s.Equal(http.StatusFound, response.Code)
	s.Equal("https://buy.stripe.com/premium_test", response.Header().Get("Location"))
}

func (s *HandlerSelfHostedCheckoutSuite) Test_RedirectsToPremiumWithRef() {
	config.Get().Stripe = &config.StripeConfig{
		PaymentLinkPremium: "https://buy.stripe.com/premium_test",
	}

	response := shttptest.Request(s.handler(), shttp.MethodGet, "/billing/checkout?plan=premium&ref=user%40example.com", nil)

	s.Equal(http.StatusFound, response.Code)
	s.Equal("https://buy.stripe.com/premium_test?client_reference_id=selfhosted%3Auser%40example.com", response.Header().Get("Location"))
}

func (s *HandlerSelfHostedCheckoutSuite) Test_RedirectsToUltimate() {
	config.Get().Stripe = &config.StripeConfig{
		PaymentLinkUltimate: "https://buy.stripe.com/ultimate_test",
	}

	response := shttptest.Request(s.handler(), shttp.MethodGet, "/billing/checkout?plan=ultimate", nil)

	s.Equal(http.StatusFound, response.Code)
	s.Equal("https://buy.stripe.com/ultimate_test", response.Header().Get("Location"))
}

func (s *HandlerSelfHostedCheckoutSuite) Test_RedirectsToUltimateWithRef() {
	config.Get().Stripe = &config.StripeConfig{
		PaymentLinkUltimate: "https://buy.stripe.com/ultimate_test",
	}

	response := shttptest.Request(s.handler(), shttp.MethodGet, "/billing/checkout?plan=ultimate&ref=user%40example.com", nil)

	s.Equal(http.StatusFound, response.Code)
	s.Equal("https://buy.stripe.com/ultimate_test?client_reference_id=selfhosted%3Auser%40example.com", response.Header().Get("Location"))
}

func (s *HandlerSelfHostedCheckoutSuite) Test_BadRequestOnInvalidPlan() {
	config.Get().Stripe = &config.StripeConfig{}
	response := shttptest.Request(s.handler(), shttp.MethodGet, "/billing/checkout?plan=unknown", nil)
	s.Equal(http.StatusBadRequest, response.Code)
}

func (s *HandlerSelfHostedCheckoutSuite) Test_ServiceUnavailableWhenStripeNotConfigured() {
	config.Get().Stripe = nil
	response := shttptest.Request(s.handler(), shttp.MethodGet, "/billing/checkout?plan=premium", nil)
	s.Equal(http.StatusServiceUnavailable, response.Code)
}

func TestHandlerSelfHostedCheckoutSuite(t *testing.T) {
	suite.Run(t, new(HandlerSelfHostedCheckoutSuite))
}
