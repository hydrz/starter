package httpserver

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gopkg.in/yaml.v3"

	"github.com/hydrz/starter/internal/announcement"
	"github.com/hydrz/starter/internal/api/announcementsapi"
	"github.com/hydrz/starter/internal/api/authapi"
	"github.com/hydrz/starter/internal/api/billingapi"
	"github.com/hydrz/starter/internal/api/organizationsapi"
	"github.com/hydrz/starter/internal/api/systemapi"
	"github.com/hydrz/starter/internal/auth"
	"github.com/hydrz/starter/internal/authorization"
	"github.com/hydrz/starter/internal/billing"
	"github.com/hydrz/starter/internal/organization"
	"github.com/hydrz/starter/internal/platform/webui"
)

type HealthChecker interface {
	Ping(context.Context) error
}

//go:embed assets/*
var docsAssets embed.FS

type SystemHandler struct {
	readiness HealthChecker
}

var _ systemapi.Handler = (*SystemHandler)(nil)

func (h *SystemHandler) GetHealth(_ context.Context) (*systemapi.HealthResponse, error) {
	return &systemapi.HealthResponse{Status: systemapi.HealthResponseStatusOk}, nil
}

func (h *SystemHandler) GetReadiness(ctx context.Context) (systemapi.GetReadinessRes, error) {
	if h.readiness == nil || h.readiness.Ping(ctx) != nil {
		return &systemapi.ApiError{
			Code:    "not_ready",
			Message: "Service dependencies are not ready.",
		}, nil
	}
	return &systemapi.HealthResponse{Status: systemapi.HealthResponseStatusOk}, nil
}

