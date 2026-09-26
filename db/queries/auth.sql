-- name: CreateUser :one
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING id, email, password_hash, email_verified_at, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, email_verified_at, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, email_verified_at, created_at, updated_at
FROM users
WHERE id = $1;

-- name: MarkUserEmailVerified :execrows
UPDATE users
SET email_verified_at = now(), updated_at = now()
WHERE id = $1 AND email_verified_at IS NULL;

-- name: UpdateUserPassword :execrows
UPDATE users
SET password_hash = $2, updated_at = now()
WHERE id = $1;

-- name: CreateRefreshTokenFamily :one
INSERT INTO refresh_token_families (user_id)
VALUES ($1)
RETURNING id, user_id, revoked_at, revoke_reason, created_at;

-- name: CreateRefreshSession :one
INSERT INTO refresh_sessions (family_id, token_digest, expires_at, user_agent, ip_address)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, family_id, token_digest, expires_at, replaced_at, revoked_at, created_at, last_used_at, user_agent, ip_address;

-- name: ConsumeRefreshSession :one
UPDATE refresh_sessions
SET replaced_at = now(), last_used_at = now()
WHERE token_digest = $1
  AND replaced_at IS NULL
  AND revoked_at IS NULL
  AND expires_at > now()
  AND EXISTS (
      SELECT 1 FROM refresh_token_families
      WHERE id = refresh_sessions.family_id AND revoked_at IS NULL
  )
RETURNING id, family_id, expires_at;

-- name: FindRefreshSession :one
SELECT session.id, session.family_id, session.expires_at, session.replaced_at, session.revoked_at,
       family.user_id, family.revoked_at AS family_revoked_at
FROM refresh_sessions AS session
JOIN refresh_token_families AS family ON family.id = session.family_id
WHERE session.token_digest = $1;

-- name: RevokeRefreshFamily :execrows
UPDATE refresh_token_families
SET revoked_at = now(), revoke_reason = $2
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeRefreshFamiliesForUser :execrows
UPDATE refresh_token_families
SET revoked_at = now(), revoke_reason = $2
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: ListRefreshSessionsForUser :many
SELECT session.id, session.created_at, session.last_used_at, session.expires_at, session.user_agent, session.ip_address
FROM refresh_sessions AS session
JOIN refresh_token_families AS family ON family.id = session.family_id
WHERE family.user_id = $1
  AND family.revoked_at IS NULL
  AND session.revoked_at IS NULL
  AND session.replaced_at IS NULL
  AND session.expires_at > now()
ORDER BY session.created_at DESC;

-- name: RevokeRefreshSessionForUser :execrows
UPDATE refresh_token_families AS family
SET revoked_at = now(), revoke_reason = 'user_revoked'
FROM refresh_sessions AS session
WHERE session.id = $1
  AND session.family_id = family.id
  AND family.user_id = $2
  AND family.revoked_at IS NULL;

-- name: CreateOneTimeToken :one
INSERT INTO one_time_tokens (user_id, purpose, token_digest, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, purpose, token_digest, expires_at, consumed_at, created_at;

-- name: ConsumeOneTimeToken :one
UPDATE one_time_tokens
SET consumed_at = now()
WHERE token_digest = $1
  AND purpose = $2
  AND consumed_at IS NULL
  AND expires_at > now()
RETURNING id, user_id, purpose, expires_at;

-- name: CreateAPIKey :one
INSERT INTO api_keys (user_id, organization_id, name, key_prefix, secret_digest, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, organization_id, name, key_prefix, expires_at, revoked_at, last_used_at, created_at;

-- name: ListAPIKeysForUser :many
SELECT id, user_id, organization_id, name, key_prefix, expires_at, revoked_at, last_used_at, created_at
FROM api_keys
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: RevokeAPIKeyForUser :execrows
UPDATE api_keys
SET revoked_at = now()
WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL;

-- name: GetActiveAPIKeyByDigest :one
UPDATE api_keys
SET last_used_at = now()
WHERE secret_digest = $1
  AND revoked_at IS NULL
  AND (expires_at IS NULL OR expires_at > now())
RETURNING id, user_id, organization_id, name, key_prefix, expires_at, created_at;
