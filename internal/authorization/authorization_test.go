package authorization

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/hydrz/starter/internal/auth"
)

func setupTestEnforcer(t *testing.T) *Service {
	t.Helper()

	adapter := NewMemoryAdapter(DefaultSeedRules())
	enforcer, err := NewEnforcer(adapter)
	if err != nil {
		t.Fatalf("failed to create test enforcer: %v", err)
	}

	return NewService(enforcer)
}

func TestCasbinRBACMatrix(t *testing.T) {
	service := setupTestEnforcer(t)
	enforcer := service.Enforcer()
	ctx := context.Background()

	org1 := "org-1111"
	org2 := "org-2222"

	userOwner := "user-owner"
	userAdmin := "user-admin"
	userMember := "user-member"
	userViewer := "user-viewer"
	userExternal := "user-external"

	// Assign roles in org1
	if _, err := enforcer.AddGroupingPolicy(userOwner, "owner", org1); err != nil {
		t.Fatalf("add grouping policy: %v", err)
	}
	if _, err := enforcer.AddGroupingPolicy(userAdmin, "admin", org1); err != nil {
		t.Fatalf("add grouping policy: %v", err)
	}
	if _, err := enforcer.AddGroupingPolicy(userMember, "member", org1); err != nil {
		t.Fatalf("add grouping policy: %v", err)
	}
	if _, err := enforcer.AddGroupingPolicy(userViewer, "viewer", org1); err != nil {
		t.Fatalf("add grouping policy: %v", err)
	}

	tests := []struct {
		name    string
		sub     string
		dom     string
		obj     string
		act     string
		allowed bool
	}{
		// Owner permissions in org1
		{"owner can read org", userOwner, org1, "organizations", "read", true},
		{"owner can update org", userOwner, org1, "organizations", "update", true},
		{"owner can delete org", userOwner, org1, "organizations", "delete", true},
		{"owner can manage members", userOwner, org1, "members", "delete", true},
		{"owner can manage invitations", userOwner, org1, "invitations", "create", true},
		{"owner can manage announcements", userOwner, org1, "announcements", "delete", true},

		// Admin permissions in org1
		{"admin can read org", userAdmin, org1, "organizations", "read", true},
		{"admin can update org", userAdmin, org1, "organizations", "update", true},
		{"admin cannot delete org", userAdmin, org1, "organizations", "delete", false},
		{"admin can manage members", userAdmin, org1, "members", "create", true},
		{"admin can manage announcements", userAdmin, org1, "announcements", "create", true},

		// Member permissions in org1
		{"member can read org", userMember, org1, "organizations", "read", true},
		{"member cannot update org", userMember, org1, "organizations", "update", false},
		{"member can read members", userMember, org1, "members", "read", true},
		{"member cannot manage members", userMember, org1, "members", "create", false},
		{"member cannot invite", userMember, org1, "invitations", "create", false},
		{"member can read announcements", userMember, org1, "announcements", "read", true},
		{"member can create announcements", userMember, org1, "announcements", "create", true},
		{"member can update announcements", userMember, org1, "announcements", "update", true},
		{"member cannot delete announcements", userMember, org1, "announcements", "delete", false},

		// Viewer permissions in org1
		{"viewer can read org", userViewer, org1, "organizations", "read", true},
		{"viewer can read members", userViewer, org1, "members", "read", true},
		{"viewer can read announcements", userViewer, org1, "announcements", "read", true},
		{"viewer cannot create announcements", userViewer, org1, "announcements", "create", false},
		{"viewer cannot update announcements", userViewer, org1, "announcements", "update", false},

		// External user in org1
		{"external cannot read org1", userExternal, org1, "organizations", "read", false},

		// Cross-domain isolation: org1 owner in org2
		{"org1 owner has no rights in org2", userOwner, org2, "organizations", "read", false},
		{"org1 owner cannot read announcements in org2", userOwner, org2, "announcements", "read", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := service.Authorize(ctx, tt.sub, tt.dom, tt.obj, tt.act)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tt.allowed {
				t.Errorf("Authorize(%s, %s, %s, %s) = %v; want %v", tt.sub, tt.dom, tt.obj, tt.act, allowed, tt.allowed)
			}
		})
	}
}

func TestRequirePermissionMiddleware(t *testing.T) {
	service := setupTestEnforcer(t)
	enforcer := service.Enforcer()

	orgID := "11111111-1111-1111-1111-111111111111"
	userID := "user-123"

	// Assign member role
	if _, err := enforcer.AddGroupingPolicy(userID, "member", orgID); err != nil {
		t.Fatalf("add grouping policy: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedOrgID, ok := OrganizationIDFromContext(r.Context())
		if !ok || capturedOrgID != orgID {
			http.Error(w, "missing captured org", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		r := chi.NewRouter()
		r.Route("/api/organizations/{organizationId}/announcements", func(r chi.Router) {
			r.Use(RequirePermission(service, "announcements", "read"))
			r.Get("/", handler)
		})

		req := httptest.NewRequest(http.MethodGet, "/api/organizations/"+orgID+"/announcements", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("non-member request returns 403", func(t *testing.T) {
		r := chi.NewRouter()
		r.Route("/api/organizations/{organizationId}/announcements", func(r chi.Router) {
			r.Use(RequirePermission(service, "announcements", "read"))
			r.Get("/", handler)
		})

		req := httptest.NewRequest(http.MethodGet, "/api/organizations/"+orgID+"/announcements", nil)
		ctx := auth.ContextWithPrincipal(req.Context(), "user-stranger")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("insufficient permission returns 403", func(t *testing.T) {
		r := chi.NewRouter()
		r.Route("/api/organizations/{organizationId}/announcements", func(r chi.Router) {
			r.Use(RequirePermission(service, "announcements", "delete"))
			r.Delete("/", handler)
		})

		req := httptest.NewRequest(http.MethodDelete, "/api/organizations/"+orgID+"/announcements", nil)
		ctx := auth.ContextWithPrincipal(req.Context(), userID)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("authorized member succeeds", func(t *testing.T) {
		r := chi.NewRouter()
		r.Route("/api/organizations/{organizationId}/announcements", func(r chi.Router) {
			r.Use(RequirePermission(service, "announcements", "read"))
			r.Get("/", handler)
		})

		req := httptest.NewRequest(http.MethodGet, "/api/organizations/"+orgID+"/announcements", nil)
		ctx := auth.ContextWithPrincipal(req.Context(), userID)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		if rec.Body.String() != "ok" {
			t.Errorf("body = %s; want ok", rec.Body.String())
		}
	})
}
