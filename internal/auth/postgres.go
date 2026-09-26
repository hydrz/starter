package auth

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/store"
)

// pgUniqueViolation is the PostgreSQL SQLSTATE for a unique_violation.
const pgUniqueViolation = "23505"

// PostgresUserRepository implements UserRepository over internal/store.
type PostgresUserRepository struct {
	queries *store.Queries
}

func NewPostgresUserRepository(queries *store.Queries) *PostgresUserRepository {
	return &PostgresUserRepository{queries: queries}
}

func (repository *PostgresUserRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return repository.queries.WithTx(tx)
	}
	return repository.queries
}

func (repository *PostgresUserRepository) Create(ctx context.Context, email, passwordHash string) (User, error) {
	row, err := repository.q(ctx).CreateUser(ctx, store.CreateUserParams{Email: email, PasswordHash: passwordHash})
	if err != nil {
		if isUniqueViolationError(err) {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	return userFromStore(row), nil
}

func (repository *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	row, err := repository.q(ctx).GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	return userFromStore(row), nil
}

func (repository *PostgresUserRepository) FindByID(ctx context.Context, id string) (User, error) {
	parsed, err := parseUUID(id)
	if err != nil {
		return User{}, ErrUserNotFound
	}
	row, err := repository.q(ctx).GetUserByID(ctx, parsed)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	return userFromStore(row), nil
}

func (repository *PostgresUserRepository) MarkEmailVerified(ctx context.Context, userID string) (bool, error) {
	parsed, err := parseUUID(userID)
	if err != nil {
		return false, ErrUserNotFound
	}
	count, err := repository.q(ctx).MarkUserEmailVerified(ctx, parsed)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (repository *PostgresUserRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	parsed, err := parseUUID(userID)
	if err != nil {
		return ErrUserNotFound
	}
	_, err = repository.q(ctx).UpdateUserPassword(ctx, store.UpdateUserPasswordParams{ID: parsed, PasswordHash: passwordHash})
	return err
}

func userFromStore(row store.User) User {
	user := User{
		ID:           uuid.UUID(row.ID.Bytes).String(),
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
	if row.EmailVerifiedAt.Valid {
		verifiedAt := row.EmailVerifiedAt.Time
		user.EmailVerifiedAt = &verifiedAt
	}
	return user
}

// PostgresRefreshTokenRepository implements RefreshTokenRepository.
type PostgresRefreshTokenRepository struct {
	queries *store.Queries
}

func NewPostgresRefreshTokenRepository(queries *store.Queries) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{queries: queries}
}

func (repository *PostgresRefreshTokenRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return repository.queries.WithTx(tx)
	}
	return repository.queries
}

func (repository *PostgresRefreshTokenRepository) CreateFamily(ctx context.Context, userID string) (string, error) {
	parsed, err := parseUUID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}
	row, err := repository.q(ctx).CreateRefreshTokenFamily(ctx, parsed)
	if err != nil {
		return "", err
	}
	return uuid.UUID(row.ID.Bytes).String(), nil
}

func (repository *PostgresRefreshTokenRepository) CreateSession(ctx context.Context, familyID string, tokenDigest []byte, expiresAt time.Time, metadata RefreshSessionMetadata) (string, error) {
	parsedFamily, err := parseUUID(familyID)
	if err != nil {
		return "", ErrSessionNotFound
	}
	var userAgent *string
	if metadata.UserAgent != "" {
		userAgent = &metadata.UserAgent
	}
	var ipAddress *netip.Addr
	if metadata.IPAddress != "" {
		if parsedIP, err := netip.ParseAddr(metadata.IPAddress); err == nil {
			ipAddress = &parsedIP
		}
	}
	row, err := repository.q(ctx).CreateRefreshSession(ctx, store.CreateRefreshSessionParams{
		FamilyID:    parsedFamily,
		TokenDigest: tokenDigest,
		ExpiresAt:   pgtype.Timestamptz{Time: expiresAt, Valid: true},
		UserAgent:   userAgent,
		IpAddress:   ipAddress,
	})
	if err != nil {
		return "", err
	}
	return uuid.UUID(row.ID.Bytes).String(), nil
}

func (repository *PostgresRefreshTokenRepository) ConsumeSession(ctx context.Context, tokenDigest []byte) (string, string, bool, error) {
	row, err := repository.q(ctx).ConsumeRefreshSession(ctx, tokenDigest)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	return uuid.UUID(row.FamilyID.Bytes).String(), uuid.UUID(row.ID.Bytes).String(), true, nil
}

func (repository *PostgresRefreshTokenRepository) FindByDigest(ctx context.Context, tokenDigest []byte) (string, string, bool, error) {
	row, err := repository.q(ctx).FindRefreshSession(ctx, tokenDigest)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	return uuid.UUID(row.UserID.Bytes).String(), uuid.UUID(row.FamilyID.Bytes).String(), true, nil
}

func (repository *PostgresRefreshTokenRepository) RevokeFamily(ctx context.Context, familyID, reason string) error {
	parsed, err := parseUUID(familyID)
	if err != nil {
		return ErrSessionNotFound
	}
	_, err = repository.q(ctx).RevokeRefreshFamily(ctx, store.RevokeRefreshFamilyParams{ID: parsed, RevokeReason: &reason})
	return err
}

func (repository *PostgresRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID, reason string) error {
	parsed, err := parseUUID(userID)
	if err != nil {
		return ErrUserNotFound
	}
	_, err = repository.q(ctx).RevokeRefreshFamiliesForUser(ctx, store.RevokeRefreshFamiliesForUserParams{UserID: parsed, RevokeReason: &reason})
	return err
}

func (repository *PostgresRefreshTokenRepository) ListSessionsForUser(ctx context.Context, userID string) ([]RefreshSession, error) {
	parsed, err := parseUUID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	rows, err := repository.q(ctx).ListRefreshSessionsForUser(ctx, parsed)
	if err != nil {
		return nil, err
	}
	sessions := make([]RefreshSession, 0, len(rows))
	for _, row := range rows {
		session := RefreshSession{
			ID:        uuid.UUID(row.ID.Bytes).String(),
			CreatedAt: row.CreatedAt.Time,
			ExpiresAt: row.ExpiresAt.Time,
		}
		if row.LastUsedAt.Valid {
			lastUsedAt := row.LastUsedAt.Time
			session.LastUsedAt = &lastUsedAt
		}
		if row.UserAgent != nil {
			session.UserAgent = *row.UserAgent
		}
		if row.IpAddress != nil {
			session.IPAddress = row.IpAddress.String()
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (repository *PostgresRefreshTokenRepository) RevokeSessionForUser(ctx context.Context, sessionID, userID string) (bool, error) {
	parsedSession, err := parseUUID(sessionID)
	if err != nil {
		return false, nil
	}
	parsedUser, err := parseUUID(userID)
	if err != nil {
		return false, nil
	}
	count, err := repository.q(ctx).RevokeRefreshSessionForUser(ctx, store.RevokeRefreshSessionForUserParams{ID: parsedSession, UserID: parsedUser})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// PostgresOneTimeTokenRepository implements OneTimeTokenRepository.
type PostgresOneTimeTokenRepository struct {
	queries *store.Queries
}

func NewPostgresOneTimeTokenRepository(queries *store.Queries) *PostgresOneTimeTokenRepository {
	return &PostgresOneTimeTokenRepository{queries: queries}
}

func (repository *PostgresOneTimeTokenRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return repository.queries.WithTx(tx)
	}
	return repository.queries
}

func (repository *PostgresOneTimeTokenRepository) Create(ctx context.Context, userID string, purpose OneTimeTokenPurpose, tokenDigest []byte, expiresAt time.Time) error {
	parsed, err := parseUUID(userID)
	if err != nil {
		return ErrUserNotFound
	}
	_, err = repository.q(ctx).CreateOneTimeToken(ctx, store.CreateOneTimeTokenParams{
		UserID:      parsed,
		Purpose:     string(purpose),
		TokenDigest: tokenDigest,
		ExpiresAt:   pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	return err
}

func (repository *PostgresOneTimeTokenRepository) Consume(ctx context.Context, tokenDigest []byte, purpose OneTimeTokenPurpose) (string, bool, error) {
	row, err := repository.q(ctx).ConsumeOneTimeToken(ctx, store.ConsumeOneTimeTokenParams{TokenDigest: tokenDigest, Purpose: string(purpose)})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return uuid.UUID(row.UserID.Bytes).String(), true, nil
}

// PostgresAPIKeyRepository implements APIKeyRepository.
type PostgresAPIKeyRepository struct {
	queries *store.Queries
}

func NewPostgresAPIKeyRepository(queries *store.Queries) *PostgresAPIKeyRepository {
	return &PostgresAPIKeyRepository{queries: queries}
}

func (repository *PostgresAPIKeyRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return repository.queries.WithTx(tx)
	}
	return repository.queries
}

func (repository *PostgresAPIKeyRepository) Create(ctx context.Context, userID, name, keyPrefix string, secretDigest []byte, expiresAt *time.Time) (APIKey, error) {
	parsedUser, err := parseUUID(userID)
	if err != nil {
		return APIKey{}, ErrUserNotFound
	}
	row, err := repository.q(ctx).CreateAPIKey(ctx, store.CreateAPIKeyParams{
		UserID:       parsedUser,
		Name:         name,
		KeyPrefix:    keyPrefix,
		SecretDigest: secretDigest,
		ExpiresAt:    timestamptzFromPointer(expiresAt),
	})
	if err != nil {
		return APIKey{}, err
	}
	return APIKey{
		ID:        uuid.UUID(row.ID.Bytes).String(),
		UserID:    uuid.UUID(row.UserID.Bytes).String(),
		Name:      row.Name,
		KeyPrefix: row.KeyPrefix,
		ExpiresAt: pointerFromTimestamptz(row.ExpiresAt),
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (repository *PostgresAPIKeyRepository) ListForUser(ctx context.Context, userID string) ([]APIKey, error) {
	parsed, err := parseUUID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	rows, err := repository.q(ctx).ListAPIKeysForUser(ctx, parsed)
	if err != nil {
		return nil, err
	}
	keys := make([]APIKey, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, APIKey{
			ID:         uuid.UUID(row.ID.Bytes).String(),
			UserID:     uuid.UUID(row.UserID.Bytes).String(),
			Name:       row.Name,
			KeyPrefix:  row.KeyPrefix,
			ExpiresAt:  pointerFromTimestamptz(row.ExpiresAt),
			RevokedAt:  pointerFromTimestamptz(row.RevokedAt),
			LastUsedAt: pointerFromTimestamptz(row.LastUsedAt),
			CreatedAt:  row.CreatedAt.Time,
		})
	}
	return keys, nil
}

func (repository *PostgresAPIKeyRepository) RevokeForUser(ctx context.Context, id, userID string) (bool, error) {
	parsedID, err := parseUUID(id)
	if err != nil {
		return false, nil
	}
	parsedUser, err := parseUUID(userID)
	if err != nil {
		return false, nil
	}
	count, err := repository.q(ctx).RevokeAPIKeyForUser(ctx, store.RevokeAPIKeyForUserParams{ID: parsedID, UserID: parsedUser})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (repository *PostgresAPIKeyRepository) FindActiveByDigest(ctx context.Context, secretDigest []byte) (APIKey, bool, error) {
	row, err := repository.q(ctx).GetActiveAPIKeyByDigest(ctx, secretDigest)
	if errors.Is(err, pgx.ErrNoRows) {
		return APIKey{}, false, nil
	}
	if err != nil {
		return APIKey{}, false, err
	}
	return APIKey{
		ID:        uuid.UUID(row.ID.Bytes).String(),
		UserID:    uuid.UUID(row.UserID.Bytes).String(),
		Name:      row.Name,
		KeyPrefix: row.KeyPrefix,
		ExpiresAt: pointerFromTimestamptz(row.ExpiresAt),
		CreatedAt: row.CreatedAt.Time,
	}, true, nil
}

// PostgresOutboxWriter implements OutboxWriter by inserting a row into
// outbox_events. It never performs delivery itself (ADR-0005/0007); a
// separate worker in internal/delivery claims and processes these rows.
type PostgresOutboxWriter struct {
	queries *store.Queries
}

func NewPostgresOutboxWriter(queries *store.Queries) *PostgresOutboxWriter {
	return &PostgresOutboxWriter{queries: queries}
}

func (writer *PostgresOutboxWriter) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return writer.queries.WithTx(tx)
	}
	return writer.queries
}

func (writer *PostgresOutboxWriter) WriteEvent(ctx context.Context, topic, aggregateType, aggregateID string, payload []byte, idempotencyKey string) error {
	parsedAggregate, err := parseUUID(aggregateID)
	if err != nil {
		return fmt.Errorf("invalid aggregate id: %w", err)
	}
	_, err = writer.q(ctx).CreateOutboxEvent(ctx, store.CreateOutboxEventParams{
		Topic:          topic,
		AggregateType:  aggregateType,
		AggregateID:    parsedAggregate,
		Payload:        payload,
		IdempotencyKey: idempotencyKey,
		AvailableAt:    pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil && isUniqueViolationError(err) {
		// Idempotency key already recorded: treat as already-queued rather
		// than a failure so a retried caller does not observe an error.
		return nil
	}
	return err
}

func parseUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid id: %w", err)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func timestamptzFromPointer(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func pointerFromTimestamptz(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func isUniqueViolationError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}
