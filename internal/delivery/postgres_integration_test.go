//go:build integration

package delivery_test

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/testdb"
	"github.com/hydrz/starter/internal/store"
)

func TestClaimOutboxEventsDoesNotDoubleClaimOrClaimFutureEvents(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := testdb.Context(t)

	available := make([]store.OutboxEvent, 2)
	for i := range available {
		event, err := db.Queries.CreateOutboxEvent(ctx, outboxEventParams(t, i, time.Now().Add(-time.Minute)))
		if err != nil {
			t.Fatalf("create available outbox event %d: %v", i, err)
		}
		available[i] = event
	}
	future, err := db.Queries.CreateOutboxEvent(ctx, outboxEventParams(t, 3, time.Now().Add(time.Hour)))
	if err != nil {
		t.Fatalf("create future outbox event: %v", err)
	}

	results := make(chan []store.OutboxEvent, 2)
	errs := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	start := make(chan struct{})
	for range 2 {
		go func() {
			ready.Done()
			<-start
			events, err := db.Queries.ClaimOutboxEvents(ctx, store.ClaimOutboxEventsParams{
				ClaimToken:   pgtype.UUID{Bytes: uuid.New(), Valid: true},
				ClaimTimeout: pgtype.Interval{Microseconds: int64(time.Minute / time.Microsecond), Valid: true},
				BatchSize:    2,
			})
			errs <- err
			results <- events
		}()
	}
	ready.Wait()
	close(start)

	claimed := make(map[[16]byte]struct{})
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("claim events: %v", err)
		}
		for _, event := range <-results {
			if _, exists := claimed[event.ID.Bytes]; exists {
				t.Fatalf("event %s claimed by both workers", uuid.UUID(event.ID.Bytes))
			}
			claimed[event.ID.Bytes] = struct{}{}
		}
	}
	if len(claimed) != len(available) {
		t.Fatalf("claimed %d events, want %d", len(claimed), len(available))
	}
	if _, claimedFuture := claimed[future.ID.Bytes]; claimedFuture {
		t.Fatal("future outbox event was claimed")
	}
}

func TestClaimOutboxEventsReclaimsExpiredClaim(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := testdb.Context(t)
	event, err := db.Queries.CreateOutboxEvent(ctx, outboxEventParams(t, 1, time.Now().Add(-time.Minute)))
	if err != nil {
		t.Fatalf("create outbox event: %v", err)
	}
	testdb.RequireExec(t, ctx, db.Pool, "UPDATE outbox_events SET claimed_at = now() - interval '2 minutes' WHERE id = $1", event.ID)

	claimed, err := db.Queries.ClaimOutboxEvents(ctx, store.ClaimOutboxEventsParams{
		ClaimToken:   pgtype.UUID{Bytes: uuid.New(), Valid: true},
		ClaimTimeout: pgtype.Interval{Microseconds: int64(time.Minute / time.Microsecond), Valid: true},
		BatchSize:    1,
	})
	if err != nil {
		t.Fatalf("reclaim expired event: %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID.Bytes != event.ID.Bytes || claimed[0].Attempts != 1 {
		t.Fatalf("reclaimed events = %#v, want event %s with one attempt", claimed, uuid.UUID(event.ID.Bytes))
	}
}

func outboxEventParams(t *testing.T, sequence int, availableAt time.Time) store.CreateOutboxEventParams {
	t.Helper()
	return store.CreateOutboxEventParams{
		Topic:          "notification.created",
		AggregateType:  "notification",
		AggregateID:    pgtype.UUID{Bytes: uuid.New(), Valid: true},
		Payload:        []byte(`{"sequence": 1}`),
		IdempotencyKey: uuid.NewString(),
		AvailableAt:    pgtype.Timestamptz{Time: availableAt, Valid: true},
	}
}
