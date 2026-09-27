package billing

import (
	"context"
	"fmt"

	"github.com/go-chi/chi/v5"

	"github.com/hydrz/starter/internal/api/billingapi"
	"github.com/hydrz/starter/internal/delivery"
	"github.com/hydrz/starter/internal/platform/config"
	platformmodule "github.com/hydrz/starter/internal/platform/module"
)

const serviceName = "billing"

// Module composes the billing feature: Stripe-gated construction, contract-
// declared route mounting, and the raw Stripe webhook endpoint.
type Module struct{ service *Service }

var _ platformmodule.Module = (*Module)(nil)

func NewModule() *Module { return &Module{} }

// NewServiceModule wraps an already-constructed service for focused route tests.
func NewServiceModule(service *Service) *Module { return &Module{service: service} }

func (*Module) Name() string { return serviceName }

func (*Module) Enabled(cfg config.Config) bool { return cfg.Stripe.Enabled }

func (m *Module) Init(_ context.Context, deps platformmodule.Deps) error {
	cfg := deps.Config
	gateway, err := NewLiveGateway(cfg.Stripe.SecretKey)
	if err != nil {
		return fmt.Errorf("create stripe gateway: %w", err)
	}
	catalog, err := ParseCatalogJSON(cfg.Billing.PriceCatalog)
	if err != nil {
		return fmt.Errorf("parse price catalog: %w", err)
	}
	service, err := NewService(Dependencies{
		BillingAccounts:  NewPostgresBillingAccountRepository(deps.Queries),
		CheckoutSessions: NewPostgresCheckoutSessionRepository(deps.Queries),
		Subscriptions:    NewPostgresSubscriptionRepository(deps.Queries),
		OneTimePurchases: NewPostgresOneTimePurchaseRepository(deps.Queries),
		WebhookEvents:    NewPostgresWebhookEventRepository(deps.Queries),
		Entitlements:     NewPostgresEntitlementRepository(deps.Queries),
		Transactor:       deps.Transactor,
		Outbox:           NewPostgresOutboxWriter(deps.Queries),
		Gateway:          gateway,
		Catalog:          catalog,
		Clock:            SystemClock{},
		WebhookSecret:    cfg.Stripe.WebhookSecret,
		SuccessURL:       cfg.Billing.SuccessURL,
		CancelURL:        cfg.Billing.CancelURL,
		PortalReturnURL:  cfg.Billing.PortalReturnURL,
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
	server, err := billingapi.NewServer(NewHTTPHandler(m.service), billingapi.WithErrorHandler(mw.APIErrorHandler))
	if err != nil {
		panic(err)
	}
	router.Route("/api/organizations/{organizationId}/billing", func(r chi.Router) {
		if mw.Authenticate != nil {
			r.Use(mw.Authenticate)
		}
		r.With(mw.RequirePermission(billingapi.Permissions, "getBillingSummary")).Get("/summary", server.ServeHTTP)
		r.With(mw.RequirePermission(billingapi.Permissions, "createCheckoutSession")).Post("/checkout-sessions", server.ServeHTTP)
		r.With(mw.RequirePermission(billingapi.Permissions, "createPortalSession")).Post("/portal-sessions", server.ServeHTTP)
	})
	// Stripe webhook delivery: a raw-body handler outside the
	// TypeSpec/ogen JSON router (see billing.WebhookHTTPHandler), never
	// behind session auth — Stripe authenticates itself via the
	// Stripe-Signature HMAC header, verified before any JSON parsing.
	router.Handle("/api/billing/webhooks/stripe", NewWebhookHTTPHandler(m.service))
}

func (*Module) OutboxHandlers() map[string]delivery.Handler { return nil }

func (*Module) Shutdown(context.Context) error { return nil }
