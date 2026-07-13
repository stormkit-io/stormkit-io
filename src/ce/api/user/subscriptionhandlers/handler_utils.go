package subscriptionhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/client"
)

type StripeClient interface {
	Customers(string, *stripe.CustomerParams) (*stripe.Customer, error)
	Subscriptions(string, *stripe.SubscriptionParams) (*stripe.Subscription, error)
}

type Stripe struct {
	client *client.API
}

var CachedClient StripeClient

func stripeClient() StripeClient {
	if CachedClient == nil {
		sc := &client.API{}
		sc.Init(config.Get().Stripe.ClientSecret, nil)
		CachedClient = &Stripe{
			client: sc,
		}
	}

	return CachedClient
}

func (s *Stripe) Customers(customerID string, params *stripe.CustomerParams) (*stripe.Customer, error) {
	return s.client.Customers.Get(customerID, params)
}

func (s *Stripe) Subscriptions(subscriptionID string, params *stripe.SubscriptionParams) (*stripe.Subscription, error) {
	return s.client.Subscriptions.Get(subscriptionID, params)
}
