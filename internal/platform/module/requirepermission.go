package module

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// PrincipalResolver extracts the authenticated subject from a request
// context. It is installed by the composition root (which wires internal/auth)
// because the platform layer must not import the auth package directly.
type PrincipalResolver func(ctx context.Context) (string, bool)

// principalResolver is set by apps/server during module registration.
var principalResolver PrincipalResolver

// SetPrincipalResolver installs the subject extractor used by permission
// middleware. It must be called before routes are mounted.
func SetPrincipalResolver(resolver PrincipalResolver) {
	principalResolver = resolver
}

// permissionErrorResponse mirrors the shared ApiError shape declared in
// packages/contracts/common/responses.tsp ({code, message}) so permission
// rejections use the same stable public error body as the rest of the API.
type permissionErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writePermissionError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(permissionErrorResponse{Code: code, Message: message})
}

// requirePermission verifies, in order: the caller is authenticated
// (principal present), the URL carries an organizationId domain, the caller
// is an organization member, and Casbin allows (sub, dom, resource, action).
// It mirrors the previous hand-written authorization.RequirePermission route
// middleware; behavior and error bodies are unchanged.
func requirePermission(enforcer PermissionEnforcer, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sub, ok := principalResolver(r.Context())
			if !ok || sub == "" {
				writePermissionError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
				return
			}

			dom := chi.URLParam(r, "organizationId")
			if dom == "" {
				writePermissionError(w, http.StatusBadRequest, "invalid_request", "Organization ID is required in URL path.")
				return
			}

			if enforcer != nil {
				roles := enforcer.GetRolesForUserInDomain(sub, dom)
				if len(roles) == 0 {
					writePermissionError(w, http.StatusForbidden, "forbidden", "You are not a member of this organization.")
					return
				}

				allowed, err := enforcer.Enforce(sub, dom, resource, action)
				if err != nil || !allowed {
					writePermissionError(w, http.StatusForbidden, "forbidden", "Insufficient permissions for the requested action.")
					return
				}
			}

			ctx := ContextWithOrganizationID(r.Context(), dom)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// organizationIDContextKey carries the URL's organization domain so domain
// services (organization, announcements, billing) can resolve it.
type organizationIDContextKey struct{}

// ContextWithOrganizationID returns a new context carrying the organization ID.
func ContextWithOrganizationID(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, organizationIDContextKey{}, orgID)
}

// OrganizationIDFromContext extracts the organization ID from context.
func OrganizationIDFromContext(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(organizationIDContextKey{}).(string)
	return val, ok && val != ""
}
