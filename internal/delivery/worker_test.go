package delivery_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hydrz/starter/internal/delivery"
)

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

type transition struct {
	eventID     string
	claimToken  string
	availableAt time.Time
	lastError   string
}

type fakeStore struct {
	mu           sync.Mutex
	events       []delivery.OutboxEvent
	claimErr     error
	processed    []transition
	retried      []transition
	deadLettered []transition
}

func (f *fakeStore) ClaimDueEvents(_ context.Context, claimToken string, _ time.Duration, batchSize int) ([]delivery.OutboxEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	if len(f.events) == 0 {
		return nil, nil
	}
	count := min(batchSize, len(f.events))
	batch := append([]delivery.OutboxEvent(nil), f.events[:count]...)
	f.events = f.events[count:]
	for index := range batch {
		batch[index].Attempts++
		batch[index].ClaimToken = &claimToken
	}
	return batch, nil
}

func (f *fakeStore) MarkProcessed(_ context.Context, eventID, claimToken string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.processed = append(f.processed, transition{eventID: eventID, claimToken: claimToken})
	return nil
}

func (f *fakeStore) Retry(_ context.Context, eventID, claimToken string, availableAt time.Time, lastError string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.retried = append(f.retried, transition{eventID: eventID, claimToken: claimToken, availableAt: availableAt, lastError: lastError})
	return nil
}

func (f *fakeStore) DeadLetter(_ context.Context, eventID, claimToken, lastError string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deadLettered = append(f.deadLettered, transition{eventID: eventID, claimToken: claimToken, lastError: lastError})
	return nil
}

func newWorker(store *fakeStore, handler delivery.Handler, opts delivery.WorkerOptions) *delivery.Worker {
	dispatcher := delivery.NewDispatcher()
	dispatcher.Register("test.event", handler)
	return delivery.NewWorker(store, dispatcher, opts)
}

func TestWorkerRunOnceMarksSuccessfulEventProcessed(t *testing.T) {
	t.Parallel()

	store := &fakeStore{events: []delivery.OutboxEvent{{ID: "event-1", Topic: "test.event"}}}
	worker := newWorker(store, delivery.HandlerFunc(func(context.Context, delivery.OutboxEvent) error { return nil }), delivery.WorkerOptions{})

	count, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("RunOnce() count = %d, want 1", count)
	}
	if len(store.processed) != 1 || store.processed[0].eventID != "event-1" || store.processed[0].claimToken == "" {
		t.Fatalf("processed transitions = %#v, want event-1 with claim token", store.processed)
	}
	if len(store.retried) != 0 || len(store.deadLettered) != 0 {
		t.Fatalf("unexpected failure transitions: retried=%#v deadLettered=%#v", store.retried, store.deadLettered)
	}
}

func TestWorkerRunOnceRetriesHandlerFailureWithBackoff(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{events: []delivery.OutboxEvent{{ID: "event-1", Topic: "test.event", Attempts: 1}}}
	handlerErr := errors.New("temporary failure")
	worker := newWorker(store, delivery.HandlerFunc(func(context.Context, delivery.OutboxEvent) error { return handlerErr }), delivery.WorkerOptions{
		MaxAttempts: 10,
		Clock:       fakeClock{now: now},
		Jitter:      func(delay time.Duration) time.Duration { return delay / 2 },
	})

	if _, err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(store.retried) != 1 {
		t.Fatalf("retried transitions = %#v, want one", store.retried)
	}
	wantAvailableAt := now.Add(delivery.RetryBackoff(2) / 2)
	if got := store.retried[0]; got.eventID != "event-1" || !got.availableAt.Equal(wantAvailableAt) || got.lastError != handlerErr.Error() {
		t.Errorf("retry = %#v, want event-1 at %v with %q", got, wantAvailableAt, handlerErr)
	}
	if len(store.processed) != 0 || len(store.deadLettered) != 0 {
		t.Fatalf("unexpected terminal transitions: processed=%#v deadLettered=%#v", store.processed, store.deadLettered)
	}
}

