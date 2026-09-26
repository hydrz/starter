package delivery

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/store"
)

// PostgresClaimer claims due outbox events from PostgreSQL using FOR UPDATE SKIP LOCKED.
type PostgresClaimer struct {
	queries *store.Queries
}

// NewPostgresClaimer creates a PostgresClaimer backed by the given queries.
func NewPostgresClaimer(queries *store.Queries) *PostgresClaimer {
	return &PostgresClaimer{queries: queries}
}

func (c *PostgresClaimer) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return c.queries.WithTx(tx)
	}
	return c.queries
}

// ClaimDueEvents claims a batch of due events with claimToken and claimTimeout.
func (c *PostgresClaimer) ClaimDueEvents(ctx context.Context, claimToken string, claimTimeout time.Duration, batchSize int) ([]OutboxEvent, error) {
	tokenUUID, err := uuid.Parse(claimToken)
	if err != nil {
		return nil, fmt.Errorf("invalid claim token uuid %q: %w", claimToken, err)
	}

	interval := pgtype.Interval{
		Microseconds: claimTimeout.Microseconds(),
		Valid:        true,
	}

	rows, err := c.q(ctx).ClaimOutboxEvents(ctx, store.ClaimOutboxEventsParams{
		ClaimToken:   pgtype.UUID{Bytes: tokenUUID, Valid: true},
		ClaimTimeout: interval,
		BatchSize:    int32(batchSize),
	})
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}

	events := make([]OutboxEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, mapStoreOutboxEvent(row))
	}
	return events, nil
}

func mapStoreOutboxEvent(row store.OutboxEvent) OutboxEvent {
	return OutboxEvent{
		ID:             uuid.UUID(row.ID.Bytes).String(),
		Topic:          row.Topic,
		AggregateType:  row.AggregateType,
		AggregateID:    uuid.UUID(row.AggregateID.Bytes).String(),
		Payload:        row.Payload,
		IdempotencyKey: row.IdempotencyKey,
		AvailableAt:    row.AvailableAt.Time,
		ClaimedAt:      pointerFromTimestamptz(row.ClaimedAt),
		ClaimToken:     pointerFromUUID(row.ClaimToken),
		Attempts:       int(row.Attempts),
		ProcessedAt:    pointerFromTimestamptz(row.ProcessedAt),
		LastError:      row.LastError,
		CreatedAt:      row.CreatedAt.Time,
	}
}

func pointerFromTimestamptz(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func pointerFromUUID(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}
	str := uuid.UUID(value.Bytes).String()
	return &str
}
