package delivery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const (
	defaultPollInterval = 1 * time.Second
	defaultBatchSize    = 50
	defaultClaimTimeout = 5 * time.Minute
	defaultMaxAttempts  = 10
	baseRetryDelay      = 5 * time.Second
	maxRetryDelay       = time.Hour
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
	DeadLetteredAt *time.Time
	LastError      *string
	CreatedAt      time.Time
}

// Store defines the persistence operations needed by Worker.
type Store interface {
	ClaimDueEvents(ctx context.Context, claimToken string, claimTimeout time.Duration, batchSize int) ([]OutboxEvent, error)
	MarkProcessed(ctx context.Context, eventID, claimToken string) error
	Retry(ctx context.Context, eventID, claimToken string, availableAt time.Time, lastError string) error
	DeadLetter(ctx context.Context, eventID, claimToken, lastError string) error
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

// Clock provides the current time, enabling deterministic worker tests.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// WorkerOptions configures the worker runtime.
type WorkerOptions struct {
	PollInterval time.Duration
	BatchSize    int
	ClaimTimeout time.Duration
	MaxAttempts  int
	Clock        Clock
	Jitter       func(time.Duration) time.Duration
	Logger       *slog.Logger
}

// Worker runs the outbox processing loop in the background.
type Worker struct {
	store        Store
	dispatcher   *Dispatcher
	pollInterval time.Duration
	batchSize    int
	claimTimeout time.Duration
	maxAttempts  int
	clock        Clock
	jitter       func(time.Duration) time.Duration
	logger       *slog.Logger

	running atomic.Bool
	stopCh  chan struct{}
	doneCh  chan struct{}
	mu      sync.Mutex
}

// NewWorker creates an outbox Worker with the given store and dispatcher.
func NewWorker(store Store, dispatcher *Dispatcher, opts WorkerOptions) *Worker {
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
	maxAttempts := opts.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
	}
	clock := opts.Clock
	if clock == nil {
		clock = systemClock{}
	}
	jitter := opts.Jitter
	if jitter == nil {
		jitter = fullJitter
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	return &Worker{
		store:        store,
		dispatcher:   dispatcher,
		pollInterval: pollInterval,
		batchSize:    batchSize,
		claimTimeout: claimTimeout,
		maxAttempts:  maxAttempts,
		clock:        clock,
		jitter:       jitter,
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

// RunOnce executes a single claim and processing cycle. It returns the number
// of claimed events whose handlers and state transitions were attempted.
func (w *Worker) RunOnce(ctx context.Context) (int, error) {
	claimToken := uuid.New().String()
	events, err := w.store.ClaimDueEvents(ctx, claimToken, w.claimTimeout, w.batchSize)
	if err != nil {
		return 0, fmt.Errorf("claim due events: %w", err)
	}

	for index, event := range events {
		if err := ctx.Err(); err != nil {
			return index, err
		}
		if event.ClaimToken == nil {
			event.ClaimToken = &claimToken
		}
		if err := w.processEvent(ctx, event); err != nil {
			w.logger.Error("process outbox event failed",
				"event_id", event.ID,
				"topic", event.Topic,
				"attempts", event.Attempts,
				"error", err,
			)
		}
	}

	return len(events), nil
}

func (w *Worker) processEvent(ctx context.Context, event OutboxEvent) error {
	claimToken := ""
	if event.ClaimToken != nil {
		claimToken = *event.ClaimToken
	}

	err := w.dispatcher.Handle(ctx, event)
	if err == nil {
		if err := w.store.MarkProcessed(ctx, event.ID, claimToken); err != nil {
			return fmt.Errorf("mark outbox event processed: %w", err)
		}
		return nil
	}

	if errors.Is(err, ErrTopicNotRegistered) || event.Attempts >= w.maxAttempts {
		if stateErr := w.store.DeadLetter(ctx, event.ID, claimToken, err.Error()); stateErr != nil {
			return errors.Join(err, fmt.Errorf("dead letter outbox event: %w", stateErr))
		}
		w.logger.Error("outbox event dead lettered",
			"event_id", event.ID,
			"topic", event.Topic,
			"attempts", event.Attempts,
			"error", err,
		)
		return nil
	}

	delay := w.jitter(RetryBackoff(event.Attempts))
	availableAt := w.clock.Now().Add(delay)
	if stateErr := w.store.Retry(ctx, event.ID, claimToken, availableAt, err.Error()); stateErr != nil {
		return errors.Join(err, fmt.Errorf("retry outbox event: %w", stateErr))
	}
	w.logger.Warn("outbox event scheduled for retry",
		"event_id", event.ID,
		"topic", event.Topic,
		"attempts", event.Attempts,
		"retry_at", availableAt,
		"error", err,
	)
	return nil
}

// RetryBackoff computes capped exponential backoff before jitter is applied.
func RetryBackoff(attempt int) time.Duration {
	if attempt <= 1 {
		return baseRetryDelay
	}

	delay := baseRetryDelay
	for current := 1; current < attempt; current++ {
		if delay >= maxRetryDelay/2 {
			return maxRetryDelay
		}
		delay *= 2
	}
	return min(delay, maxRetryDelay)
}

func fullJitter(maxDelay time.Duration) time.Duration {
	if maxDelay <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(maxDelay) + 1))
}

func (w *Worker) run(ctx context.Context) {
	defer close(w.doneCh)
	w.logger.Info("outbox worker started",
		"poll_interval", w.pollInterval,
		"batch_size", w.batchSize,
		"claim_timeout", w.claimTimeout,
		"max_attempts", w.maxAttempts,
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
