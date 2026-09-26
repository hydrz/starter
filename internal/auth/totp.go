package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/store"
)

const (
	totpFactorStatusPending  = "pending"
	totpFactorStatusVerified = "verified"

	mfaChallengeTTL   = 10 * time.Minute
	recoveryCodeCount = 10
)

var (
	// ErrTOTPEnrollmentNotFound is returned when the pending enrollment
	// identified by factorID does not belong to userID or is not pending.
	ErrTOTPEnrollmentNotFound = errors.New("auth: totp enrollment not found")
	// ErrTOTPInvalidCode is returned for a TOTP code or recovery code that
	// does not verify.
	ErrTOTPInvalidCode = errors.New("auth: invalid totp code")
	// ErrMFAChallengeNotFound is returned for an unknown, expired, or
	// already-consumed MFA challenge.
	ErrMFAChallengeNotFound = errors.New("auth: mfa challenge not found")
	// ErrMFARequired is returned by a primary sign-in method when the
	// account has an active second factor; the caller must complete
	// VerifyMFAChallenge with the returned challenge ID before receiving
	// access/refresh tokens.
	ErrMFARequired = errors.New("auth: multi-factor authentication required")
)

// TOTPFactorRecord is the narrow record Service needs; the plaintext secret
// never appears here.
type TOTPFactorRecord struct {
	ID               string
	UserID           string
	SecretCiphertext []byte
	SecretNonce      []byte
	Status           string
}

// TOTPFactorRepository persists per-user TOTP enrollments. Only one
// verified factor per user is permitted (enforced by a partial unique index
// in the migration); a pending enrollment is replaced, never stacked.
type TOTPFactorRepository interface {
	CreatePending(ctx context.Context, userID string, secretCiphertext, secretNonce []byte) (factorID string, err error)
	GetByID(ctx context.Context, factorID string) (TOTPFactorRecord, bool, error)
	GetVerifiedForUser(ctx context.Context, userID string) (TOTPFactorRecord, bool, error)
	// Activate atomically transitions factorID from pending to verified,
	// scoped to userID. ok is false if no such pending factor exists.
	Activate(ctx context.Context, factorID, userID string) (ok bool, err error)
	DeletePendingForUser(ctx context.Context, userID string) error
}

// TOTPRecoveryCodeRepository persists single-use recovery code digests tied
// to a verified TOTP factor.
type TOTPRecoveryCodeRepository interface {
	CreateMany(ctx context.Context, factorID string, codeDigests [][]byte) error
	// Consume atomically marks one recovery code used-once.
	Consume(ctx context.Context, factorID string, codeDigest []byte) (ok bool, err error)
}

// MFAChallengeRecord ties a partially-authenticated sign-in attempt to the
// factor that must complete it.
type MFAChallengeRecord struct {
	ID       string
	UserID   string
	FactorID string
}

// MFAChallengeRepository persists short-lived, single-use MFA challenges.
type MFAChallengeRepository interface {
	Create(ctx context.Context, userID, factorID string, expiresAt time.Time) (challengeID string, err error)
	// Consume atomically marks the challenge used-once, returning its
	// user/factor on success.
	Consume(ctx context.Context, challengeID string) (MFAChallengeRecord, bool, error)
}

// TOTPProvisioning is returned by BeginTOTPEnrollment. Secret is the
// base32-encoded TOTP secret and URL is the otpauth:// provisioning URI a
// client renders as a QR code; both are shown to the authenticated caller
// exactly once and never persisted in plaintext by this package's callers.
type TOTPProvisioning struct {
	FactorID string
	Secret   string
	URL      string
}

