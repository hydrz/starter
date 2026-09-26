package billing_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	stripesdk "github.com/stripe/stripe-go/v81"

	"github.com/hydrz/starter/internal/billing"
)

// signPayload computes a valid Stripe-Signature header for payload using
// Stripe's documented v1 signing scheme (HMAC-SHA256 over "{t}.{payload}"),
// the same scheme webhook.ConstructEvent verifies.
func signPayload(payload []byte, secret string, ts time.Time) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d", ts.Unix())))
	mac.Write([]byte("."))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d,v1=%s", ts.Unix(), sig)
}

func checkoutCompletedPayload(eventID, sessionID, customerID string, amountTotal int64, currency string, mode string, created int64) []byte {
	return []byte(fmt.Sprintf(`{
		"id": %q,
		"type": "checkout.session.completed",
		"created": %d,
		"api_version": %q,
		"data": {"object": {
			"id": %q,
			"mode": %q,
			"customer": %q,
			"amount_total": %d,
			"currency": %q
		}}
	}`, eventID, created, stripesdk.APIVersion, sessionID, mode, customerID, amountTotal, currency))
}

func subscriptionEventPayload(eventType, eventID, subscriptionID, customerID, status, priceID string, currentPeriodEnd int64, cancelAtPeriodEnd bool, created int64) []byte {
	return []byte(fmt.Sprintf(`{
		"id": %q,
		"type": %q,
		"created": %d,
		"api_version": %q,
		"data": {"object": {
			"id": %q,
			"customer": %q,
			"status": %q,
			"current_period_end": %d,
			"cancel_at_period_end": %t,
			"items": {"object": "list", "data": [{"id": "si_1", "object": "subscription_item", "price": {"id": %q, "object": "price"}}]}
		}}
	}`, eventID, eventType, created, stripesdk.APIVersion, subscriptionID, customerID, status, currentPeriodEnd, cancelAtPeriodEnd, priceID))
}

func TestHandleWebhook_InvalidSignatureRejected(t *testing.T) {
	t.Parallel()
	h := newHarness(t, billing.Catalog{})

	payload := checkoutCompletedPayload("evt_1", "cs_1", "cus_1", 1000, "usd", "payment", time.Now().Unix())
	err := h.service.HandleWebhook(context.Background(), payload, "t=1,v1=deadbeef")
	if err == nil {
		t.Fatal("HandleWebhook() error = nil, want ErrInvalidSignature")
	}
}

func TestHandleWebhook_DuplicateEventIsNoOp(t *testing.T) {
	t.Parallel()
	catalog := billing.Catalog{"lifetime": {StripePriceID: "price_lifetime", FeatureKey: "premium", Mode: billing.ModePayment}}
	h := newHarness(t, catalog)

	// Seed a billing account and a checkout session as CreateCheckoutSession would.
	account, err := h.billingAccounts.GetOrCreate(context.Background(), "org-1", "cus_1")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if _, err := h.checkoutSessions.Create(context.Background(), account.ID, "cs_1", "lifetime", billing.ModePayment); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	now := time.Now()
	payload := checkoutCompletedPayload("evt_dup_1", "cs_1", "cus_1", 5000, "usd", "payment", now.Unix())
	header := signPayload(payload, testWebhookSecret, now)

	if err := h.service.HandleWebhook(context.Background(), payload, header); err != nil {
		t.Fatalf("first HandleWebhook() error = %v", err)
	}
	has, err := h.entitlements.HasEntitlement(context.Background(), "org-1", "premium")
	if err != nil || !has {
		t.Fatalf("HasEntitlement() = %v, %v; want true, nil", has, err)
	}

	// Redeliver the exact same event: must be a no-op, not a double-grant
	// or an error.
	if err := h.service.HandleWebhook(context.Background(), payload, header); err != nil {
		t.Fatalf("duplicate HandleWebhook() error = %v", err)
	}
	if got := len(h.oneTimePurchases.byStripeID); got != 1 {
		t.Fatalf("one-time purchases recorded = %d, want 1 (duplicate must not double-insert)", got)
	}
}

