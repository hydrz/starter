package organization

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"

	"github.com/hydrz/starter/internal/auth"
	"github.com/hydrz/starter/internal/platform/database"
)

const defaultInvitationTTL = 7 * 24 * time.Hour

type Dependencies struct {
	Repository    Repository
	Transactor    database.Transactor
	Enforcer      *casbin.Enforcer
	Digester      *auth.SecretDigester
	Clock         auth.Clock
	InvitationTTL time.Duration
}

type Service struct {
	repo          Repository
	transactor    database.Transactor
	enforcer      *casbin.Enforcer
	digester      *auth.SecretDigester
	clock         auth.Clock
	invitationTTL time.Duration
}

var _ auth.PersonalOrgCreator = (*Service)(nil)

func NewService(deps Dependencies) (*Service, error) {
	if deps.Repository == nil {
		return nil, errors.New("organization: repository is required")
	}

	clock := deps.Clock
	if clock == nil {
		clock = auth.SystemClock{}
	}

	ttl := deps.InvitationTTL
	if ttl <= 0 {
		ttl = defaultInvitationTTL
	}

	return &Service{
		repo:          deps.Repository,
		transactor:    deps.Transactor,
		enforcer:      deps.Enforcer,
		digester:      deps.Digester,
		clock:         clock,
		invitationTTL: ttl,
	}, nil
}

func (s *Service) withinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if s.transactor != nil {
		return s.transactor.WithinTransaction(ctx, fn)
	}
	return fn(ctx)
}

func (s *Service) digest(raw []byte) []byte {
	if s.digester != nil {
		return s.digester.Digest(raw)
	}
	sum := sha256.Sum256(raw)
	return sum[:]
}

