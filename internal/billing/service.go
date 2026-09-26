package billing

import (
	"context"
	"errors"
	"fmt"
)

// Service implements the billing/entitlement use cases described by
// docs/adr/0006-stripe-payments-subscriptions-and-entitlements.md. It
// depends only on narrow repository ports, a StripeGateway, and a Catalog,
// so it can be unit tested without PostgreSQL or the Stripe network.
type Service struct {
	billingAccounts  BillingAccountRepository
	checkoutSessions CheckoutSessionRepository
	subscriptions    SubscriptionRepository
	oneTimePurchases OneTimePurchaseRepository
	webhookEvents    WebhookEventRepository
	entitlements     EntitlementRepository

	transactor Transactor
	outbox     OutboxWriter // optional
	gateway    StripeGateway
	catalog    Catalog
	clock      Clock

	webhookSecret   string
	successURL      string
	cancelURL       string
	portalReturnURL string
}

// Dependencies bundles the ports Service needs.
type Dependencies struct {
	BillingAccounts  BillingAccountRepository
	CheckoutSessions CheckoutSessionRepository
	Subscriptions    SubscriptionRepository
	OneTimePurchases OneTimePurchaseRepository
	WebhookEvents    WebhookEventRepository
	Entitlements     EntitlementRepository

	Transactor Transactor
	Outbox     OutboxWriter // optional: billing works without notifications
	Gateway    StripeGateway
	Catalog    Catalog
	Clock      Clock

	// WebhookSecret verifies the Stripe-Signature header (STRIPE_WEBHOOK_SECRET).
	WebhookSecret string
	// SuccessURL/CancelURL/PortalReturnURL are server-configured redirect
	// targets. They are never accepted from the client per ADR-0006 and the
	// checkout/portal security invariants.
	SuccessURL      string
	CancelURL       string
	PortalReturnURL string
}

// NewService validates deps and returns a ready Service.
func NewService(deps Dependencies) (*Service, error) {
	switch {
	case deps.BillingAccounts == nil, deps.CheckoutSessions == nil, deps.Subscriptions == nil,
		deps.OneTimePurchases == nil, deps.WebhookEvents == nil, deps.Entitlements == nil,
		deps.Transactor == nil, deps.Gateway == nil:
		return nil, errors.New("billing: all service dependencies are required")
	}
	if deps.WebhookSecret == "" {
		return nil, errors.New("billing: webhook secret is required")
	}
	if deps.SuccessURL == "" || deps.CancelURL == "" || deps.PortalReturnURL == "" {
		return nil, errors.New("billing: success, cancel, and portal return urls are required")
	}
	clock := deps.Clock
	if clock == nil {
		clock = SystemClock{}
	}
	catalog := deps.Catalog
	if catalog == nil {
		catalog = Catalog{}
	}
	return &Service{
		billingAccounts:  deps.BillingAccounts,
		checkoutSessions: deps.CheckoutSessions,
		subscriptions:    deps.Subscriptions,
		oneTimePurchases: deps.OneTimePurchases,
		webhookEvents:    deps.WebhookEvents,
		entitlements:     deps.Entitlements,
		transactor:       deps.Transactor,
		outbox:           deps.Outbox,
		gateway:          deps.Gateway,
		catalog:          catalog,
		clock:            clock,
		webhookSecret:    deps.WebhookSecret,
		successURL:       deps.SuccessURL,
		cancelURL:        deps.CancelURL,
		portalReturnURL:  deps.PortalReturnURL,
	}, nil
}

var _ EntitlementReader = (*Service)(nil)

// HasEntitlement satisfies EntitlementReader. It reads only the projected
// entitlements table; it never calls Stripe on the request path.
func (s *Service) HasEntitlement(ctx context.Context, organizationID, featureKey string) (bool, error) {
	if organizationID == "" || featureKey == "" {
		return false, ErrInvalidInput
	}
	return s.entitlements.HasEntitlement(ctx, organizationID, featureKey)
}

