package subscriptionhandlers_test

import (
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/subscriptionhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/mailer"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
	"github.com/stripe/stripe-go/v81"
)

type HandlerSelfHostedLicenseSuite struct {
	suite.Suite
	*factory.Factory

	conn       databasetest.TestDB
	mockClient mocks.StripeClient
	req        *shttp.RequestContext
}

func (s *HandlerSelfHostedLicenseSuite) SetupSuite() {
	s.mockClient = mocks.StripeClient{}
}

func (s *HandlerSelfHostedLicenseSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	s.req = &shttp.RequestContext{
		Request: &http.Request{},
	}

	config.Get().Stripe = &config.StripeConfig{
		ClientSecret: "sk_test_placeholder",
	}

	subscriptionhandlers.CachedClient = &s.mockClient
}

func (s *HandlerSelfHostedLicenseSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	subscriptionhandlers.CachedClient = nil
	config.Get().Stripe = nil
	config.Get().SMTP = nil
	mailer.SendMailFunc = smtp.SendMail
}

func (s *HandlerSelfHostedLicenseSuite) Test_GeneratesAndEmailsLicense() {
	var capturedTo, capturedBody string

	mailer.SendMailFunc = func(_ string, _ smtp.Auth, _ string, to []string, msg []byte) error {
		capturedTo = to[0]
		capturedBody = string(msg)
		return nil
	}

	config.Get().SMTP = &config.SMTPConfig{
		Host:     "smtp.example.com",
		Username: "user",
		Password: "pass",
	}

	customer := &stripe.Customer{
		ID:    "cus_abc123",
		Email: "owner@example.com",
	}

	subscription := stripe.Subscription{
		Status: stripe.SubscriptionStatusActive,
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{
					Plan: &stripe.Plan{
						Product: &stripe.Product{
							ID: "prod_THDhiOfzmRa6xD",
						},
					},
					Quantity: 10,
				},
			},
		},
	}

	res := subscriptionhandlers.SendSelfHostedLicense(s.req, customer, subscription)

	s.Equal(http.StatusOK, res.Status)
	s.Equal("owner@example.com", capturedTo)
	s.True(strings.Contains(capturedBody, "Thank you for subscribing"))
	s.True(strings.Contains(capturedBody, "Admin"))
	s.True(strings.Contains(capturedBody, "License"))
}

func (s *HandlerSelfHostedLicenseSuite) Test_RollsBackLicenseOnEmailFailure() {
	mailer.SendMailFunc = func(_ string, _ smtp.Auth, _ string, _ []string, _ []byte) error {
		return fmt.Errorf("smtp unavailable")
	}

	config.Get().SMTP = &config.SMTPConfig{
		Host:     "smtp.example.com",
		Username: "user",
		Password: "pass",
	}

	customer := &stripe.Customer{
		ID:    "cus_rollback",
		Email: "rollback@example.com",
	}

	subscription := stripe.Subscription{
		Status: stripe.SubscriptionStatusActive,
	}

	res := subscriptionhandlers.SendSelfHostedLicense(s.req, customer, subscription)

	s.Equal(http.StatusInternalServerError, res.Status)

	// License row must have been deleted so the next Stripe retry can recreate it.
	store := user.NewStore()
	license, err := store.License(s.req.Context(), user.LicenseParams{Email: customer.Email})
	s.NoError(err)
	s.Nil(license)
}

func (s *HandlerSelfHostedLicenseSuite) Test_GeneratesLicenseWithoutItems() {
	mailer.SendMailFunc = func(_ string, _ smtp.Auth, _ string, _ []string, _ []byte) error {
		return nil
	}

	config.Get().SMTP = &config.SMTPConfig{
		Host:     "smtp.example.com",
		Username: "user",
		Password: "pass",
	}

	customer := &stripe.Customer{
		ID:    "cus_abc123",
		Email: "noitems@example.com",
	}

	subscription := stripe.Subscription{
		Status: stripe.SubscriptionStatusActive,
	}

	res := subscriptionhandlers.SendSelfHostedLicense(s.req, customer, subscription)

	s.Equal(http.StatusOK, res.Status)
}

func TestHandlerSelfHostedLicenseSuite(t *testing.T) {
	suite.Run(t, &HandlerSelfHostedLicenseSuite{})
}
