package notification

import (
	"context"
	"time"
)

// CreateDeliveryMessageParams contains parameters to persist a new DeliveryMessage.
type CreateDeliveryMessageParams struct {
	NotificationIntentID *string
	OutboxEventID        string
	Channel              string
	Recipient            string
	Subject              string
	TextBody             string
	HTMLBody             string
	IdempotencyKey       string
}

// Repository abstracts database operations for notification intents, delivery messages,
// delivery attempts, and outbox event state transitions.
type Repository interface {
	CreateNotificationIntent(ctx context.Context, recipientUserID *string, kind string, payload []byte) (string, error)
	GetDeliveryMessageByOutboxEvent(ctx context.Context, outboxEventID string) (*DeliveryMessage, error)
	CreateDeliveryMessage(ctx context.Context, params CreateDeliveryMessageParams) (*DeliveryMessage, error)
	CreateDeliveryAttempt(ctx context.Context, messageID string, attemptNumber int, status string) (string, error)
	CompleteDeliveryAttempt(ctx context.Context, attemptID string, status string, providerResponse, errorMessage *string) error
	MarkOutboxEventProcessed(ctx context.Context, outboxEventID, claimToken string) error
	RetryOutboxEvent(ctx context.Context, outboxEventID, claimToken string, availableAt time.Time, lastError *string) error
}

// Clock provides the current time, enabling deterministic testing.
type Clock interface {
	Now() time.Time
}

// SystemClock implements Clock using the real system clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}