// BeginTOTPEnrollment generates a new TOTP secret for userID, encrypts it at
// rest with AES-256-GCM, and stores it as a pending factor. Any previous
// pending enrollment for the user is discarded first so restarting
// enrollment never leaves orphaned secrets. It requires an authenticated
// caller; userID must come from a verified access token (enforced by the
// HTTP handler, which only resolves this method from an authenticated
// context).
func (service *Service) BeginTOTPEnrollment(ctx context.Context, userID string) (TOTPProvisioning, error) {
	if service.totpFactors == nil || service.totpCipher == nil {
		return TOTPProvisioning{}, ErrInvalidInput
	}
	user, err := service.users.FindByID(ctx, userID)
	if err != nil {
		return TOTPProvisioning{}, fmt.Errorf("find user: %w", err)
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      service.totpIssuer,
		AccountName: user.Email,
	})
	if err != nil {
		return TOTPProvisioning{}, fmt.Errorf("generate totp secret: %w", err)
	}

	ciphertext, nonce, err := service.totpCipher.Encrypt([]byte(key.Secret()))
	if err != nil {
		return TOTPProvisioning{}, fmt.Errorf("encrypt totp secret: %w", err)
	}

	if err := service.totpFactors.DeletePendingForUser(ctx, userID); err != nil {
		return TOTPProvisioning{}, fmt.Errorf("clear previous pending enrollment: %w", err)
	}
	factorID, err := service.totpFactors.CreatePending(ctx, userID, ciphertext, nonce)
	if err != nil {
		return TOTPProvisioning{}, fmt.Errorf("create pending totp factor: %w", err)
	}

	return TOTPProvisioning{FactorID: factorID, Secret: key.Secret(), URL: key.URL()}, nil
}

// ConfirmTOTPEnrollment verifies code against the pending enrollment
// factorID, activates it, and returns a fresh set of recovery codes exactly
// once (only their digests are ever persisted).
func (service *Service) ConfirmTOTPEnrollment(ctx context.Context, userID, factorID, code string) ([]string, error) {
	if service.totpFactors == nil || service.recoveryCodes == nil || service.totpCipher == nil {
		return nil, ErrInvalidInput
	}
	factor, ok, err := service.totpFactors.GetByID(ctx, factorID)
	if err != nil {
		return nil, fmt.Errorf("find totp factor: %w", err)
	}
	if !ok || factor.UserID != userID || factor.Status != totpFactorStatusPending {
		return nil, ErrTOTPEnrollmentNotFound
	}

	secret, err := service.totpCipher.Decrypt(factor.SecretCiphertext, factor.SecretNonce)
	if err != nil {
		return nil, fmt.Errorf("decrypt totp secret: %w", err)
	}
	if !service.validateTOTPCode(code, string(secret)) {
		return nil, ErrTOTPInvalidCode
	}

	activated, err := service.totpFactors.Activate(ctx, factorID, userID)
	if err != nil {
		return nil, fmt.Errorf("activate totp factor: %w", err)
	}
	if !activated {
		return nil, ErrTOTPEnrollmentNotFound
	}

	rawCodes := make([]string, 0, recoveryCodeCount)
	digests := make([][]byte, 0, recoveryCodeCount)
	for range recoveryCodeCount {
		raw, rawBytes, err := GenerateSecret()
		if err != nil {
			return nil, fmt.Errorf("generate recovery code: %w", err)
		}
		rawCodes = append(rawCodes, raw)
		digests = append(digests, service.digester.Digest(rawBytes))
	}
	if err := service.recoveryCodes.CreateMany(ctx, factorID, digests); err != nil {
		return nil, fmt.Errorf("store recovery codes: %w", err)
	}

	return rawCodes, nil
}

// validateTOTPCode checks code against secret at the service's injected
// clock's current time (not real wall-clock time), so tests using a fake
// Clock validate deterministically regardless of when they actually run.
func (service *Service) validateTOTPCode(code, secret string) bool {
	ok, err := totp.ValidateCustom(code, secret, service.clock.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return err == nil && ok
}

// checkMFARequired returns a non-nil MFA challenge ID when userID has a
// verified TOTP factor, in which case the caller must stop issuing tokens
// and instead surface ErrMFARequired with that challenge ID. It is a no-op
// (never required) when MFA dependencies are not wired, so MFA remains
// entirely optional infrastructure until an operator configures it.
func (service *Service) checkMFARequired(ctx context.Context, userID string) (string, error) {
	if service.totpFactors == nil || service.mfaChallenges == nil {
		return "", nil
	}
	factor, ok, err := service.totpFactors.GetVerifiedForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("check mfa enrollment: %w", err)
	}
	if !ok {
		return "", nil
	}
	expiresAt := service.clock.Now().UTC().Add(mfaChallengeTTL)
	challengeID, err := service.mfaChallenges.Create(ctx, userID, factor.ID, expiresAt)
	if err != nil {
		return "", fmt.Errorf("create mfa challenge: %w", err)
	}
	return challengeID, nil
}

