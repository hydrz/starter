package notification_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hydrz/starter/internal/delivery"
	"github.com/hydrz/starter/internal/notification"
)

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

type fakeChannel struct {
	name        string
	deliverFunc func(ctx context.Context, msg delivery.Message) (string, error)
	delivered   []delivery.Message
	mu          sync.Mutex
}

func newFakeChannel(name string) *fakeChannel {
	return &fakeChannel{name: name}
}

func (c *fakeChannel) Name() string {
	return c.name
}

func (c *fakeChannel) Deliver(ctx context.Context, msg delivery.Message) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.delivered = append(c.delivered, msg)
	if c.deliverFunc != nil {
		return c.deliverFunc(ctx, msg)
	}
	return "250 OK mock", nil
}

type fakeRepository struct {
	mu sync.Mutex

	intents   []notification.NotificationIntent
	messages  map[string]*notification.DeliveryMessage // by outboxEventID
	attempts  map[string]*notification.DeliveryAttempt // by ID
	processed map[string]string                        // eventID -> claimToken
	retried   map[string]retriedEvent                  // eventID -> details
	nextID    int
}

type retriedEvent struct {
	claimToken  string
	availableAt time.Time
	lastError   *string
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		messages:  make(map[string]*notification.DeliveryMessage),
		attempts:  make(map[string]*notification.DeliveryAttempt),
		processed: make(map[string]string),
		retried:   make(map[string]retriedEvent),
	}
}

func (r *fakeRepository) CreateNotificationIntent(ctx context.Context, recipientUserID *string, kind string, payload []byte) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	id := string(rune('a' + r.nextID))
	r.intents = append(r.intents, notification.NotificationIntent{
		ID:              id,
		RecipientUserID: recipientUserID,
		Kind:            kind,
		Payload:         payload,
		CreatedAt:       time.Now(),
	})
	return id, nil
}

func (r *fakeRepository) GetDeliveryMessageByOutboxEvent(ctx context.Context, outboxEventID string) (*notification.DeliveryMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	msg, ok := r.messages[outboxEventID]
	if !ok {
		return nil, notification.ErrNotFound
	}
	return msg, nil
}

func (r *fakeRepository) CreateDeliveryMessage(ctx context.Context, params notification.CreateDeliveryMessageParams) (*notification.DeliveryMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	id := string(rune('m' + r.nextID))
	msg := &notification.DeliveryMessage{
		ID:                   id,
		NotificationIntentID: params.NotificationIntentID,
		OutboxEventID:        &params.OutboxEventID,
		Channel:              params.Channel,
		Recipient:            params.Recipient,
		Subject:              params.Subject,
		TextBody:             params.TextBody,
		HTMLBody:             params.HTMLBody,
		IdempotencyKey:       params.IdempotencyKey,
		CreatedAt:            time.Now(),
	}
	r.messages[params.OutboxEventID] = msg
	return msg, nil
}

func (r *fakeRepository) CreateDeliveryAttempt(ctx context.Context, messageID string, attemptNumber int, status string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	id := string(rune('t' + r.nextID))
	r.attempts[id] = &notification.DeliveryAttempt{
		ID:                id,
		DeliveryMessageID: messageID,
		AttemptNumber:     attemptNumber,
		Status:            status,
		StartedAt:         time.Now(),
	}
	return id, nil
}

func (r *fakeRepository) CompleteDeliveryAttempt(ctx context.Context, attemptID string, status string, providerResponse, errorMessage *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	att, ok := r.attempts[attemptID]
	if !ok {
		return errors.New("attempt not found")
	}
	att.Status = status
	att.ProviderResponse = providerResponse
	att.ErrorMessage = errorMessage
	now := time.Now()
	att.CompletedAt = &now
	return nil
}

func (r *fakeRepository) MarkOutboxEventProcessed(ctx context.Context, outboxEventID, claimToken string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.processed[outboxEventID] = claimToken
	return nil
}

