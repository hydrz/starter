package organization_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hydrz/starter/internal/auth"
	"github.com/hydrz/starter/internal/authorization"
	"github.com/hydrz/starter/internal/organization"
)

type memoryRepository struct {
	orgs        map[string]organization.Organization
	memberships map[string]map[string]organization.Membership // orgID -> userID -> Membership
	invitations map[string]organization.Invitation
	invByDigest map[string]organization.Invitation
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		orgs:        make(map[string]organization.Organization),
		memberships: make(map[string]map[string]organization.Membership),
		invitations: make(map[string]organization.Invitation),
		invByDigest: make(map[string]organization.Invitation),
	}
}

func (m *memoryRepository) Create(_ context.Context, slug, name string) (organization.Organization, error) {
	for _, o := range m.orgs {
		if o.Slug == slug {
			return organization.Organization{}, organization.ErrSlugTaken
		}
	}
	org := organization.Organization{
		ID:        uuid.New().String(),
		Slug:      slug,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.orgs[org.ID] = org
	m.memberships[org.ID] = make(map[string]organization.Membership)
	return org, nil
}

func (m *memoryRepository) GetByID(_ context.Context, id string) (organization.Organization, error) {
	org, ok := m.orgs[id]
	if !ok {
		return organization.Organization{}, organization.ErrNotFound
	}
	return org, nil
}

func (m *memoryRepository) GetBySlug(_ context.Context, slug string) (organization.Organization, error) {
	for _, o := range m.orgs {
		if o.Slug == slug {
			return o, nil
		}
	}
	return organization.Organization{}, organization.ErrNotFound
}

func (m *memoryRepository) Update(_ context.Context, id, slug, name string) (organization.Organization, error) {
	org, ok := m.orgs[id]
	if !ok {
		return organization.Organization{}, organization.ErrNotFound
	}
	for _, o := range m.orgs {
		if o.ID != id && o.Slug == slug {
			return organization.Organization{}, organization.ErrSlugTaken
		}
	}
	org.Slug = slug
	org.Name = name
	org.UpdatedAt = time.Now()
	m.orgs[id] = org
	return org, nil
}

func (m *memoryRepository) Delete(_ context.Context, id string) error {
	if _, ok := m.orgs[id]; !ok {
		return organization.ErrNotFound
	}
	delete(m.orgs, id)
	delete(m.memberships, id)
	return nil
}

func (m *memoryRepository) ListForUser(_ context.Context, userID string) ([]organization.OrganizationWithRole, error) {
	var result []organization.OrganizationWithRole
	for orgID, members := range m.memberships {
		if mem, ok := members[userID]; ok {
			org := m.orgs[orgID]
			result = append(result, organization.OrganizationWithRole{
				ID:        org.ID,
				Slug:      org.Slug,
				Name:      org.Name,
				Role:      mem.Role,
				CreatedAt: org.CreatedAt,
				UpdatedAt: org.UpdatedAt,
			})
		}
	}
	return result, nil
}

func (m *memoryRepository) CreateMembership(_ context.Context, orgID, userID string, role organization.Role) (organization.Membership, error) {
	members, ok := m.memberships[orgID]
	if !ok {
		return organization.Membership{}, organization.ErrNotFound
	}
	if _, exists := members[userID]; exists {
		return organization.Membership{}, organization.ErrAlreadyMember
	}
	mem := organization.Membership{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	members[userID] = mem
	return mem, nil
}

func (m *memoryRepository) GetMembership(_ context.Context, orgID, userID string) (organization.Membership, error) {
	members, ok := m.memberships[orgID]
	if !ok {
		return organization.Membership{}, organization.ErrMembershipNotFound
	}
	mem, ok := members[userID]
	if !ok {
		return organization.Membership{}, organization.ErrMembershipNotFound
	}
	return mem, nil
}

func (m *memoryRepository) ListMemberships(_ context.Context, orgID string) ([]organization.Membership, error) {
	members, ok := m.memberships[orgID]
	if !ok {
		return nil, organization.ErrNotFound
	}
	var list []organization.Membership
	for _, mem := range members {
		list = append(list, mem)
	}
	return list, nil
}

func (m *memoryRepository) UpdateMembershipRole(_ context.Context, orgID, userID string, role organization.Role) (organization.Membership, error) {
	members, ok := m.memberships[orgID]
	if !ok {
		return organization.Membership{}, organization.ErrMembershipNotFound
	}
	mem, ok := members[userID]
	if !ok {
		return organization.Membership{}, organization.ErrMembershipNotFound
	}
	mem.Role = role
	mem.UpdatedAt = time.Now()
	members[userID] = mem
	return mem, nil
}

func (m *memoryRepository) DeleteMembership(_ context.Context, orgID, userID string) error {
	members, ok := m.memberships[orgID]
	if !ok {
		return organization.ErrMembershipNotFound
	}
	if _, ok := members[userID]; !ok {
		return organization.ErrMembershipNotFound
	}
	delete(members, userID)
	return nil
}

func (m *memoryRepository) CountOwners(_ context.Context, orgID string) (int64, error) {
	members, ok := m.memberships[orgID]
	if !ok {
		return 0, nil
	}
	var count int64
	for _, mem := range members {
		if mem.Role == organization.RoleOwner {
			count++
		}
	}
	return count, nil
}

func (m *memoryRepository) CreateInvitation(_ context.Context, orgID, email string, role organization.Role, tokenDigest []byte, expiresAt time.Time) (organization.Invitation, error) {
	inv := organization.Invitation{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		Email:          email,
		Role:           role,
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now(),
	}
	m.invitations[inv.ID] = inv
	m.invByDigest[string(tokenDigest)] = inv
	return inv, nil
}

func (m *memoryRepository) GetInvitationByID(_ context.Context, id string) (organization.Invitation, error) {
	inv, ok := m.invitations[id]
	if !ok {
		return organization.Invitation{}, organization.ErrNotFound
	}
	return inv, nil
}

func (m *memoryRepository) GetInvitationByDigest(_ context.Context, digest []byte) (organization.Invitation, error) {
	inv, ok := m.invByDigest[string(digest)]
	if !ok {
		return organization.Invitation{}, organization.ErrNotFound
	}
	return inv, nil
}

func (m *memoryRepository) ListInvitations(_ context.Context, orgID string) ([]organization.Invitation, error) {
	var list []organization.Invitation
	for _, inv := range m.invitations {
		if inv.OrganizationID == orgID {
			list = append(list, inv)
		}
	}
	return list, nil
}

func (m *memoryRepository) AcceptInvitation(_ context.Context, id string) (organization.Invitation, error) {
	inv, ok := m.invitations[id]
	if !ok {
		return organization.Invitation{}, organization.ErrNotFound
	}
	delete(m.invitations, id)
	return inv, nil
}

func (m *memoryRepository) RevokeInvitation(_ context.Context, orgID, id string) error {
	inv, ok := m.invitations[id]
	if !ok || inv.OrganizationID != orgID {
		return organization.ErrNotFound
	}
	delete(m.invitations, id)
	return nil
}

func setupTestOrgService(t *testing.T) (*organization.Service, *memoryRepository, *authorization.Service) {
	t.Helper()

	repo := newMemoryRepository()
	authService := authorization.NewService(nil)
	adapter := authorization.NewMemoryAdapter(authorization.DefaultSeedRules())
	enforcer, err := authorization.NewEnforcer(adapter)
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}
	authService = authorization.NewService(enforcer)

	orgService, err := organization.NewService(organization.Dependencies{
		Repository: repo,
		Enforcer:   enforcer,
	})
	if err != nil {
		t.Fatalf("create org service: %v", err)
	}

	return orgService, repo, authService
}

func TestOrganizationService_CRUD(t *testing.T) {
	service, _, authService := setupTestOrgService(t)
	ctx := context.Background()
	userID := "user-alice"

	// 1. Create Organization
	org, err := service.Create(ctx, userID, organization.CreateInput{
		Slug: "acme-corp",
		Name: "Acme Corporation",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if org.Slug != "acme-corp" || org.Name != "Acme Corporation" {
		t.Errorf("unexpected org: %+v", org)
	}

	// Verify Casbin role assignment
	roles := authService.GetRolesForUserInDomain(userID, org.ID)
	if len(roles) != 1 || roles[0] != "owner" {
		t.Errorf("roles = %v, want [owner]", roles)
	}

	// 2. Get Organization
	got, err := service.Get(ctx, org.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != org.ID {
		t.Errorf("got id = %s, want %s", got.ID, org.ID)
	}

	// 3. Update Organization
	updated, err := service.Update(ctx, org.ID, organization.UpdateInput{
		Slug: "acme-inc",
		Name: "Acme Inc.",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Slug != "acme-inc" || updated.Name != "Acme Inc." {
		t.Errorf("unexpected updated org: %+v", updated)
	}

	// 4. ListForUser
	list, err := service.ListForUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}
	if len(list) != 1 || list[0].Role != organization.RoleOwner {
		t.Errorf("list = %+v, want 1 item with owner role", list)
	}

	// 5. Delete Organization
	if err := service.Delete(ctx, org.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	rolesAfterDelete := authService.GetRolesForUserInDomain(userID, org.ID)
	if len(rolesAfterDelete) != 0 {
		t.Errorf("roles after delete = %v, want empty", rolesAfterDelete)
	}
}

func TestOrganizationService_CreatePersonalOrg(t *testing.T) {
	service, _, authService := setupTestOrgService(t)
	ctx := context.Background()

	user := auth.User{
		ID:    "user-uuid-1234",
		Email: "alice@example.com",
	}

	if err := service.CreatePersonalOrg(ctx, user); err != nil {
		t.Fatalf("CreatePersonalOrg() error = %v", err)
	}

	list, err := service.ListForUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].Role != organization.RoleOwner {
		t.Errorf("role = %s, want owner", list[0].Role)
	}
	if list[0].Slug != "personal-user-uuid-1234" {
		t.Errorf("slug = %s, want personal-user-uuid-1234", list[0].Slug)
	}

	// Verify Casbin role assignment
	roles := authService.GetRolesForUserInDomain(user.ID, list[0].ID)
	if len(roles) != 1 || roles[0] != "owner" {
		t.Errorf("roles = %v, want [owner]", roles)
	}
}

func TestOrganizationService_LastOwnerProtection(t *testing.T) {
	service, _, _ := setupTestOrgService(t)
	ctx := context.Background()
	ownerID := "user-owner"

	org, err := service.Create(ctx, ownerID, organization.CreateInput{
		Slug: "solo-org",
		Name: "Solo Org",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Cannot demote the only owner
	_, err = service.UpdateMemberRole(ctx, org.ID, ownerID, organization.RoleAdmin)
	if err != organization.ErrLastOwner {
		t.Errorf("UpdateMemberRole() error = %v, want ErrLastOwner", err)
	}

	// Cannot delete the only owner
	err = service.DeleteMember(ctx, org.ID, ownerID)
	if err != organization.ErrLastOwner {
		t.Errorf("DeleteMember() error = %v, want ErrLastOwner", err)
	}
}

func TestOrganizationService_Invitations(t *testing.T) {
	service, _, authService := setupTestOrgService(t)
	ctx := context.Background()
	ownerID := "user-owner"
	inviteeID := "user-bob"

	org, err := service.Create(ctx, ownerID, organization.CreateInput{
		Slug: "invite-org",
		Name: "Invite Org",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 1. Create invitation
	createdInv, err := service.CreateInvitation(ctx, org.ID, organization.CreateInvitationInput{
		Email: "bob@example.com",
		Role:  organization.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("CreateInvitation() error = %v", err)
	}
	if createdInv.Token == "" {
		t.Fatal("expected non-empty token")
	}

	// 2. Accept invitation
	membership, err := service.AcceptInvitation(ctx, inviteeID, createdInv.Token)
	if err != nil {
		t.Fatalf("AcceptInvitation() error = %v", err)
	}
	if membership.Role != organization.RoleAdmin {
		t.Errorf("role = %s, want admin", membership.Role)
	}

	// Verify Casbin role assignment
	roles := authService.GetRolesForUserInDomain(inviteeID, org.ID)
	if len(roles) != 1 || roles[0] != "admin" {
		t.Errorf("roles = %v, want [admin]", roles)
	}

	// 3. Accepting again fails
	_, err = service.AcceptInvitation(ctx, inviteeID, createdInv.Token)
	if err == nil {
		t.Fatal("expected error on re-accepting invitation")
	}
}
