package module

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/hydrz/starter/internal/delivery"
	"github.com/hydrz/starter/internal/platform/config"
)

type stubHandler struct{ err error }

func (h stubHandler) Handle(context.Context, delivery.OutboxEvent) error { return h.err }

func TestServices(t *testing.T) {
	t.Parallel()
	services := NewServices()
	services.Set("organization", "service")

	got, ok := services.Get("organization")
	if !ok || got != "service" {
		t.Fatalf("Get() = (%v, %t), want (service, true)", got, ok)
	}
	if _, ok := services.Get("missing"); ok {
		t.Fatal("Get(missing) reported a service")
	}
}

func TestPermissionMiddlewareMissingDeclarationFailsClosed(t *testing.T) {
	t.Parallel()
	handler := PermissionMiddleware(nil, nil, "undeclared")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestDispatcher(t *testing.T) {
	t.Parallel()
	want := errors.New("handler failed")
	dispatcher, err := NewDispatcher([]Module{testModule{handlers: map[string]delivery.Handler{"topic": stubHandler{err: want}}}})
	if err != nil {
		t.Fatalf("NewDispatcher() error = %v", err)
	}
	if err := dispatcher.Handle(context.Background(), delivery.OutboxEvent{Topic: "topic"}); !errors.Is(err, want) {
		t.Fatalf("Handle() error = %v, want %v", err, want)
	}
	if err := dispatcher.Handle(context.Background(), delivery.OutboxEvent{Topic: "missing"}); err == nil {
		t.Fatal("Handle(missing) error = nil, want error")
	}
}

func TestDispatcherRejectsDuplicateTopics(t *testing.T) {
	t.Parallel()
	_, err := NewDispatcher([]Module{
		testModule{handlers: map[string]delivery.Handler{"topic": stubHandler{}}},
		testModule{handlers: map[string]delivery.Handler{"topic": stubHandler{}}},
	})
	if err == nil {
		t.Fatal("NewDispatcher() error = nil, want duplicate-topic error")
	}
}

type testModule struct{ handlers map[string]delivery.Handler }

func (testModule) Name() string                                  { return "test" }
func (testModule) Enabled(config.Config) bool                    { return true }
func (testModule) Init(context.Context, Deps) error              { return nil }
func (testModule) Routes(chi.Router, Middlewares)                {}
func (m testModule) OutboxHandlers() map[string]delivery.Handler { return m.handlers }
func (testModule) Shutdown(context.Context) error                { return nil }