func (r *fakeRepository) RetryOutboxEvent(ctx context.Context, outboxEventID, claimToken string, availableAt time.Time, lastError *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.retried[outboxEventID] = retriedEvent{
		claimToken:  claimToken,
		availableAt: availableAt,
		lastError:   lastError,
	}
	return nil
}

func TestService_VerificationRequested_Success(t *testing.T) {
	repo := newFakeRepository()
	renderer, err := delivery.NewTemplateRenderer("Starter")
	if err != nil {
		t.Fatalf("NewTemplateRenderer: %v", err)
	}

	smtpChan := newFakeChannel("smtp")
	clock := &fakeClock{now: time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)}

	svc, err := notification.NewService(notification.Dependencies{
		Repository: repo,
		Renderer:   renderer,
		Channels:   map[string]delivery.Channel{"smtp": smtpChan},
		Clock:      clock,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	payload, _ := json.Marshal(notification.AuthPayload{
		UserID: "usr-123",
		Email:  "alice@example.com",
		Token:  "verify-token-abc",
	})
	claimToken := "claim-tok-1"

	event := delivery.OutboxEvent{
		ID:         "event-1",
		Topic:      notification.TopicAuthVerificationRequested,
		Payload:    payload,
		Attempts:   1,
		ClaimToken: &claimToken,
	}

	if err := svc.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	// 1. Check intent
	if len(repo.intents) != 1 {
		t.Fatalf("expected 1 intent, got %d", len(repo.intents))
	}
	if repo.intents[0].Kind != notification.KindEmailVerification {
		t.Errorf("intent kind = %s, want %s", repo.intents[0].Kind, notification.KindEmailVerification)
	}

	// 2. Check message
	msg := repo.messages["event-1"]
	if msg == nil {
		t.Fatal("expected delivery message for event-1")
	}
	if msg.Recipient != "alice@example.com" {
		t.Errorf("recipient = %s, want alice@example.com", msg.Recipient)
	}
	if !strings.Contains(msg.Subject, "Verify your email address") {
		t.Errorf("subject %q missing verification title", msg.Subject)
	}
	if !strings.Contains(msg.TextBody, "verify-token-abc") {
		t.Errorf("text body missing token: %s", msg.TextBody)
	}

	// 3. Check channel delivery
	if len(smtpChan.delivered) != 1 {
		t.Fatalf("expected 1 delivered message, got %d", len(smtpChan.delivered))
	}
	if smtpChan.delivered[0].Recipient != "alice@example.com" {
		t.Errorf("channel received recipient %s", smtpChan.delivered[0].Recipient)
	}

	// 4. Check attempt completed as sent
	if len(repo.attempts) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(repo.attempts))
	}
	for _, att := range repo.attempts {
		if att.Status != notification.StatusSent {
			t.Errorf("attempt status = %s, want %s", att.Status, notification.StatusSent)
		}
	}

	// 5. Check outbox event marked processed
	if repo.processed["event-1"] != claimToken {
		t.Errorf("expected outbox event processed with claimToken, got %s", repo.processed["event-1"])
	}
}

