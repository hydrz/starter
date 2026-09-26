package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/hydrz/starter/internal/delivery"
)

// Dependencies holds components required by the notification Service.
type Dependencies struct {
	Repository Repository
	Renderer   *delivery.TemplateRenderer
	Channels   map[string]delivery.Channel
	Clock      Clock
	Logger     *slog.Logger
}

// Service routes outbox events to appropriate notification channels and tracks attempts.
type Service struct {
	repo     Repository
	renderer *delivery.TemplateRenderer
	channels map[string]delivery.Channel
	clock    Clock
	logger   *slog.Logger
}

// NewService constructs a notification Service.
func NewService(deps Dependencies) (*Service, error) {
	if deps.Repository == nil {
		return nil, errors.New("notification: repository is required")
	}
	if deps.Renderer == nil {
		return nil, errors.New("notification: template renderer is required")
	}
	if deps.Channels == nil {
		deps.Channels = make(map[string]delivery.Channel)
	}
	if deps.Clock == nil {
		deps.Clock = SystemClock{}
	}
	if deps.Logger == nil {
		deps.Logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	return &Service{
		repo:     deps.Repository,
		renderer: deps.Renderer,
		channels: deps.Channels,
		clock:    deps.Clock,
		logger:   deps.Logger,
	}, nil
}

// Handle processes an outbox event, fulfilling the delivery.Handler interface.
func (s *Service) Handle(ctx context.Context, event delivery.OutboxEvent) error {
	switch event.Topic {
	case TopicAuthVerificationRequested, TopicAuthPasswordResetRequested:
		return s.handleAuthEvent(ctx, event)
	default:
		s.logger.Warn("unsupported outbox event topic", "topic", event.Topic, "event_id", event.ID)
		return fmt.Errorf("%w: %s", ErrUnknownTopic, event.Topic)
	}
}

func (s *Service) handleAuthEvent(ctx context.Context, event delivery.OutboxEvent) error {
	var payload AuthPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	smtpChan, ok := s.channels[ChannelSMTP]
	if !ok || smtpChan == nil {
		return fmt.Errorf("%w: %s", ErrChannelMissing, ChannelSMTP)
	}

	// Determine kind
	var kind string
	if event.Topic == TopicAuthVerificationRequested {
		kind = KindEmailVerification
	} else {
		kind = KindPasswordReset
	}

	// 1. Persist notification intent
	var recipientUserID *string
	if payload.UserID != "" {
		recipientUserID = &payload.UserID
	}
	intentID, err := s.repo.CreateNotificationIntent(ctx, recipientUserID, kind, event.Payload)
	if err != nil {
		s.logger.Warn("failed to create notification intent", "error", err, "event_id", event.ID)
	}

	// 2. Render template
	rendered, err := s.renderer.Render(event.Topic, delivery.EmailTemplateData{
		Email: payload.Email,
		Token: payload.Token,
	})
	if err != nil {
		return fmt.Errorf("render email template: %w", err)
	}

	// 3. Find or create delivery message
	deliveryMsg, err := s.repo.GetDeliveryMessageByOutboxEvent(ctx, event.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("get delivery message: %w", err)
	}
	if deliveryMsg == nil {
		var intentRef *string
		if intentID != "" {
			intentRef = &intentID
		}
		deliveryMsg, err = s.repo.CreateDeliveryMessage(ctx, CreateDeliveryMessageParams{
			NotificationIntentID: intentRef,
			OutboxEventID:        event.ID,
			Channel:              ChannelSMTP,
			Recipient:            payload.Email,
			Subject:              rendered.Subject,
			TextBody:             rendered.TextBody,
			HTMLBody:             rendered.HTMLBody,
			IdempotencyKey:       fmt.Sprintf("outbox:%s:smtp", event.ID),
		})
		if err != nil {
			return fmt.Errorf("create delivery message: %w", err)
		}
	}

	// 4. Record delivery attempt in 'claimed' status
	attemptNumber := event.Attempts
	if attemptNumber <= 0 {
		attemptNumber = 1
	}
	attemptID, err := s.repo.CreateDeliveryAttempt(ctx, deliveryMsg.ID, attemptNumber, StatusClaimed)
	if err != nil {
		return fmt.Errorf("create delivery attempt: %w", err)
	}

	// 5. Dispatch via Channel
	msg := delivery.Message{
		ID:             deliveryMsg.ID,
		Recipient:      deliveryMsg.Recipient,
		Subject:        deliveryMsg.Subject,
		TextBody:       deliveryMsg.TextBody,
		HTMLBody:       deliveryMsg.HTMLBody,
		IdempotencyKey: deliveryMsg.IdempotencyKey,
	}

	claimToken := ""
	if event.ClaimToken != nil {
		claimToken = *event.ClaimToken
	}

	resp, deliverErr := smtpChan.Deliver(ctx, msg)
	if deliverErr != nil {
		errMsg := deliverErr.Error()
		if err := s.repo.CompleteDeliveryAttempt(ctx, attemptID, StatusFailed, nil, &errMsg); err != nil {
			s.logger.Error("failed to complete failed delivery attempt", "error", err)
		}

		backoff := CalculateBackoff(event.Attempts)
		nextAvailableAt := s.clock.Now().UTC().Add(backoff)
		if err := s.repo.RetryOutboxEvent(ctx, event.ID, claimToken, nextAvailableAt, &errMsg); err != nil {
			return fmt.Errorf("retry outbox event: %w", err)
		}
		return deliverErr
	}

	// Success: record attempt as 'sent' and mark outbox event processed
	if err := s.repo.CompleteDeliveryAttempt(ctx, attemptID, StatusSent, &resp, nil); err != nil {
		s.logger.Error("failed to complete sent delivery attempt", "error", err)
	}

	if err := s.repo.MarkOutboxEventProcessed(ctx, event.ID, claimToken); err != nil {
		return fmt.Errorf("mark outbox event processed: %w", err)
	}

	return nil
}

// CalculateBackoff computes exponential retry delay based on the attempt count.
func CalculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	base := 5 * time.Second
	multiplier := 3
	delay := base
	for i := 1; i < attempt; i++ {
		delay *= time.Duration(multiplier)
		if delay > time.Hour {
			return time.Hour
		}
	}
	return delay
}
