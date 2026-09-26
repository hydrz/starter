package billing

import (
	"context"
	"time"
)

// Clock abstracts wall-clock time so Service is deterministic under test.
type Clock interface {
	Now() time.Time
}

// SystemClock is the production Clock backed by time.Now.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

// Transactor runs fn within an atomic unit of work, matching
// internal/platform/database.Transactor so Service can compose multi-
// repository writes (recording a webhook event and updating projections)
// without depending on pgx directly.
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// OutboxWriter records a durable event in the same transaction as the
// domain write that produced it. Billing never calls a delivery channel
// directly (ADR-0005/0007); it only writes outbox rows for a future
// notification consumer.
type OutboxWriter interface {
	WriteEvent(ctx context.Context, topic, aggregateType, aggregateID string, payload []byte, idempotencyKey string) error
}

// BillingAccountRepository persists the one-per-organization link to a
// Stripe customer.
type BillingAccountRepository interface {
	// GetOrCreate returns the organization's billing account, creating one
	// with stripeCustomerID if none exists yet. It is idempotent: calling it
	// again for the same organization returns the existing row untouched.
	GetOrCreate(ctx context.Context, organizationID, stripeCustomerID string) (BillingAccount, error)
	GetByOrganization(ctx context.Context, organizationID string) (BillingAccount, error)
	GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (BillingAccount, error)
}

// CheckoutSessionRepository persists server-initiated Checkout sessions.
type CheckoutSessionRepository interface {
	Create(ctx context.Context, billingAccountID, stripeCheckoutSessionID, priceKey, mode string) (CheckoutSession, error)
	GetByStripeID(ctx context.Context, stripeCheckoutSessionID string) (CheckoutSession, bool, error)
	UpdateStatus(ctx context.Context, stripeCheckoutSessionID, status string) error
}

// UpsertSubscriptionInput carries the fields needed to project one verified
// subscription webhook event.
type UpsertSubscriptionInput struct {
	BillingAccountID     string
	StripeSubscriptionID string
	PriceKey             string
	Status               string
	CurrentPeriodEnd     *time.Time
	CancelAtPeriodEnd    bool
	EventCreatedAt       time.Time
}

// SubscriptionRepository persists the subscription projection.
type SubscriptionRepository interface {
	// Upsert applies input if and only if input.EventCreatedAt is at least as
	// new as the subscription's currently recorded event timestamp (the
	// monotonic out-of-order guard). applied is false when the guard
	// rejected a stale event; that is success, not an error.
	Upsert(ctx context.Context, input UpsertSubscriptionInput) (subscription Subscription, applied bool, err error)
	GetByStripeID(ctx context.Context, stripeSubscriptionID string) (Subscription, bool, error)
	// GetLatestForBillingAccount returns the most recently created
	// subscription for a billing account, for summary display.
	GetLatestForBillingAccount(ctx context.Context, billingAccountID string) (Subscription, bool, error)
}

// InsertOneTimePurchaseInput carries the fields needed to record one
// verified one-time payment webhook event.
type InsertOneTimePurchaseInput struct {
	BillingAccountID        string
	StripeCheckoutSessionID string
	PriceKey                string
	AmountMinorUnits        int64
	Currency                string
}

// OneTimePurchaseRepository persists completed one-time purchases.
type OneTimePurchaseRepository interface {
	// Insert is idempotent on StripeCheckoutSessionID: inserted is false
	// when the row already existed (re-delivered event), which is success,
	// not an error.
	Insert(ctx context.Context, input InsertOneTimePurchaseInput) (purchase OneTimePurchase, inserted bool, err error)
}

// WebhookEventRepository persists the Stripe event ledger used for
// idempotency. It never stores raw webhook payloads.
type WebhookEventRepository interface {
	// Insert is the single-statement, constraint-based idempotency check:
	// inserted is false when stripeEventID was already recorded (a
	// duplicate delivery), which is success, not an error.
	Insert(ctx context.Context, stripeEventID, eventType string, eventCreatedAt time.Time) (inserted bool, err error)
}

// UpsertEntitlementInput carries the fields needed to project one
// feature/source grant.
type UpsertEntitlementInput struct {
	OrganizationID    string
	FeatureKey        string
	Source            string
	Enabled           bool
	ExpiresAt         *time.Time
	SubscriptionID    *string
	OneTimePurchaseID *string
}

// EntitlementRepository persists and reads entitlement projections.
type EntitlementRepository interface {
	Upsert(ctx context.Context, input UpsertEntitlementInput) (Entitlement, error)
	HasEntitlement(ctx context.Context, organizationID, featureKey string) (bool, error)
	ListForOrganization(ctx context.Context, organizationID string) ([]Entitlement, error)
}

// CheckoutParams describes the server-controlled fields needed to create a
// Stripe Checkout session. The client never supplies a price, amount,
// customer/subscription ID, or success/cancel URL.
type CheckoutParams struct {
	CustomerID     string
	PriceID        string
	Mode           string // "payment" or "subscription"
	SuccessURL     string
	CancelURL      string
	OrganizationID string
}

// StripeGateway is the narrow boundary over outbound Stripe API calls. It is
// the only interface a production implementation may satisfy using the
// Stripe Go SDK; tests use an in-memory fake so Service is unit-testable
// without network access.
type StripeGateway interface {
	CreateCustomer(ctx context.Context, organizationID string) (stripeCustomerID string, err error)
	CreateCheckoutSession(ctx context.Context, params CheckoutParams) (stripeCheckoutSessionID, url string, err error)
	CreatePortalSession(ctx context.Context, stripeCustomerID, returnURL string) (url string, err error)
}
