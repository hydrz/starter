package httpserver

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/hydrz/starter/internal/api/systemapi"
	"github.com/hydrz/starter/internal/auth"
	"github.com/hydrz/starter/internal/platform/module"
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

func init() {
	// Permission middleware is platform-owned but intentionally resolves the
	// principal through auth's context boundary. Install it once so parallel
	// handler tests never mutate shared resolver state.
	module.SetPrincipalResolver(auth.PrincipalFromContext)
}

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

// NewHandler mounts global middleware, contract documentation, the health
// endpoints, the web UI, and each enabled module's routes. Route ownership
// lives in the modules themselves (see internal/platform/module and each
// domain package's module.go); this function no longer knows about
// individual domain services or hand-written permission tables.
func NewHandler(
	modules []module.Module,
	readiness HealthChecker,
	trustedProxies []netip.Prefix,
) (http.Handler, error) {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(ClientIP(trustedProxies))
	router.Use(AccessLog(slog.Default()))
	router.Use(middleware.Recoverer)
	// SecurityHeaders sets baseline response headers (nosniff, referrer
	// policy, frame denial, and conditional HSTS) on every response; see
	// security_headers.go for why HSTS is conditional.
	router.Use(SecurityHeaders)
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

	// errorHandler replaces every generated *api.NewServer(...)'s default
	// error handler so an unhandled (return nil, err) fallback never
	// serializes raw internal/vendor error text to the client; see
	// errors.go.
	errorHandler := NewAPIErrorHandler(slog.Default())

	systemServer, err := systemapi.NewServer(&SystemHandler{readiness: readiness}, systemapi.WithErrorHandler(errorHandler))
	if err != nil {
		return nil, fmt.Errorf("initialize system api server: %w", err)
	}
	router.Handle("/api/healthz", systemServer)
	router.Handle("/api/readyz", systemServer)

	// The actual authentication middleware is contributed by auth.Module below,
	// so this package never constructs or imports a domain service directly.
	mw := module.Middlewares{
		Authenticate: func(next http.Handler) http.Handler { return next },
		RequirePermission: func(_ map[string]string, _ string) func(http.Handler) http.Handler {
			return func(next http.Handler) http.Handler { return next }
		},
		APIErrorHandler: errorHandler,
	}
	for _, m := range modules {
		provider, ok := m.(module.RouteMiddlewareProvider)
		if !ok {
			continue
		}
		provided := provider.RouteMiddlewares()
		if provided.Authenticate != nil {
			mw.Authenticate = provided.Authenticate
		}
		if provided.RequirePermission != nil {
			mw.RequirePermission = provided.RequirePermission
		}
	}
	for _, m := range modules {
		m.Routes(router, mw)
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
)