func TestService_PasswordResetRequested_Success(t *testing.T) {
	repo := newFakeRepository()
	renderer, _ := delivery.NewTemplateRenderer("Starter")
	smtpChan := newFakeChannel("smtp")
	clock := &fakeClock{now: time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)}

	svc, err := notification.NewService(notification.Dependencies{
		Repository: repo,
		Renderer:   renderer,
		Channels:   map[string]delivery.Channel{"smtp": smtpChan},
		Clock:      clock,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	payload, _ := json.Marshal(notification.AuthPayload{
		UserID: "usr-456",
		Email:  "bob@example.com",
		Token:  "reset-token-xyz",
	})
	claimToken := "claim-tok-2"

	event := delivery.OutboxEvent{
		ID:         "event-2",
		Topic:      notification.TopicAuthPasswordResetRequested,
		Payload:    payload,
		Attempts:   1,
		ClaimToken: &claimToken,
	}

	if err := svc.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	if len(repo.intents) != 1 || repo.intents[0].Kind != notification.KindPasswordReset {
		t.Errorf("intent not recorded as password_reset")
	}

	msg := repo.messages["event-2"]
	if msg == nil || !strings.Contains(msg.Subject, "Reset your password") {
		t.Errorf("expected reset password subject, got %v", msg)
	}
	if repo.processed["event-2"] != claimToken {
		t.Errorf("expected event-2 processed")
	}
}

func TestService_DeliveryFailure_RetriesWithBackoff(t *testing.T) {
	repo := newFakeRepository()
	renderer, _ := delivery.NewTemplateRenderer("Starter")
	smtpChan := newFakeChannel("smtp")
	deliveryErr := errors.New("smtp connection timed out")
	smtpChan.deliverFunc = func(ctx context.Context, msg delivery.Message) (string, error) {
		return "", deliveryErr
	}

	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	clock := &fakeClock{now: now}

	svc, err := notification.NewService(notification.Dependencies{
		Repository: repo,
		Renderer:   renderer,
		Channels:   map[string]delivery.Channel{"smtp": smtpChan},
		Clock:      clock,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	payload, _ := json.Marshal(notification.AuthPayload{
		UserID: "usr-fail",
		Email:  "fail@example.com",
		Token:  "fail-token",
	})
	claimToken := "claim-fail"

	event := delivery.OutboxEvent{
		ID:         "event-fail",
		Topic:      notification.TopicAuthVerificationRequested,
		Payload:    payload,
		Attempts:   2, // 2nd attempt
		ClaimToken: &claimToken,
	}

	err = svc.Handle(context.Background(), event)
	if err == nil {
		t.Fatal("expected Handle to return delivery error, got nil")
	}
	if !errors.Is(err, deliveryErr) {
		t.Errorf("expected error %v, got %v", deliveryErr, err)
	}

	// Attempt should be recorded as failed
	for _, att := range repo.attempts {
		if att.Status != notification.StatusFailed {
			t.Errorf("attempt status = %s, want %s", att.Status, notification.StatusFailed)
		}
		if att.ErrorMessage == nil || !strings.Contains(*att.ErrorMessage, "smtp connection timed out") {
			t.Errorf("unexpected error message: %v", att.ErrorMessage)
		}
	}

	// Outbox event should NOT be marked processed
	if _, ok := repo.processed["event-fail"]; ok {
		t.Error("failed event should not be marked processed")
	}

	// Outbox event should be marked for retry with backoff
	retry, ok := repo.retried["event-fail"]
	if !ok {
		t.Fatal("expected event-fail to be retried")
	}
	if retry.claimToken != claimToken {
		t.Errorf("retried claimToken = %s, want %s", retry.claimToken, claimToken)
	}

	// Backoff for attempt 2: 5s * 3^1 = 15s
	expectedAvailableAt := now.Add(15 * time.Second)
	if !retry.availableAt.Equal(expectedAvailableAt) {
		t.Errorf("availableAt = %v, want %v", retry.availableAt, expectedAvailableAt)
	}
}

func TestService_Idempotency_ReusesExistingDeliveryMessage(t *testing.T) {
	repo := newFakeRepository()
	renderer, _ := delivery.NewTemplateRenderer("Starter")
	smtpChan := newFakeChannel("smtp")
	clock := &fakeClock{now: time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)}

	svc, err := notification.NewService(notification.Dependencies{
		Repository: repo,
		Renderer:   renderer,
		Channels:   map[string]delivery.Channel{"smtp": smtpChan},
		Clock:      clock,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	// Pre-create delivery message (as if attempt 1 created it, then crashed)
	preExistingMsg, _ := repo.CreateDeliveryMessage(context.Background(), notification.CreateDeliveryMessageParams{
		OutboxEventID:  "event-existing",
		Channel:        "smtp",
		Recipient:      "existing@example.com",
		Subject:        "Pre-existing Subject",
		TextBody:       "Pre-existing Text",
		HTMLBody:       "Pre-existing HTML",
		IdempotencyKey: "outbox:event-existing:smtp",
	})

	payload, _ := json.Marshal(notification.AuthPayload{
		UserID: "usr-existing",
		Email:  "existing@example.com",
		Token:  "token",
	})
	claimToken := "claim-existing"

	event := delivery.OutboxEvent{
		ID:         "event-existing",
		Topic:      notification.TopicAuthVerificationRequested,
		Payload:    payload,
		Attempts:   2, // attempt 2
		ClaimToken: &claimToken,
	}

	if err := svc.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	// Check that the delivered message reused the pre-existing message ID
	if len(smtpChan.delivered) != 1 {
		t.Fatalf("expected 1 delivered message, got %d", len(smtpChan.delivered))
	}
	if smtpChan.delivered[0].ID != preExistingMsg.ID {
		t.Errorf("delivered message ID = %s, want pre-existing ID %s", smtpChan.delivered[0].ID, preExistingMsg.ID)
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{attempt: 0, expected: 5 * time.Second},
		{attempt: 1, expected: 5 * time.Second},
		{attempt: 2, expected: 15 * time.Second},
		{attempt: 3, expected: 45 * time.Second},
		{attempt: 4, expected: 135 * time.Second},
		{attempt: 5, expected: 405 * time.Second},
		{attempt: 10, expected: 1 * time.Hour}, // capped at 1 hour
		{attempt: 20, expected: 1 * time.Hour},
	}

	for _, tc := range tests {
		got := notification.CalculateBackoff(tc.attempt)
		if got != tc.expected {
			t.Errorf("CalculateBackoff(%d) = %v, want %v", tc.attempt, got, tc.expected)
		}
	}
}

func TestService_UnknownTopic(t *testing.T) {
	repo := newFakeRepository()
	renderer, _ := delivery.NewTemplateRenderer("Starter")
	svc, _ := notification.NewService(notification.Dependencies{
		Repository: repo,
		Renderer:   renderer,
	})

	event := delivery.OutboxEvent{
		ID:    "event-unknown",
		Topic: "some.random.topic",
	}

	err := svc.Handle(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for unknown topic, got nil")
	}
	if !errors.Is(err, notification.ErrUnknownTopic) {
		t.Errorf("expected ErrUnknownTopic, got %v", err)
	}
}

func TestService_MalformedPayload(t *testing.T) {
	repo := newFakeRepository()
	renderer, _ := delivery.NewTemplateRenderer("Starter")
	smtpChan := newFakeChannel("smtp")
	svc, _ := notification.NewService(notification.Dependencies{
		Repository: repo,
		Renderer:   renderer,
		Channels:   map[string]delivery.Channel{"smtp": smtpChan},
	})

	event := delivery.OutboxEvent{
		ID:      "event-malformed",
		Topic:   notification.TopicAuthVerificationRequested,
		Payload: []byte("not valid json"),
	}

	err := svc.Handle(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for malformed payload, got nil")
	}
}

func TestService_MissingSMTPChannel(t *testing.T) {
	repo := newFakeRepository()
	renderer, _ := delivery.NewTemplateRenderer("Starter")
	svc, _ := notification.NewService(notification.Dependencies{
		Repository: repo,
		Renderer:   renderer,
		Channels:   map[string]delivery.Channel{}, // no smtp channel
	})

	payload, _ := json.Marshal(notification.AuthPayload{
		UserID: "usr-1",
		Email:  "test@example.com",
		Token:  "tok",
	})
	event := delivery.OutboxEvent{
		ID:      "event-no-chan",
		Topic:   notification.TopicAuthVerificationRequested,
		Payload: payload,
	}

	err := svc.Handle(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for missing channel, got nil")
	}
	if !errors.Is(err, notification.ErrChannelMissing) {
		t.Errorf("expected ErrChannelMissing, got %v", err)
	}
}
