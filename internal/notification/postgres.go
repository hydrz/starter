package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/store"
)

// PostgresRepository implements Repository using generated sqlc queries.
type PostgresRepository struct {
	queries *store.Queries
}

// NewPostgresRepository creates a PostgresRepository backed by store.Queries.
func NewPostgresRepository(queries *store.Queries) *PostgresRepository {
	return &PostgresRepository{queries: queries}
}

func (r *PostgresRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *PostgresRepository) CreateNotificationIntent(ctx context.Context, recipientUserID *string, kind string, payload []byte) (string, error) {
	var userUUID pgtype.UUID
	if recipientUserID != nil && *recipientUserID != "" {
		parsed, err := uuid.Parse(*recipientUserID)
		if err == nil {
			userUUID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	intent, err := r.q(ctx).CreateNotificationIntent(ctx, store.CreateNotificationIntentParams{
		RecipientUserID: userUUID,
		Kind:            kind,
		Payload:         payload,
	})
	if err != nil {
		return "", fmt.Errorf("create notification intent: %w", err)
	}
	return uuid.UUID(intent.ID.Bytes).String(), nil
}

func (r *PostgresRepository) GetDeliveryMessageByOutboxEvent(ctx context.Context, outboxEventID string) (*DeliveryMessage, error) {
	eventUUID, err := uuid.Parse(outboxEventID)
	if err != nil {
		return nil, fmt.Errorf("invalid outbox event uuid %q: %w", outboxEventID, err)
	}

	row, err := r.q(ctx).GetDeliveryMessageByOutboxEvent(ctx, pgtype.UUID{Bytes: eventUUID, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get delivery message by outbox event: %w", err)
	}

	msg := mapStoreDeliveryMessage(row)
	return &msg, nil
}

func (r *PostgresRepository) CreateDeliveryMessage(ctx context.Context, params CreateDeliveryMessageParams) (*DeliveryMessage, error) {
	eventUUID, err := uuid.Parse(params.OutboxEventID)
	if err != nil {
		return nil, fmt.Errorf("invalid outbox event id %q: %w", params.OutboxEventID, err)
	}

	var intentUUID pgtype.UUID
	if params.NotificationIntentID != nil && *params.NotificationIntentID != "" {
		parsed, err := uuid.Parse(*params.NotificationIntentID)
		if err == nil {
			intentUUID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	row, err := r.q(ctx).CreateDeliveryMessage(ctx, store.CreateDeliveryMessageParams{
		NotificationIntentID: intentUUID,
		OutboxEventID:        pgtype.UUID{Bytes: eventUUID, Valid: true},
		Channel:              params.Channel,
		Recipient:            params.Recipient,
		Subject:              params.Subject,
		TextBody:             params.TextBody,
		HtmlBody:             params.HTMLBody,
		IdempotencyKey:       params.IdempotencyKey,
	})
	if err != nil {
		return nil, fmt.Errorf("create delivery message: %w", err)
	}

	msg := mapStoreDeliveryMessage(row)
	return &msg, nil
}

func (r *PostgresRepository) CreateDeliveryAttempt(ctx context.Context, messageID string, attemptNumber int, status string) (string, error) {
	msgUUID, err := uuid.Parse(messageID)
	if err != nil {
		return "", fmt.Errorf("invalid message uuid %q: %w", messageID, err)
	}

	row, err := r.q(ctx).CreateDeliveryAttempt(ctx, store.CreateDeliveryAttemptParams{
		DeliveryMessageID: pgtype.UUID{Bytes: msgUUID, Valid: true},
		AttemptNumber:     int32(attemptNumber),
		Status:            status,
	})
	if err != nil {
		return "", fmt.Errorf("create delivery attempt: %w", err)
	}
	return uuid.UUID(row.ID.Bytes).String(), nil
}

func (r *PostgresRepository) CompleteDeliveryAttempt(ctx context.Context, attemptID string, status string, providerResponse, errorMessage *string) error {
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return fmt.Errorf("invalid attempt uuid %q: %w", attemptID, err)
	}

	_, err = r.q(ctx).CompleteDeliveryAttempt(ctx, store.CompleteDeliveryAttemptParams{
		ID:               pgtype.UUID{Bytes: attemptUUID, Valid: true},
		Status:           status,
		ProviderResponse: providerResponse,
		ErrorMessage:     errorMessage,
	})
	if err != nil {
		return fmt.Errorf("complete delivery attempt: %w", err)
	}
	return nil
}

func (r *PostgresRepository) MarkOutboxEventProcessed(ctx context.Context, outboxEventID, claimToken string) error {
	eventUUID, err := uuid.Parse(outboxEventID)
	if err != nil {
		return fmt.Errorf("invalid outbox event uuid %q: %w", outboxEventID, err)
	}
	tokenUUID, err := uuid.Parse(claimToken)
	if err != nil {
		return fmt.Errorf("invalid claim token uuid %q: %w", claimToken, err)
	}

	_, err = r.q(ctx).MarkOutboxEventProcessed(ctx, store.MarkOutboxEventProcessedParams{
		ID:         pgtype.UUID{Bytes: eventUUID, Valid: true},
		ClaimToken: pgtype.UUID{Bytes: tokenUUID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("mark outbox event processed: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RetryOutboxEvent(ctx context.Context, outboxEventID, claimToken string, availableAt time.Time, lastError *string) error {
	eventUUID, err := uuid.Parse(outboxEventID)
	if err != nil {
		return fmt.Errorf("invalid outbox event uuid %q: %w", outboxEventID, err)
	}
	tokenUUID, err := uuid.Parse(claimToken)
	if err != nil {
		return fmt.Errorf("invalid claim token uuid %q: %w", claimToken, err)
	}

	_, err = r.q(ctx).RetryOutboxEvent(ctx, store.RetryOutboxEventParams{
		ID:          pgtype.UUID{Bytes: eventUUID, Valid: true},
		ClaimToken:  pgtype.UUID{Bytes: tokenUUID, Valid: true},
		AvailableAt: pgtype.Timestamptz{Time: availableAt, Valid: true},
		LastError:   lastError,
	})
	if err != nil {
		return fmt.Errorf("retry outbox event: %w", err)
	}
	return nil
}

func mapStoreDeliveryMessage(row store.DeliveryMessage) DeliveryMessage {
	return DeliveryMessage{
		ID:                   uuid.UUID(row.ID.Bytes).String(),
		NotificationIntentID: pointerFromUUID(row.NotificationIntentID),
		OutboxEventID:        pointerFromUUID(row.OutboxEventID),
		Channel:              row.Channel,
		Recipient:            row.Recipient,
		Subject:              row.Subject,
		TextBody:             row.TextBody,
		HTMLBody:             row.HtmlBody,
		IdempotencyKey:       row.IdempotencyKey,
		CreatedAt:            row.CreatedAt.Time,
	}
}

func pointerFromUUID(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}
	str := uuid.UUID(value.Bytes).String()
	return &str
}