func TestHandleWebhook_OutOfOrderSubscriptionEventDoesNotRegress(t *testing.T) {
	t.Parallel()
	catalog := billing.Catalog{"pro": {StripePriceID: "price_pro", FeatureKey: "pro", Mode: billing.ModeSubscription}}
	h := newHarness(t, catalog)

	if _, err := h.billingAccounts.GetOrCreate(context.Background(), "org-1", "cus_1"); err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}

	// The HMAC signature timestamp (used only for the replay-tolerance
	// window) is always "now"; the event's own `created` field (embedded in
	// the payload, independent of the signature timestamp) is what the
	// monotonic out-of-order guard compares.
	signNow := time.Now()
	base := signNow.Add(-time.Hour)
	periodEnd := base.Add(30 * 24 * time.Hour).Unix()

	// Newer event arrives first: subscription becomes active.
	newerCreated := base.Add(2 * time.Minute).Unix()
	newerPayload := subscriptionEventPayload("customer.subscription.updated", "evt_newer", "sub_1", "cus_1", "active", "price_pro", periodEnd, false, newerCreated)
	newerHeader := signPayload(newerPayload, testWebhookSecret, signNow)
	if err := h.service.HandleWebhook(context.Background(), newerPayload, newerHeader); err != nil {
		t.Fatalf("newer event HandleWebhook() error = %v", err)
	}
	has, err := h.entitlements.HasEntitlement(context.Background(), "org-1", "pro")
	if err != nil || !has {
		t.Fatalf("after newer event HasEntitlement() = %v, %v; want true, nil", has, err)
	}

	// An older event (e.g. redelivered/retried out of order) claims
	// past_due: it must NOT regress the already-applied active state.
	olderCreated := base.Unix()
	olderPayload := subscriptionEventPayload("customer.subscription.updated", "evt_older", "sub_1", "cus_1", "past_due", "price_pro", periodEnd, false, olderCreated)
	olderHeader := signPayload(olderPayload, testWebhookSecret, signNow)
	if err := h.service.HandleWebhook(context.Background(), olderPayload, olderHeader); err != nil {
		t.Fatalf("older event HandleWebhook() error = %v", err)
	}

	sub, ok, err := h.subscriptions.GetByStripeID(context.Background(), "sub_1")
	if err != nil || !ok {
		t.Fatalf("GetByStripeID() = %v, %v, %v", sub, ok, err)
	}
	if sub.Status != "active" {
		t.Fatalf("subscription status = %q, want %q (out-of-order event must not regress state)", sub.Status, "active")
	}
	has, err = h.entitlements.HasEntitlement(context.Background(), "org-1", "pro")
	if err != nil || !has {
		t.Fatalf("after older event HasEntitlement() = %v, %v; want true, nil", has, err)
	}
}

