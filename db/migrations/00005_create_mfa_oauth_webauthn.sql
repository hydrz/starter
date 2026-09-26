-- +goose Up
-- Email OTP challenges: hashed code, purpose, expiry, attempt counter,
-- single-use, keyed by email + optional IP for rate limiting. Codes are
-- never stored raw (see auth.SecretDigester usage in internal/auth/otp.go).
CREATE TABLE email_otp_challenges (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL,
    purpose text NOT NULL CHECK (purpose IN ('sign_in')),
    code_digest bytea NOT NULL,
    ip_address inet,
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    max_attempts integer NOT NULL DEFAULT 5 CHECK (max_attempts > 0),
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX email_otp_challenges_email_idx ON email_otp_challenges (email, created_at DESC);
CREATE INDEX email_otp_challenges_ip_idx ON email_otp_challenges (ip_address, created_at DESC) WHERE ip_address IS NOT NULL;

-- TOTP factors: one per user, AES-GCM encrypted secret ciphertext + nonce.
-- status distinguishes an enrollment awaiting confirmation ('pending') from
-- an active second factor ('verified'); only one verified factor per user.
CREATE TABLE totp_factors (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    secret_ciphertext bytea NOT NULL,
    secret_nonce bytea NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'verified')),
    created_at timestamptz NOT NULL DEFAULT now(),
    verified_at timestamptz
);
CREATE INDEX totp_factors_user_id_idx ON totp_factors (user_id);
CREATE UNIQUE INDEX totp_factors_one_verified_per_user ON totp_factors (user_id) WHERE status = 'verified';

-- Recovery codes: hash only, single-use, tied to a verified TOTP factor.
CREATE TABLE totp_recovery_codes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    factor_id uuid NOT NULL REFERENCES totp_factors (id) ON DELETE CASCADE,
    code_digest bytea NOT NULL UNIQUE,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX totp_recovery_codes_factor_id_idx ON totp_recovery_codes (factor_id);

-- MFA challenges: short-lived, ties a partially-authenticated sign-in
-- attempt to the factor that must complete it. Single-use.
CREATE TABLE mfa_challenges (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    factor_id uuid NOT NULL REFERENCES totp_factors (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX mfa_challenges_user_id_idx ON mfa_challenges (user_id);

-- OAuth accounts: (provider, provider_subject) is the identity key. Email is
-- stored only for display/diagnostics; account resolution never keys on it.
CREATE TABLE oauth_accounts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider text NOT NULL CHECK (provider IN ('google', 'github')),
    provider_subject text NOT NULL,
    email text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT oauth_accounts_provider_subject_key UNIQUE (provider, provider_subject)
);
CREATE INDEX oauth_accounts_user_id_idx ON oauth_accounts (user_id);

-- OAuth authorization state: one-time state/nonce/PKCE verifier, single-use,
-- short expiry. intent='link' carries the authenticated user to attach the
-- provider identity to, so an unauthenticated callback can never link.
CREATE TABLE oauth_authorization_states (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider text NOT NULL CHECK (provider IN ('google', 'github')),
    state_digest bytea NOT NULL UNIQUE,
    nonce text NOT NULL,
    code_verifier text NOT NULL,
    intent text NOT NULL CHECK (intent IN ('sign_in', 'link')),
    linking_user_id uuid REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT oauth_authorization_states_link_requires_user CHECK (
        (intent = 'link' AND linking_user_id IS NOT NULL) OR
        (intent = 'sign_in' AND linking_user_id IS NULL)
    )
);

-- WebAuthn credentials: created only from an authenticated session.
CREATE TABLE webauthn_credentials (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    credential_id bytea NOT NULL UNIQUE,
    public_key bytea NOT NULL,
    attestation_type text NOT NULL DEFAULT '',
    aaguid bytea,
    sign_count bigint NOT NULL DEFAULT 0,
    transports text,
    user_handle bytea NOT NULL,
    name text,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz
);
CREATE INDEX webauthn_credentials_user_id_idx ON webauthn_credentials (user_id);
CREATE INDEX webauthn_credentials_user_handle_idx ON webauthn_credentials (user_handle);

-- WebAuthn challenges: server-persisted ceremony state (the full
-- webauthn.SessionData, serialized), short TTL, single-use, tied to either a
-- registration (user_id set, authenticated caller) or an authentication
-- ceremony (user_id null for discoverable/usernameless login).
CREATE TABLE webauthn_challenges (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    ceremony text NOT NULL CHECK (ceremony IN ('registration', 'authentication')),
    user_id uuid REFERENCES users (id) ON DELETE CASCADE,
    challenge bytea NOT NULL UNIQUE,
    session_data jsonb NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX webauthn_challenges_user_id_idx ON webauthn_challenges (user_id);

-- +goose Down
DROP TABLE webauthn_challenges;
DROP TABLE webauthn_credentials;
DROP TABLE oauth_authorization_states;
DROP TABLE oauth_accounts;
DROP TABLE mfa_challenges;
DROP TABLE totp_recovery_codes;
DROP TABLE totp_factors;
DROP TABLE email_otp_challenges;
