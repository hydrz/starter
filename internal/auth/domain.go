package auth

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors returned by the service. Handlers translate these into
// transport responses; they never leak persistence or crypto detail.
var (
	ErrInvalidCredentials = errors.New("auth: invalid email or password")
	ErrEmailTaken         = errors.New("auth: email is already registered")
	ErrInvalidInput       = errors.New("auth: invalid input")
	ErrSessionNotFound    = errors.New("auth: refresh session not found")
	ErrSessionReused      = errors.New("auth: refresh token was already used")
	ErrTokenNotFound      = errors.New("auth: token not found or expired")
	ErrAPIKeyNotFound     = errors.New("auth: api key not found")
)

// User is the identity domain's account record. It never leaves this
// package with its password hash attached to a transport response.
type User struct {
	ID              string
	Email           string
	PasswordHash    string
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// RefreshSession describes one issued opaque refresh token for session
// listing. The raw token and its digest never appear here.
type RefreshSession struct {
	ID         string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	ExpiresAt  time.Time
	UserAgent  string
	IPAddress  string
}

// APIKey describes an API key's metadata. The raw secret is only ever
// returned once, at creation, by CreateAPIKeyResult.
type APIKey struct {
	ID         string
	UserID     string
	Name       string
	KeyPrefix  string
	ExpiresAt  *time.Time
	RevokedAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
}

// CreatedAPIKey pairs an APIKey's metadata with its one-time raw secret.
type CreatedAPIKey struct {
	APIKey
	Key string
}

// OneTimeTokenPurpose distinguishes verification tokens from reset tokens so
// a token minted for one purpose can never be consumed for another.
type OneTimeTokenPurpose string

const (
	PurposeEmailVerification OneTimeTokenPurpose = "email_verification"
	PurposeResetPassword     OneTimeTokenPurpose = "password_reset"
)

// SignUpInput is the validated input to Service.SignUp.
type SignUpInput struct {
	Email    string
	Password string
}

// PasswordSignInInput is the validated input to Service.PasswordSignIn.
type PasswordSignInInput struct {
	Email    string
	Password string
}

// RefreshSessionMetadata captures request context stored alongside a refresh
// session for later display (session listing) and audit.
type RefreshSessionMetadata struct {
	UserAgent string
	IPAddress string
}

// IssuedTokens is returned whenever the service mints a new access/refresh
// token pair (sign-up, sign-in, refresh).
type IssuedTokens struct {
	AccessToken     string
	AccessExpiresAt time.Time
	RefreshToken    string
	RefreshExpires  time.Time
	SessionID       string
	UserID          string
}

// UserRepository persists accounts and password credentials.
type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByID(ctx context.Context, id string) (User, error)
	MarkEmailVerified(ctx context.Context, userID string) (bool, error)
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
}

// RefreshTokenRepository persists refresh token families and sessions.
// ConsumeSession must atomically mark a refresh token used-once: it must
// not be possible for two concurrent callers to both succeed for the same
// token. A caller detecting reuse (the digest exists but consumption failed)
// must revoke the whole family via RevokeFamily.
type RefreshTokenRepository interface {
	CreateFamily(ctx context.Context, userID string) (familyID string, err error)
	CreateSession(ctx context.Context, familyID string, tokenDigest []byte, expiresAt time.Time, metadata RefreshSessionMetadata) (sessionID string, err error)
	// ConsumeSession atomically marks the session identified by tokenDigest
	// as replaced, returning its family and session IDs on success. It
	// returns ok=false without error when the digest is unknown, already
	// consumed/revoked, expired, or its family is revoked.
	ConsumeSession(ctx context.Context, tokenDigest []byte) (familyID, sessionID string, ok bool, err error)
	// FindByDigest reports whether tokenDigest names a session at all
	// (regardless of consumption state) and which user/family it belongs
	// to, so reuse of an already-consumed token can trigger family
	// revocation instead of a generic invalid-token response.
	FindByDigest(ctx context.Context, tokenDigest []byte) (userID, familyID string, found bool, err error)
	RevokeFamily(ctx context.Context, familyID, reason string) error
	RevokeAllForUser(ctx context.Context, userID, reason string) error
	ListSessionsForUser(ctx context.Context, userID string) ([]RefreshSession, error)
	RevokeSessionForUser(ctx context.Context, sessionID, userID string) (bool, error)
}

// OneTimeTokenRepository persists single-use verification/reset tokens.
type OneTimeTokenRepository interface {
	Create(ctx context.Context, userID string, purpose OneTimeTokenPurpose, tokenDigest []byte, expiresAt time.Time) error
	// Consume atomically marks the token used-once, returning the owning
	// user ID on success. ok is false when the digest/purpose pair is
	// unknown, already consumed, or expired.
	Consume(ctx context.Context, tokenDigest []byte, purpose OneTimeTokenPurpose) (userID string, ok bool, err error)
}

// APIKeyRepository persists API key metadata and secret digests.
type APIKeyRepository interface {
	Create(ctx context.Context, userID, name, keyPrefix string, secretDigest []byte, expiresAt *time.Time) (APIKey, error)
	ListForUser(ctx context.Context, userID string) ([]APIKey, error)
	RevokeForUser(ctx context.Context, id, userID string) (bool, error)
	FindActiveByDigest(ctx context.Context, secretDigest []byte) (APIKey, bool, error)
}

// OutboxWriter records a durable event in the same transaction as the
// domain write that produced it. Auth writes exactly one outbox row per
// notification-worthy event (sign-up verification, password reset request);
// it never sends email inline. See internal/delivery for the consumer side.
type OutboxWriter interface {
	WriteEvent(ctx context.Context, topic, aggregateType, aggregateID string, payload []byte, idempotencyKey string) error
}

// Transactor runs fn within an atomic unit of work, matching
// internal/platform/database.Transactor so the service can compose
// multi-repository writes (e.g. create user + write outbox event) without
// depending on pgx directly.
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