func NewHandler(
	announcements *announcement.Service,
	readiness HealthChecker,
	identity *auth.Service,
	orgs *organization.Service,
	authorizer authorization.PermissionEnforcer,
	billingService *billing.Service,
) (http.Handler, error) {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(AccessLog(slog.Default()))
	router.Use(middleware.Recoverer)
	// auth.WithHTTPContext exposes the raw request/response via context so
	// auth.BearerTokenFromContext (used by every AuthenticationMiddleware
	// call, not just the auth API's own routes) and the refresh-cookie
	// helpers can read/write them. It must be global: without it, bearer
	// tokens are silently invisible to every route outside /api/auth/*.
	router.Use(auth.WithHTTPContext)

	router.Get("/api/openapi.yaml", openAPISpecYAML)
	router.Get("/api/openapi.json", openAPISpecJSON)
	router.Get("/api/docs", scalarReference)
	router.Get("/api/docs/scalar.js", scalarScript)

	systemServer, err := systemapi.NewServer(&SystemHandler{readiness: readiness})
	if err != nil {
		return nil, fmt.Errorf("initialize system api server: %w", err)
	}
	router.Handle("/api/healthz", systemServer)
	router.Handle("/api/readyz", systemServer)

	var authMiddleware func(http.Handler) http.Handler
	if identity != nil {
		authMiddleware = identity.AuthenticationMiddleware
		authServer, err := authapi.NewServer(auth.NewHTTPHandler(identity))
		if err != nil {
			return nil, fmt.Errorf("initialize auth api server: %w", err)
		}
		// identity.AuthenticationMiddleware resolves an optional bearer
		// principal into context; it never itself rejects a request, so
		// authorization for other routes remains each router group's own
		// concern (Casbin RequirePermission, or the operation's own 401).
		// Operation paths are declared in TypeSpec with the full /api/auth
		// prefix, so the generated server is handled directly rather than
		// mounted with path-stripping.
		authHandler := identity.AuthenticationMiddleware(authServer)
		router.Handle("/api/auth/*", authHandler)
	}

	if orgs != nil {
		orgServer, err := organizationsapi.NewServer(organization.NewHTTPHandler(orgs))
		if err != nil {
			return nil, fmt.Errorf("initialize organizations api server: %w", err)
		}

		router.Route("/api/organizations", func(r chi.Router) {
			if authMiddleware != nil {
				r.Use(authMiddleware)
			}

			// Top-level operations
			r.Get("/", orgServer.ServeHTTP)
			r.Post("/", orgServer.ServeHTTP)
			r.Post("/invitations/accept", orgServer.ServeHTTP)

			// Domain-scoped operations
			r.Route("/{organizationId}", func(r chi.Router) {
				if authorizer != nil {
					r.With(authorization.RequirePermission(authorizer, "organizations", "read")).Get("/", orgServer.ServeHTTP)
					r.With(authorization.RequirePermission(authorizer, "organizations", "update")).Put("/", orgServer.ServeHTTP)
					r.With(authorization.RequirePermission(authorizer, "organizations", "delete")).Delete("/", orgServer.ServeHTTP)

					r.With(authorization.RequirePermission(authorizer, "members", "read")).Get("/members", orgServer.ServeHTTP)
					r.With(authorization.RequirePermission(authorizer, "members", "update")).Put("/members/{userId}", orgServer.ServeHTTP)
					r.With(authorization.RequirePermission(authorizer, "members", "delete")).Delete("/members/{userId}", orgServer.ServeHTTP)

					r.With(authorization.RequirePermission(authorizer, "invitations", "read")).Get("/invitations", orgServer.ServeHTTP)
					r.With(authorization.RequirePermission(authorizer, "invitations", "create")).Post("/invitations", orgServer.ServeHTTP)
					r.With(authorization.RequirePermission(authorizer, "invitations", "delete")).Delete("/invitations/{invitationId}", orgServer.ServeHTTP)
				} else {
					r.HandleFunc("/*", orgServer.ServeHTTP)
				}
			})
		})
	}

	if announcements != nil {
		announcementServer, err := announcementsapi.NewServer(announcement.NewHTTPHandler(announcements))
		if err != nil {
			return nil, fmt.Errorf("initialize announcements api server: %w", err)
		}

		router.Route("/api/organizations/{organizationId}/announcements", func(r chi.Router) {
			if authMiddleware != nil {
				r.Use(authMiddleware)
			}
			if authorizer != nil {
				r.With(authorization.RequirePermission(authorizer, "announcements", "read")).Get("/", announcementServer.ServeHTTP)
				r.With(authorization.RequirePermission(authorizer, "announcements", "create")).Post("/", announcementServer.ServeHTTP)
				r.With(authorization.RequirePermission(authorizer, "announcements", "read")).Get("/{id}", announcementServer.ServeHTTP)
				r.With(authorization.RequirePermission(authorizer, "announcements", "update")).Put("/{id}", announcementServer.ServeHTTP)
				r.With(authorization.RequirePermission(authorizer, "announcements", "delete")).Delete("/{id}", announcementServer.ServeHTTP)
			} else {
				r.HandleFunc("/*", announcementServer.ServeHTTP)
				r.HandleFunc("/", announcementServer.ServeHTTP)
			}
		})
	}

	if billingService != nil {
		billingServer, err := billingapi.NewServer(billing.NewHTTPHandler(billingService))
		if err != nil {
			return nil, fmt.Errorf("initialize billing api server: %w", err)
		}

		router.Route("/api/organizations/{organizationId}/billing", func(r chi.Router) {
			if authMiddleware != nil {
				r.Use(authMiddleware)
			}
			if authorizer != nil {
				r.With(authorization.RequirePermission(authorizer, "billing", "read")).Get("/summary", billingServer.ServeHTTP)
				r.With(authorization.RequirePermission(authorizer, "billing", "checkout")).Post("/checkout-sessions", billingServer.ServeHTTP)
				r.With(authorization.RequirePermission(authorizer, "billing", "portal")).Post("/portal-sessions", billingServer.ServeHTTP)
			} else {
				r.HandleFunc("/*", billingServer.ServeHTTP)
			}
		})

		// Stripe webhook delivery: a raw-body handler outside the
		// TypeSpec/ogen JSON router (see billing.WebhookHTTPHandler), never
		// behind session auth — Stripe authenticates itself via the
		// Stripe-Signature HMAC header, verified before any JSON parsing.
		router.Handle("/api/billing/webhooks/stripe", billing.NewWebhookHTTPHandler(billingService))
	}

	webHandler, err := webui.NewHandler()
	if err != nil {
		return nil, fmt.Errorf("initialize web assets: %w", err)
	}
	router.NotFound(webHandler.ServeHTTP)

	return router, nil
}

var (
	openAPIYAML []byte
	openAPIJSON []byte
	openAPIOnce sync.Once
	openAPIErr  error
)

func loadOpenAPISpecs() error {
	openAPIOnce.Do(func() {
		openAPIYAML, openAPIErr = docsAssets.ReadFile("assets/openapi.yaml")
		if openAPIErr != nil {
			return
		}
		var raw any
		if openAPIErr = yaml.Unmarshal(openAPIYAML, &raw); openAPIErr != nil {
			return
		}
		openAPIJSON, openAPIErr = json.Marshal(raw)
	})
	return openAPIErr
}

func openAPISpecYAML(response http.ResponseWriter, _ *http.Request) {
	if err := loadOpenAPISpecs(); err != nil {
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = response.Write(openAPIYAML)
}

func openAPISpecJSON(response http.ResponseWriter, _ *http.Request) {
	if err := loadOpenAPISpecs(); err != nil {
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = response.Write(openAPIJSON)
}

func scalarReference(response http.ResponseWriter, _ *http.Request) {
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src data: https:; font-src data:; connect-src 'self'")
	_, _ = response.Write([]byte(scalarHTML))
}

func scalarScript(response http.ResponseWriter, _ *http.Request) {
	script, err := docsAssets.ReadFile("assets/scalar.js")
	if err != nil {
		http.Error(response, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	response.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = response.Write(script)
}

const scalarHTML = `<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Starter API</title>
  </head>
  <body>
    <script id="api-reference" data-url="/api/openapi.yaml"></script>
    <script src="/api/docs/scalar.js"></script>
  </body>
</html>`
