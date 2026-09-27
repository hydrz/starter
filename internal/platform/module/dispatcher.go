package module

import (
	"context"
	"fmt"

	"github.com/hydrz/starter/internal/delivery"
)

// Dispatcher temporarily dispatches module-provided outbox handlers by topic.
// F1 will move retry and dead-letter policy into delivery.Dispatcher.
type Dispatcher struct {
	handlers map[string]delivery.Handler
}

func NewDispatcher(modules []Module) (*Dispatcher, error) {
	handlers := make(map[string]delivery.Handler)
	for _, feature := range modules {
		for topic, handler := range feature.OutboxHandlers() {
			if _, exists := handlers[topic]; exists {
				return nil, fmt.Errorf("module: duplicate outbox handler for topic %q", topic)
			}
			handlers[topic] = handler
		}
	}
	return &Dispatcher{handlers: handlers}, nil
}

func (d *Dispatcher) Handle(ctx context.Context, event delivery.OutboxEvent) error {
	handler, ok := d.handlers[event.Topic]
	if !ok {
		return fmt.Errorf("module: no outbox handler registered for topic %q", event.Topic)
	}
	return handler.Handle(ctx, event)
}

var _ delivery.Handler = (*Dispatcher)(nil)
