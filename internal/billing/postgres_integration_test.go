//go:build integration

package billing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/testdb"
	"github.com/hydrz/starter/internal/store"
)

func TestBillingQueriesIdempotentWebhookProjection(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := testdb.Context(t)

	organization := createOrganizationForBilling(t, ctx, db.Queries)
	account, err := db.Queries.CreateBillingAccount(ctx, store.CreateBillingAccountParams{
		OrganizationID:   organization,
		StripeCustomerID: "cus_" + uuid.NewString()[:8],
	})
	if err != nil {
		t.Fatalf("create billing account: %v", err)
	}
	if fetched, err := db.Queries.GetBillingAccountByStripeCustomerID(ctx, account.StripeCustomerID); err != nil || fetched.ID != account.ID {
		t.Fatalf("get billing account by Stripe customer = %#v, %v", fetched, err)
	}

	session, err := db.Queries.CreateCheckoutSession(ctx, store.CreateCheckoutSessionParams{
		BillingAccountID:        account.ID,
		StripeCheckoutSessionID: "cs_" + uuid.NewString()[:8],
		PriceKey:                "price_pro",
		Mode:                    "subscription",
		Status:                  "open",
	})
	if err != nil {
		t.Fatalf("create checkout session: %v", err)
	}
	if updated, err := db.Queries.UpdateCheckoutSessionStatus(ctx, store.UpdateCheckoutSessionStatusParams{
		StripeCheckoutSessionID: session.StripeCheckoutSessionID, Status: "completed",
	}); err != nil || updated != 1 {
		t.Fatalf("update checkout session status = %d, %v", updated, err)
	}

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	subscription, err := db.Queries.UpsertSubscription(ctx, store.UpsertSubscriptionParams{
		BillingAccountID: account.ID, StripeSubscriptionID: "sub_" + uuid.NewString()[:8],
		PriceKey: "price_pro", Status: "active", CurrentPeriodEnd: future(), LastEventCreatedAt: now,
	})
	if err != nil {
		t.Fatalf("upsert subscription: %v", err)
	}
	stale, err := db.Queries.UpsertSubscription(ctx, store.UpsertSubscriptionParams{
		BillingAccountID: account.ID, StripeSubscriptionID: subscription.StripeSubscriptionID,
		PriceKey: "price_pro", Status: "past_due", LastEventCreatedAt: pgtype.Timestamptz{Time: now.Time.Add(-time.Hour), Valid: true},
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("stale subscription upsert = %#v, %v; want pgx.ErrNoRows", stale, err)
	}

	purchaseKey := "cs_" + uuid.NewString()[:8]
	if _, err := db.Queries.InsertOneTimePurchase(ctx, store.InsertOneTimePurchaseParams{
		BillingAccountID: account.ID, StripeCheckoutSessionID: purchaseKey, PriceKey: "price_pack", AmountMinorUnits: 1000, Currency: "usd", Status: "completed",
	}); err != nil {
		t.Fatalf("insert one-time purchase: %v", err)
	}
	if replay, err := db.Queries.InsertOneTimePurchase(ctx, store.InsertOneTimePurchaseParams{
		BillingAccountID: account.ID, StripeCheckoutSessionID: purchaseKey, PriceKey: "price_pack", AmountMinorUnits: 1000, Currency: "usd", Status: "completed",
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("replayed purchase = %#v, %v; want pgx.ErrNoRows", replay, err)
	}

	event := store.InsertWebhookEventParams{StripeEventID: "evt_" + uuid.NewString()[:8], EventType: "checkout.session.completed", EventCreatedAt: now}
	if _, err := db.Queries.InsertWebhookEvent(ctx, event); err != nil {
		t.Fatalf("insert webhook event: %v", err)
	}
	if duplicate, err := db.Queries.InsertWebhookEvent(ctx, event); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("duplicate webhook event = %#v, %v; want pgx.ErrNoRows", duplicate, err)
	}

	entitlement, err := db.Queries.UpsertEntitlement(ctx, store.UpsertEntitlementParams{
		OrganizationID: organization, FeatureKey: "feature.pro", Source: "subscription", Enabled: true, SubscriptionID: subscription.ID,
	})
	if err != nil {
		t.Fatalf("upsert entitlement: %v", err)
	}
	if has, err := db.Queries.HasEntitlement(ctx, store.HasEntitlementParams{OrganizationID: organization, FeatureKey: "feature.pro"}); err != nil || !has {
		t.Fatalf("has entitlement = %v, %v; want true", has, err)
	}
	if _, err := db.Queries.UpsertEntitlement(ctx, store.UpsertEntitlementParams{
		OrganizationID: organization, FeatureKey: entitlement.FeatureKey, Source: "subscription", Enabled: false,
	}); err != nil {
		t.Fatalf("disable entitlement: %v", err)
	}
	if has, err := db.Queries.HasEntitlement(ctx, store.HasEntitlementParams{OrganizationID: organization, FeatureKey: "feature.pro"}); err != nil || has {
		t.Fatalf("disabled entitlement still active = %v, %v", has, err)
	}
	if listed, err := db.Queries.ListEntitlementsForOrganization(ctx, organization); err != nil || len(listed) != 1 {
		t.Fatalf("list entitlements = %d items, %v", len(listed), err)
	}
}

func createOrganizationForBilling(t *testing.T, ctx context.Context, queries *store.Queries) pgtype.UUID {
	t.Helper()
	organization, err := queries.CreateOrganization(ctx, store.CreateOrganizationParams{
		Slug: "billing-" + uuid.NewString()[:8], Name: "Billing Integration",
	})
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	return organization.ID
}

func future() pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: time.Now().Add(24 * time.Hour), Valid: true}
}