// VerifyMFAChallenge completes a sign-in that was paused by ErrMFARequired.
// It accepts either a current TOTP code or an unused recovery code for the
// challenge's factor, tried in that order, and funnels a successful
// verification through the same issueSessionTokens path as every other
// sign-in-completing flow.
func (service *Service) VerifyMFAChallenge(ctx context.Context, challengeID, code string, metadata RefreshSessionMetadata) (IssuedTokens, error) {
	if service.mfaChallenges == nil || service.totpFactors == nil {
		return IssuedTokens{}, ErrMFAChallengeNotFound
	}
	challenge, ok, err := service.mfaChallenges.Consume(ctx, challengeID)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("consume mfa challenge: %w", err)
	}
	if !ok {
		return IssuedTokens{}, ErrMFAChallengeNotFound
	}

	factor, ok, err := service.totpFactors.GetByID(ctx, challenge.FactorID)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("find totp factor: %w", err)
	}
	if !ok || factor.UserID != challenge.UserID || factor.Status != totpFactorStatusVerified {
		return IssuedTokens{}, ErrTOTPInvalidCode
	}

	verified := false
	if secret, err := service.totpCipher.Decrypt(factor.SecretCiphertext, factor.SecretNonce); err == nil {
		verified = service.validateTOTPCode(code, string(secret))
	}
	if !verified && service.recoveryCodes != nil {
		if digest, err := decodeAndDigest(service.digester, code); err == nil {
			consumed, err := service.recoveryCodes.Consume(ctx, factor.ID, digest)
			if err != nil {
				return IssuedTokens{}, fmt.Errorf("consume recovery code: %w", err)
			}
			verified = consumed
		}
	}
	if !verified {
		return IssuedTokens{}, ErrTOTPInvalidCode
	}

	familyID, err := service.refreshTokens.CreateFamily(ctx, challenge.UserID)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("create refresh family: %w", err)
	}
	return service.issueSessionTokens(ctx, challenge.UserID, familyID, metadata)
}

// --- PostgreSQL adapters ---------------------------------------------------

type PostgresTOTPFactorRepository struct{ queries *store.Queries }

func NewPostgresTOTPFactorRepository(queries *store.Queries) *PostgresTOTPFactorRepository {
	return &PostgresTOTPFactorRepository{queries: queries}
}

func (repository *PostgresTOTPFactorRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return repository.queries.WithTx(tx)
	}
	return repository.queries
}

func (repository *PostgresTOTPFactorRepository) CreatePending(ctx context.Context, userID string, secretCiphertext, secretNonce []byte) (string, error) {
	parsed, err := parseUUID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}
	row, err := repository.q(ctx).CreateTOTPFactor(ctx, store.CreateTOTPFactorParams{
		UserID:           parsed,
		SecretCiphertext: secretCiphertext,
		SecretNonce:      secretNonce,
	})
	if err != nil {
		return "", err
	}
	return uuid.UUID(row.ID.Bytes).String(), nil
}

func (repository *PostgresTOTPFactorRepository) GetByID(ctx context.Context, factorID string) (TOTPFactorRecord, bool, error) {
	parsed, err := parseUUID(factorID)
	if err != nil {
		return TOTPFactorRecord{}, false, nil
	}
	row, err := repository.q(ctx).GetTOTPFactorByID(ctx, parsed)
	if errors.Is(err, pgx.ErrNoRows) {
		return TOTPFactorRecord{}, false, nil
	}
	if err != nil {
		return TOTPFactorRecord{}, false, err
	}
	return totpFactorFromStore(row), true, nil
}

func (repository *PostgresTOTPFactorRepository) GetVerifiedForUser(ctx context.Context, userID string) (TOTPFactorRecord, bool, error) {
	parsed, err := parseUUID(userID)
	if err != nil {
		return TOTPFactorRecord{}, false, nil
	}
	row, err := repository.q(ctx).GetVerifiedTOTPFactorForUser(ctx, parsed)
	if errors.Is(err, pgx.ErrNoRows) {
		return TOTPFactorRecord{}, false, nil
	}
	if err != nil {
		return TOTPFactorRecord{}, false, err
	}
	return totpFactorFromStore(row), true, nil
}

