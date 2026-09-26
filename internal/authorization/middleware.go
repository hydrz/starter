package authorization

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/hydrz/starter/internal/auth"
)

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

// PermissionEnforcer defines the methods required for route-level permission checking.
type PermissionEnforcer interface {
	Enforce(rvals ...interface{}) (bool, error)
	GetRolesForUserInDomain(name string, domain string) []string
}

type apiErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiErrorResponse{Code: code, Message: message})
}

// RequirePermission returns Chi middleware verifying that the authenticated caller (sub)
// has permission to perform action (act) on resource (obj) within the URL's organization domain (dom).
// It verifies:
// 1. Caller is authenticated (sub from auth context). If missing -> 401 Unauthorized.
// 2. Organization ID is present in URL path ("organizationId"). If missing -> 400 Bad Request.
// 3. Caller is a member of the organization (has at least one role in domain). If not -> 403 Forbidden.
// 4. Casbin policy allows (sub, dom, resource, action). If not -> 403 Forbidden.
func RequirePermission(enforcer PermissionEnforcer, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sub, ok := auth.PrincipalFromContext(r.Context())
			if !ok || sub == "" {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
				return
			}

			dom := chi.URLParam(r, "organizationId")
			if dom == "" {
				writeJSONError(w, http.StatusBadRequest, "invalid_request", "Organization ID is required in URL path.")
				return
			}

			if enforcer != nil {
				roles := enforcer.GetRolesForUserInDomain(sub, dom)
				if len(roles) == 0 {
					writeJSONError(w, http.StatusForbidden, "forbidden", "You are not a member of this organization.")
					return
				}

				allowed, err := enforcer.Enforce(sub, dom, resource, action)
				if err != nil || !allowed {
					writeJSONError(w, http.StatusForbidden, "forbidden", "Insufficient permissions for the requested action.")
					return
				}
			}

			ctx := ContextWithOrganizationID(r.Context(), dom)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
