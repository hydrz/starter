-- +goose Up
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL,
    password_hash text NOT NULL,
    email_verified_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_normalized CHECK (email = lower(email)),
    CONSTRAINT users_email_length CHECK (char_length(email) BETWEEN 3 AND 320)
);
CREATE UNIQUE INDEX users_email_key ON users (email);

CREATE TABLE refresh_token_families (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    revoked_at timestamptz,
    revoke_reason text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX refresh_token_families_user_id_idx ON refresh_token_families (user_id);

CREATE TABLE refresh_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id uuid NOT NULL REFERENCES refresh_token_families (id) ON DELETE CASCADE,
    token_digest bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    replaced_at timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz,
    user_agent text,
    ip_address inet
);
CREATE INDEX refresh_sessions_family_id_idx ON refresh_sessions (family_id);
CREATE INDEX refresh_sessions_expires_at_idx ON refresh_sessions (expires_at);

CREATE TABLE one_time_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose text NOT NULL CHECK (purpose IN ('email_verification', 'password_reset')),
    token_digest bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX one_time_tokens_user_purpose_idx ON one_time_tokens (user_id, purpose, created_at DESC);

CREATE TABLE api_keys (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    organization_id uuid,
    name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
    key_prefix text NOT NULL,
    secret_digest bytea NOT NULL UNIQUE,
    expires_at timestamptz,
    revoked_at timestamptz,
    last_used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT api_keys_scope CHECK (organization_id IS NULL OR user_id IS NOT NULL)
);
CREATE INDEX api_keys_user_id_idx ON api_keys (user_id);
CREATE INDEX api_keys_organization_id_idx ON api_keys (organization_id) WHERE organization_id IS NOT NULL;

CREATE TABLE notification_intents (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_user_id uuid REFERENCES users (id) ON DELETE SET NULL,
    kind text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE outbox_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    topic text NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    payload jsonb NOT NULL,
    idempotency_key text NOT NULL UNIQUE,
    available_at timestamptz NOT NULL DEFAULT now(),
    claimed_at timestamptz,
    claim_token uuid,
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    processed_at timestamptz,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX outbox_events_claim_idx ON outbox_events (available_at, created_at) WHERE processed_at IS NULL;

CREATE TABLE delivery_messages (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_intent_id uuid REFERENCES notification_intents (id) ON DELETE SET NULL,
    outbox_event_id uuid UNIQUE REFERENCES outbox_events (id) ON DELETE SET NULL,
    channel text NOT NULL CHECK (channel = 'smtp'),
    recipient text NOT NULL,
    subject text NOT NULL,
    text_body text NOT NULL,
    html_body text NOT NULL,
    idempotency_key text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE delivery_attempts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_message_id uuid NOT NULL REFERENCES delivery_messages (id) ON DELETE CASCADE,
    attempt_number integer NOT NULL CHECK (attempt_number > 0),
    status text NOT NULL CHECK (status IN ('claimed', 'sent', 'failed')),
    provider_response text,
    error_message text,
    started_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    UNIQUE (delivery_message_id, attempt_number)
);

-- +goose Down
DROP TABLE delivery_attempts;
DROP TABLE delivery_messages;
DROP TABLE outbox_events;
DROP TABLE notification_intents;
DROP TABLE api_keys;
DROP TABLE one_time_tokens;
DROP TABLE refresh_sessions;
DROP TABLE refresh_token_families;
DROP TABLE users;
