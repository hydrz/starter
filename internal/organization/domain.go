package organization

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidInput       = errors.New("organization: invalid input")
	ErrNotFound           = errors.New("organization: not found")
	ErrSlugTaken          = errors.New("organization: slug is already taken")
	ErrLastOwner          = errors.New("organization: cannot remove or demote the last owner")
	ErrInvitationExpired  = errors.New("organization: invitation has expired or is invalid")
	ErrAlreadyMember      = errors.New("organization: user is already a member")
	ErrMembershipNotFound = errors.New("organization: membership not found")
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

func (r Role) Valid() bool {
	return r == RoleOwner || r == RoleAdmin || r == RoleMember || r == RoleViewer
}

type Organization struct {
	ID        string
	Slug      string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrganizationWithRole struct {
	ID        string
	Slug      string
	Name      string
	Role      Role
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Membership struct {
	ID             string
	OrganizationID string
	UserID         string
	Email          string
	Role           Role
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Invitation struct {
	ID             string
	OrganizationID string
	Email          string
	Role           Role
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

type CreatedInvitation struct {
	Invitation Invitation
	Token      string
}

type CreateInput struct {
	Slug string
	Name string
}

type UpdateInput struct {
	Slug string
	Name string
}

type CreateInvitationInput struct {
	Email string
	Role  Role
}

type Repository interface {
	Create(ctx context.Context, slug, name string) (Organization, error)
	GetByID(ctx context.Context, id string) (Organization, error)
	GetBySlug(ctx context.Context, slug string) (Organization, error)
	Update(ctx context.Context, id, slug, name string) (Organization, error)
	Delete(ctx context.Context, id string) error
	ListForUser(ctx context.Context, userID string) ([]OrganizationWithRole, error)

	CreateMembership(ctx context.Context, orgID, userID string, role Role) (Membership, error)
	GetMembership(ctx context.Context, orgID, userID string) (Membership, error)
	ListMemberships(ctx context.Context, orgID string) ([]Membership, error)
	UpdateMembershipRole(ctx context.Context, orgID, userID string, role Role) (Membership, error)
	DeleteMembership(ctx context.Context, orgID, userID string) error
	CountOwners(ctx context.Context, orgID string) (int64, error)

	CreateInvitation(ctx context.Context, orgID, email string, role Role, tokenDigest []byte, expiresAt time.Time) (Invitation, error)
	GetInvitationByID(ctx context.Context, id string) (Invitation, error)
	GetInvitationByDigest(ctx context.Context, digest []byte) (Invitation, error)
	ListInvitations(ctx context.Context, orgID string) ([]Invitation, error)
	AcceptInvitation(ctx context.Context, id string) (Invitation, error)
	RevokeInvitation(ctx context.Context, orgID, id string) error
}
