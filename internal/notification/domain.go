package notification

import (
	"errors"
	"time"
)

var (
	ErrNotFound       = errors.New("notification: not found")
	ErrUnknownTopic   = errors.New("notification: unknown topic")
	ErrChannelMissing = errors.New("notification: delivery channel missing")
)

const (
	KindEmailVerification = "email_verification"
	KindPasswordReset     = "password_reset"

	TopicAuthVerificationRequested  = "auth.verification_requested"
	TopicAuthPasswordResetRequested = "auth.password_reset_requested"

	ChannelSMTP = "smtp"

	StatusClaimed = "claimed"
	StatusSent    = "sent"
	StatusFailed  = "failed"
)

// NotificationIntent represents an intent to notify a recipient.
type NotificationIntent struct {
	ID              string
	RecipientUserID *string
	Kind            string
	Payload         []byte
	CreatedAt       time.Time
}

// DeliveryMessage represents a formatted message queued for delivery on a specific channel.
type DeliveryMessage struct {
	ID                   string
	NotificationIntentID *string
	OutboxEventID        *string
	Channel              string
	Recipient            string
	Subject              string
	TextBody             string
	HTMLBody             string
	IdempotencyKey       string
	CreatedAt            time.Time
}

// DeliveryAttempt records an individual delivery execution attempt.
type DeliveryAttempt struct {
	ID                string
	DeliveryMessageID string
	AttemptNumber     int
	Status            string
	ProviderResponse  *string
	ErrorMessage      *string
	StartedAt         time.Time
	CompletedAt       *time.Time
}

// AuthPayload matches the JSON structure emitted by internal/auth for verification and reset.
type AuthPayload struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Token  string `json:"token"`
}
