package announcement

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hydrz/starter/internal/api/announcementsapi"
)

type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

var _ announcementsapi.Handler = (*HTTPHandler)(nil)

func (h *HTTPHandler) ListAnnouncements(ctx context.Context, params announcementsapi.ListAnnouncementsParams) (announcementsapi.ListAnnouncementsRes, error) {
	filter := Filter{
		Limit:  params.Limit.Or(DefaultPageSize),
		Offset: params.Offset.Or(0),
	}
	if params.Status.IsSet() {
		status := Status(params.Status.Value)
		filter.Status = &status
	}

	page, err := h.service.List(ctx, params.OrganizationId, filter)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return &announcementsapi.ListAnnouncementsBadRequest{Code: "invalid_request", Message: err.Error()}, nil
		}
		return nil, err
	}

	items := make([]announcementsapi.Announcement, len(page.Items))
	for i, item := range page.Items {
		items[i] = toAPIAnnouncement(item)
	}

	return &announcementsapi.PageAnnouncement{
		Items:  items,
		Total:  page.Total,
		Limit:  page.Limit,
		Offset: page.Offset,
	}, nil
}

func (h *HTTPHandler) CreateAnnouncement(ctx context.Context, req *announcementsapi.AnnouncementInput, params announcementsapi.CreateAnnouncementParams) (announcementsapi.CreateAnnouncementRes, error) {
	input := Input{
		Title:   req.Title,
		Content: req.Content,
		Status:  Status(req.Status),
	}

	item, err := h.service.Create(ctx, params.OrganizationId, input)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return &announcementsapi.CreateAnnouncementBadRequest{Code: "invalid_request", Message: err.Error()}, nil
		}
		return nil, err
	}

	res := toAPIAnnouncement(item)
	return &res, nil
}

func (h *HTTPHandler) GetAnnouncement(ctx context.Context, params announcementsapi.GetAnnouncementParams) (announcementsapi.GetAnnouncementRes, error) {
	item, err := h.service.Get(ctx, params.OrganizationId, params.ID.String())
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return &announcementsapi.GetAnnouncementNotFound{Code: "not_found", Message: "Announcement was not found."}, nil
		case errors.Is(err, ErrInvalidInput):
			return &announcementsapi.GetAnnouncementBadRequest{Code: "invalid_request", Message: err.Error()}, nil
		default:
			return nil, err
		}
	}

	res := toAPIAnnouncement(item)
	return &res, nil
}

func (h *HTTPHandler) UpdateAnnouncement(ctx context.Context, req *announcementsapi.AnnouncementInput, params announcementsapi.UpdateAnnouncementParams) (announcementsapi.UpdateAnnouncementRes, error) {
	input := Input{
		Title:   req.Title,
		Content: req.Content,
		Status:  Status(req.Status),
	}

	item, err := h.service.Update(ctx, params.OrganizationId, params.ID.String(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return &announcementsapi.UpdateAnnouncementNotFound{Code: "not_found", Message: "Announcement was not found."}, nil
		case errors.Is(err, ErrInvalidInput):
			return &announcementsapi.UpdateAnnouncementBadRequest{Code: "invalid_request", Message: err.Error()}, nil
		default:
			return nil, err
		}
	}

	res := toAPIAnnouncement(item)
	return &res, nil
}

func (h *HTTPHandler) DeleteAnnouncement(ctx context.Context, params announcementsapi.DeleteAnnouncementParams) (announcementsapi.DeleteAnnouncementRes, error) {
	if err := h.service.Delete(ctx, params.OrganizationId, params.ID.String()); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return &announcementsapi.DeleteAnnouncementNotFound{Code: "not_found", Message: "Announcement was not found."}, nil
		case errors.Is(err, ErrInvalidInput):
			return &announcementsapi.DeleteAnnouncementBadRequest{Code: "invalid_request", Message: err.Error()}, nil
		default:
			return nil, err
		}
	}

	return &announcementsapi.DeleteAnnouncementNoContent{}, nil
}

func toAPIAnnouncement(item Announcement) announcementsapi.Announcement {
	orgID, _ := uuid.Parse(item.OrganizationID)
	return announcementsapi.Announcement{
		ID:             uuid.MustParse(item.ID),
		OrganizationId: orgID,
		Title:          item.Title,
		Content:        item.Content,
		Status:         announcementsapi.AnnouncementStatus(item.Status),
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}
