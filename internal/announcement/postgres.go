package announcement

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/store"
)

type PostgresRepository struct {
	queries *store.Queries
}

func NewPostgresRepository(queries *store.Queries) *PostgresRepository {
	return &PostgresRepository{queries: queries}
}

func (repository *PostgresRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return repository.queries.WithTx(tx)
	}
	return repository.queries
}

func (repository *PostgresRepository) List(ctx context.Context, organizationID uuid.UUID, filter Filter) ([]Announcement, int64, error) {
	var status *store.AnnouncementStatus
	if filter.Status != nil {
		s := store.AnnouncementStatus(*filter.Status)
		status = &s
	}
	orgUUID := pgtype.UUID{Bytes: organizationID, Valid: true}

	total, err := repository.q(ctx).CountAnnouncements(ctx, store.CountAnnouncementsParams{
		OrganizationID: orgUUID,
		Status:         status,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}
	rows, err := repository.q(ctx).ListAnnouncements(ctx, store.ListAnnouncementsParams{
		OrganizationID: orgUUID,
		Status:         status,
		Limit:          filter.Limit,
		Offset:         filter.Offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("query: %w", err)
	}

	items := make([]Announcement, 0, len(rows))
	for _, row := range rows {
		items = append(items, announcementFromRow(row.ID, row.OrganizationID, row.Title, row.Content, row.Status, row.CreatedAt, row.UpdatedAt))
	}
	return items, total, nil
}

func (repository *PostgresRepository) Get(ctx context.Context, organizationID uuid.UUID, id string) (Announcement, error) {
	databaseID, err := parseUUID(id)
	if err != nil {
		return Announcement{}, err
	}
	row, err := repository.q(ctx).GetAnnouncement(ctx, store.GetAnnouncementParams{
		ID:             databaseID,
		OrganizationID: pgtype.UUID{Bytes: organizationID, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Announcement{}, ErrNotFound
	}
	if err != nil {
		return Announcement{}, err
	}
	return announcementFromRow(row.ID, row.OrganizationID, row.Title, row.Content, row.Status, row.CreatedAt, row.UpdatedAt), nil
}

func (repository *PostgresRepository) Create(ctx context.Context, organizationID uuid.UUID, input Input) (Announcement, error) {
	row, err := repository.q(ctx).CreateAnnouncement(ctx, store.CreateAnnouncementParams{
		OrganizationID: pgtype.UUID{Bytes: organizationID, Valid: true},
		Title:          input.Title,
		Content:        input.Content,
		Status:         store.AnnouncementStatus(input.Status),
	})
	if err != nil {
		return Announcement{}, err
	}
	return announcementFromRow(row.ID, row.OrganizationID, row.Title, row.Content, row.Status, row.CreatedAt, row.UpdatedAt), nil
}

func (repository *PostgresRepository) Update(ctx context.Context, organizationID uuid.UUID, id string, input Input) (Announcement, error) {
	databaseID, err := parseUUID(id)
	if err != nil {
		return Announcement{}, err
	}
	row, err := repository.q(ctx).UpdateAnnouncement(ctx, store.UpdateAnnouncementParams{
		ID:             databaseID,
		OrganizationID: pgtype.UUID{Bytes: organizationID, Valid: true},
		Title:          input.Title,
		Content:        input.Content,
		Status:         store.AnnouncementStatus(input.Status),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Announcement{}, ErrNotFound
	}
	if err != nil {
		return Announcement{}, err
	}
	return announcementFromRow(row.ID, row.OrganizationID, row.Title, row.Content, row.Status, row.CreatedAt, row.UpdatedAt), nil
}

func (repository *PostgresRepository) Delete(ctx context.Context, organizationID uuid.UUID, id string) error {
	databaseID, err := parseUUID(id)
	if err != nil {
		return err
	}
	count, err := repository.q(ctx).DeleteAnnouncement(ctx, store.DeleteAnnouncementParams{
		ID:             databaseID,
		OrganizationID: pgtype.UUID{Bytes: organizationID, Valid: true},
	})
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func parseUUID(value string) (pgtype.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid announcement id: %w", err)
	}
	return pgtype.UUID{Bytes: id, Valid: true}, nil
}

func announcementFromRow(id, orgID pgtype.UUID, title, content string, status store.AnnouncementStatus, createdAt, updatedAt pgtype.Timestamptz) Announcement {
	return Announcement{
		ID:             uuid.UUID(id.Bytes).String(),
		OrganizationID: uuid.UUID(orgID.Bytes).String(),
		Title:          title,
		Content:        content,
		Status:         Status(status),
		CreatedAt:      createdAt.Time,
		UpdatedAt:      updatedAt.Time,
	}
}
