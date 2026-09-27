package delivery_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hydrz/starter/internal/delivery"
)

func TestDispatcherRoutesRegisteredTopic(t *testing.T) {
	t.Parallel()

	dispatcher := delivery.NewDispatcher()
	var handled delivery.OutboxEvent
	dispatcher.Register("example.created", delivery.HandlerFunc(func(_ context.Context, event delivery.OutboxEvent) error {
		handled = event
		return nil
	}))

	event := delivery.OutboxEvent{ID: "event-1", Topic: "example.created"}
	if err := dispatcher.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if handled.ID != event.ID {
		t.Errorf("handled event ID = %q, want %q", handled.ID, event.ID)
	}
}

func TestDispatcherRejectsUnknownTopic(t *testing.T) {
	t.Parallel()

	err := delivery.NewDispatcher().Handle(context.Background(), delivery.OutboxEvent{Topic: "example.unknown"})
	if !errors.Is(err, delivery.ErrTopicNotRegistered) {
		t.Fatalf("Handle() error = %v, want ErrTopicNotRegistered", err)
	}
}

func TestDispatcherRejectsDuplicateRegistration(t *testing.T) {
	t.Parallel()

	dispatcher := delivery.NewDispatcher()
	handler := delivery.HandlerFunc(func(context.Context, delivery.OutboxEvent) error { return nil })
	dispatcher.Register("example.created", handler)

	defer func() {
		if recover() == nil {
			t.Fatal("second Register() did not panic")
		}
	}()
	dispatcher.Register("example.created", handler)
}
