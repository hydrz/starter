-- name: CreateBillingAccount :one
INSERT INTO billing_accounts (organization_id, stripe_customer_id)
VALUES ($1, $2)
ON CONFLICT (organization_id) DO UPDATE
SET stripe_customer_id = billing_accounts.stripe_customer_id
RETURNING id, organization_id, stripe_customer_id, created_at, updated_at;

-- name: GetBillingAccountByOrganization :one
SELECT id, organization_id, stripe_customer_id, created_at, updated_at
FROM billing_accounts
WHERE organization_id = $1;

-- name: GetBillingAccountByStripeCustomerID :one
SELECT id, organization_id, stripe_customer_id, created_at, updated_at
FROM billing_accounts
WHERE stripe_customer_id = $1;

-- name: CreateCheckoutSession :one
INSERT INTO checkout_sessions (billing_account_id, stripe_checkout_session_id, price_key, mode, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, billing_account_id, stripe_checkout_session_id, price_key, mode, status, created_at, updated_at;

-- name: GetCheckoutSessionByStripeID :one
SELECT id, billing_account_id, stripe_checkout_session_id, price_key, mode, status, created_at, updated_at
FROM checkout_sessions
WHERE stripe_checkout_session_id = $1;

-- name: UpdateCheckoutSessionStatus :execrows
UPDATE checkout_sessions
SET status = $2,
    updated_at = now()
WHERE stripe_checkout_session_id = $1;

-- name: UpsertSubscription :one
-- Monotonic guard: the ON CONFLICT DO UPDATE only fires when the incoming
-- event is at least as new as the last one applied to this subscription, so
-- an out-of-order (older) webhook delivery can never regress state. When the
-- guard rejects the update, no row is returned (treat as a no-op, not an
-- error).
INSERT INTO subscriptions (
    billing_account_id, stripe_subscription_id, price_key, status,
    current_period_end, cancel_at_period_end, last_event_created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (stripe_subscription_id) DO UPDATE
SET price_key = EXCLUDED.price_key,
    status = EXCLUDED.status,
    current_period_end = EXCLUDED.current_period_end,
    cancel_at_period_end = EXCLUDED.cancel_at_period_end,
    last_event_created_at = EXCLUDED.last_event_created_at,
    updated_at = now()
WHERE subscriptions.last_event_created_at IS NULL
   OR subscriptions.last_event_created_at < EXCLUDED.last_event_created_at
RETURNING id, billing_account_id, stripe_subscription_id, price_key, status,
          current_period_end, cancel_at_period_end, last_event_created_at, created_at, updated_at;

-- name: GetSubscriptionByStripeID :one
SELECT id, billing_account_id, stripe_subscription_id, price_key, status,
       current_period_end, cancel_at_period_end, last_event_created_at, created_at, updated_at
FROM subscriptions
WHERE stripe_subscription_id = $1;

-- name: ListSubscriptionsForBillingAccount :many
SELECT id, billing_account_id, stripe_subscription_id, price_key, status,
       current_period_end, cancel_at_period_end, last_event_created_at, created_at, updated_at
FROM subscriptions
WHERE billing_account_id = $1
ORDER BY created_at DESC;

-- name: InsertOneTimePurchase :one
-- ON CONFLICT DO NOTHING makes re-delivery of the same checkout session's
-- completion event idempotent at the row level, independent of the
-- stripe_webhook_events dedup (defense in depth).
INSERT INTO one_time_purchases (billing_account_id, stripe_checkout_session_id, price_key, amount_minor_units, currency, status)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (stripe_checkout_session_id) DO NOTHING
RETURNING id, billing_account_id, stripe_checkout_session_id, price_key, amount_minor_units, currency, status, created_at;

-- name: ListOneTimePurchasesForBillingAccount :many
SELECT id, billing_account_id, stripe_checkout_session_id, price_key, amount_minor_units, currency, status, created_at
FROM one_time_purchases
WHERE billing_account_id = $1
ORDER BY created_at DESC;

-- name: InsertWebhookEvent :one
-- Single-statement, constraint-based idempotency check: ON CONFLICT DO
-- NOTHING means a duplicate delivery of the same Stripe event ID returns no
-- row (not an error), which the caller treats as "already processed".
INSERT INTO stripe_webhook_events (stripe_event_id, event_type, event_created_at)
VALUES ($1, $2, $3)
ON CONFLICT (stripe_event_id) DO NOTHING
RETURNING id, stripe_event_id, event_type, event_created_at, received_at;

-- name: UpsertEntitlement :one
INSERT INTO entitlements (organization_id, feature_key, source, enabled, expires_at, subscription_id, one_time_purchase_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (organization_id, feature_key, source) DO UPDATE
SET enabled = EXCLUDED.enabled,
    expires_at = EXCLUDED.expires_at,
    subscription_id = EXCLUDED.subscription_id,
    one_time_purchase_id = EXCLUDED.one_time_purchase_id,
    updated_at = now()
RETURNING id, organization_id, feature_key, source, enabled, expires_at, subscription_id, one_time_purchase_id, created_at, updated_at;

-- name: HasEntitlement :one
SELECT EXISTS (
    SELECT 1
    FROM entitlements
    WHERE organization_id = $1
      AND feature_key = $2
      AND enabled = true
      AND (expires_at IS NULL OR expires_at > now())
) AS has_entitlement;

-- name: ListEntitlementsForOrganization :many
SELECT id, organization_id, feature_key, source, enabled, expires_at, subscription_id, one_time_purchase_id, created_at, updated_at
FROM entitlements
WHERE organization_id = $1
ORDER BY feature_key, source;
