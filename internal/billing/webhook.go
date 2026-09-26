package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	stripesdk "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
)

// HandleWebhook verifies payload against sigHeader using webhookSecret
// BEFORE any further parsing (ADR-0006 / security invariant 5: only a
// verified webhook is proof of payment, never a browser redirect). If the
// event was already processed, it returns nil immediately (idempotent
// no-op). Otherwise it records the event and updates subscription/
// entitlement projections atomically in one DB transaction: a delivery is
// either fully applied or not applied at all. A transient persistence
// failure returns a non-nil error so the caller responds with a 5xx and
// Stripe retries; it is never swallowed.
func (s *Service) HandleWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	event, err := webhook.ConstructEvent(payload, sigHeader, s.webhookSecret)
	if err != nil {
		return ErrInvalidSignature
	}

	eventCreatedAt := time.Unix(event.Created, 0).UTC()

	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		inserted, err := s.webhookEvents.Insert(ctx, event.ID, string(event.Type), eventCreatedAt)
		if err != nil {
			return fmt.Errorf("record webhook event: %w", err)
		}
		if !inserted {
			// Duplicate delivery of an event already recorded: idempotent no-op.
			return nil
		}
		return s.applyEvent(ctx, event, eventCreatedAt)
	})
}

func (s *Service) applyEvent(ctx context.Context, event stripesdk.Event, eventCreatedAt time.Time) error {
	switch event.Type {
	case "checkout.session.completed":
		return s.applyCheckoutSessionCompleted(ctx, event)
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		return s.applySubscriptionEvent(ctx, event, eventCreatedAt)
	default:
		// Unhandled event types are still recorded (for idempotency/audit)
		// but require no projection change.
		return nil
	}
}

func (s *Service) applyCheckoutSessionCompleted(ctx context.Context, event stripesdk.Event) error {
	var session stripesdk.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return fmt.Errorf("decode checkout session: %w", err)
	}

	if err := s.checkoutSessions.UpdateStatus(ctx, session.ID, "completed"); err != nil {
		return fmt.Errorf("update checkout session status: %w", err)
	}

	if session.Mode != stripesdk.CheckoutSessionModePayment {
		// Subscription checkouts are projected by customer.subscription.*
		// events; the checkout completion itself never grants an
		// entitlement (checkout redirects are not proof of payment).
		return nil
	}

	stored, ok, err := s.checkoutSessions.GetByStripeID(ctx, session.ID)
	if err != nil {
		return fmt.Errorf("lookup checkout session: %w", err)
	}
	if !ok {
		// Not a session this service created: nothing to project.
		return nil
	}

	customerID := ""
	if session.Customer != nil {
		customerID = session.Customer.ID
	}
	if customerID == "" {
		return fmt.Errorf("checkout session %s has no customer", session.ID)
	}
	account, err := s.billingAccounts.GetByStripeCustomerID(ctx, customerID)
	if err != nil {
		return fmt.Errorf("lookup billing account: %w", err)
	}

	purchase, inserted, err := s.oneTimePurchases.Insert(ctx, InsertOneTimePurchaseInput{
		BillingAccountID:        account.ID,
		StripeCheckoutSessionID: session.ID,
		PriceKey:                stored.PriceKey,
		AmountMinorUnits:        session.AmountTotal,
		Currency:                string(session.Currency),
	})
	if err != nil {
		return fmt.Errorf("insert one-time purchase: %w", err)
	}
	if !inserted {
		// Already recorded by an earlier delivery of this same event/session.
		return nil
	}

	entry, err := s.catalog.Lookup(stored.PriceKey)
	if err != nil {
		// Purchase is recorded; there is simply no feature mapped to grant.
		return nil
	}

	purchaseID := purchase.ID
	if _, err := s.entitlements.Upsert(ctx, UpsertEntitlementInput{
		OrganizationID:    account.OrganizationID,
		FeatureKey:        entry.FeatureKey,
		Source:            EntitlementSourceOneTime,
		Enabled:           true,
		ExpiresAt:         nil,
		OneTimePurchaseID: &purchaseID,
	}); err != nil {
		return fmt.Errorf("upsert entitlement: %w", err)
	}

	return s.writeOutboxEvent(ctx, "billing.one_time_purchase_completed", account.OrganizationID, purchase.ID)
}

// applySubscriptionEvent handles customer.subscription.created/updated/
// deleted uniformly: the subscription's own Status field already reflects
// cancellation on a "deleted" event, so a single upsert-plus-entitlement
// path is correct for all three. Cancelling a subscription only disables
// its own (source=subscription) entitlement row — a separate one-time
// purchase's (source=one_time) row for the same feature is never touched.
func (s *Service) applySubscriptionEvent(ctx context.Context, event stripesdk.Event, eventCreatedAt time.Time) error {
	var sub stripesdk.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("decode subscription: %w", err)
	}
	if sub.Customer == nil || sub.Customer.ID == "" {
		return fmt.Errorf("subscription %s event missing customer id", sub.ID)
	}
	account, err := s.billingAccounts.GetByStripeCustomerID(ctx, sub.Customer.ID)
	if err != nil {
		return fmt.Errorf("lookup billing account: %w", err)
	}

	stripePriceID := ""
	if sub.Items != nil && len(sub.Items.Data) > 0 && sub.Items.Data[0].Price != nil {
		stripePriceID = sub.Items.Data[0].Price.ID
	}
	featureKey, priceKey, found := s.catalog.FeatureKeyForPrice(stripePriceID)

	var periodEnd *time.Time
	if sub.CurrentPeriodEnd > 0 {
		t := time.Unix(sub.CurrentPeriodEnd, 0).UTC()
		periodEnd = &t
	}

	projected, applied, err := s.subscriptions.Upsert(ctx, UpsertSubscriptionInput{
		BillingAccountID:     account.ID,
		StripeSubscriptionID: sub.ID,
		PriceKey:             priceKey,
		Status:               string(sub.Status),
		CurrentPeriodEnd:     periodEnd,
		CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
		EventCreatedAt:       eventCreatedAt,
	})
	if err != nil {
		return fmt.Errorf("upsert subscription: %w", err)
	}
	if !applied {
		// A newer event already projected this subscription's state:
		// out-of-order delivery must never regress it.
		return nil
	}
	if !found {
		// The subscription's price is not in the catalog: subscription
		// state is still projected, but there is no feature to grant.
		return nil
	}

	enabled := isActiveStatus(string(sub.Status))
	subscriptionID := projected.ID
	if _, err := s.entitlements.Upsert(ctx, UpsertEntitlementInput{
		OrganizationID: account.OrganizationID,
		FeatureKey:     featureKey,
		Source:         EntitlementSourceSubscription,
		Enabled:        enabled,
		ExpiresAt:      periodEnd,
		SubscriptionID: &subscriptionID,
	}); err != nil {
		return fmt.Errorf("upsert entitlement: %w", err)
	}

	return s.writeOutboxEvent(ctx, "billing.subscription_updated", account.OrganizationID, projected.ID)
}

func (s *Service) writeOutboxEvent(ctx context.Context, topic, organizationID, aggregateID string) error {
	if s.outbox == nil {
		return nil
	}
	payload, err := json.Marshal(struct {
		OrganizationID string `json:"organization_id"`
	}{OrganizationID: organizationID})
	if err != nil {
		return fmt.Errorf("encode outbox payload: %w", err)
	}
	idempotencyKey := topic + "|" + aggregateID
	if err := s.outbox.WriteEvent(ctx, topic, "billing", aggregateID, payload, idempotencyKey); err != nil {
		return fmt.Errorf("write outbox event: %w", err)
	}
	return nil
}
