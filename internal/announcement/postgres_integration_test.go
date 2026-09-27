//go:build integration

package announcement_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/testdb"
	"github.com/hydrz/starter/internal/store"
)

func TestAnnouncementQueriesCRUDAndOrganizationScope(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := testdb.Context(t)
	org := createOrganization(t, ctx, db.Queries, "announcements")
	otherOrg := createOrganization(t, ctx, db.Queries, "other-announcements")

	created, err := db.Queries.CreateAnnouncement(ctx, store.CreateAnnouncementParams{
		OrganizationID: org,
		Title:          "Initial title",
		Content:        "Initial content",
		Status:         store.AnnouncementStatusDraft,
	})
	if err != nil {
		t.Fatalf("create announcement: %v", err)
	}
	if _, err := db.Queries.CreateAnnouncement(ctx, store.CreateAnnouncementParams{
		OrganizationID: otherOrg, Title: "Other", Content: "Other content", Status: store.AnnouncementStatusPublished,
	}); err != nil {
		t.Fatalf("create other organization announcement: %v", err)
	}

	count, err := db.Queries.CountAnnouncements(ctx, store.CountAnnouncementsParams{OrganizationID: org})
	if err != nil || count != 1 {
		t.Fatalf("count announcements = %d, %v; want 1, nil", count, err)
	}
	items, err := db.Queries.ListAnnouncements(ctx, store.ListAnnouncementsParams{OrganizationID: org, Limit: 10})
	if err != nil || len(items) != 1 || items[0].ID != created.ID {
		t.Fatalf("list announcements = %#v, %v; want created announcement", items, err)
	}
	if _, err := db.Queries.GetAnnouncement(ctx, store.GetAnnouncementParams{ID: created.ID, OrganizationID: otherOrg}); err != pgx.ErrNoRows {
		t.Fatalf("cross-organization get error = %v, want pgx.ErrNoRows", err)
	}
	updated, err := db.Queries.UpdateAnnouncement(ctx, store.UpdateAnnouncementParams{
		ID: created.ID, OrganizationID: org, Title: "Updated title", Content: "Updated content", Status: store.AnnouncementStatusPublished,
	})
	if err != nil || updated.Status != store.AnnouncementStatusPublished {
		t.Fatalf("update announcement = %#v, %v", updated, err)
	}
	deleted, err := db.Queries.DeleteAnnouncement(ctx, store.DeleteAnnouncementParams{ID: created.ID, OrganizationID: org})
	if err != nil || deleted != 1 {
		t.Fatalf("delete announcement = %d, %v; want 1, nil", deleted, err)
	}
}

func createOrganization(t *testing.T, ctx context.Context, queries *store.Queries, suffix string) pgtype.UUID {
	t.Helper()
	organization, err := queries.CreateOrganization(ctx, store.CreateOrganizationParams{
		Slug: "integration-" + suffix,
		Name: "Integration " + suffix,
	})
	if err != nil {
		t.Fatalf("create organization %q: %v", suffix, err)
	}
	return organization.ID
}
