package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hydrz/starter/internal/announcement"
	"github.com/hydrz/starter/internal/auth"
	"github.com/hydrz/starter/internal/authorization"
	"github.com/hydrz/starter/internal/organization"
	"github.com/hydrz/starter/internal/platform/httpserver"
)

type announcementRepository struct{}

type healthChecker struct{ err error }

func (checker healthChecker) Ping(context.Context) error { return checker.err }

func (*announcementRepository) List(context.Context, uuid.UUID, announcement.Filter) ([]announcement.Announcement, int64, error) {
	return []announcement.Announcement{}, 0, nil
}
func (*announcementRepository) Get(context.Context, uuid.UUID, string) (announcement.Announcement, error) {
	return announcement.Announcement{}, announcement.ErrNotFound
}
func (*announcementRepository) Create(_ context.Context, orgID uuid.UUID, input announcement.Input) (announcement.Announcement, error) {
	return announcement.Announcement{
		ID: "9b2e66b7-495d-4d44-b353-9952fa4c2a37", OrganizationID: orgID.String(), Title: input.Title, Content: input.Content, Status: input.Status,
	}, nil
}
func (*announcementRepository) Update(context.Context, uuid.UUID, string, announcement.Input) (announcement.Announcement, error) {
	return announcement.Announcement{}, announcement.ErrNotFound
}
func (*announcementRepository) Delete(context.Context, uuid.UUID, string) error {
	return announcement.ErrNotFound
}

func TestHealth(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	recorder := httptest.NewRecorder()

	newHandler(t, nil, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status value = %q, want %q", body.Status, "ok")
	}
}

func TestOpenAPISpec(t *testing.T) {
	t.Parallel()

	// 1. Test YAML endpoint
	yamlRequest := httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil)
	yamlRecorder := httptest.NewRecorder()
	newHandler(t, nil, nil).ServeHTTP(yamlRecorder, yamlRequest)
	if yamlRecorder.Code != http.StatusOK {
		t.Fatalf("yaml status = %d, want %d", yamlRecorder.Code, http.StatusOK)
	}
	if !bytes.Contains(yamlRecorder.Body.Bytes(), []byte("openapi: 3.0.0")) {
		t.Errorf("yaml body missing openapi: 3.0.0")
	}

	// 2. Test JSON endpoint
	jsonRequest := httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil)
	jsonRecorder := httptest.NewRecorder()
	newHandler(t, nil, nil).ServeHTTP(jsonRecorder, jsonRequest)
	if jsonRecorder.Code != http.StatusOK {
		t.Fatalf("json status = %d, want %d", jsonRecorder.Code, http.StatusOK)
	}
	var document struct {
		OpenAPI string `json:"openapi"`
	}
	if err := json.NewDecoder(jsonRecorder.Body).Decode(&document); err != nil {
		t.Fatalf("decode OpenAPI document: %v", err)
	}
	if document.OpenAPI != "3.0.0" {
		t.Errorf("OpenAPI version = %q, want %q", document.OpenAPI, "3.0.0")
	}
}

