//go:build integration

package notification_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/testdb"
	"github.com/hydrz/starter/internal/store"
)

func TestNotificationIntentAndDeliveryQueries(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := testdb.Context(t)

	intent, err := db.Queries.CreateNotificationIntent(ctx, store.CreateNotificationIntentParams{
		Kind:    "email_verification",
		Payload: []byte(`{"email":"user@example.com"}`),
	})
	if err != nil {
		t.Fatalf("create notification intent: %v", err)
	}
	event, err := db.Queries.CreateOutboxEvent(ctx, store.CreateOutboxEventParams{
		Topic: "notification.created", AggregateType: "notification", AggregateID: intent.ID,
		Payload: []byte(`{}`), IdempotencyKey: uuid.NewString(), AvailableAt: pgtype.Timestamptz{Time: testdb.Now().Add(-time.Minute), Valid: true},
	})
	if err != nil {
		t.Fatalf("create outbox event: %v", err)
	}
	message, err := db.Queries.CreateDeliveryMessage(ctx, store.CreateDeliveryMessageParams{
		OutboxEventID: event.ID, Channel: "smtp", Recipient: "user@example.com",
		Subject: "Verify your email", TextBody: "text", HtmlBody: "<p>html</p>", IdempotencyKey: uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("create delivery message: %v", err)
	}
	attempt, err := db.Queries.CreateDeliveryAttempt(ctx, store.CreateDeliveryAttemptParams{
		DeliveryMessageID: message.ID, AttemptNumber: 1, Status: "claimed",
	})
	if err != nil {
		t.Fatalf("create delivery attempt: %v", err)
	}
	if completed, err := db.Queries.CompleteDeliveryAttempt(ctx, store.CompleteDeliveryAttemptParams{ID: attempt.ID, Status: "sent"}); err != nil || completed != 1 {
		t.Fatalf("complete delivery attempt = %d, %v", completed, err)
	}
	if fetched, err := db.Queries.GetDeliveryMessageByOutboxEvent(ctx, event.ID); err != nil || fetched.ID != message.ID {
		t.Fatalf("get delivery message by outbox event = %#v, %v", fetched, err)
	}
	if attempts, err := db.Queries.ListDeliveryAttemptsByMessage(ctx, message.ID); err != nil || len(attempts) != 1 {
		t.Fatalf("list delivery attempts = %d items, %v", len(attempts), err)
	}
	if fetched, err := db.Queries.GetOutboxEventByID(ctx, event.ID); err != nil || fetched.ProcessedAt.Valid {
		t.Fatalf("get outbox event = %#v, %v", fetched, err)
	}
	claimToken := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	claimed, err := db.Queries.ClaimOutboxEvents(ctx, store.ClaimOutboxEventsParams{
		ClaimToken:   claimToken,
		ClaimTimeout: pgtype.Interval{Microseconds: int64(time.Minute / time.Microsecond), Valid: true},
		BatchSize:    1,
	})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim outbox events = %d items, %v", len(claimed), err)
	}
	if processed, err := db.Queries.MarkOutboxEventProcessed(ctx, store.MarkOutboxEventProcessedParams{
		ID: event.ID, ClaimToken: claimToken,
	}); err != nil || processed != 1 {
		t.Fatalf("mark outbox event processed = %d, %v", processed, err)
	}
}
