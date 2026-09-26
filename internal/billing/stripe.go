package billing

import (
	"context"
	"errors"
	"fmt"

	stripesdk "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/client"
)

// LiveGateway implements StripeGateway using the real Stripe API. It is the
// only production implementation of StripeGateway and the only place in
// this package (besides webhook.go's event verification/parsing) that calls
// out to Stripe.
type LiveGateway struct {
	client *client.API
}

var _ StripeGateway = (*LiveGateway)(nil)

// NewLiveGateway builds a StripeGateway backed by the given secret key.
func NewLiveGateway(secretKey string) (*LiveGateway, error) {
	if secretKey == "" {
		return nil, errors.New("billing: stripe secret key is required")
	}
	return &LiveGateway{client: client.New(secretKey, nil)}, nil
}

// CreateCustomer creates a Stripe Customer for a new billing account. The
// organization ID is stored only as opaque Stripe metadata (never as a
// business record on the Stripe side, and Stripe secrets/IDs are never
// logged).
func (g *LiveGateway) CreateCustomer(_ context.Context, organizationID string) (string, error) {
	params := &stripesdk.CustomerParams{
		Metadata: map[string]string{"organization_id": organizationID},
	}
	created, err := g.client.Customers.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe: create customer: %w", err)
	}
	return created.ID, nil
}

// CreateCheckoutSession creates a Stripe Checkout Session for a single
// catalog price. Success/cancel URLs and the customer/price are entirely
// server-controlled (params); nothing here is client-supplied.
func (g *LiveGateway) CreateCheckoutSession(_ context.Context, checkoutParams CheckoutParams) (string, string, error) {
	quantity := int64(1)
	params := &stripesdk.CheckoutSessionParams{
		Customer:   stripesdk.String(checkoutParams.CustomerID),
		Mode:       stripesdk.String(checkoutParams.Mode),
		SuccessURL: stripesdk.String(checkoutParams.SuccessURL),
		CancelURL:  stripesdk.String(checkoutParams.CancelURL),
		LineItems: []*stripesdk.CheckoutSessionLineItemParams{
			{Price: stripesdk.String(checkoutParams.PriceID), Quantity: &quantity},
		},
		Metadata: map[string]string{"organization_id": checkoutParams.OrganizationID},
	}
	created, err := g.client.CheckoutSessions.New(params)
	if err != nil {
		return "", "", fmt.Errorf("stripe: create checkout session: %w", err)
	}
	return created.ID, created.URL, nil
}

// CreatePortalSession creates a Stripe Customer Portal session. returnURL is
// server-configured, never client-supplied.
func (g *LiveGateway) CreatePortalSession(_ context.Context, stripeCustomerID, returnURL string) (string, error) {
	params := &stripesdk.BillingPortalSessionParams{
		Customer:  stripesdk.String(stripeCustomerID),
		ReturnURL: stripesdk.String(returnURL),
	}
	created, err := g.client.BillingPortalSessions.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe: create portal session: %w", err)
	}
	return created.URL, nil
}
