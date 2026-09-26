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

type fakeClaimer struct {
	mu     sync.Mutex
	events []delivery.OutboxEvent
}

func (f *fakeClaimer) ClaimDueEvents(ctx context.Context, claimToken string, claimTimeout time.Duration, batchSize int) ([]delivery.OutboxEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.events) == 0 {
		return nil, nil
	}

	count := batchSize
	if count > len(f.events) {
		count = len(f.events)
	}

	claimed := make([]delivery.OutboxEvent, count)
	copy(claimed, f.events[:count])
	f.events = f.events[count:]

	for i := range claimed {
		claimed[i].ClaimToken = &claimToken
		claimed[i].Attempts++
	}

	return claimed, nil
}

func (f *fakeClaimer) addEvent(e delivery.OutboxEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
}

func TestWorker_RunOnce_ProcessesEvents(t *testing.T) {
	claimer := &fakeClaimer{}
	claimer.addEvent(delivery.OutboxEvent{
		ID:    "event-1",
		Topic: "auth.verification_requested",
	})
	claimer.addEvent(delivery.OutboxEvent{
		ID:    "event-2",
		Topic: "auth.password_reset_requested",
	})

	var handledCount atomic.Int32
	handler := delivery.HandlerFunc(func(ctx context.Context, event delivery.OutboxEvent) error {
		handledCount.Add(1)
		return nil
	})

	worker := delivery.NewWorker(claimer, handler, delivery.WorkerOptions{
		BatchSize: 10,
	})

	count, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 events processed, got %d", count)
	}
	if handledCount.Load() != 2 {
		t.Errorf("expected handler invoked 2 times, got %d", handledCount.Load())
	}

	// Next run should see 0 events
	count2, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("second RunOnce failed: %v", err)
	}
	if count2 != 0 {
		t.Errorf("expected 0 events on second run, got %d", count2)
	}
}

func TestWorker_RunOnce_HandlerErrorContinuesBatch(t *testing.T) {
	claimer := &fakeClaimer{}
	claimer.addEvent(delivery.OutboxEvent{ID: "event-1", Topic: "topic-1"})
	claimer.addEvent(delivery.OutboxEvent{ID: "event-2", Topic: "topic-2"})

	var handled []string
	var mu sync.Mutex

	handler := delivery.HandlerFunc(func(ctx context.Context, event delivery.OutboxEvent) error {
		mu.Lock()
		handled = append(handled, event.ID)
		mu.Unlock()

		if event.ID == "event-1" {
			return errors.New("simulated handler error")
		}
		return nil
	})

	worker := delivery.NewWorker(claimer, handler, delivery.WorkerOptions{
		BatchSize: 10,
	})

	count, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2 despite error on first event, got %d", count)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(handled) != 2 {
		t.Errorf("expected both events handled, got %v", handled)
	}
}

func TestWorker_StartStop(t *testing.T) {
	claimer := &fakeClaimer{}
	handler := delivery.HandlerFunc(func(ctx context.Context, event delivery.OutboxEvent) error {
		return nil
	})

	worker := delivery.NewWorker(claimer, handler, delivery.WorkerOptions{
		PollInterval: 50 * time.Millisecond,
		BatchSize:    5,
	})

	ctx := context.Background()
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Double start should fail
	if err := worker.Start(ctx); err == nil {
		t.Fatal("expected error on second Start, got nil")
	}

	// Stop cleanly
	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := worker.Stop(stopCtx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestWorker_ContextCancellation(t *testing.T) {
	claimer := &fakeClaimer{}
	handler := delivery.HandlerFunc(func(ctx context.Context, event delivery.OutboxEvent) error {
		return nil
	})

	worker := delivery.NewWorker(claimer, handler, delivery.WorkerOptions{
		PollInterval: 20 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Cancel context to stop worker
	cancel()

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()

	if err := worker.Stop(stopCtx); err != nil {
		t.Fatalf("Stop failed after context cancel: %v", err)
	}
}
