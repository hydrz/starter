-- name: CreateNotificationIntent :one
INSERT INTO notification_intents (recipient_user_id, kind, payload)
VALUES ($1, $2, $3)
RETURNING id, recipient_user_id, kind, payload, created_at;

-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (topic, aggregate_type, aggregate_id, payload, idempotency_key, available_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, topic, aggregate_type, aggregate_id, payload, idempotency_key, available_at, claimed_at, claim_token, attempts, processed_at, last_error, created_at;

-- name: ClaimOutboxEvents :many
WITH candidates AS (
    SELECT id
    FROM outbox_events
    WHERE processed_at IS NULL
      AND available_at <= now()
      AND (claimed_at IS NULL OR claimed_at < now() - sqlc.arg('claim_timeout')::interval)
    ORDER BY available_at, created_at
    LIMIT sqlc.arg('batch_size')
    FOR UPDATE SKIP LOCKED
)
UPDATE outbox_events AS event
SET claimed_at = now(), claim_token = sqlc.arg('claim_token')::uuid, attempts = attempts + 1
FROM candidates
WHERE event.id = candidates.id
RETURNING event.id, event.topic, event.aggregate_type, event.aggregate_id, event.payload, event.idempotency_key,
          event.available_at, event.claimed_at, event.claim_token, event.attempts, event.processed_at, event.last_error, event.created_at;

-- name: MarkOutboxEventProcessed :execrows
UPDATE outbox_events
SET processed_at = now(), claimed_at = NULL, claim_token = NULL, last_error = NULL
WHERE id = $1 AND claim_token = $2 AND processed_at IS NULL;

-- name: RetryOutboxEvent :execrows
UPDATE outbox_events
SET available_at = $3, claimed_at = NULL, claim_token = NULL, last_error = $4
WHERE id = $1 AND claim_token = $2 AND processed_at IS NULL;

-- name: CreateDeliveryMessage :one
INSERT INTO delivery_messages (notification_intent_id, outbox_event_id, channel, recipient, subject, text_body, html_body, idempotency_key)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, notification_intent_id, outbox_event_id, channel, recipient, subject, text_body, html_body, idempotency_key, created_at;

-- name: CreateDeliveryAttempt :one
INSERT INTO delivery_attempts (delivery_message_id, attempt_number, status)
VALUES ($1, $2, $3)
RETURNING id, delivery_message_id, attempt_number, status, provider_response, error_message, started_at, completed_at;

-- name: CompleteDeliveryAttempt :execrows
UPDATE delivery_attempts
SET status = $2, provider_response = $3, error_message = $4, completed_at = now()
WHERE id = $1 AND status = 'claimed';

-- name: GetDeliveryMessageByOutboxEvent :one
SELECT id, notification_intent_id, outbox_event_id, channel, recipient, subject, text_body, html_body, idempotency_key, created_at
FROM delivery_messages
WHERE outbox_event_id = $1;