func (s *Service) Create(ctx context.Context, userID string, input CreateInput) (Organization, error) {
	slug := strings.TrimSpace(input.Slug)
	name := strings.TrimSpace(input.Name)

	if slug == "" || len([]rune(slug)) > 100 {
		return Organization{}, fmt.Errorf("%w: slug must be between 1 and 100 characters", ErrInvalidInput)
	}
	if name == "" || len([]rune(name)) > 120 {
		return Organization{}, fmt.Errorf("%w: name must be between 1 and 120 characters", ErrInvalidInput)
	}

	var created Organization
	err := s.withinTx(ctx, func(ctx context.Context) error {
		org, err := s.repo.Create(ctx, slug, name)
		if err != nil {
			return err
		}
		created = org

		if _, err := s.repo.CreateMembership(ctx, org.ID, userID, RoleOwner); err != nil {
			return err
		}

		if s.enforcer != nil {
			if _, err := s.enforcer.AddGroupingPolicy(userID, string(RoleOwner), org.ID); err != nil {
				return fmt.Errorf("add role grouping policy: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return Organization{}, err
	}
	return created, nil
}

func (s *Service) CreatePersonalOrg(ctx context.Context, user auth.User) error {
	slug := "personal-" + user.ID
	name := "Personal"

	return s.withinTx(ctx, func(ctx context.Context) error {
		org, err := s.repo.Create(ctx, slug, name)
		if err != nil {
			return err
		}

		if _, err := s.repo.CreateMembership(ctx, org.ID, user.ID, RoleOwner); err != nil {
			return err
		}

		if s.enforcer != nil {
			if _, err := s.enforcer.AddGroupingPolicy(user.ID, string(RoleOwner), org.ID); err != nil {
				return fmt.Errorf("add personal org grouping policy: %w", err)
			}
		}
		return nil
	})
}

func (s *Service) ListForUser(ctx context.Context, userID string) ([]OrganizationWithRole, error) {
	return s.repo.ListForUser(ctx, userID)
}

func (s *Service) Get(ctx context.Context, orgID string) (Organization, error) {
	return s.repo.GetByID(ctx, orgID)
}

func (s *Service) Update(ctx context.Context, orgID string, input UpdateInput) (Organization, error) {
	slug := strings.TrimSpace(input.Slug)
	name := strings.TrimSpace(input.Name)

	if slug == "" || len([]rune(slug)) > 100 {
		return Organization{}, fmt.Errorf("%w: slug must be between 1 and 100 characters", ErrInvalidInput)
	}
	if name == "" || len([]rune(name)) > 120 {
		return Organization{}, fmt.Errorf("%w: name must be between 1 and 120 characters", ErrInvalidInput)
	}

	return s.repo.Update(ctx, orgID, slug, name)
}

func (s *Service) Delete(ctx context.Context, orgID string) error {
	return s.withinTx(ctx, func(ctx context.Context) error {
		if err := s.repo.Delete(ctx, orgID); err != nil {
			return err
		}

		if s.enforcer != nil {
			if _, err := s.enforcer.RemoveFilteredNamedGroupingPolicy("g", 2, orgID); err != nil {
				return fmt.Errorf("remove org grouping policies: %w", err)
			}
		}
		return nil
	})
}

func (s *Service) ListMembers(ctx context.Context, orgID string) ([]Membership, error) {
	return s.repo.ListMemberships(ctx, orgID)
}

func (s *Service) UpdateMemberRole(ctx context.Context, orgID, userID string, newRole Role) (Membership, error) {
	if !newRole.Valid() {
		return Membership{}, fmt.Errorf("%w: invalid role %s", ErrInvalidInput, newRole)
	}

	old, err := s.repo.GetMembership(ctx, orgID, userID)
	if err != nil {
		return Membership{}, err
	}

	if old.Role == RoleOwner && newRole != RoleOwner {
		count, err := s.repo.CountOwners(ctx, orgID)
		if err != nil {
			return Membership{}, err
		}
		if count <= 1 {
			return Membership{}, ErrLastOwner
		}
	}

	var updated Membership
	err = s.withinTx(ctx, func(ctx context.Context) error {
		m, err := s.repo.UpdateMembershipRole(ctx, orgID, userID, newRole)
		if err != nil {
			return err
		}
		updated = m

		if s.enforcer != nil {
			_, _ = s.enforcer.RemoveGroupingPolicy(userID, string(old.Role), orgID)
			if _, err := s.enforcer.AddGroupingPolicy(userID, string(newRole), orgID); err != nil {
				return fmt.Errorf("add updated role grouping policy: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return Membership{}, err
	}
	return updated, nil
}

func (s *Service) DeleteMember(ctx context.Context, orgID, userID string) error {
	old, err := s.repo.GetMembership(ctx, orgID, userID)
	if err != nil {
		return err
	}

	if old.Role == RoleOwner {
		count, err := s.repo.CountOwners(ctx, orgID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastOwner
		}
	}

	return s.withinTx(ctx, func(ctx context.Context) error {
		if err := s.repo.DeleteMembership(ctx, orgID, userID); err != nil {
			return err
		}

		if s.enforcer != nil {
			_, _ = s.enforcer.RemoveGroupingPolicy(userID, string(old.Role), orgID)
		}
		return nil
	})
}

func (s *Service) CreateInvitation(ctx context.Context, orgID string, input CreateInvitationInput) (CreatedInvitation, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return CreatedInvitation{}, fmt.Errorf("%w: invalid email address", ErrInvalidInput)
	}
	if !input.Role.Valid() {
		return CreatedInvitation{}, fmt.Errorf("%w: invalid role %s", ErrInvalidInput, input.Role)
	}

	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return CreatedInvitation{}, fmt.Errorf("generate random token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(rawBytes)
	tokenDigest := s.digest(rawBytes)
	expiresAt := s.clock.Now().UTC().Add(s.invitationTTL)

	inv, err := s.repo.CreateInvitation(ctx, orgID, email, input.Role, tokenDigest, expiresAt)
	if err != nil {
		return CreatedInvitation{}, err
	}

	return CreatedInvitation{
		Invitation: inv,
		Token:      token,
	}, nil
}

func (s *Service) ListInvitations(ctx context.Context, orgID string) ([]Invitation, error) {
	return s.repo.ListInvitations(ctx, orgID)
}

func (s *Service) RevokeInvitation(ctx context.Context, orgID, id string) error {
	return s.repo.RevokeInvitation(ctx, orgID, id)
}

func (s *Service) AcceptInvitation(ctx context.Context, userID, rawToken string) (Membership, error) {
	rawBytes, err := base64.RawURLEncoding.DecodeString(rawToken)
	if err != nil {
		return Membership{}, ErrInvitationExpired
	}
	tokenDigest := s.digest(rawBytes)

	inv, err := s.repo.GetInvitationByDigest(ctx, tokenDigest)
	if err != nil {
		return Membership{}, ErrInvitationExpired
	}

	// Verify not already member
	if _, err := s.repo.GetMembership(ctx, inv.OrganizationID, userID); err == nil {
		return Membership{}, ErrAlreadyMember
	}

	var created Membership
	err = s.withinTx(ctx, func(ctx context.Context) error {
		if _, err := s.repo.AcceptInvitation(ctx, inv.ID); err != nil {
			return ErrInvitationExpired
		}

		m, err := s.repo.CreateMembership(ctx, inv.OrganizationID, userID, inv.Role)
		if err != nil {
			return err
		}
		created = m

		if s.enforcer != nil {
			if _, err := s.enforcer.AddGroupingPolicy(userID, string(inv.Role), inv.OrganizationID); err != nil {
				return fmt.Errorf("add invitation role grouping policy: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return Membership{}, err
	}
	return created, nil
}
