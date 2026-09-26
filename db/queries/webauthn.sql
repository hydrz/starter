-- name: CreateWebAuthnChallenge :one
INSERT INTO webauthn_challenges (ceremony, user_id, challenge, session_data, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, ceremony, user_id, expires_at;

-- name: ConsumeWebAuthnChallenge :one
UPDATE webauthn_challenges
SET consumed_at = now()
WHERE challenge = $1
  AND ceremony = $2
  AND consumed_at IS NULL
  AND expires_at > now()
RETURNING id, ceremony, user_id, session_data;

-- name: CreateWebAuthnCredential :one
INSERT INTO webauthn_credentials (user_id, credential_id, public_key, attestation_type, aaguid, sign_count, transports, user_handle, name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, user_id, credential_id, public_key, sign_count, user_handle, created_at;

-- name: ListWebAuthnCredentialsForUser :many
SELECT id, user_id, credential_id, public_key, attestation_type, aaguid, sign_count, transports, user_handle, name, created_at, last_used_at
FROM webauthn_credentials
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetWebAuthnCredentialByCredentialID :one
SELECT id, user_id, credential_id, public_key, attestation_type, aaguid, sign_count, transports, user_handle, name, created_at, last_used_at
FROM webauthn_credentials
WHERE credential_id = $1;

-- name: UpdateWebAuthnCredentialSignCount :execrows
UPDATE webauthn_credentials
SET sign_count = $2, last_used_at = now()
WHERE id = $1 AND sign_count < $2;
