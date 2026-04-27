package subscriptionhandlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/mailer"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
)

var productIDToPackage = map[string]string{
	// test
	"prod_THDhiOfzmRa6xD_test": config.PackagePremium,
	"prod_Rw6no7lokoLIVD_test": config.PackageUltimate,

	// prod
	"prod_THDhiOfzmRa6xD": config.PackagePremium,
	"prod_Rw6no7lokoLIVD": config.PackageUltimate,
}

// handlerSubscriptionUpdate updates the subscription of the user to the given one.
// It also makes sure that a stripe client exists.
func handlerSubscriptionUpdate(req *shttp.RequestContext) *shttp.Response {
	const MaxBodyBytes = int64(65536)
	req.Body = http.MaxBytesReader(req.Writer(), req.Body, MaxBodyBytes)
	payload, err := io.ReadAll(req.Body)

	if err != nil {
		slog.Errorf("error while reading stripe body: %s", err.Error())
		return shttp.Error(err)
	}

	// Pass the request body and Stripe-Signature header to ConstructEvent, along
	// with the webhook signing key.
	event, err := webhook.ConstructEvent(
		payload,
		req.Header.Get("Stripe-Signature"),
		config.Get().Stripe.WebhooksSecret,
	)

	if err != nil {
		slog.Errorf("error while constructing stripe event: %s", err.Error())
		return shttp.Error(err)
	}

	// Testing this locally:
	// $ gem install ultrahook
	// $ export GEM_HOME="$HOME/.gem"
	// $ cd ~/.gem/ruby/2.6.0/bin
	// $ ./ultrahook stripe 8080

	switch event.Type {

	case
		"customer.subscription.created",
		"customer.subscription.updated":
		var subscription stripe.Subscription

		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			slog.Errorf("error while unmarshaling stripe event: %s", err.Error())
			return shttp.Error(err)
		}

		return UpdateSubscription(req, subscription)

	case "customer.subscription.deleted":
		var subscription stripe.Subscription

		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			slog.Errorf("error while unmarshaling stripe event: %s", err.Error())
			return shttp.Error(err)
		}

		return CancelSubscription(req, subscription)

	default:
		return shttp.NoContent()
	}
}

// UpdateSubscription updates the subscription of the user based on the provided subscription info.
// For cloud users it updates their account metadata. For self-hosted customers (no cloud account)
// it generates or updates a license and emails it on first creation.
func UpdateSubscription(req *shttp.RequestContext, subscription stripe.Subscription) *shttp.Response {
	client := stripeClient()
	customer, err := client.Customers(subscription.Customer.ID, nil)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("error while retrieving customer: %v", err))
	}

	if customer == nil {
		if subscription.Customer == nil {
			slog.Errorf("error while looking for stripe customer: nil customer in subscription")
		} else {
			slog.Errorf("error while looking for stripe customer: %s", subscription.Customer.ID)
		}

		return shttp.NotFound()
	}

	store := user.NewStore()
	usr, err := store.UserByEmail(req.Context(), []string{customer.Email})

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("error while retrieving user from stripe email: %s, err: %v", customer.Email, err))
	}

	if usr != nil {
		packageName := config.PackageFree
		quantity := int64(0)

		if subscription.Items != nil {
			for _, sub := range subscription.Items.Data {
				if pck := productIDToPackage[sub.Plan.Product.ID]; pck != "" {
					packageName = pck
					quantity = sub.Quantity
				}
			}
		}

		meta := usr.Metadata
		meta.PackageName = packageName
		meta.StripeCustomerID = customer.ID
		meta.SeatsPurchased = int(quantity)

		if err := store.UpdateSubscription(req.Context(), usr.ID, meta); err != nil {
			return shttp.Error(err, fmt.Sprintf("error while updating subscription: %v", err))
		}

		return shttp.OK()
	}

	// No cloud account — self-hosted customer.
	return SendSelfHostedLicense(req, customer, subscription)
}

