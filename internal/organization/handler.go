package organization

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/hydrz/starter/internal/api/organizationsapi"
	"github.com/hydrz/starter/internal/auth"
)

type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

var _ organizationsapi.Handler = (*HTTPHandler)(nil)

func (h *HTTPHandler) ListUserOrganizations(ctx context.Context) (organizationsapi.ListUserOrganizationsRes, error) {
	userID, ok := auth.PrincipalFromContext(ctx)
	if !ok || userID == "" {
		return &organizationsapi.ApiError{
			Code:    "unauthorized",
			Message: "Authentication is required.",
		}, nil
	}

	items, err := h.service.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	apiItems := make([]organizationsapi.OrganizationWithRole, len(items))
	for i, item := range items {
		apiItems[i] = organizationsapi.OrganizationWithRole{
			ID:        uuid.MustParse(item.ID),
			Slug:      item.Slug,
			Name:      item.Name,
			Role:      organizationsapi.OrganizationRole(item.Role),
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		}
	}
	res := organizationsapi.ListUserOrganizationsOKApplicationJSON(apiItems)
	return &res, nil
}

func (h *HTTPHandler) CreateOrganization(ctx context.Context, req *organizationsapi.CreateOrganizationInput) (organizationsapi.CreateOrganizationRes, error) {
	userID, ok := auth.PrincipalFromContext(ctx)
	if !ok || userID == "" {
		return &organizationsapi.CreateOrganizationUnauthorized{
			Code:    "unauthorized",
			Message: "Authentication is required.",
		}, nil
	}

	org, err := h.service.Create(ctx, userID, CreateInput{
		Slug: req.Slug,
		Name: req.Name,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrSlugTaken):
			return &organizationsapi.CreateOrganizationConflict{
				Code:    "slug_taken",
				Message: "An organization with this slug already exists.",
			}, nil
		case errors.Is(err, ErrInvalidInput):
			return &organizationsapi.CreateOrganizationBadRequest{
				Code:    "invalid_request",
				Message: err.Error(),
			}, nil
		default:
			return nil, err
		}
	}

	res := toAPIOrganization(org)
	return &res, nil
}

func (h *HTTPHandler) GetOrganization(ctx context.Context, params organizationsapi.GetOrganizationParams) (organizationsapi.GetOrganizationRes, error) {
	org, err := h.service.Get(ctx, params.OrganizationId.String())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return &organizationsapi.GetOrganizationNotFound{
				Code:    "not_found",
				Message: "Organization not found.",
			}, nil
		}
		return nil, err
	}

	res := toAPIOrganization(org)
	return &res, nil
}

func (h *HTTPHandler) UpdateOrganization(ctx context.Context, req *organizationsapi.UpdateOrganizationInput, params organizationsapi.UpdateOrganizationParams) (organizationsapi.UpdateOrganizationRes, error) {
	org, err := h.service.Update(ctx, params.OrganizationId.String(), UpdateInput{
		Slug: req.Slug,
		Name: req.Name,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return &organizationsapi.UpdateOrganizationNotFound{
				Code:    "not_found",
				Message: "Organization not found.",
			}, nil
		case errors.Is(err, ErrSlugTaken):
			return &organizationsapi.UpdateOrganizationConflict{
				Code:    "slug_taken",
				Message: "An organization with this slug already exists.",
			}, nil
		case errors.Is(err, ErrInvalidInput):
			return &organizationsapi.UpdateOrganizationBadRequest{
				Code:    "invalid_request",
				Message: err.Error(),
			}, nil
		default:
			return nil, err
		}
	}

	res := toAPIOrganization(org)
	return &res, nil
}

func (h *HTTPHandler) DeleteOrganization(ctx context.Context, params organizationsapi.DeleteOrganizationParams) (organizationsapi.DeleteOrganizationRes, error) {
	if err := h.service.Delete(ctx, params.OrganizationId.String()); err != nil {
		if errors.Is(err, ErrNotFound) {
			return &organizationsapi.DeleteOrganizationNotFound{
				Code:    "not_found",
				Message: "Organization not found.",
			}, nil
		}
		return nil, err
	}

	return &organizationsapi.DeleteOrganizationNoContent{}, nil
}

func (h *HTTPHandler) ListMembers(ctx context.Context, params organizationsapi.ListMembersParams) (organizationsapi.ListMembersRes, error) {
	members, err := h.service.ListMembers(ctx, params.OrganizationId.String())
	if err != nil {
		return nil, err
	}

	apiMembers := make([]organizationsapi.OrganizationMembership, len(members))
	for i, m := range members {
		apiMembers[i] = toAPIMembership(m)
	}
	res := organizationsapi.ListMembersOKApplicationJSON(apiMembers)
	return &res, nil
}

func (h *HTTPHandler) UpdateMemberRole(ctx context.Context, req *organizationsapi.UpdateMembershipRoleInput, params organizationsapi.UpdateMemberRoleParams) (organizationsapi.UpdateMemberRoleRes, error) {
	m, err := h.service.UpdateMemberRole(ctx, params.OrganizationId.String(), params.UserId.String(), Role(req.Role))
	if err != nil {
		switch {
		case errors.Is(err, ErrLastOwner):
			return &organizationsapi.UpdateMemberRoleBadRequest{
				Code:    "last_owner",
				Message: "Cannot demote the last owner of the organization.",
			}, nil
		case errors.Is(err, ErrMembershipNotFound):
			return &organizationsapi.UpdateMemberRoleNotFound{
				Code:    "not_found",
				Message: "Member not found in organization.",
			}, nil
		case errors.Is(err, ErrInvalidInput):
			return &organizationsapi.UpdateMemberRoleBadRequest{
				Code:    "invalid_request",
				Message: err.Error(),
			}, nil
		default:
			return nil, err
		}
	}

	res := toAPIMembership(m)
	return &res, nil
}

