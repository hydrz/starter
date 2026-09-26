package delivery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const (
	defaultPollInterval = 1 * time.Second
	defaultBatchSize    = 50
	defaultClaimTimeout = 5 * time.Minute
)

// OutboxEvent represents an event claimed from the outbox table.
type OutboxEvent struct {
	ID             string
	Topic          string
	AggregateType  string
	AggregateID    string
	Payload        []byte
	IdempotencyKey string
	AvailableAt    time.Time
	ClaimedAt      *time.Time
	ClaimToken     *string
	Attempts       int
	ProcessedAt    *time.Time
	LastError      *string
	CreatedAt      time.Time
}

// Claimer defines the contract for atomically claiming due outbox events.
type Claimer interface {
	ClaimDueEvents(ctx context.Context, claimToken string, claimTimeout time.Duration, batchSize int) ([]OutboxEvent, error)
}

// Handler processes a single claimed outbox event.
type Handler interface {
	Handle(ctx context.Context, event OutboxEvent) error
}

// HandlerFunc is an adapter to allow the use of ordinary functions as Handlers.
type HandlerFunc func(ctx context.Context, event OutboxEvent) error

func (f HandlerFunc) Handle(ctx context.Context, event OutboxEvent) error {
	return f(ctx, event)
}

// WorkerOptions configures the worker runtime.
type WorkerOptions struct {
	PollInterval time.Duration
	BatchSize    int
	ClaimTimeout time.Duration
	Logger       *slog.Logger
}

// Worker runs the outbox processing loop in the background.
type Worker struct {
	claimer      Claimer
	handler      Handler
	pollInterval time.Duration
	batchSize    int
	claimTimeout time.Duration
	logger       *slog.Logger

	running atomic.Bool
	stopCh  chan struct{}
	doneCh  chan struct{}
	mu      sync.Mutex
}

// NewWorker creates an outbox Worker with the given claimer and handler.
func NewWorker(claimer Claimer, handler Handler, opts WorkerOptions) *Worker {
	pollInterval := opts.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	claimTimeout := opts.ClaimTimeout
	if claimTimeout <= 0 {
		claimTimeout = defaultClaimTimeout
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	return &Worker{
		claimer:      claimer,
		handler:      handler,
		pollInterval: pollInterval,
		batchSize:    batchSize,
		claimTimeout: claimTimeout,
		logger:       logger,
	}
}

// Start spawns the worker loop in a background goroutine.
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running.Swap(true) {
		return errors.New("delivery: worker is already running")
	}

	w.stopCh = make(chan struct{})
	w.doneCh = make(chan struct{})

	go w.run(ctx)
	return nil
}

// Stop gracefully signals the worker to stop and waits for in-flight processing to complete.
func (w *Worker) Stop(ctx context.Context) error {
	w.mu.Lock()
	if !w.running.Load() {
		w.mu.Unlock()
		return nil
	}
	close(w.stopCh)
	w.mu.Unlock()

	select {
	case <-w.doneCh:
		w.running.Store(false)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RunOnce executes a single claim and processing cycle. Returns the number of events processed.
func (w *Worker) RunOnce(ctx context.Context) (int, error) {
	claimToken := uuid.New().String()
	events, err := w.claimer.ClaimDueEvents(ctx, claimToken, w.claimTimeout, w.batchSize)
	if err != nil {
		return 0, fmt.Errorf("claim due events: %w", err)
	}

	if len(events) == 0 {
		return 0, nil
	}

	for _, event := range events {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		if err := w.handler.Handle(ctx, event); err != nil {
			w.logger.Error("handle outbox event failed",
				"event_id", event.ID,
				"topic", event.Topic,
				"attempts", event.Attempts,
				"error", err,
			)
		}
	}

	return len(events), nil
}

func (w *Worker) run(ctx context.Context) {
	defer close(w.doneCh)
	w.logger.Info("outbox worker started",
		"poll_interval", w.pollInterval,
		"batch_size", w.batchSize,
		"claim_timeout", w.claimTimeout,
	)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("outbox worker context cancelled")
			return
		case <-w.stopCh:
			w.logger.Info("outbox worker stop requested")
			return
		case <-ticker.C:
			count, err := w.RunOnce(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				w.logger.Error("outbox worker iteration failed", "error", err)
			}
			// Drain backlog immediately without waiting for next tick
			for count >= w.batchSize {
				select {
				case <-ctx.Done():
					return
				case <-w.stopCh:
					return
				default:
				}
				count, err = w.RunOnce(ctx)
				if err != nil && !errors.Is(err, context.Canceled) {
					w.logger.Error("outbox worker backlog iteration failed", "error", err)
					break
				}
			}
		}
	}
}
