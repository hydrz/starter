package billing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hydrz/starter/internal/billing"
)

func TestCreateCheckoutSession_UnknownPriceKeyRejected(t *testing.T) {
	t.Parallel()
	h := newHarness(t, billing.Catalog{})

	_, err := h.service.CreateCheckoutSession(context.Background(), "org-1", "does-not-exist")
	if !errors.Is(err, billing.ErrUnknownPriceKey) {
		t.Fatalf("CreateCheckoutSession() error = %v, want ErrUnknownPriceKey", err)
	}
}

func TestCreateCheckoutSession_NeverAcceptsClientAmountOrURLs(t *testing.T) {
	t.Parallel()
	// This test documents (and enforces via the type system) that
	// CreateCheckoutSession's only client-facing input is a price key: the
	// Stripe price, customer ID, mode, and success/cancel URLs are all
	// resolved from server-side configuration (the catalog and the
	// Service's own SuccessURL/CancelURL), never from caller-supplied
	// parameters.
	catalog := billing.Catalog{"pro": {StripePriceID: "price_pro", FeatureKey: "pro", Mode: billing.ModeSubscription}}
	h := newHarness(t, catalog)

	created, err := h.service.CreateCheckoutSession(context.Background(), "org-1", "pro")
	if err != nil {
		t.Fatalf("CreateCheckoutSession() error = %v", err)
	}
	if created.URL == "" {
		t.Fatal("CreateCheckoutSession() returned empty URL")
	}
	if created.PriceKey != "pro" {
		t.Fatalf("stored PriceKey = %q, want %q", created.PriceKey, "pro")
	}

	account, err := h.billingAccounts.GetByOrganization(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("GetByOrganization() error = %v", err)
	}
	if account.StripeCustomerID == "" {
		t.Fatal("expected a billing account with a server-created Stripe customer ID")
	}
}

func TestCreatePortalSession_UnknownOrganizationReturnsNotFound(t *testing.T) {
	t.Parallel()
	h := newHarness(t, billing.Catalog{})

	_, err := h.service.CreatePortalSession(context.Background(), "org-without-billing-account")
	if !errors.Is(err, billing.ErrNotFound) {
		t.Fatalf("CreatePortalSession() error = %v, want ErrNotFound", err)
	}
}

func TestBillingAccountIsolationAcrossOrganizations(t *testing.T) {
	t.Parallel()
	catalog := billing.Catalog{"pro": {StripePriceID: "price_pro", FeatureKey: "pro", Mode: billing.ModeSubscription}}
	h := newHarness(t, catalog)

	if _, err := h.service.CreateCheckoutSession(context.Background(), "org-a", "pro"); err != nil {
		t.Fatalf("CreateCheckoutSession(org-a) error = %v", err)
	}

	// A billing account created for org-a must never be visible when
	// resolving org-b: every operation resolves the billing account
	// strictly through the caller's own organization ID, never through a
	// client-supplied billing-account/customer ID (see internal/billing's
	// HTTPHandler doc comment for the enforcement chain).
	_, err := h.billingAccounts.GetByOrganization(context.Background(), "org-b")
	if !errors.Is(err, billing.ErrNotFound) {
		t.Fatalf("GetByOrganization(org-b) error = %v, want ErrNotFound", err)
	}

	summary, err := h.service.GetSummary(context.Background(), "org-b")
	if err != nil {
		t.Fatalf("GetSummary(org-b) error = %v", err)
	}
	if summary.HasActiveSubscription || summary.Subscription != nil || len(summary.Entitlements) != 0 {
		t.Fatalf("GetSummary(org-b) = %+v, want an empty summary (no leakage from org-a)", summary)
	}

	if _, err := h.service.CreatePortalSession(context.Background(), "org-b"); !errors.Is(err, billing.ErrNotFound) {
		t.Fatalf("CreatePortalSession(org-b) error = %v, want ErrNotFound", err)
	}
}

func TestGetSummary_NoBillingAccountYetReturnsEmptySummaryNotError(t *testing.T) {
	t.Parallel()
	h := newHarness(t, billing.Catalog{})

	summary, err := h.service.GetSummary(context.Background(), "brand-new-org")
	if err != nil {
		t.Fatalf("GetSummary() error = %v", err)
	}
	if summary.HasActiveSubscription {
		t.Fatal("HasActiveSubscription = true for an organization with no billing history")
	}
}

func TestHasEntitlement_RequiresOrganizationAndFeatureKey(t *testing.T) {
	t.Parallel()
	h := newHarness(t, billing.Catalog{})

	if _, err := h.service.HasEntitlement(context.Background(), "", "pro"); !errors.Is(err, billing.ErrInvalidInput) {
		t.Fatalf("HasEntitlement() error = %v, want ErrInvalidInput", err)
	}
	has, err := h.service.HasEntitlement(context.Background(), "org-1", "pro")
	if err != nil || has {
		t.Fatalf("HasEntitlement() = %v, %v; want false, nil for an org with no entitlements", has, err)
	}
}