func (h *HTTPHandler) RemoveMember(ctx context.Context, params organizationsapi.RemoveMemberParams) (organizationsapi.RemoveMemberRes, error) {
	if err := h.service.DeleteMember(ctx, params.OrganizationId.String(), params.UserId.String()); err != nil {
		switch {
		case errors.Is(err, ErrLastOwner):
			return &organizationsapi.RemoveMemberBadRequest{
				Code:    "last_owner",
				Message: "Cannot remove the last owner of the organization.",
			}, nil
		case errors.Is(err, ErrMembershipNotFound):
			return &organizationsapi.RemoveMemberNotFound{
				Code:    "not_found",
				Message: "Member not found in organization.",
			}, nil
		default:
			return nil, err
		}
	}

	return &organizationsapi.RemoveMemberNoContent{}, nil
}

func (h *HTTPHandler) ListInvitations(ctx context.Context, params organizationsapi.ListInvitationsParams) (organizationsapi.ListInvitationsRes, error) {
	invs, err := h.service.ListInvitations(ctx, params.OrganizationId.String())
	if err != nil {
		return nil, err
	}

	apiInvs := make([]organizationsapi.OrganizationInvitation, len(invs))
	for i, inv := range invs {
		apiInvs[i] = toAPIInvitation(inv)
	}
	res := organizationsapi.ListInvitationsOKApplicationJSON(apiInvs)
	return &res, nil
}

func (h *HTTPHandler) CreateInvitation(ctx context.Context, req *organizationsapi.CreateInvitationInput, params organizationsapi.CreateInvitationParams) (organizationsapi.CreateInvitationRes, error) {
	created, err := h.service.CreateInvitation(ctx, params.OrganizationId.String(), CreateInvitationInput{
		Email: req.Email,
		Role:  Role(req.Role),
	})
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return &organizationsapi.CreateInvitationBadRequest{
				Code:    "invalid_request",
				Message: err.Error(),
			}, nil
		}
		return nil, err
	}

	return &organizationsapi.CreatedInvitationResponse{
		Invitation: toAPIInvitation(created.Invitation),
		Token:      created.Token,
	}, nil
}

func (h *HTTPHandler) RevokeInvitation(ctx context.Context, params organizationsapi.RevokeInvitationParams) (organizationsapi.RevokeInvitationRes, error) {
	if err := h.service.RevokeInvitation(ctx, params.OrganizationId.String(), params.InvitationId.String()); err != nil {
		if errors.Is(err, ErrNotFound) {
			return &organizationsapi.RevokeInvitationNotFound{
				Code:    "not_found",
				Message: "Invitation not found.",
			}, nil
		}
		return nil, err
	}

	return &organizationsapi.RevokeInvitationNoContent{}, nil
}

func (h *HTTPHandler) AcceptInvitation(ctx context.Context, req *organizationsapi.AcceptInvitationInput) (organizationsapi.AcceptInvitationRes, error) {
	userID, ok := auth.PrincipalFromContext(ctx)
	if !ok || userID == "" {
		return &organizationsapi.AcceptInvitationUnauthorized{
			Code:    "unauthorized",
			Message: "Authentication is required.",
		}, nil
	}

	m, err := h.service.AcceptInvitation(ctx, userID, req.Token)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvitationExpired):
			return &organizationsapi.AcceptInvitationBadRequest{
				Code:    "invalid_token",
				Message: "Invitation is expired, revoked, or invalid.",
			}, nil
		case errors.Is(err, ErrAlreadyMember):
			return &organizationsapi.AcceptInvitationBadRequest{
				Code:    "already_member",
				Message: "User is already a member of this organization.",
			}, nil
		default:
			return nil, err
		}
	}

	res := toAPIMembership(m)
	return &res, nil
}

func toAPIOrganization(org Organization) organizationsapi.Organization {
	return organizationsapi.Organization{
		ID:        uuid.MustParse(org.ID),
		Slug:      org.Slug,
		Name:      org.Name,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}
}

func toAPIMembership(m Membership) organizationsapi.OrganizationMembership {
	var emailOpt organizationsapi.OptString
	if m.Email != "" {
		emailOpt = organizationsapi.NewOptString(m.Email)
	}
	return organizationsapi.OrganizationMembership{
		ID:             uuid.MustParse(m.ID),
		OrganizationId: uuid.MustParse(m.OrganizationID),
		UserId:         uuid.MustParse(m.UserID),
		Email:          emailOpt,
		Role:           organizationsapi.OrganizationRole(m.Role),
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func toAPIInvitation(inv Invitation) organizationsapi.OrganizationInvitation {
	return organizationsapi.OrganizationInvitation{
		ID:             uuid.MustParse(inv.ID),
		OrganizationId: uuid.MustParse(inv.OrganizationID),
		Email:          inv.Email,
		Role:           organizationsapi.OrganizationRole(inv.Role),
		ExpiresAt:      inv.ExpiresAt,
		CreatedAt:      inv.CreatedAt,
	}
}
