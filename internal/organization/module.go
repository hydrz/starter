package organization

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hydrz/starter/internal/api/organizationsapi"
	"github.com/hydrz/starter/internal/delivery"
	"github.com/hydrz/starter/internal/platform/config"
	platformmodule "github.com/hydrz/starter/internal/platform/module"
)

const serviceName = "organization"

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
	service, err := NewService(Dependencies{
		Repository:    NewPostgresRepository(deps.Queries),
		Transactor:    deps.Transactor,
		Enforcer:      deps.Enforcer,
		Clock:         systemClock{clock: deps.Clock},
		InvitationTTL: 0,
	})
	if err != nil {
		return err
	}
	m.service = service
	deps.Services.Set(serviceName, service)
	return nil
}

func (m *Module) Routes(router chi.Router, mw platformmodule.Middlewares) {
	if m.service == nil {
		return
	}
	server, err := organizationsapi.NewServer(NewHTTPHandler(m.service), organizationsapi.WithErrorHandler(mw.APIErrorHandler))
	if err != nil {
		panic(err)
	}
	router.Route("/api/organizations", func(r chi.Router) {
		if mw.Authenticate != nil {
			r.Use(mw.Authenticate)
		}
		r.Get("/", server.ServeHTTP)
		r.Post("/", server.ServeHTTP)
		r.Post("/invitations/accept", server.ServeHTTP)
		r.Route("/{organizationId}", func(r chi.Router) {
			r.With(mw.RequirePermission(organizationsapi.Permissions, "getOrganization")).Get("/", server.ServeHTTP)
			r.With(mw.RequirePermission(organizationsapi.Permissions, "updateOrganization")).Put("/", server.ServeHTTP)
			r.With(mw.RequirePermission(organizationsapi.Permissions, "deleteOrganization")).Delete("/", server.ServeHTTP)
			r.With(mw.RequirePermission(organizationsapi.Permissions, "listMembers")).Get("/members", server.ServeHTTP)
			r.With(mw.RequirePermission(organizationsapi.Permissions, "updateMemberRole")).Put("/members/{userId}", server.ServeHTTP)
			r.With(mw.RequirePermission(organizationsapi.Permissions, "removeMember")).Delete("/members/{userId}", server.ServeHTTP)
			r.With(mw.RequirePermission(organizationsapi.Permissions, "listInvitations")).Get("/invitations", server.ServeHTTP)
			r.With(mw.RequirePermission(organizationsapi.Permissions, "createInvitation")).Post("/invitations", server.ServeHTTP)
			r.With(mw.RequirePermission(organizationsapi.Permissions, "revokeInvitation")).Delete("/invitations/{invitationId}", server.ServeHTTP)
		})
	})
}

func (*Module) OutboxHandlers() map[string]delivery.Handler { return nil }

func (*Module) Shutdown(context.Context) error { return nil }

type systemClock struct{ clock platformmodule.Clock }

func (c systemClock) Now() time.Time { return c.clock.Now() }
