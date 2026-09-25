package announcement_test

import (
	"context"
	"errors"
	"testing"

	"github.com/example/enterprise-platform/internal/announcement"
)

type repositoryStub struct {
	created announcement.Input
}

func (*repositoryStub) List(context.Context, announcement.Filter) ([]announcement.Announcement, int64, error) {
	return []announcement.Announcement{}, 0, nil
}
func (*repositoryStub) Get(context.Context, string) (announcement.Announcement, error) {
	return announcement.Announcement{}, announcement.ErrNotFound
}
func (repository *repositoryStub) Create(_ context.Context, input announcement.Input) (announcement.Announcement, error) {
	repository.created = input
	return announcement.Announcement{Title: input.Title}, nil
}
func (*repositoryStub) Update(context.Context, string, announcement.Input) (announcement.Announcement, error) {
	return announcement.Announcement{}, errors.New("not implemented")
}
func (*repositoryStub) Delete(context.Context, string) error { return nil }

func TestCreateValidatesAndNormalizesInput(t *testing.T) {
	t.Parallel()

	repository := &repositoryStub{}
	service := announcement.NewService(repository)
	item, err := service.Create(context.Background(), announcement.Input{
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

	service := announcement.NewService(&repositoryStub{})
	_, err := service.Create(context.Background(), announcement.Input{
		Content: "Content",
		Status:  announcement.StatusDraft,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
}

func TestListRejectsOversizedPage(t *testing.T) {
	t.Parallel()

	service := announcement.NewService(&repositoryStub{})
	_, err := service.List(context.Background(), announcement.Filter{Limit: announcement.MaxPageSize + 1})
	if err == nil {
		t.Fatal("List() error = nil, want validation error")
	}
}