// GetSummary returns the organization's current subscription (if any) and
// its resolved entitlements. It never fails just because the organization
// has no billing account yet (a fresh organization with no purchases).
func (s *Service) GetSummary(ctx context.Context, organizationID string) (Summary, error) {
	if organizationID == "" {
		return Summary{}, ErrInvalidInput
	}

	entitlements, err := s.entitlements.ListForOrganization(ctx, organizationID)
	if err != nil {
		return Summary{}, fmt.Errorf("list entitlements: %w", err)
	}
	summary := Summary{OrganizationID: organizationID, Entitlements: entitlements}

	account, err := s.billingAccounts.GetByOrganization(ctx, organizationID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return summary, nil
		}
		return Summary{}, fmt.Errorf("get billing account: %w", err)
	}

	subscription, ok, err := s.subscriptions.GetLatestForBillingAccount(ctx, account.ID)
	if err != nil {
		return Summary{}, fmt.Errorf("get latest subscription: %w", err)
	}
	if ok {
		sub := subscription
		summary.Subscription = &sub
		summary.HasActiveSubscription = isActiveStatus(subscription.Status)
	}
	return summary, nil
}

// CreateCheckoutSession maps a server-controlled priceKey to a real Stripe
// Price via the catalog and creates a Checkout Session. The client can never
// supply a Stripe price/customer ID, an amount, or a success/cancel URL —
// those are resolved entirely server-side.
func (s *Service) CreateCheckoutSession(ctx context.Context, organizationID, priceKey string) (CreatedCheckoutSession, error) {
	if organizationID == "" {
		return CreatedCheckoutSession{}, ErrInvalidInput
	}
	entry, err := s.catalog.Lookup(priceKey)
	if err != nil {
		return CreatedCheckoutSession{}, err
	}

	account, err := s.getOrCreateBillingAccount(ctx, organizationID)
	if err != nil {
		return CreatedCheckoutSession{}, fmt.Errorf("get or create billing account: %w", err)
	}

	stripeSessionID, url, err := s.gateway.CreateCheckoutSession(ctx, CheckoutParams{
		CustomerID:     account.StripeCustomerID,
		PriceID:        entry.StripePriceID,
		Mode:           entry.Mode,
		SuccessURL:     s.successURL,
		CancelURL:      s.cancelURL,
		OrganizationID: organizationID,
	})
	if err != nil {
		return CreatedCheckoutSession{}, fmt.Errorf("create stripe checkout session: %w", err)
	}

	stored, err := s.checkoutSessions.Create(ctx, account.ID, stripeSessionID, priceKey, entry.Mode)
	if err != nil {
		return CreatedCheckoutSession{}, fmt.Errorf("record checkout session: %w", err)
	}

	return CreatedCheckoutSession{CheckoutSession: stored, URL: url}, nil
}

// CreatePortalSession opens a Stripe Customer Portal session for an
// organization that already has a billing account. It never grants or
// revokes an entitlement itself — only a verified webhook does that.
func (s *Service) CreatePortalSession(ctx context.Context, organizationID string) (CreatedPortalSession, error) {
	if organizationID == "" {
		return CreatedPortalSession{}, ErrInvalidInput
	}
	account, err := s.billingAccounts.GetByOrganization(ctx, organizationID)
	if err != nil {
		return CreatedPortalSession{}, err
	}
	url, err := s.gateway.CreatePortalSession(ctx, account.StripeCustomerID, s.portalReturnURL)
	if err != nil {
		return CreatedPortalSession{}, fmt.Errorf("create stripe portal session: %w", err)
	}
	return CreatedPortalSession{URL: url}, nil
}

func (s *Service) getOrCreateBillingAccount(ctx context.Context, organizationID string) (BillingAccount, error) {
	account, err := s.billingAccounts.GetByOrganization(ctx, organizationID)
	if err == nil {
		return account, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return BillingAccount{}, err
	}

	stripeCustomerID, err := s.gateway.CreateCustomer(ctx, organizationID)
	if err != nil {
		return BillingAccount{}, fmt.Errorf("create stripe customer: %w", err)
	}
	return s.billingAccounts.GetOrCreate(ctx, organizationID, stripeCustomerID)
}

func isActiveStatus(status string) bool {
	return status == "active" || status == "trialing"
}
