package delivery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/store"
)

// ErrClaimLost indicates that an event is no longer owned by the worker claim.
var ErrClaimLost = errors.New("delivery: outbox event claim lost")

// PostgresStore persists worker claims and outbox state transitions.
type PostgresStore struct {
	queries *store.Queries
}

// NewPostgresStore creates a PostgresStore backed by the given queries.
func NewPostgresStore(queries *store.Queries) *PostgresStore {
	return &PostgresStore{queries: queries}
}

func (s *PostgresStore) q(ctx context.Context) *store.Queries {
	return database.Queries(ctx, s.queries)
}

// ClaimDueEvents claims a batch of due events with claimToken and claimTimeout.
func (s *PostgresStore) ClaimDueEvents(ctx context.Context, claimToken string, claimTimeout time.Duration, batchSize int) ([]OutboxEvent, error) {
	token, err := parseUUID(claimToken, "claim token")
	if err != nil {
		return nil, err
	}

	rows, err := s.q(ctx).ClaimOutboxEvents(ctx, store.ClaimOutboxEventsParams{
		ClaimToken:   token,
		ClaimTimeout: pgtype.Interval{Microseconds: claimTimeout.Microseconds(), Valid: true},
		BatchSize:    int32(batchSize),
	})
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}

	events := make([]OutboxEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, mapStoreOutboxEvent(store.OutboxEvent{
			ID:             row.ID,
			Topic:          row.Topic,
			AggregateType:  row.AggregateType,
			AggregateID:    row.AggregateID,
			Payload:        row.Payload,
			IdempotencyKey: row.IdempotencyKey,
			AvailableAt:    row.AvailableAt,
			ClaimedAt:      row.ClaimedAt,
			ClaimToken:     row.ClaimToken,
			Attempts:       row.Attempts,
			ProcessedAt:    row.ProcessedAt,
			DeadLetteredAt: row.DeadLetteredAt,
			LastError:      row.LastError,
			CreatedAt:      row.CreatedAt,
		}))
	}
	return events, nil
}

// MarkProcessed records successful handling while the caller still owns the claim.
func (s *PostgresStore) MarkProcessed(ctx context.Context, eventID, claimToken string) error {
	id, token, err := parseTransitionIDs(eventID, claimToken)
	if err != nil {
		return err
	}
	updated, err := s.q(ctx).MarkOutboxEventProcessed(ctx, store.MarkOutboxEventProcessedParams{ID: id, ClaimToken: token})
	return transitionResult("mark processed", eventID, updated, err)
}

// Retry releases a failed event until availableAt while the caller owns the claim.
func (s *PostgresStore) Retry(ctx context.Context, eventID, claimToken string, availableAt time.Time, lastError string) error {
	id, token, err := parseTransitionIDs(eventID, claimToken)
	if err != nil {
		return err
	}
	updated, err := s.q(ctx).RetryOutboxEvent(ctx, store.RetryOutboxEventParams{
		ID:          id,
		ClaimToken:  token,
		AvailableAt: pgtype.Timestamptz{Time: availableAt, Valid: true},
		LastError:   &lastError,
	})
	return transitionResult("retry", eventID, updated, err)
}

// DeadLetter moves a permanently failed event to the dead-letter terminal state.
func (s *PostgresStore) DeadLetter(ctx context.Context, eventID, claimToken, lastError string) error {
	id, token, err := parseTransitionIDs(eventID, claimToken)
	if err != nil {
		return err
	}
	updated, err := s.q(ctx).MarkOutboxEventDeadLettered(ctx, store.MarkOutboxEventDeadLetteredParams{
		ID:         id,
		ClaimToken: token,
		LastError:  &lastError,
	})
	return transitionResult("dead letter", eventID, updated, err)
}

func parseTransitionIDs(eventID, claimToken string) (pgtype.UUID, pgtype.UUID, error) {
	id, err := parseUUID(eventID, "outbox event")
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	token, err := parseUUID(claimToken, "claim token")
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	return id, token, nil
}

func parseUUID(value, field string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid %s uuid %q: %w", field, value, err)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func transitionResult(operation, eventID string, updated int64, err error) error {
	if err != nil {
		return fmt.Errorf("%s outbox event: %w", operation, err)
	}
	if updated == 0 {
		return fmt.Errorf("%w: %s", ErrClaimLost, eventID)
	}
	return nil
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
		DeadLetteredAt: pointerFromTimestamptz(row.DeadLetteredAt),
		LastError:      row.LastError,
		CreatedAt:      row.CreatedAt.Time,
	}
}

func pointerFromTimestamptz(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func pointerFromUUID(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}
	result := uuid.UUID(value.Bytes).String()
	return &result
}