// CancelSubscription downgrades a cloud user to the free tier and removes any
// self-hosted license associated with the customer, if they exist.
func CancelSubscription(req *shttp.RequestContext, subscription stripe.Subscription) *shttp.Response {
	client := stripeClient()
	customer, err := client.Customers(subscription.Customer.ID, nil)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("error while retrieving customer: %v", err))
	}

	if customer == nil {
		slog.Errorf("error while looking for stripe customer: %s", subscription.Customer.ID)
		return shttp.NotFound()
	}

	store := user.NewStore()
	usr, err := store.UserByEmail(req.Context(), []string{customer.Email})

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("error while retrieving user from stripe email: %s, err: %v", customer.Email, err))
	}

	if usr != nil {
		meta := usr.Metadata
		meta.PackageName = config.PackageFree
		meta.SeatsPurchased = 0

		if err := store.UpdateSubscription(req.Context(), usr.ID, meta); err != nil {
			return shttp.Error(err, fmt.Sprintf("error while downgrading subscription: %v", err))
		}
	}

	if err := store.DeleteSelfHostedLicense(req.Context(), customer.Email); err != nil {
		return shttp.Error(err, fmt.Sprintf("error while deleting self-hosted license for %s: %v", customer.Email, err))
	}

	return shttp.OK()
}

const mailTemplateLicense = `
<p>Thank you for subscribing to Stormkit!</p>
<p>
	Your license key is ready. Copy it and paste it into your self-hosted
	instance under <strong>Admin &rsaquo; License</strong>.
</p>
<p><strong>%s</strong></p>
<p>
	If you have any questions, feel free to reach out to us at
	<a href="mailto:hello@stormkit.io">hello@stormkit.io</a>.
</p>
<p>The Stormkit Team</p>`

// SendSelfHostedLicense generates or updates a license for a self-hosted customer and
// emails the key on first creation. On subscription deletion it is a no-op.
func SendSelfHostedLicense(req *shttp.RequestContext, customer *stripe.Customer, subscription stripe.Subscription) *shttp.Response {
	packageName, quantity := selfHostedPlan(subscription)
	store := user.NewStore()

	existing, err := store.License(req.Context(), user.LicenseParams{Email: customer.Email})

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("error while looking up license for %s: %v", customer.Email, err))
	}

	if existing != nil {
		if err := store.UpdateSelfHostedLicense(req.Context(), customer.Email, int(quantity), packageName); err != nil {
			return shttp.Error(err, fmt.Sprintf("error while updating license for %s: %v", customer.Email, err))
		}

		return shttp.OK()
	}

	meta := map[string]any{"email": customer.Email}

	if customer.ID != "" {
		meta["stripeCustomerId"] = customer.ID
	}

	license, err := store.GenerateSelfHostedLicense(req.Context(), int(quantity), packageName, meta)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("error while generating license: %v", err))
	}

	if err := mailer.SendEmail(mailer.SendEmailParams{
		To:      customer.Email,
		Subject: "Your Stormkit License Key",
		Body:    fmt.Sprintf(mailTemplateLicense, license.Key),
	}); err != nil {
		// Roll back the license row so the next Stripe retry can recreate it
		// and attempt the email again.
		if deleteErr := store.DeleteSelfHostedLicense(req.Context(), customer.Email); deleteErr != nil {
			slog.Errorf("error while rolling back license for %s after email failure: %v", customer.Email, deleteErr)
		}

		return shttp.Error(err, fmt.Sprintf("error while sending license email to %s: %v", customer.Email, err))
	}

	return shttp.OK()
}

// selfHostedPlan resolves the package name and seat count from a subscription's items.
func selfHostedPlan(subscription stripe.Subscription) (packageName string, quantity int64) {
	packageName = config.PackageFree

	if subscription.Items == nil {
		return
	}

	for _, sub := range subscription.Items.Data {
		if pck := productIDToPackage[sub.Plan.Product.ID]; pck != "" {
			packageName = pck
			quantity = sub.Quantity
		}
	}

	return
}
