package announcement

import (
	"context"

	"github.com/go-chi/chi/v5"

	"github.com/hydrz/starter/internal/api/announcementsapi"
	"github.com/hydrz/starter/internal/delivery"
	"github.com/hydrz/starter/internal/platform/config"
	platformmodule "github.com/hydrz/starter/internal/platform/module"
)

const serviceName = "announcement"

type Module struct {
	service *Service
}

var _ platformmodule.Module = (*Module)(nil)

func NewModule() *Module { return &Module{} }

// NewServiceModule wraps an already-constructed service for focused route tests.
func NewServiceModule(service *Service) *Module { return &Module{service: service} }

func (*Module) Name() string { return serviceName }

func (*Module) Enabled(config.Config) bool { return true }

func (m *Module) Init(_ context.Context, deps platformmodule.Deps) error {
	m.service = NewService(NewPostgresRepository(deps.Queries))
	deps.Services.Set(serviceName, m.service)
	return nil
}

func (m *Module) Routes(router chi.Router, mw platformmodule.Middlewares) {
	if m.service == nil {
		return
	}
	server, err := announcementsapi.NewServer(NewHTTPHandler(m.service), announcementsapi.WithErrorHandler(mw.APIErrorHandler))
	if err != nil {
		panic(err)
	}
	router.Route("/api/organizations/{organizationId}/announcements", func(r chi.Router) {
		if mw.Authenticate != nil {
			r.Use(mw.Authenticate)
		}
		r.With(mw.RequirePermission(announcementsapi.Permissions, "listAnnouncements")).Get("/", server.ServeHTTP)
		r.With(mw.RequirePermission(announcementsapi.Permissions, "createAnnouncement")).Post("/", server.ServeHTTP)
		r.With(mw.RequirePermission(announcementsapi.Permissions, "getAnnouncement")).Get("/{id}", server.ServeHTTP)
		r.With(mw.RequirePermission(announcementsapi.Permissions, "updateAnnouncement")).Put("/{id}", server.ServeHTTP)
		r.With(mw.RequirePermission(announcementsapi.Permissions, "deleteAnnouncement")).Delete("/{id}", server.ServeHTTP)
	})
}

func (*Module) OutboxHandlers() map[string]delivery.Handler { return nil }

func (*Module) Shutdown(context.Context) error { return nil }
