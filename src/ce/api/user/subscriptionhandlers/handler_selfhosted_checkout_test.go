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

func (s *HandlerSelfHostedCheckoutSuite) Test_RedirectsToCloudPremium() {
	config.Get().Stripe = &config.StripeConfig{
		PaymentLinkPremiumCloud: "https://buy.stripe.com/cloud_premium_test",
	}

	response := shttptest.Request(s.handler(), shttp.MethodGet, "/billing/checkout?plan=premium", nil)

	s.Equal(http.StatusFound, response.Code)
	s.Equal("https://buy.stripe.com/cloud_premium_test", response.Header().Get("Location"))
}

func (s *HandlerSelfHostedCheckoutSuite) Test_RedirectsToCloudPremiumWithRef() {
	config.Get().Stripe = &config.StripeConfig{
		PaymentLinkPremiumCloud: "https://buy.stripe.com/cloud_premium_test",
	}

	response := shttptest.Request(s.handler(), shttp.MethodGet, "/billing/checkout?plan=premium&ref=user%40example.com", nil)

	s.Equal(http.StatusFound, response.Code)
	s.Equal("https://buy.stripe.com/cloud_premium_test?client_reference_id=user%40example.com&prefilled_email=user%40example.com", response.Header().Get("Location"))
}

func (s *HandlerSelfHostedCheckoutSuite) Test_RedirectsToSelfHostedPremium() {
	config.Get().Stripe = &config.StripeConfig{
		PaymentLinkPremiumSH: "https://buy.stripe.com/sh_premium_test",
	}

	response := shttptest.Request(s.handler(), shttp.MethodGet, "/billing/checkout?plan=premium&edition=self-hosted", nil)

	s.Equal(http.StatusFound, response.Code)
	s.Equal("https://buy.stripe.com/sh_premium_test", response.Header().Get("Location"))
}

func (s *HandlerSelfHostedCheckoutSuite) Test_RedirectsToSelfHostedPremiumWithRef() {
	config.Get().Stripe = &config.StripeConfig{
		PaymentLinkPremiumSH: "https://buy.stripe.com/sh_premium_test",
	}

	response := shttptest.Request(s.handler(), shttp.MethodGet, "/billing/checkout?plan=premium&edition=self-hosted&ref=user%40example.com", nil)

	s.Equal(http.StatusFound, response.Code)
	s.Equal("https://buy.stripe.com/sh_premium_test?client_reference_id=user%40example.com&prefilled_email=user%40example.com", response.Header().Get("Location"))
}

func (s *HandlerSelfHostedCheckoutSuite) Test_RedirectsToUltimate() {
	config.Get().Stripe = &config.StripeConfig{
		PaymentLinkUltimateSH: "https://buy.stripe.com/ultimate_test",
	}

	response := shttptest.Request(s.handler(), shttp.MethodGet, "/billing/checkout?plan=ultimate", nil)

	s.Equal(http.StatusFound, response.Code)
	s.Equal("https://buy.stripe.com/ultimate_test", response.Header().Get("Location"))
}

func (s *HandlerSelfHostedCheckoutSuite) Test_RedirectsToUltimateWithRef() {
	config.Get().Stripe = &config.StripeConfig{
		PaymentLinkUltimateSH: "https://buy.stripe.com/ultimate_test",
	}

	response := shttptest.Request(s.handler(), shttp.MethodGet, "/billing/checkout?plan=ultimate&ref=user%40example.com", nil)

	s.Equal(http.StatusFound, response.Code)
	s.Equal("https://buy.stripe.com/ultimate_test?client_reference_id=user%40example.com&prefilled_email=user%40example.com", response.Header().Get("Location"))
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
