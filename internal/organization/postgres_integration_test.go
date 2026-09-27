//go:build integration

package organization_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/testdb"
	"github.com/hydrz/starter/internal/store"
)

func TestOrganizationMembershipAndInvitationQueries(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := testdb.Context(t)

	organization, err := db.Queries.CreateOrganization(ctx, store.CreateOrganizationParams{
		Slug: "org-" + uuid.NewString()[:8], Name: "Integration Organization",
	})
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	owner := createUserForOrganization(t, ctx, db.Queries)
	member := createUserForOrganization(t, ctx, db.Queries)

	if _, err := db.Queries.CreateMembership(ctx, store.CreateMembershipParams{OrganizationID: organization.ID, UserID: owner.ID, Role: "owner"}); err != nil {
		t.Fatalf("create owner membership: %v", err)
	}
	if _, err := db.Queries.CreateMembership(ctx, store.CreateMembershipParams{OrganizationID: organization.ID, UserID: member.ID, Role: "member"}); err != nil {
		t.Fatalf("create member membership: %v", err)
	}
	if owners, err := db.Queries.CountOrganizationOwners(ctx, organization.ID); err != nil || owners != 1 {
		t.Fatalf("count owners = %d, %v", owners, err)
	}
	if listed, err := db.Queries.ListOrganizationsForUser(ctx, owner.ID); err != nil || len(listed) != 1 {
		t.Fatalf("list organizations for user = %d items, %v", len(listed), err)
	}
	updated, err := db.Queries.UpdateMembershipRole(ctx, store.UpdateMembershipRoleParams{OrganizationID: organization.ID, UserID: member.ID, Role: "admin"})
	if err != nil || updated.Role != "admin" {
		t.Fatalf("update membership role = %#v, %v", updated, err)
	}
	if deleted, err := db.Queries.DeleteMembership(ctx, store.DeleteMembershipParams{OrganizationID: organization.ID, UserID: member.ID}); err != nil || deleted != 1 {
		t.Fatalf("delete membership = %d, %v", deleted, err)
	}

	invitation, err := db.Queries.CreateInvitation(ctx, store.CreateInvitationParams{
		OrganizationID: organization.ID, Email: "invitee@example.com", Role: "member", TokenDigest: []byte("invitation-digest"), ExpiresAt: future(),
	})
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}
	if pending, err := db.Queries.ListInvitations(ctx, organization.ID); err != nil || len(pending) != 1 {
		t.Fatalf("list pending invitations = %d items, %v", len(pending), err)
	}
	accepted, err := db.Queries.AcceptInvitation(ctx, invitation.ID)
	if err != nil || !accepted.AcceptedAt.Valid {
		t.Fatalf("accept invitation = %#v, %v", accepted, err)
	}
	if again, err := db.Queries.AcceptInvitation(ctx, invitation.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("re-accept invitation = %#v, %v; want pgx.ErrNoRows", again, err)
	}

	revocable, err := db.Queries.CreateInvitation(ctx, store.CreateInvitationParams{
		OrganizationID: organization.ID, Email: "revoked@example.com", Role: "viewer", TokenDigest: []byte("revoked-digest"), ExpiresAt: future(),
	})
	if err != nil {
		t.Fatalf("create revocable invitation: %v", err)
	}
	if revoked, err := db.Queries.RevokeInvitation(ctx, store.RevokeInvitationParams{ID: revocable.ID, OrganizationID: organization.ID}); err != nil || revoked != 1 {
		t.Fatalf("revoke invitation = %d, %v", revoked, err)
	}
	renamed, err := db.Queries.UpdateOrganization(ctx, store.UpdateOrganizationParams{ID: organization.ID, Name: "Renamed", Slug: organization.Slug})
	if err != nil || renamed.Name != "Renamed" {
		t.Fatalf("rename organization = %#v, %v", renamed, err)
	}
	if deleted, err := db.Queries.DeleteOrganization(ctx, organization.ID); err != nil || deleted != 1 {
		t.Fatalf("delete organization = %d, %v", deleted, err)
	}
}

func createUserForOrganization(t *testing.T, ctx context.Context, queries *store.Queries) store.User {
	t.Helper()
	user, err := queries.CreateUser(ctx, store.CreateUserParams{Email: uuid.NewString()[:12] + "@example.com", PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func future() pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}
}
