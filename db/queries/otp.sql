-- name: CreateEmailOTPChallenge :one
INSERT INTO email_otp_challenges (email, purpose, code_digest, ip_address, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email, purpose, expires_at, created_at;

-- name: CountRecentEmailOTPChallenges :one
SELECT count(*)
FROM email_otp_challenges
WHERE email = $1 AND created_at > $2;

-- name: CountRecentEmailOTPChallengesByIP :one
SELECT count(*)
FROM email_otp_challenges
WHERE ip_address = $1 AND created_at > $2;

-- name: FindActiveEmailOTPChallenge :one
SELECT id, email, purpose, code_digest, attempt_count, max_attempts, expires_at
FROM email_otp_challenges
WHERE email = $1
  AND purpose = $2
  AND consumed_at IS NULL
  AND expires_at > now()
ORDER BY created_at DESC
LIMIT 1;

-- name: IncrementEmailOTPAttempt :one
UPDATE email_otp_challenges
SET attempt_count = attempt_count + 1
WHERE id = $1
  AND consumed_at IS NULL
  AND expires_at > now()
  AND attempt_count < max_attempts
RETURNING id, attempt_count, max_attempts;

-- name: ConsumeEmailOTPChallengeByID :one
UPDATE email_otp_challenges
SET consumed_at = now()
WHERE id = $1
  AND consumed_at IS NULL
  AND expires_at > now()
RETURNING id, email;