func TestHandleWebhook_CancellingSubscriptionNeverRevokesOneTimePurchase(t *testing.T) {
	t.Parallel()
	catalog := billing.Catalog{
		"pro":      {StripePriceID: "price_pro", FeatureKey: "premium", Mode: billing.ModeSubscription},
		"lifetime": {StripePriceID: "price_lifetime", FeatureKey: "premium", Mode: billing.ModePayment},
	}
	h := newHarness(t, catalog)

	account, err := h.billingAccounts.GetOrCreate(context.Background(), "org-1", "cus_1")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}

	// 1. One-time lifetime purchase grants "premium".
	if _, err := h.checkoutSessions.Create(context.Background(), account.ID, "cs_1", "lifetime", billing.ModePayment); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	now := time.Now()
	purchasePayload := checkoutCompletedPayload("evt_purchase", "cs_1", "cus_1", 9900, "usd", "payment", now.Unix())
	purchaseHeader := signPayload(purchasePayload, testWebhookSecret, now)
	if err := h.service.HandleWebhook(context.Background(), purchasePayload, purchaseHeader); err != nil {
		t.Fatalf("purchase HandleWebhook() error = %v", err)
	}

	// 2. A subscription ALSO grants "premium" (independent source).
	subCreated := now.Add(time.Minute)
	periodEnd := subCreated.Add(30 * 24 * time.Hour).Unix()
	subPayload := subscriptionEventPayload("customer.subscription.created", "evt_sub_created", "sub_1", "cus_1", "active", "price_pro", periodEnd, false, subCreated.Unix())
	subHeader := signPayload(subPayload, testWebhookSecret, subCreated)
	if err := h.service.HandleWebhook(context.Background(), subPayload, subHeader); err != nil {
		t.Fatalf("subscription created HandleWebhook() error = %v", err)
	}
	has, err := h.entitlements.HasEntitlement(context.Background(), "org-1", "premium")
	if err != nil || !has {
		t.Fatalf("HasEntitlement() after both grants = %v, %v; want true, nil", has, err)
	}

	// 3. Cancel the subscription.
	canceled := subCreated.Add(time.Minute)
	cancelPayload := subscriptionEventPayload("customer.subscription.deleted", "evt_sub_deleted", "sub_1", "cus_1", "canceled", "price_pro", periodEnd, false, canceled.Unix())
	cancelHeader := signPayload(cancelPayload, testWebhookSecret, canceled)
	if err := h.service.HandleWebhook(context.Background(), cancelPayload, cancelHeader); err != nil {
		t.Fatalf("subscription deleted HandleWebhook() error = %v", err)
	}

	// The feature must still be entitled: the one-time purchase's row is
	// untouched, even though the subscription's own row is now disabled.
	has, err = h.entitlements.HasEntitlement(context.Background(), "org-1", "premium")
	if err != nil || !has {
		t.Fatalf("HasEntitlement() after cancellation = %v, %v; want true, nil (one-time purchase must survive)", has, err)
	}

	subEntitlement := h.entitlements.rows[entitlementKey{"org-1", "premium", billing.EntitlementSourceSubscription}]
	if subEntitlement.Enabled {
		t.Fatal("subscription-sourced entitlement row still enabled after cancellation")
	}
	oneTimeEntitlement := h.entitlements.rows[entitlementKey{"org-1", "premium", billing.EntitlementSourceOneTime}]
	if !oneTimeEntitlement.Enabled {
		t.Fatal("one-time-sourced entitlement row was revoked by subscription cancellation")
	}
}

func TestHandleWebhook_TransientPersistenceFailureSurfacesAsError(t *testing.T) {
	t.Parallel()
	h := newHarness(t, billing.Catalog{})
	h.service = nil // ensure we don't reuse the harness's working service below

	// Rebuild a service whose entitlements repository fails, to prove a
	// mid-transaction persistence failure is never swallowed as success.
	failingEntitlements := &failingEntitlementRepository{}
	catalog := billing.Catalog{"lifetime": {StripePriceID: "price_lifetime", FeatureKey: "premium", Mode: billing.ModePayment}}

	h2 := newHarness(t, catalog)
	svc, err := billing.NewService(billing.Dependencies{
		BillingAccounts:  h2.billingAccounts,
		CheckoutSessions: h2.checkoutSessions,
		Subscriptions:    h2.subscriptions,
		OneTimePurchases: h2.oneTimePurchases,
		WebhookEvents:    h2.webhookEvents,
		Entitlements:     failingEntitlements,
		Transactor:       fakeTransactor{},
		Gateway:          h2.gateway,
		Catalog:          catalog,
		Clock:            &fakeClock{now: time.Now()},
		WebhookSecret:    testWebhookSecret,
		SuccessURL:       "https://app.test/billing/success",
		CancelURL:        "https://app.test/billing/cancel",
		PortalReturnURL:  "https://app.test/billing",
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	account, err := h2.billingAccounts.GetOrCreate(context.Background(), "org-1", "cus_1")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if _, err := h2.checkoutSessions.Create(context.Background(), account.ID, "cs_1", "lifetime", billing.ModePayment); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	now := time.Now()
	payload := checkoutCompletedPayload("evt_fail", "cs_1", "cus_1", 1000, "usd", "payment", now.Unix())
	header := signPayload(payload, testWebhookSecret, now)

	if err := svc.HandleWebhook(context.Background(), payload, header); err == nil {
		t.Fatal("HandleWebhook() error = nil, want a persistence error surfaced (never swallowed as 2xx)")
	}
}

type failingEntitlementRepository struct{}

func (failingEntitlementRepository) Upsert(context.Context, billing.UpsertEntitlementInput) (billing.Entitlement, error) {
	return billing.Entitlement{}, fmt.Errorf("simulated transient database failure")
}
func (failingEntitlementRepository) HasEntitlement(context.Context, string, string) (bool, error) {
	return false, nil
}
func (failingEntitlementRepository) ListForOrganization(context.Context, string) ([]billing.Entitlement, error) {
	return nil, nil
}