func TestScalarReference(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
	recorder := httptest.NewRecorder()

	newHandler(t, nil, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	for _, expected := range [][]byte{[]byte(`data-url="/api/openapi.yaml"`), []byte(`/api/docs/scalar.js`)} {
		if !bytes.Contains(body, expected) {
			t.Errorf("Scalar reference response does not contain %q", expected)
		}
	}
}

func TestScalarScript(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/api/docs/scalar.js", nil)
	recorder := httptest.NewRecorder()

	newHandler(t, nil, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	contentType := recorder.Header().Get("Content-Type")
	if !bytes.Contains([]byte(contentType), []byte("javascript")) {
		t.Errorf("Content-Type = %q, want javascript", contentType)
	}
}

func TestUnknownRoute(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/api/unknown", nil)
	recorder := httptest.NewRecorder()

	newHandler(t, nil, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestCreateAnnouncement(t *testing.T) {
	t.Parallel()

	orgID := "9b2e66b7-495d-4d44-b353-9952fa4c2a37"
	body := bytes.NewBufferString(`{"title":"Maintenance","content":"Tonight at 22:00","status":"draft"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/organizations/"+orgID+"/announcements", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	service := announcement.NewService(&announcementRepository{})

	newHandler(t, service, healthChecker{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"title":"Maintenance"`)) {
		t.Errorf("response = %s, want created announcement", recorder.Body.String())
	}
}

func TestReadiness(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		checker healthChecker
		status  int
	}{
		{name: "ready", checker: healthChecker{}, status: http.StatusOK},
		{name: "dependency unavailable", checker: healthChecker{err: errors.New("unavailable")}, status: http.StatusServiceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/readyz", nil)
			response := httptest.NewRecorder()
			newHandler(t, nil, test.checker).ServeHTTP(response, request)
			if response.Code != test.status {
				t.Errorf("status = %d, want %d", response.Code, test.status)
			}
		})
	}
}

func newHandler(t *testing.T, service *announcement.Service, checker httpserver.HealthChecker) http.Handler {
	t.Helper()
	handler, err := httpserver.NewHandler(service, checker, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return handler
}

func TestAccessLogMiddleware(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logMiddleware := httpserver.AccessLog(logger)

	handler := logMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	request := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	request.Header.Set("User-Agent", "test-agent")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	logOutput := buf.String()
	for _, expected := range []string{`"method":"GET"`, `"path":"/test/path"`, `"status":200`, `"user_agent":"test-agent"`} {
		if !bytes.Contains([]byte(logOutput), []byte(expected)) {
			t.Errorf("log output %s missing expected substring %s", logOutput, expected)
		}
	}
}

type testOrgRepo struct {
	org organization.Organization
}

func (s *testOrgRepo) Create(context.Context, string, string) (organization.Organization, error) {
	return s.org, nil
}
func (s *testOrgRepo) GetByID(context.Context, string) (organization.Organization, error) {
	return s.org, nil
}
func (s *testOrgRepo) GetBySlug(context.Context, string) (organization.Organization, error) {
	return s.org, nil
}
func (s *testOrgRepo) Update(context.Context, string, string, string) (organization.Organization, error) {
	return s.org, nil
}
func (s *testOrgRepo) Delete(context.Context, string) error { return nil }
func (s *testOrgRepo) ListForUser(context.Context, string) ([]organization.OrganizationWithRole, error) {
	return []organization.OrganizationWithRole{{ID: s.org.ID, Slug: s.org.Slug, Name: s.org.Name, Role: organization.RoleOwner}}, nil
}
func (s *testOrgRepo) CreateMembership(context.Context, string, string, organization.Role) (organization.Membership, error) {
	return organization.Membership{}, nil
}
func (s *testOrgRepo) GetMembership(context.Context, string, string) (organization.Membership, error) {
	return organization.Membership{}, nil
}
func (s *testOrgRepo) ListMemberships(context.Context, string) ([]organization.Membership, error) {
	return nil, nil
}
func (s *testOrgRepo) UpdateMembershipRole(context.Context, string, string, organization.Role) (organization.Membership, error) {
	return organization.Membership{}, nil
}
func (s *testOrgRepo) DeleteMembership(context.Context, string, string) error { return nil }
func (s *testOrgRepo) CountOwners(context.Context, string) (int64, error)     { return 1, nil }
func (s *testOrgRepo) CreateInvitation(context.Context, string, string, organization.Role, []byte, time.Time) (organization.Invitation, error) {
	return organization.Invitation{}, nil
}
func (s *testOrgRepo) GetInvitationByID(context.Context, string) (organization.Invitation, error) {
	return organization.Invitation{}, nil
}
func (s *testOrgRepo) GetInvitationByDigest(context.Context, []byte) (organization.Invitation, error) {
	return organization.Invitation{}, nil
}
func (s *testOrgRepo) ListInvitations(context.Context, string) ([]organization.Invitation, error) {
	return nil, nil
}
func (s *testOrgRepo) AcceptInvitation(context.Context, string) (organization.Invitation, error) {
	return organization.Invitation{}, nil
}
func (s *testOrgRepo) RevokeInvitation(context.Context, string, string) error { return nil }

func TestOrganizationAndAuthorizationRouting(t *testing.T) {
	t.Parallel()

	orgID := "a0000000-0000-0000-0000-000000000001"
	annID := "b0000000-0000-0000-0000-000000000002"
	ownerUserID := "user-owner-1"
	memberUserID := "user-member-2"
	outsiderUserID := "user-outsider-3"

	adapter := authorization.NewMemoryAdapter(authorization.DefaultSeedRules())
	enforcer, err := authorization.NewEnforcer(adapter)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}

	// Assign roles in Casbin
	if _, err := enforcer.AddGroupingPolicy(ownerUserID, "owner", orgID); err != nil {
		t.Fatalf("AddGroupingPolicy() error = %v", err)
	}
	if _, err := enforcer.AddGroupingPolicy(memberUserID, "member", orgID); err != nil {
		t.Fatalf("AddGroupingPolicy() error = %v", err)
	}

	authzService := authorization.NewService(enforcer)

	orgRepo := &testOrgRepo{
		org: organization.Organization{
			ID:        orgID,
			Slug:      "test-org",
			Name:      "Test Organization",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	orgService, err := organization.NewService(organization.Dependencies{
		Repository: orgRepo,
		Enforcer:   enforcer,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	annService := announcement.NewService(&announcementRepository{})

	handler, err := httpserver.NewHandler(annService, healthChecker{}, nil, orgService, authzService)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	t.Run("GET /api/organizations requires authentication", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/organizations", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("GET /api/organizations with authenticated user succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/organizations", nil)
		ctx := auth.ContextWithPrincipal(req.Context(), ownerUserID)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("GET /api/organizations/{orgID} for outsider returns 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/organizations/"+orgID, nil)
		ctx := auth.ContextWithPrincipal(req.Context(), outsiderUserID)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("GET /api/organizations/{orgID} for member succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/organizations/"+orgID, nil)
		ctx := auth.ContextWithPrincipal(req.Context(), memberUserID)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("GET /api/organizations/{orgID}/announcements for member succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/organizations/"+orgID+"/announcements", nil)
		ctx := auth.ContextWithPrincipal(req.Context(), memberUserID)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("DELETE /api/organizations/{orgID}/announcements/{annID} for member returns 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/organizations/"+orgID+"/announcements/"+annID, nil)
		ctx := auth.ContextWithPrincipal(req.Context(), memberUserID)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("DELETE /api/organizations/{orgID}/announcements/{annID} for owner succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/organizations/"+orgID+"/announcements/"+annID, nil)
		ctx := auth.ContextWithPrincipal(req.Context(), ownerUserID)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code == http.StatusForbidden || rec.Code == http.StatusUnauthorized {
			t.Errorf("status = %d, expected authorized (non-401/403)", rec.Code)
		}
	})
}
