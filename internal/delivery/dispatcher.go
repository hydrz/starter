package delivery

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrTopicNotRegistered indicates that no handler is registered for an outbox topic.
var ErrTopicNotRegistered = errors.New("delivery: outbox topic is not registered")

// Dispatcher routes outbox events to handlers registered by topic.
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewDispatcher creates an empty Dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{handlers: make(map[string]Handler)}
}

// Register associates topic with h. Register panics for invalid or duplicate
// registrations because registrations are application-startup configuration.
func (d *Dispatcher) Register(topic string, h Handler) {
	if topic == "" {
		panic("delivery: cannot register an empty topic")
	}
	if h == nil {
		panic("delivery: cannot register a nil handler")
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.handlers[topic]; exists {
		panic(fmt.Sprintf("delivery: topic %q is already registered", topic))
	}
	d.handlers[topic] = h
}

// Handle dispatches event to its registered topic handler.
func (d *Dispatcher) Handle(ctx context.Context, event OutboxEvent) error {
	d.mu.RLock()
	handler, ok := d.handlers[event.Topic]
	d.mu.RUnlock()
	if !ok {
		return fmt.Errorf("%w: %s", ErrTopicNotRegistered, event.Topic)
	}
	return handler.Handle(ctx, event)
}
