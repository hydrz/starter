package announcement_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hydrz/starter/internal/announcement"
)

type repositoryStub struct {
	created announcement.Input
}

func (*repositoryStub) List(context.Context, uuid.UUID, announcement.Filter) ([]announcement.Announcement, int64, error) {
	return []announcement.Announcement{}, 0, nil
}
func (*repositoryStub) Get(context.Context, uuid.UUID, string) (announcement.Announcement, error) {
	return announcement.Announcement{}, announcement.ErrNotFound
}
func (repository *repositoryStub) Create(_ context.Context, _ uuid.UUID, input announcement.Input) (announcement.Announcement, error) {
	repository.created = input
	return announcement.Announcement{Title: input.Title}, nil
}
func (*repositoryStub) Update(context.Context, uuid.UUID, string, announcement.Input) (announcement.Announcement, error) {
	return announcement.Announcement{}, errors.New("not implemented")
}
func (*repositoryStub) Delete(context.Context, uuid.UUID, string) error { return nil }

func TestCreateValidatesAndNormalizesInput(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	repository := &repositoryStub{}
	service := announcement.NewService(repository)
	item, err := service.Create(context.Background(), orgID, announcement.Input{
		Title:   "  Planned maintenance  ",
		Content: "  The service will be unavailable.  ",
		Status:  announcement.StatusDraft,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if item.Title != "Planned maintenance" {
		t.Errorf("title = %q, want trimmed title", item.Title)
	}
}

func TestCreateRejectsEmptyTitle(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	service := announcement.NewService(&repositoryStub{})
	_, err := service.Create(context.Background(), orgID, announcement.Input{
		Content: "Content",
		Status:  announcement.StatusDraft,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
}

func TestListRejectsOversizedPage(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	service := announcement.NewService(&repositoryStub{})
	_, err := service.List(context.Background(), orgID, announcement.Filter{Limit: announcement.MaxPageSize + 1})
	if err == nil {
		t.Fatal("List() error = nil, want validation error")
	}
}
