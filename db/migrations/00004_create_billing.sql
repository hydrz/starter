-- +goose Up
-- Billing account: exactly one per organization, linking it to a Stripe
-- customer. Never stores card/payment data — only the Stripe customer ID.
CREATE TABLE billing_accounts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL UNIQUE REFERENCES organizations (id) ON DELETE CASCADE,
    stripe_customer_id text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Checkout sessions initiated by an organization. Server-controlled price_key
-- and mode only; never a client-supplied Stripe price/amount. status is
-- updated only from verified webhook facts, never from the browser redirect.
CREATE TABLE checkout_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    billing_account_id uuid NOT NULL REFERENCES billing_accounts (id) ON DELETE CASCADE,
    stripe_checkout_session_id text NOT NULL UNIQUE,
    price_key text NOT NULL,
    mode text NOT NULL CHECK (mode IN ('payment', 'subscription')),
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'completed', 'expired')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX checkout_sessions_billing_account_id_idx ON checkout_sessions (billing_account_id);

-- Subscription projection derived only from verified webhook events.
-- last_event_created_at is the Stripe event `created` timestamp that most
-- recently updated this row: it is a monotonic guard so an out-of-order
-- (older) webhook delivery can never regress a newer state.
CREATE TABLE subscriptions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    billing_account_id uuid NOT NULL REFERENCES billing_accounts (id) ON DELETE CASCADE,
    stripe_subscription_id text NOT NULL UNIQUE,
    price_key text NOT NULL,
    status text NOT NULL,
    current_period_end timestamptz,
    cancel_at_period_end boolean NOT NULL DEFAULT false,
    last_event_created_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX subscriptions_billing_account_id_idx ON subscriptions (billing_account_id);

-- One-time purchases derived only from verified webhook events. Amounts are
-- integer minor units with an ISO currency code, never a float.
CREATE TABLE one_time_purchases (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    billing_account_id uuid NOT NULL REFERENCES billing_accounts (id) ON DELETE CASCADE,
    stripe_checkout_session_id text NOT NULL UNIQUE,
    price_key text NOT NULL,
    amount_minor_units bigint NOT NULL CHECK (amount_minor_units >= 0),
    currency text NOT NULL,
    status text NOT NULL DEFAULT 'completed',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX one_time_purchases_billing_account_id_idx ON one_time_purchases (billing_account_id);

-- Stripe webhook event ledger. stripe_event_id is the idempotency key: a
-- single UNIQUE constraint plus INSERT ... ON CONFLICT DO NOTHING is the
-- constraint-based idempotency check (never check-then-insert). Raw webhook
-- payloads are never stored here — only identifying/ledger fields.
CREATE TABLE stripe_webhook_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stripe_event_id text NOT NULL UNIQUE,
    event_type text NOT NULL,
    event_created_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now()
);

-- Entitlements are organization-scoped and derived only from webhook-verified
-- state. A feature can be granted by more than one independent source (e.g.
-- an active subscription AND a separate one-time purchase); each source's
-- grant is tracked as its own row so cancelling a subscription can never
-- revoke a one-time purchase's entitlement (it only disables the
-- subscription-sourced row for that feature).
CREATE TABLE entitlements (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    feature_key text NOT NULL,
    source text NOT NULL CHECK (source IN ('subscription', 'one_time')),
    enabled boolean NOT NULL,
    expires_at timestamptz,
    subscription_id uuid REFERENCES subscriptions (id) ON DELETE SET NULL,
    one_time_purchase_id uuid REFERENCES one_time_purchases (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT entitlements_org_feature_source_key UNIQUE (organization_id, feature_key, source)
);
CREATE INDEX entitlements_org_feature_idx ON entitlements (organization_id, feature_key);

-- Seed billing Casbin policies: owner/admin can read billing state and
-- initiate checkout/portal sessions; member/viewer can only read.
INSERT INTO casbin_rules (ptype, v0, v1, v2, v3) VALUES
    ('p', 'owner', '*', 'billing', 'read'),
    ('p', 'owner', '*', 'billing', 'checkout'),
    ('p', 'owner', '*', 'billing', 'portal'),

    ('p', 'admin', '*', 'billing', 'read'),
    ('p', 'admin', '*', 'billing', 'checkout'),
    ('p', 'admin', '*', 'billing', 'portal'),

    ('p', 'member', '*', 'billing', 'read'),
    ('p', 'viewer', '*', 'billing', 'read')
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM casbin_rules WHERE v2 = 'billing';
DROP TABLE IF EXISTS entitlements;
DROP TABLE IF EXISTS stripe_webhook_events;
DROP TABLE IF EXISTS one_time_purchases;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS checkout_sessions;
DROP TABLE IF EXISTS billing_accounts;
