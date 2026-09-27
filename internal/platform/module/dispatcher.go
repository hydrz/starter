package module

import (
	"fmt"

	"github.com/hydrz/starter/internal/delivery"
)

// NewDispatcher collects module-provided outbox handlers into a
// delivery.Dispatcher. Duplicate topic registrations are startup errors.
func NewDispatcher(modules []Module) (*delivery.Dispatcher, error) {
	seen := make(map[string]struct{})
	dispatcher := delivery.NewDispatcher()
	for _, feature := range modules {
		for topic, handler := range feature.OutboxHandlers() {
			if _, exists := seen[topic]; exists {
				return nil, fmt.Errorf("module: duplicate outbox handler for topic %q", topic)
			}
			seen[topic] = struct{}{}
			dispatcher.Register(topic, handler)
		}
	}
	return dispatcher, nil
}
