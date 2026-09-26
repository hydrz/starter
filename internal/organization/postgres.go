package organization

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/store"
)

const pgUniqueViolation = "23505"

type PostgresRepository struct {
	queries *store.Queries
}

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(queries *store.Queries) *PostgresRepository {
	return &PostgresRepository{queries: queries}
}

func (r *PostgresRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}

func parseUUID(val string) (pgtype.UUID, error) {
	id, err := uuid.Parse(val)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid %q: %w", val, ErrInvalidInput)
	}
	return pgtype.UUID{Bytes: id, Valid: true}, nil
}

func (r *PostgresRepository) Create(ctx context.Context, slug, name string) (Organization, error) {
	row, err := r.q(ctx).CreateOrganization(ctx, store.CreateOrganizationParams{
		Slug: slug,
		Name: name,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return Organization{}, ErrSlugTaken
		}
		return Organization{}, fmt.Errorf("create organization: %w", err)
	}
	return orgFromStore(row), nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (Organization, error) {
	orgUUID, err := parseUUID(id)
	if err != nil {
		return Organization{}, err
	}
	row, err := r.q(ctx).GetOrganizationByID(ctx, orgUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Organization{}, ErrNotFound
	}
	if err != nil {
		return Organization{}, fmt.Errorf("get organization by id: %w", err)
	}
	return orgFromStore(row), nil
}

func (r *PostgresRepository) GetBySlug(ctx context.Context, slug string) (Organization, error) {
	row, err := r.q(ctx).GetOrganizationBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return Organization{}, ErrNotFound
	}
	if err != nil {
		return Organization{}, fmt.Errorf("get organization by slug: %w", err)
	}
	return orgFromStore(row), nil
}

func (r *PostgresRepository) Update(ctx context.Context, id, slug, name string) (Organization, error) {
	orgUUID, err := parseUUID(id)
	if err != nil {
		return Organization{}, err
	}
	row, err := r.q(ctx).UpdateOrganization(ctx, store.UpdateOrganizationParams{
		ID:   orgUUID,
		Name: name,
		Slug: slug,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Organization{}, ErrNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return Organization{}, ErrSlugTaken
		}
		return Organization{}, fmt.Errorf("update organization: %w", err)
	}
	return orgFromStore(row), nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	orgUUID, err := parseUUID(id)
	if err != nil {
		return err
	}
	count, err := r.q(ctx).DeleteOrganization(ctx, orgUUID)
	if err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) ListForUser(ctx context.Context, userID string) ([]OrganizationWithRole, error) {
	userUUID, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q(ctx).ListOrganizationsForUser(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("list organizations for user: %w", err)
	}

	items := make([]OrganizationWithRole, len(rows))
	for i, row := range rows {
		items[i] = OrganizationWithRole{
			ID:        uuid.UUID(row.ID.Bytes).String(),
			Slug:      row.Slug,
			Name:      row.Name,
			Role:      Role(row.Role),
			CreatedAt: row.CreatedAt.Time,
			UpdatedAt: row.UpdatedAt.Time,
		}
	}
	return items, nil
}

func (r *PostgresRepository) CreateMembership(ctx context.Context, orgID, userID string, role Role) (Membership, error) {
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return Membership{}, err
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		return Membership{}, err
	}

	row, err := r.q(ctx).CreateMembership(ctx, store.CreateMembershipParams{
		OrganizationID: orgUUID,
		UserID:         userUUID,
		Role:           string(role),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return Membership{}, ErrAlreadyMember
		}
		return Membership{}, fmt.Errorf("create membership: %w", err)
	}
	return membershipFromStore(row), nil
}

func (r *PostgresRepository) GetMembership(ctx context.Context, orgID, userID string) (Membership, error) {
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return Membership{}, err
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		return Membership{}, err
	}

	row, err := r.q(ctx).GetMembership(ctx, store.GetMembershipParams{
		OrganizationID: orgUUID,
		UserID:         userUUID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrMembershipNotFound
	}
	if err != nil {
		return Membership{}, fmt.Errorf("get membership: %w", err)
	}
	return membershipFromStore(row), nil
}

func (r *PostgresRepository) ListMemberships(ctx context.Context, orgID string) ([]Membership, error) {
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return nil, err
	}

	rows, err := r.q(ctx).ListMemberships(ctx, orgUUID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}

	items := make([]Membership, len(rows))
	for i, row := range rows {
		items[i] = Membership{
			ID:             uuid.UUID(row.ID.Bytes).String(),
			OrganizationID: uuid.UUID(row.OrganizationID.Bytes).String(),
			UserID:         uuid.UUID(row.UserID.Bytes).String(),
			Email:          row.Email,
			Role:           Role(row.Role),
			CreatedAt:      row.CreatedAt.Time,
			UpdatedAt:      row.UpdatedAt.Time,
		}
	}
	return items, nil
}

