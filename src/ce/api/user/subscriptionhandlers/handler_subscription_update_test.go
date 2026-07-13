package subscriptionhandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/subscriptionhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
	"github.com/stripe/stripe-go/v81"
)

type HandlerUpdateSubscriptionSuite struct {
	suite.Suite
	*factory.Factory

	conn       databasetest.TestDB
	mockClient mocks.StripeClient
	req        *shttp.RequestContext
	user       *factory.MockUser
	customer   *stripe.Customer
}

func (s *HandlerUpdateSubscriptionSuite) BeforeTest(suiteName, _ string) {
	s.mockClient = mocks.StripeClient{}
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.user = s.MockUser(map[string]any{
		"Metadata": user.UserMeta{
			PackageName: config.PackageFree,
		},
	})

	s.req = &shttp.RequestContext{
		Request: &http.Request{},
	}

	s.customer = &stripe.Customer{
		ID:    "cus_test123",
		Email: s.user.PrimaryEmail(),
	}

	subscriptionhandlers.CachedClient = &s.mockClient
}

func (s *HandlerUpdateSubscriptionSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	subscriptionhandlers.CachedClient = nil
}

func (s *HandlerUpdateSubscriptionSuite) Test_UpdateSubscription_Success() {
	subscription := stripe.Subscription{
		ID:       "sub_test123",
		Customer: s.customer,
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{
					Plan: &stripe.Plan{
						Product: &stripe.Product{
							ID: "prod_UQ3S5tzO70RHPK", // Cloud premium package
						},
					},
					Quantity: 5,
				},
			},
		},
	}

	s.mockClient.On("Customers", s.customer.ID, (*stripe.CustomerParams)(nil)).Return(s.customer, nil)
	s.Equal(http.StatusOK, subscriptionhandlers.UpdateSubscription(s.req, subscription).Status)

	// Verify the user's subscription was updated
	store := user.NewStore()
	updatedUser, err := store.UserByID(s.user.ID)
	s.NoError(err)
	s.Equal(config.PackagePremium, updatedUser.Metadata.PackageName)
}

func (s *HandlerUpdateSubscriptionSuite) Test_CancelSubscription_DowngradesCloudUser() {
	subscription := stripe.Subscription{
		ID:       "sub_test123",
		Status:   stripe.SubscriptionStatusCanceled,
		Customer: s.customer,
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{
					Plan: &stripe.Plan{
						Product: &stripe.Product{
							ID: "prod_UQ3S5tzO70RHPK", // Cloud premium package
						},
					},
				},
			},
		},
	}

	s.mockClient.On("Customers", s.customer.ID, (*stripe.CustomerParams)(nil)).Return(s.customer, nil)

	response := subscriptionhandlers.CancelSubscription(s.req, subscription)

	s.Equal(http.StatusOK, response.Status)

	store := user.NewStore()
	updatedUser, err := store.UserByID(s.user.ID)
	s.NoError(err)
	s.Equal(config.PackageFree, updatedUser.Metadata.PackageName)
}

func (s *HandlerUpdateSubscriptionSuite) Test_CancelSubscription_NoCloudUser() {
	customer := &stripe.Customer{
		ID:    "cus_selfhosted123",
		Email: "selfhosted@example.com",
	}

	subscription := stripe.Subscription{
		ID:       "sub_test123",
		Status:   stripe.SubscriptionStatusCanceled,
		Customer: customer,
	}

	s.mockClient.On("Customers", customer.ID, (*stripe.CustomerParams)(nil)).Return(customer, nil)

	// No license in DB for this email — DeleteSelfHostedLicense should be a no-op.
	response := subscriptionhandlers.CancelSubscription(s.req, subscription)

	s.Equal(http.StatusOK, response.Status)
}

func (s *HandlerUpdateSubscriptionSuite) Test_UpdateSubscription_CustomerNotFound() {
	customer := &stripe.Customer{
		ID:    "cus_test345",
		Email: "email_not_found@example.com",
	}

	subscription := stripe.Subscription{
		ID:       "sub_test123",
		Customer: customer,
	}

	// Setup mock client to return nil customer
	s.mockClient.On("Customers", customer.ID, (*stripe.CustomerParams)(nil)).Return(nil, nil)
	response := subscriptionhandlers.UpdateSubscription(s.req, subscription)

	// Assert error
	s.Equal(http.StatusNotFound, response.Status)
}

func TestHandlerUpdateSubscriptionSuite(t *testing.T) {
	suite.Run(t, &HandlerUpdateSubscriptionSuite{})
}
