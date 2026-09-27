package notification

import "context"

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

// Repository abstracts persistence for notification intents, delivery messages,
// and delivery attempts. Outbox state transitions belong to delivery.Worker.
type Repository interface {
	CreateNotificationIntent(ctx context.Context, recipientUserID *string, kind string, payload []byte) (string, error)
	GetDeliveryMessageByOutboxEvent(ctx context.Context, outboxEventID string) (*DeliveryMessage, error)
	CreateDeliveryMessage(ctx context.Context, params CreateDeliveryMessageParams) (*DeliveryMessage, error)
	CreateDeliveryAttempt(ctx context.Context, messageID string, attemptNumber int, status string) (string, error)
	CompleteDeliveryAttempt(ctx context.Context, attemptID string, status string, providerResponse, errorMessage *string) error
}