func (repository *PostgresTOTPFactorRepository) Activate(ctx context.Context, factorID, userID string) (bool, error) {
	parsedFactor, err := parseUUID(factorID)
	if err != nil {
		return false, nil
	}
	parsedUser, err := parseUUID(userID)
	if err != nil {
		return false, nil
	}
	count, err := repository.q(ctx).ActivateTOTPFactor(ctx, store.ActivateTOTPFactorParams{ID: parsedFactor, UserID: parsedUser})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (repository *PostgresTOTPFactorRepository) DeletePendingForUser(ctx context.Context, userID string) error {
	parsed, err := parseUUID(userID)
	if err != nil {
		return ErrUserNotFound
	}
	_, err = repository.q(ctx).DeletePendingTOTPFactorsForUser(ctx, parsed)
	return err
}

func totpFactorFromStore(row store.TotpFactor) TOTPFactorRecord {
	return TOTPFactorRecord{
		ID:               uuid.UUID(row.ID.Bytes).String(),
		UserID:           uuid.UUID(row.UserID.Bytes).String(),
		SecretCiphertext: row.SecretCiphertext,
		SecretNonce:      row.SecretNonce,
		Status:           row.Status,
	}
}

type PostgresTOTPRecoveryCodeRepository struct{ queries *store.Queries }

func NewPostgresTOTPRecoveryCodeRepository(queries *store.Queries) *PostgresTOTPRecoveryCodeRepository {
	return &PostgresTOTPRecoveryCodeRepository{queries: queries}
}

func (repository *PostgresTOTPRecoveryCodeRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return repository.queries.WithTx(tx)
	}
	return repository.queries
}

func (repository *PostgresTOTPRecoveryCodeRepository) CreateMany(ctx context.Context, factorID string, codeDigests [][]byte) error {
	parsed, err := parseUUID(factorID)
	if err != nil {
		return ErrInvalidInput
	}
	q := repository.q(ctx)
	for _, digest := range codeDigests {
		if err := q.CreateTOTPRecoveryCode(ctx, store.CreateTOTPRecoveryCodeParams{FactorID: parsed, CodeDigest: digest}); err != nil {
			return err
		}
	}
	return nil
}

func (repository *PostgresTOTPRecoveryCodeRepository) Consume(ctx context.Context, factorID string, codeDigest []byte) (bool, error) {
	parsed, err := parseUUID(factorID)
	if err != nil {
		return false, nil
	}
	_, err = repository.q(ctx).ConsumeTOTPRecoveryCode(ctx, store.ConsumeTOTPRecoveryCodeParams{FactorID: parsed, CodeDigest: codeDigest})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

type PostgresMFAChallengeRepository struct{ queries *store.Queries }

func NewPostgresMFAChallengeRepository(queries *store.Queries) *PostgresMFAChallengeRepository {
	return &PostgresMFAChallengeRepository{queries: queries}
}

func (repository *PostgresMFAChallengeRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return repository.queries.WithTx(tx)
	}
	return repository.queries
}

func (repository *PostgresMFAChallengeRepository) Create(ctx context.Context, userID, factorID string, expiresAt time.Time) (string, error) {
	parsedUser, err := parseUUID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}
	parsedFactor, err := parseUUID(factorID)
	if err != nil {
		return "", ErrTOTPEnrollmentNotFound
	}
	row, err := repository.q(ctx).CreateMFAChallenge(ctx, store.CreateMFAChallengeParams{
		UserID:    parsedUser,
		FactorID:  parsedFactor,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return "", err
	}
	return uuid.UUID(row.ID.Bytes).String(), nil
}

func (repository *PostgresMFAChallengeRepository) Consume(ctx context.Context, challengeID string) (MFAChallengeRecord, bool, error) {
	parsed, err := parseUUID(challengeID)
	if err != nil {
		return MFAChallengeRecord{}, false, nil
	}
	row, err := repository.q(ctx).ConsumeMFAChallenge(ctx, parsed)
	if errors.Is(err, pgx.ErrNoRows) {
		return MFAChallengeRecord{}, false, nil
	}
	if err != nil {
		return MFAChallengeRecord{}, false, err
	}
	return MFAChallengeRecord{
		ID:       uuid.UUID(row.ID.Bytes).String(),
		UserID:   uuid.UUID(row.UserID.Bytes).String(),
		FactorID: uuid.UUID(row.FactorID.Bytes).String(),
	}, true, nil
}
