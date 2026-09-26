package auth

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/store"
)

const (
	// PurposeEmailOTPSignIn is the OneTimeToken-style purpose tag for a
	// stand-alone email OTP sign-in code. It is a distinct table
	// (email_otp_challenges) from OneTimeTokenRepository because OTP codes
	// are short, numeric, rate-limited per email/IP, and attempt-limited,
	// none of which apply to the opaque verification/reset tokens.
	PurposeEmailOTPSignIn = "sign_in"

	otpCodeDigits                = 6
	otpTTL                       = 10 * time.Minute
	otpMaxAttempts               = 5
	otpEmailWindow               = 10 * time.Minute
	otpMaxPerEmail               = 3
	otpIPWindow                  = time.Hour
	otpMaxPerIP                  = 10
	outboxTopicEmailOTPRequested = "auth.email_otp_requested"
)

// ErrRateLimited is returned when a caller-visible rate limit (never an
// email-existence-dependent one) has been exceeded.
var ErrRateLimited = errors.New("auth: too many requests")

// EmailOTPChallenge is the narrow record Service needs to verify a code. It
// never carries the raw code, only its digest.
type EmailOTPChallenge struct {
	ID           string
	Email        string
	CodeDigest   []byte
	AttemptCount int
	MaxAttempts  int
	ExpiresAt    time.Time
}

// EmailOTPRepository persists rate-limited, attempt-limited, single-use
// email OTP sign-in challenges.
type EmailOTPRepository interface {
	CreateChallenge(ctx context.Context, email, purpose string, codeDigest []byte, ipAddress string, expiresAt time.Time) error
	CountRecentForEmail(ctx context.Context, email string, since time.Time) (int, error)
	CountRecentForIP(ctx context.Context, ipAddress string, since time.Time) (int, error)
	FindActive(ctx context.Context, email, purpose string) (EmailOTPChallenge, bool, error)
	// IncrementAttempt atomically increments the attempt counter, returning
	// ok=false when the challenge is missing, expired, or attempts are
	// already exhausted (so a caller cannot keep guessing past the limit).
	IncrementAttempt(ctx context.Context, id string) (ok bool, err error)
	// Consume atomically marks the challenge used-once.
	Consume(ctx context.Context, id string) (ok bool, err error)
}

// RequestEmailOTP issues a fresh one-time sign-in code by email, subject to
// per-IP and per-email rate limits, and writes an outbox event for delivery.
// Anti-enumeration: the response is identical for a known or unknown email,
// and for an email that is silently rate-limited (soft-dropped rather than
// surfaced), matching RequestPasswordReset's semantics. Only an IP-level
// limit — which never depends on whether the email exists — is surfaced to
// the caller as ErrRateLimited.
func (service *Service) RequestEmailOTP(ctx context.Context, email, ipAddress string) error {
	if service.emailOTP == nil {
		return ErrInvalidInput
	}
	now := service.clock.Now().UTC()
	if ipAddress != "" {
		count, err := service.emailOTP.CountRecentForIP(ctx, ipAddress, now.Add(-otpIPWindow))
		if err != nil {
			return fmt.Errorf("count recent otp requests by ip: %w", err)
		}
		if count >= otpMaxPerIP {
			return ErrRateLimited
		}
	}

	normalized, err := normalizeEmail(email)
	if err != nil {
		return nil // anti-enumeration: invalid input looks identical to unknown email
	}

	return service.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		user, err := service.users.FindByEmail(ctx, normalized)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				return nil // anti-enumeration
			}
			return fmt.Errorf("find user: %w", err)
		}

		emailCount, err := service.emailOTP.CountRecentForEmail(ctx, normalized, now.Add(-otpEmailWindow))
		if err != nil {
			return fmt.Errorf("count recent otp requests by email: %w", err)
		}
		if emailCount >= otpMaxPerEmail {
			return nil // soft rate limit: identical response, no new code issued
		}

		code, err := generateNumericCode(otpCodeDigits)
		if err != nil {
			return fmt.Errorf("generate otp code: %w", err)
		}
		digest := service.digester.Digest([]byte(code))
		expiresAt := now.Add(otpTTL)
		if err := service.emailOTP.CreateChallenge(ctx, normalized, PurposeEmailOTPSignIn, digest, ipAddress, expiresAt); err != nil {
			return fmt.Errorf("create email otp challenge: %w", err)
		}

		payload := struct {
			UserID string `json:"user_id"`
			Email  string `json:"email"`
			Code   string `json:"code"`
		}{UserID: user.ID, Email: user.Email, Code: code}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode outbox payload: %w", err)
		}
		idempotencyKey := idempotencyKeyFor(outboxTopicEmailOTPRequested, user.ID, digest)
		if err := service.outbox.WriteEvent(ctx, outboxTopicEmailOTPRequested, "user", user.ID, payloadBytes, idempotencyKey); err != nil {
			return fmt.Errorf("write outbox event: %w", err)
		}
		return nil
	})
}