func TestWorkerRunOnceDeadLettersAtAttemptLimit(t *testing.T) {
	t.Parallel()

	store := &fakeStore{events: []delivery.OutboxEvent{{ID: "event-1", Topic: "test.event", Attempts: 2}}}
	handlerErr := errors.New("still failing")
	worker := newWorker(store, delivery.HandlerFunc(func(context.Context, delivery.OutboxEvent) error { return handlerErr }), delivery.WorkerOptions{MaxAttempts: 3})

	if _, err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(store.deadLettered) != 1 || store.deadLettered[0].eventID != "event-1" || store.deadLettered[0].lastError != handlerErr.Error() {
		t.Fatalf("dead-letter transitions = %#v, want event-1", store.deadLettered)
	}
	if len(store.retried) != 0 || len(store.processed) != 0 {
		t.Fatalf("unexpected non-dead-letter transitions: retried=%#v processed=%#v", store.retried, store.processed)
	}
}

func TestWorkerRunOnceDeadLettersUnknownTopicWithoutRetry(t *testing.T) {
	t.Parallel()

	store := &fakeStore{events: []delivery.OutboxEvent{{ID: "event-unknown", Topic: "unknown.event"}}}
	dispatcher := delivery.NewDispatcher()
	worker := delivery.NewWorker(store, dispatcher, delivery.WorkerOptions{MaxAttempts: 10})

	if _, err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(store.deadLettered) != 1 || store.deadLettered[0].eventID != "event-unknown" {
		t.Fatalf("dead-letter transitions = %#v, want event-unknown", store.deadLettered)
	}
	if !errors.Is(errors.New(store.deadLettered[0].lastError), delivery.ErrTopicNotRegistered) && store.deadLettered[0].lastError == "" {
		t.Error("dead-letter reason is empty")
	}
	if len(store.retried) != 0 {
		t.Fatalf("unknown topic retried: %#v", store.retried)
	}
}

func TestWorkerRunOnceContinuesAfterHandlerFailure(t *testing.T) {
	t.Parallel()

	store := &fakeStore{events: []delivery.OutboxEvent{
		{ID: "event-1", Topic: "test.event"},
		{ID: "event-2", Topic: "test.event"},
	}}
	var handled atomic.Int32
	worker := newWorker(store, delivery.HandlerFunc(func(_ context.Context, event delivery.OutboxEvent) error {
		handled.Add(1)
		if event.ID == "event-1" {
			return errors.New("temporary failure")
		}
		return nil
	}), delivery.WorkerOptions{Jitter: func(time.Duration) time.Duration { return 0 }})

	count, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if count != 2 || handled.Load() != 2 {
		t.Fatalf("count = %d, handled = %d, want 2 and 2", count, handled.Load())
	}
	if len(store.retried) != 1 || len(store.processed) != 1 {
		t.Fatalf("retried=%#v processed=%#v, want one each", store.retried, store.processed)
	}
}

func TestWorkerStartStop(t *testing.T) {
	store := &fakeStore{}
	worker := newWorker(store, delivery.HandlerFunc(func(context.Context, delivery.OutboxEvent) error { return nil }), delivery.WorkerOptions{PollInterval: 20 * time.Millisecond})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := worker.Start(ctx); err == nil {
		t.Fatal("second Start() error = nil, want error")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	if err := worker.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestWorkerContextCancellation(t *testing.T) {
	store := &fakeStore{}
	worker := newWorker(store, delivery.HandlerFunc(func(context.Context, delivery.OutboxEvent) error { return nil }), delivery.WorkerOptions{PollInterval: 20 * time.Millisecond})

	ctx, cancel := context.WithCancel(context.Background())
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	cancel()

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	if err := worker.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestRetryBackoffIsCapped(t *testing.T) {
	t.Parallel()

	for attempt, want := range map[int]time.Duration{
		0:  5 * time.Second,
		1:  5 * time.Second,
		2:  10 * time.Second,
		3:  20 * time.Second,
		20: time.Hour,
	} {
		if got := delivery.RetryBackoff(attempt); got != want {
			t.Errorf("RetryBackoff(%d) = %s, want %s", attempt, got, want)
		}
	}
}
