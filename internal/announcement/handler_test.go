package announcement_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hydrz/starter/internal/announcement"
	"github.com/hydrz/starter/internal/api/announcementsapi"
)

type mockRepository struct {
	items []announcement.Announcement
}

func (m *mockRepository) List(_ context.Context, _ announcement.Filter) ([]announcement.Announcement, int64, error) {
	return m.items, int64(len(m.items)), nil
}

func (m *mockRepository) Get(_ context.Context, id string) (announcement.Announcement, error) {
	for _, item := range m.items {
		if item.ID == id {
			return item, nil
		}
	}
	return announcement.Announcement{}, announcement.ErrNotFound
}

func (m *mockRepository) Create(_ context.Context, input announcement.Input) (announcement.Announcement, error) {
	item := announcement.Announcement{
		ID:        uuid.New().String(),
		Title:     input.Title,
		Content:   input.Content,
		Status:    input.Status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.items = append(m.items, item)
	return item, nil
}

func (m *mockRepository) Update(_ context.Context, id string, input announcement.Input) (announcement.Announcement, error) {
	for i, item := range m.items {
		if item.ID == id {
			item.Title = input.Title
			item.Content = input.Content
			item.Status = input.Status
			item.UpdatedAt = time.Now()
			m.items[i] = item
			return item, nil
		}
	}
	return announcement.Announcement{}, announcement.ErrNotFound
}

func (m *mockRepository) Delete(_ context.Context, id string) error {
	for i, item := range m.items {
		if item.ID == id {
			m.items = append(m.items[:i], m.items[i+1:]...)
			return nil
		}
	}
	return announcement.ErrNotFound
}

func TestHTTPHandlerCRUD(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	service := announcement.NewService(repo)
	handler := announcement.NewHTTPHandler(service)
	ctx := context.Background()

	// 1. Create
	createRes, err := handler.CreateAnnouncement(ctx, &announcementsapi.AnnouncementInput{
		Title:   "Release v1.0",
		Content: "Major release",
		Status:  announcementsapi.AnnouncementStatusPublished,
	})
	if err != nil {
		t.Fatalf("CreateAnnouncement failed: %v", err)
	}
	created, ok := createRes.(*announcementsapi.Announcement)
	if !ok {
		t.Fatalf("expected *Announcement, got %T", createRes)
	}
	if created.Title != "Release v1.0" {
		t.Errorf("title = %q, want Release v1.0", created.Title)
	}

	// 2. List
	listRes, err := handler.ListAnnouncements(ctx, announcementsapi.ListAnnouncementsParams{})
	if err != nil {
		t.Fatalf("ListAnnouncements failed: %v", err)
	}
	page, ok := listRes.(*announcementsapi.PageAnnouncement)
	if !ok {
		t.Fatalf("expected *PageAnnouncement, got %T", listRes)
	}
	if page.Total != 1 {
		t.Errorf("total = %d, want 1", page.Total)
	}

	// 3. Get
	getRes, err := handler.GetAnnouncement(ctx, announcementsapi.GetAnnouncementParams{ID: created.ID})
	if err != nil {
		t.Fatalf("GetAnnouncement failed: %v", err)
	}
	got, ok := getRes.(*announcementsapi.Announcement)
	if !ok {
		t.Fatalf("expected *Announcement, got %T", getRes)
	}
	if got.ID != created.ID {
		t.Errorf("id = %v, want %v", got.ID, created.ID)
	}

	// 4. Update
	updateRes, err := handler.UpdateAnnouncement(ctx, &announcementsapi.AnnouncementInput{
		Title:   "Release v1.1",
		Content: "Patch release",
		Status:  announcementsapi.AnnouncementStatusDraft,
	}, announcementsapi.UpdateAnnouncementParams{ID: created.ID})
	if err != nil {
		t.Fatalf("UpdateAnnouncement failed: %v", err)
	}
	updated, ok := updateRes.(*announcementsapi.Announcement)
	if !ok {
		t.Fatalf("expected *Announcement, got %T", updateRes)
	}
	if updated.Title != "Release v1.1" {
		t.Errorf("title = %q, want Release v1.1", updated.Title)
	}

	// 5. Delete
	deleteRes, err := handler.DeleteAnnouncement(ctx, announcementsapi.DeleteAnnouncementParams{ID: created.ID})
	if err != nil {
		t.Fatalf("DeleteAnnouncement failed: %v", err)
	}
	if _, ok := deleteRes.(*announcementsapi.DeleteAnnouncementNoContent); !ok {
		t.Fatalf("expected *DeleteAnnouncementNoContent, got %T", deleteRes)
	}

	// 6. Get not found
	getNotFoundRes, err := handler.GetAnnouncement(ctx, announcementsapi.GetAnnouncementParams{ID: created.ID})
	if err != nil {
		t.Fatalf("GetAnnouncement failed: %v", err)
	}
	if _, ok := getNotFoundRes.(*announcementsapi.GetAnnouncementNotFound); !ok {
		t.Fatalf("expected *GetAnnouncementNotFound, got %T", getNotFoundRes)
	}
}
