-- name: CreateOAuthAuthorizationState :one
INSERT INTO oauth_authorization_states (provider, state_digest, nonce, code_verifier, intent, linking_user_id, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, provider, expires_at;

-- name: ConsumeOAuthAuthorizationState :one
UPDATE oauth_authorization_states
SET consumed_at = now()
WHERE state_digest = $1
  AND provider = $2
  AND consumed_at IS NULL
  AND expires_at > now()
RETURNING id, provider, nonce, code_verifier, intent, linking_user_id;

-- name: FindOAuthAccount :one
SELECT id, user_id, provider, provider_subject, email, created_at
FROM oauth_accounts
WHERE provider = $1 AND provider_subject = $2;

-- name: CreateOAuthAccount :one
INSERT INTO oauth_accounts (user_id, provider, provider_subject, email)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, provider, provider_subject, email, created_at;

-- name: ListOAuthAccountsForUser :many
SELECT id, user_id, provider, provider_subject, email, created_at
FROM oauth_accounts
WHERE user_id = $1
ORDER BY created_at DESC;
