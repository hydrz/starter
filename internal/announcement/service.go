package announcement

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

var (
	ErrInvalidInput = errors.New("invalid announcement input")
	ErrNotFound     = errors.New("announcement not found")
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
)

type Announcement struct {
	ID             string
	OrganizationID string
	Title          string
	Content        string
	Status         Status
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Input struct {
	Title   string
	Content string
	Status  Status
}

type Filter struct {
	Limit  int32
	Offset int32
	Status *Status
}

type Page struct {
	Items  []Announcement
	Total  int64
	Limit  int32
	Offset int32
}

type Repository interface {
	List(context.Context, uuid.UUID, Filter) ([]Announcement, int64, error)
	Get(context.Context, uuid.UUID, string) (Announcement, error)
	Create(context.Context, uuid.UUID, Input) (Announcement, error)
	Update(context.Context, uuid.UUID, string, Input) (Announcement, error)
	Delete(context.Context, uuid.UUID, string) error
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) List(ctx context.Context, organizationID uuid.UUID, filter Filter) (Page, error) {
	if filter.Limit == 0 {
		filter.Limit = DefaultPageSize
	}
	if filter.Limit < 1 || filter.Limit > MaxPageSize {
		return Page{}, fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalidInput, MaxPageSize)
	}
	if filter.Offset < 0 {
		return Page{}, fmt.Errorf("%w: offset must be greater than or equal to 0", ErrInvalidInput)
	}
	if filter.Status != nil && !filter.Status.Valid() {
		return Page{}, fmt.Errorf("%w: status must be draft or published", ErrInvalidInput)
	}

	items, total, err := service.repository.List(ctx, organizationID, filter)
	if err != nil {
		return Page{}, fmt.Errorf("list announcements: %w", err)
	}
	return Page{Items: items, Total: total, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func (service *Service) Get(ctx context.Context, organizationID uuid.UUID, id string) (Announcement, error) {
	item, err := service.repository.Get(ctx, organizationID, id)
	if err != nil {
		return Announcement{}, fmt.Errorf("get announcement: %w", err)
	}
	return item, nil
}

func (service *Service) Create(ctx context.Context, organizationID uuid.UUID, input Input) (Announcement, error) {
	input, err := validateInput(input)
	if err != nil {
		return Announcement{}, err
	}
	item, err := service.repository.Create(ctx, organizationID, input)
	if err != nil {
		return Announcement{}, fmt.Errorf("create announcement: %w", err)
	}
	return item, nil
}

func (service *Service) Update(ctx context.Context, organizationID uuid.UUID, id string, input Input) (Announcement, error) {
	input, err := validateInput(input)
	if err != nil {
		return Announcement{}, err
	}
	item, err := service.repository.Update(ctx, organizationID, id, input)
	if err != nil {
		return Announcement{}, fmt.Errorf("update announcement: %w", err)
	}
	return item, nil
}

func (service *Service) Delete(ctx context.Context, organizationID uuid.UUID, id string) error {
	if err := service.repository.Delete(ctx, organizationID, id); err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}
	return nil
}

func (status Status) Valid() bool {
	return status == StatusDraft || status == StatusPublished
}

func validateInput(input Input) (Input, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	if input.Title == "" || len([]rune(input.Title)) > 120 {
		return Input{}, fmt.Errorf("%w: title must contain between 1 and 120 characters", ErrInvalidInput)
	}
	if input.Content == "" || len([]rune(input.Content)) > 10000 {
		return Input{}, fmt.Errorf("%w: content must contain between 1 and 10000 characters", ErrInvalidInput)
	}
	if !input.Status.Valid() {
		return Input{}, fmt.Errorf("%w: status must be draft or published", ErrInvalidInput)
	}
	return input, nil
}
