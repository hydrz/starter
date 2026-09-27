// Package module defines the application composition boundary for feature modules.
package module

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/hydrz/starter/internal/delivery"
	"github.com/hydrz/starter/internal/platform/config"
	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/store"
)

// PermissionEnforcer is the authorization boundary required by the
// RequirePermission route middleware. Declared here (consumer side) rather
// than imported from internal/authorization to avoid an import cycle with
// internal/auth via the platform layer.
type PermissionEnforcer interface {
	Enforce(rvals ...interface{}) (bool, error)
	GetRolesForUserInDomain(name string, domain string) []string
}

// Module is a self-contained application feature registered by the composition root.
type Module interface {
	Name() string
	Enabled(config.Config) bool
	Init(context.Context, Deps) error
	Routes(chi.Router, Middlewares)
	OutboxHandlers() map[string]delivery.Handler
	Shutdown(context.Context) error
}

// Deps contains process-wide infrastructure shared with modules.
type Deps struct {
	Pool       *pgxpool.Pool
	Queries    *store.Queries
	Transactor database.Transactor
	Logger     *slog.Logger
	Clock      Clock
	Config     config.Config
	Authorizer PermissionEnforcer
	Enforcer   *casbin.Enforcer
	Services   *Services
}

// Clock is the narrow time boundary available to infrastructure modules.
type Clock interface {
	Now() time.Time
}

// Services lets modules publish explicitly named domain services without making
// the platform package depend on any domain package.
type Services struct {
	mu     sync.RWMutex
	values map[string]any
}

func NewServices() *Services {
	return &Services{values: make(map[string]any)}
}

func (s *Services) Set(name string, service any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[name] = service
}

func (s *Services) Get(name string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	service, ok := s.values[name]
	return service, ok
}

// Middlewares contains route concerns supplied by the HTTP platform.
type Middlewares struct {
	Authenticate      func(http.Handler) http.Handler
	RequirePermission func(map[string]string, string) func(http.Handler) http.Handler
	APIErrorHandler   ogenerrors.ErrorHandler
}

// RouteMiddlewareProvider lets the composition root provide process-wide route
// middleware without coupling httpserver to a domain package.
type RouteMiddlewareProvider interface {
	RouteMiddlewares() Middlewares
}

// MiddlewareModule contributes platform-owned route middleware to a module list.
// It has no routes or lifecycle work of its own.
type MiddlewareModule struct{ enforcer PermissionEnforcer }

var _ Module = (*MiddlewareModule)(nil)
var _ RouteMiddlewareProvider = (*MiddlewareModule)(nil)

func NewMiddlewareModule(enforcer PermissionEnforcer) *MiddlewareModule {
	return &MiddlewareModule{enforcer: enforcer}
}

func (*MiddlewareModule) Name() string { return "route-middleware" }

func (*MiddlewareModule) Enabled(config.Config) bool { return true }

func (*MiddlewareModule) Init(context.Context, Deps) error { return nil }

func (*MiddlewareModule) Routes(chi.Router, Middlewares) {}

func (*MiddlewareModule) OutboxHandlers() map[string]delivery.Handler { return nil }

func (*MiddlewareModule) Shutdown(context.Context) error { return nil }

func (m *MiddlewareModule) RouteMiddlewares() Middlewares {
	return Middlewares{
		RequirePermission: func(permissions map[string]string, operationID string) func(http.Handler) http.Handler {
			return PermissionMiddleware(m.enforcer, permissions, operationID)
		},
	}
}

// PermissionMiddleware resolves the permission declared by a generated API
// package for one operation ID before applying Casbin authorization.
func PermissionMiddleware(enforcer PermissionEnforcer, permissions map[string]string, operationID string) func(http.Handler) http.Handler {
	permission, ok := permissions[operationID]
	if !ok || permission == "" {
		return func(http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeJSONError(w, http.StatusInternalServerError, "internal_error", "An internal error occurred.")
			})
		}
	}

	resource, action, ok := splitPermission(permission)
	if !ok {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeJSONError(w, http.StatusInternalServerError, "internal_error", "An internal error occurred.")
			})
		}
	}
	return requirePermission(enforcer, resource, action)
}

func splitPermission(permission string) (string, string, bool) {
	for i := range permission {
		if permission[i] == ':' && i > 0 && i < len(permission)-1 {
			return permission[:i], permission[i+1:], true
		}
	}
	return "", "", false
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{Code: code, Message: message})
}

// NewAPIErrorHandler prevents ogen fallback errors from exposing internal
// implementation details while leaving declared typed responses untouched.
func NewAPIErrorHandler(logger *slog.Logger) ogenerrors.ErrorHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
		status := ogenerrors.ErrorCode(err)
		if status == http.StatusInternalServerError {
			logger.ErrorContext(ctx, "unhandled api error", "method", r.Method, "path", r.URL.Path, "error", err)
			writeJSONError(w, status, "internal_error", "An internal error occurred.")
			return
		}
		logger.WarnContext(ctx, "rejected api request", "method", r.Method, "path", r.URL.Path, "status", status, "error", err)
		writeJSONError(w, status, "request_error", http.StatusText(status))
	}
}
