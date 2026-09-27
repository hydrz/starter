package notification

import (
	"context"
	"fmt"

	"github.com/go-chi/chi/v5"

	"github.com/hydrz/starter/internal/delivery"
	"github.com/hydrz/starter/internal/platform/config"
	platformmodule "github.com/hydrz/starter/internal/platform/module"
)

const serviceName = "notification"

// Module composes the notification feature for outbox processing. It mounts
// no HTTP routes; its contribution is the set of outbox topic handlers the
// worker dispatcher resolves.
type Module struct{ service *Service }

var _ platformmodule.Module = (*Module)(nil)

func NewModule() *Module { return &Module{} }

func (*Module) Name() string { return serviceName }

func (*Module) Enabled(cfg config.Config) bool { return cfg.Worker.Enabled || cfg.SMTP.Enabled }

func (m *Module) Init(_ context.Context, deps platformmodule.Deps) error {
	renderer, err := delivery.NewTemplateRenderer(deps.Config.App.Name)
	if err != nil {
		return fmt.Errorf("create template renderer: %w", err)
	}
	channels := make(map[string]delivery.Channel)
	if deps.Config.SMTP.Enabled {
		channels[ChannelSMTP] = delivery.NewSMTPChannel(deps.Config.SMTP)
	} else {
		channels[ChannelSMTP] = delivery.NewNoopChannel("smtp-noop", deps.Logger)
	}
	service, err := NewService(Dependencies{
		Repository: NewPostgresRepository(deps.Queries),
		Renderer:   renderer,
		Channels:   channels,
		Logger:     deps.Logger,
	})
	if err != nil {
		return err
	}
	m.service = service
	deps.Services.Set(serviceName, service)
	return nil
}

func (*Module) Routes(chi.Router, platformmodule.Middlewares) {}

// OutboxHandlers exposes the topics this module processes. F1 will replace
// this transitional per-topic map with delivery.Dispatcher semantics
// (retry with backoff and dead-lettering) owned by the worker.
func (m *Module) OutboxHandlers() map[string]delivery.Handler {
	if m.service == nil {
		return nil
	}
	return map[string]delivery.Handler{
		TopicAuthVerificationRequested:  m.service,
		TopicAuthPasswordResetRequested: m.service,
		TopicAuthEmailOTPRequested:      m.service,
	}
}

func (*Module) Shutdown(context.Context) error { return nil }
