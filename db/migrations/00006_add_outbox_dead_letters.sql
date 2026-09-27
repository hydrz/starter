-- +goose Up

ALTER TABLE outbox_events
    ADD COLUMN dead_lettered_at timestamptz;

DROP INDEX outbox_events_claim_idx;
CREATE INDEX outbox_events_claim_idx
    ON outbox_events (available_at, created_at)
    WHERE processed_at IS NULL AND dead_lettered_at IS NULL;

-- +goose Down

DROP INDEX outbox_events_claim_idx;
CREATE INDEX outbox_events_claim_idx
    ON outbox_events (available_at, created_at)
    WHERE processed_at IS NULL;

ALTER TABLE outbox_events
    DROP COLUMN dead_lettered_at;
