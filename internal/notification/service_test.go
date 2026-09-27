package notification_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hydrz/starter/internal/delivery"
	"github.com/hydrz/starter/internal/notification"
)

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

	intents  []notification.NotificationIntent
	messages map[string]*notification.DeliveryMessage // by outboxEventID
	attempts map[string]*notification.DeliveryAttempt // by ID
	nextID   int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		messages: make(map[string]*notification.DeliveryMessage),
		attempts: make(map[string]*notification.DeliveryAttempt),
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

func TestService_VerificationRequested_Success(t *testing.T) {
	repo := newFakeRepository()
	renderer, err := delivery.NewTemplateRenderer("Starter")
	if err != nil {
		t.Fatalf("NewTemplateRenderer: %v", err)
	}

	smtpChan := newFakeChannel("smtp")

	svc, err := notification.NewService(notification.Dependencies{
		Repository: repo,
		Renderer:   renderer,
		Channels:   map[string]delivery.Channel{"smtp": smtpChan},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	payload, _ := json.Marshal(notification.AuthPayload{
		UserID: "usr-123",
		Email:  "alice@example.com",
		Token:  "verify-token-abc",
	})

	event := delivery.OutboxEvent{
		ID:       "event-1",
		Topic:    notification.TopicAuthVerificationRequested,
		Payload:  payload,
		Attempts: 1,
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
}

func TestService_EmailOTPRequested_SendsSMTPMessage(t *testing.T) {
	repo := newFakeRepository()
	renderer, err := delivery.NewTemplateRenderer("Starter")
	if err != nil {
		t.Fatalf("NewTemplateRenderer: %v", err)
	}
	smtpChan := newFakeChannel("smtp")

	svc, err := notification.NewService(notification.Dependencies{
		Repository: repo,
		Renderer:   renderer,
		Channels:   map[string]delivery.Channel{"smtp": smtpChan},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	payload, err := json.Marshal(struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Code   string `json:"code"`
	}{
		UserID: "usr-otp",
		Email:  "otp@example.com",
		Code:   "123456",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	event := delivery.OutboxEvent{
		ID:       "event-otp",
		Topic:    notification.TopicAuthEmailOTPRequested,
		Payload:  payload,
		Attempts: 1,
	}

	if err := svc.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if len(repo.intents) != 1 {
		t.Fatalf("expected one notification intent, got %d", len(repo.intents))
	}
	if got, want := repo.intents[0].Kind, notification.KindEmailOTP; got != want {
		t.Errorf("intent kind = %q, want %q", got, want)
	}
	if len(smtpChan.delivered) != 1 {
		t.Fatalf("expected one SMTP delivery, got %d", len(smtpChan.delivered))
	}
	message := smtpChan.delivered[0]
	if got, want := message.Recipient, "otp@example.com"; got != want {
		t.Errorf("recipient = %q, want %q", got, want)
	}
	if !strings.Contains(message.Subject, "sign-in code") {
		t.Errorf("email subject missing OTP purpose: %q", message.Subject)
	}
	for _, body := range []string{message.TextBody, message.HTMLBody} {
		if !strings.Contains(body, "123456") {
			t.Errorf("email body missing OTP code: %q", body)
		}
		if !strings.Contains(body, "10 minutes") {
			t.Errorf("email body missing OTP validity period: %q", body)
		}
	}
	if len(smtpChan.delivered) == 0 {
		t.Error("expected a delivered message")
	}
}

func TestService_HandlesEveryAuthOutboxTopic(t *testing.T) {
	authDir := filepath.Join(repositoryRoot(t), "internal", "auth")
	topicPattern := regexp.MustCompile(`outboxTopic\w+\s*=\s*"([^"]+)"`)

	entries, err := os.ReadDir(authDir)
	if err != nil {
		t.Fatalf("read auth directory: %v", err)
	}

	topics := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(authDir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		for _, match := range topicPattern.FindAllStringSubmatch(string(source), -1) {
			topics[match[1]] = entry.Name()
		}
	}

	for topic, sourceFile := range topics {
		t.Run(topic, func(t *testing.T) {
			repo := newFakeRepository()
			svc, err := notification.NewService(notification.Dependencies{
				Repository: repo,
				Renderer:   mustNewTemplateRenderer(t),
				Channels:   map[string]delivery.Channel{"smtp": newFakeChannel("smtp")},
			})
			if err != nil {
				t.Fatalf("NewService: %v", err)
			}

			payload := []byte(`{"user_id":"usr-topic","email":"topic@example.com","token":"token","code":"123456"}`)
			err = svc.Handle(context.Background(), delivery.OutboxEvent{ID: "event-" + topic, Topic: topic, Payload: payload})
			if err != nil {
				t.Errorf("notification cannot handle auth outbox topic %q from %s: %v", topic, sourceFile, err)
			}
		})
	}
}

func mustNewTemplateRenderer(t *testing.T) *delivery.TemplateRenderer {
	t.Helper()
	renderer, err := delivery.NewTemplateRenderer("Starter")
	if err != nil {
		t.Fatalf("NewTemplateRenderer: %v", err)
	}
	return renderer
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("determine test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestService_PasswordResetRequested_Success(t *testing.T) {
	repo := newFakeRepository()
	renderer, _ := delivery.NewTemplateRenderer("Starter")
	smtpChan := newFakeChannel("smtp")

	svc, err := notification.NewService(notification.Dependencies{
		Repository: repo,
		Renderer:   renderer,
		Channels:   map[string]delivery.Channel{"smtp": smtpChan},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	payload, _ := json.Marshal(notification.AuthPayload{
		UserID: "usr-456",
		Email:  "bob@example.com",
		Token:  "reset-token-xyz",
	})

	event := delivery.OutboxEvent{
		ID:       "event-2",
		Topic:    notification.TopicAuthPasswordResetRequested,
		Payload:  payload,
		Attempts: 1,
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
}

func TestService_DeliveryFailure_RetriesWithBackoff(t *testing.T) {
	repo := newFakeRepository()
	renderer, _ := delivery.NewTemplateRenderer("Starter")
	smtpChan := newFakeChannel("smtp")
	deliveryErr := errors.New("smtp connection timed out")
	smtpChan.deliverFunc = func(ctx context.Context, msg delivery.Message) (string, error) {
		return "", deliveryErr
	}

	svc, err := notification.NewService(notification.Dependencies{
		Repository: repo,
		Renderer:   renderer,
		Channels:   map[string]delivery.Channel{"smtp": smtpChan},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	payload, _ := json.Marshal(notification.AuthPayload{
		UserID: "usr-fail",
		Email:  "fail@example.com",
		Token:  "fail-token",
	})

	event := delivery.OutboxEvent{
		ID:       "event-fail",
		Topic:    notification.TopicAuthVerificationRequested,
		Payload:  payload,
		Attempts: 2, // 2nd attempt
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
}

func TestService_Idempotency_ReusesExistingDeliveryMessage(t *testing.T) {
	repo := newFakeRepository()
	renderer, _ := delivery.NewTemplateRenderer("Starter")
	smtpChan := newFakeChannel("smtp")

	svc, err := notification.NewService(notification.Dependencies{
		Repository: repo,
		Renderer:   renderer,
		Channels:   map[string]delivery.Channel{"smtp": smtpChan},
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
	event := delivery.OutboxEvent{
		ID:       "event-existing",
		Topic:    notification.TopicAuthVerificationRequested,
		Payload:  payload,
		Attempts: 2, // attempt 2
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
