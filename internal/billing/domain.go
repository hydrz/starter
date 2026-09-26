// Package billing implements ADR-0006 (Stripe payments, subscriptions, and
// entitlement projections). It is the ONLY package permitted to import the
// Stripe Go SDK: every other domain package that needs to know whether an
// organization has a feature enabled depends only on the narrow
// EntitlementReader interface below, never on Stripe or store/ogen types.
//
// Stripe objects, webhooks, and price identifiers never become the
// authorization model (see internal/authorization): entitlements are only an
// input to a business capability check, derived exclusively from
// webhook-verified state.
package billing

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors returned by Service. Handlers translate these into
// transport responses; they never leak persistence or Stripe SDK detail.
var (
	ErrNotFound          = errors.New("billing: not found")
	ErrUnknownPriceKey   = errors.New("billing: unknown price key")
	ErrInvalidInput      = errors.New("billing: invalid input")
	ErrInvalidSignature  = errors.New("billing: invalid webhook signature")
	ErrOrganizationInUse = errors.New("billing: organization already has a billing account")
)

// BillingAccount is the one-per-organization link to a Stripe customer. It
// never stores card/payment data, only the Stripe customer ID.
type BillingAccount struct {
	ID               string
	OrganizationID   string
	StripeCustomerID string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CheckoutSession is a server-initiated Stripe Checkout session for a
// catalog price. Its status is only ever advanced by verified webhook facts.
type CheckoutSession struct {
	ID                      string
	BillingAccountID        string
	StripeCheckoutSessionID string
	PriceKey                string
	Mode                    string
	Status                  string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// Subscription is a projection of Stripe subscription state, derived only
// from verified webhook events. LastEventCreatedAt is the monotonic guard
// against out-of-order webhook delivery.
type Subscription struct {
	ID                   string
	BillingAccountID     string
	StripeSubscriptionID string
	PriceKey             string
	Status               string
	CurrentPeriodEnd     *time.Time
	CancelAtPeriodEnd    bool
	LastEventCreatedAt   *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// OneTimePurchase is a completed one-time payment, derived only from a
// verified webhook event. Amounts are integer minor units with an ISO
// currency code, never a float.
type OneTimePurchase struct {
	ID                      string
	BillingAccountID        string
	StripeCheckoutSessionID string
	PriceKey                string
	AmountMinorUnits        int64
	Currency                string
	Status                  string
	CreatedAt               time.Time
}

const (
	EntitlementSourceSubscription = "subscription"
	EntitlementSourceOneTime      = "one_time"
)

// Entitlement expresses whether one feature is granted to an organization by
// one independent source. A feature can be granted by more than one source
// at once (e.g. an active subscription AND a separate one-time purchase);
// cancelling the subscription only disables its own row, never the one-time
// purchase's row for the same feature.
type Entitlement struct {
	ID                string
	OrganizationID    string
	FeatureKey        string
	Source            string
	Enabled           bool
	ExpiresAt         *time.Time
	SubscriptionID    *string
	OneTimePurchaseID *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Summary is the read-facing projection returned to callers: the
// organization's current subscription (if any) and its resolved
// entitlements.
type Summary struct {
	OrganizationID        string
	HasActiveSubscription bool
	Subscription          *Subscription
	Entitlements          []Entitlement
}

// CreatedCheckoutSession pairs the persisted checkout session with the
// Stripe-hosted URL the client should redirect to.
type CreatedCheckoutSession struct {
	CheckoutSession
	URL string
}

// CreatedPortalSession is the Stripe-hosted Customer Portal URL.
type CreatedPortalSession struct {
	URL string
}

// EntitlementReader is the narrow boundary other domain packages (future
// authorization/product checks) depend on. It never exposes Stripe or
// store/ogen types.
type EntitlementReader interface {
	HasEntitlement(ctx context.Context, organizationID, featureKey string) (bool, error)
}