// VerifyEmailOTP consumes a pending email OTP challenge and, on success,
// completes sign-in through the same issueSessionTokens path every other
// sign-in-completing flow uses. Wrong codes and unknown/expired challenges
// return the same ErrInvalidCredentials so a caller cannot distinguish them.
func (service *Service) VerifyEmailOTP(ctx context.Context, email, code string, metadata RefreshSessionMetadata) (IssuedTokens, error) {
	if service.emailOTP == nil {
		return IssuedTokens{}, ErrInvalidCredentials
	}
	normalized, err := normalizeEmail(email)
	if err != nil {
		return IssuedTokens{}, ErrInvalidCredentials
	}

	challenge, ok, err := service.emailOTP.FindActive(ctx, normalized, PurposeEmailOTPSignIn)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("find email otp challenge: %w", err)
	}
	if !ok {
		return IssuedTokens{}, ErrInvalidCredentials
	}

	attemptOK, err := service.emailOTP.IncrementAttempt(ctx, challenge.ID)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("increment email otp attempt: %w", err)
	}
	if !attemptOK {
		return IssuedTokens{}, ErrInvalidCredentials
	}

	if !service.digester.Verify(challenge.CodeDigest, []byte(code)) {
		return IssuedTokens{}, ErrInvalidCredentials
	}

	consumed, err := service.emailOTP.Consume(ctx, challenge.ID)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("consume email otp challenge: %w", err)
	}
	if !consumed {
		return IssuedTokens{}, ErrInvalidCredentials
	}

	user, err := service.users.FindByEmail(ctx, normalized)
	if err != nil {
		return IssuedTokens{}, ErrInvalidCredentials
	}

	familyID, err := service.refreshTokens.CreateFamily(ctx, user.ID)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("create refresh family: %w", err)
	}
	return service.issueSessionTokens(ctx, user.ID, familyID, metadata)
}

func generateNumericCode(digits int) (string, error) {
	max := int64(1)
	for range digits {
		max *= 10
	}
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", digits, n.Int64()), nil
}

// PostgresEmailOTPRepository implements EmailOTPRepository over internal/store.
type PostgresEmailOTPRepository struct {
	queries *store.Queries
}

func NewPostgresEmailOTPRepository(queries *store.Queries) *PostgresEmailOTPRepository {
	return &PostgresEmailOTPRepository{queries: queries}
}

func (repository *PostgresEmailOTPRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return repository.queries.WithTx(tx)
	}
	return repository.queries
}

func (repository *PostgresEmailOTPRepository) CreateChallenge(ctx context.Context, email, purpose string, codeDigest []byte, ipAddress string, expiresAt time.Time) error {
	_, err := repository.q(ctx).CreateEmailOTPChallenge(ctx, store.CreateEmailOTPChallengeParams{
		Email:      email,
		Purpose:    purpose,
		CodeDigest: codeDigest,
		IpAddress:  parseOptionalIP(ipAddress),
		ExpiresAt:  pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	return err
}

func (repository *PostgresEmailOTPRepository) CountRecentForEmail(ctx context.Context, email string, since time.Time) (int, error) {
	count, err := repository.q(ctx).CountRecentEmailOTPChallenges(ctx, store.CountRecentEmailOTPChallengesParams{
		Email:     email,
		CreatedAt: pgtype.Timestamptz{Time: since, Valid: true},
	})
	return int(count), err
}

func (repository *PostgresEmailOTPRepository) CountRecentForIP(ctx context.Context, ipAddress string, since time.Time) (int, error) {
	count, err := repository.q(ctx).CountRecentEmailOTPChallengesByIP(ctx, store.CountRecentEmailOTPChallengesByIPParams{
		IpAddress: parseOptionalIP(ipAddress),
		CreatedAt: pgtype.Timestamptz{Time: since, Valid: true},
	})
	return int(count), err
}

func (repository *PostgresEmailOTPRepository) FindActive(ctx context.Context, email, purpose string) (EmailOTPChallenge, bool, error) {
	row, err := repository.q(ctx).FindActiveEmailOTPChallenge(ctx, store.FindActiveEmailOTPChallengeParams{Email: email, Purpose: purpose})
	if errors.Is(err, pgx.ErrNoRows) {
		return EmailOTPChallenge{}, false, nil
	}
	if err != nil {
		return EmailOTPChallenge{}, false, err
	}
	return EmailOTPChallenge{
		ID:           uuid.UUID(row.ID.Bytes).String(),
		Email:        row.Email,
		CodeDigest:   row.CodeDigest,
		AttemptCount: int(row.AttemptCount),
		MaxAttempts:  int(row.MaxAttempts),
		ExpiresAt:    row.ExpiresAt.Time,
	}, true, nil
}

func (repository *PostgresEmailOTPRepository) IncrementAttempt(ctx context.Context, id string) (bool, error) {
	parsed, err := parseUUID(id)
	if err != nil {
		return false, nil
	}
	_, err = repository.q(ctx).IncrementEmailOTPAttempt(ctx, parsed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (repository *PostgresEmailOTPRepository) Consume(ctx context.Context, id string) (bool, error) {
	parsed, err := parseUUID(id)
	if err != nil {
		return false, nil
	}
	_, err = repository.q(ctx).ConsumeEmailOTPChallengeByID(ctx, parsed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func parseOptionalIP(raw string) *netip.Addr {
	if raw == "" {
		return nil
	}
	parsed, err := netip.ParseAddr(raw)
	if err != nil {
		return nil
	}
	return &parsed
}