func (r *PostgresRepository) UpdateMembershipRole(ctx context.Context, orgID, userID string, role Role) (Membership, error) {
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return Membership{}, err
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		return Membership{}, err
	}

	row, err := r.q(ctx).UpdateMembershipRole(ctx, store.UpdateMembershipRoleParams{
		OrganizationID: orgUUID,
		UserID:         userUUID,
		Role:           string(role),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrMembershipNotFound
	}
	if err != nil {
		return Membership{}, fmt.Errorf("update membership role: %w", err)
	}
	return membershipFromStore(row), nil
}

func (r *PostgresRepository) DeleteMembership(ctx context.Context, orgID, userID string) error {
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return err
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		return err
	}

	count, err := r.q(ctx).DeleteMembership(ctx, store.DeleteMembershipParams{
		OrganizationID: orgUUID,
		UserID:         userUUID,
	})
	if err != nil {
		return fmt.Errorf("delete membership: %w", err)
	}
	if count == 0 {
		return ErrMembershipNotFound
	}
	return nil
}

func (r *PostgresRepository) CountOwners(ctx context.Context, orgID string) (int64, error) {
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return 0, err
	}
	return r.q(ctx).CountOrganizationOwners(ctx, orgUUID)
}

func (r *PostgresRepository) CreateInvitation(ctx context.Context, orgID, email string, role Role, tokenDigest []byte, expiresAt time.Time) (Invitation, error) {
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return Invitation{}, err
	}

	row, err := r.q(ctx).CreateInvitation(ctx, store.CreateInvitationParams{
		OrganizationID: orgUUID,
		Email:          email,
		Role:           string(role),
		TokenDigest:    tokenDigest,
		ExpiresAt:      pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return Invitation{}, fmt.Errorf("create invitation: %w", err)
	}
	return invitationFromStore(row), nil
}

func (r *PostgresRepository) GetInvitationByID(ctx context.Context, id string) (Invitation, error) {
	invUUID, err := parseUUID(id)
	if err != nil {
		return Invitation{}, err
	}
	row, err := r.q(ctx).GetInvitationByID(ctx, invUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invitation{}, ErrNotFound
	}
	if err != nil {
		return Invitation{}, fmt.Errorf("get invitation by id: %w", err)
	}
	return invitationFromStore(row), nil
}

func (r *PostgresRepository) GetInvitationByDigest(ctx context.Context, digest []byte) (Invitation, error) {
	row, err := r.q(ctx).GetInvitationByDigest(ctx, digest)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invitation{}, ErrNotFound
	}
	if err != nil {
		return Invitation{}, fmt.Errorf("get invitation by digest: %w", err)
	}
	return invitationFromStore(row), nil
}

func (r *PostgresRepository) ListInvitations(ctx context.Context, orgID string) ([]Invitation, error) {
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q(ctx).ListInvitations(ctx, orgUUID)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}

	items := make([]Invitation, len(rows))
	for i, row := range rows {
		items[i] = invitationFromStore(row)
	}
	return items, nil
}

func (r *PostgresRepository) AcceptInvitation(ctx context.Context, id string) (Invitation, error) {
	invUUID, err := parseUUID(id)
	if err != nil {
		return Invitation{}, err
	}
	row, err := r.q(ctx).AcceptInvitation(ctx, invUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invitation{}, ErrInvitationExpired
	}
	if err != nil {
		return Invitation{}, fmt.Errorf("accept invitation: %w", err)
	}
	return invitationFromStore(row), nil
}

func (r *PostgresRepository) RevokeInvitation(ctx context.Context, orgID, id string) error {
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return err
	}
	invUUID, err := parseUUID(id)
	if err != nil {
		return err
	}
	count, err := r.q(ctx).RevokeInvitation(ctx, store.RevokeInvitationParams{
		ID:             invUUID,
		OrganizationID: orgUUID,
	})
	if err != nil {
		return fmt.Errorf("revoke invitation: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func orgFromStore(row store.Organization) Organization {
	return Organization{
		ID:        uuid.UUID(row.ID.Bytes).String(),
		Slug:      row.Slug,
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

func membershipFromStore(row store.OrganizationMembership) Membership {
	return Membership{
		ID:             uuid.UUID(row.ID.Bytes).String(),
		OrganizationID: uuid.UUID(row.OrganizationID.Bytes).String(),
		UserID:         uuid.UUID(row.UserID.Bytes).String(),
		Role:           Role(row.Role),
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}
}

func invitationFromStore(row store.OrganizationInvitation) Invitation {
	return Invitation{
		ID:             uuid.UUID(row.ID.Bytes).String(),
		OrganizationID: uuid.UUID(row.OrganizationID.Bytes).String(),
		Email:          row.Email,
		Role:           Role(row.Role),
		ExpiresAt:      row.ExpiresAt.Time,
		CreatedAt:      row.CreatedAt.Time,
	}
}
