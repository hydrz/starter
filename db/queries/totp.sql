-- name: CreateTOTPFactor :one
INSERT INTO totp_factors (user_id, secret_ciphertext, secret_nonce, status)
VALUES ($1, $2, $3, 'pending')
RETURNING id, user_id, secret_ciphertext, secret_nonce, status, created_at, verified_at;

-- name: GetTOTPFactorByID :one
SELECT id, user_id, secret_ciphertext, secret_nonce, status, created_at, verified_at
FROM totp_factors
WHERE id = $1;

-- name: GetVerifiedTOTPFactorForUser :one
SELECT id, user_id, secret_ciphertext, secret_nonce, status, created_at, verified_at
FROM totp_factors
WHERE user_id = $1 AND status = 'verified';

-- name: ActivateTOTPFactor :execrows
UPDATE totp_factors
SET status = 'verified', verified_at = now()
WHERE id = $1 AND user_id = $2 AND status = 'pending';

-- name: DeletePendingTOTPFactorsForUser :execrows
DELETE FROM totp_factors
WHERE user_id = $1 AND status = 'pending';

-- name: CreateTOTPRecoveryCode :exec
INSERT INTO totp_recovery_codes (factor_id, code_digest)
VALUES ($1, $2);

-- name: ConsumeTOTPRecoveryCode :one
UPDATE totp_recovery_codes
SET consumed_at = now()
WHERE factor_id = $1
  AND code_digest = $2
  AND consumed_at IS NULL
RETURNING id;

-- name: CreateMFAChallenge :one
INSERT INTO mfa_challenges (user_id, factor_id, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, factor_id, expires_at, created_at;

-- name: GetActiveMFAChallenge :one
SELECT id, user_id, factor_id, expires_at
FROM mfa_challenges
WHERE id = $1
  AND consumed_at IS NULL
  AND expires_at > now();

-- name: ConsumeMFAChallenge :one
UPDATE mfa_challenges
SET consumed_at = now()
WHERE id = $1
  AND consumed_at IS NULL
  AND expires_at > now()
RETURNING id, user_id, factor_id;
