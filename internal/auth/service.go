package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

const (
	MinPasswordLength = 8
	MaxPasswordLength = 128

	verificationTokenTTL = 24 * time.Hour
	passwordResetTTL     = time.Hour

	apiKeyPrefixLength = 8

	// audienceAPI is the JWT "aud" claim value for access tokens. It is a
	// fixed, internal constant rather than configuration because this
	// package issues tokens for exactly one audience.
	audienceAPI = "starter-api"

	outboxTopicVerificationRequested  = "auth.verification_requested"
	outboxTopicPasswordResetRequested = "auth.password_reset_requested"
)

// Service implements the identity/session use cases described by
// docs/architecture/identity-platform.md and ADR-0003. It depends only on
// narrow repository ports plus an Issuer/Verifier pair and a Clock, so it
// can be unit tested without PostgreSQL.
type Service struct {
	users         UserRepository
	refreshTokens RefreshTokenRepository
	oneTimeTokens OneTimeTokenRepository
	apiKeys       APIKeyRepository
	outbox        OutboxWriter
	transactor    Transactor

	issuer   *Issuer
	verifier *Verifier
	digester *SecretDigester
	clock    Clock

	personalOrgCreator PersonalOrgCreator

	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// Dependencies bundles the ports Service needs. All fields are required.
type Dependencies struct {
	Users         UserRepository
	RefreshTokens RefreshTokenRepository
	OneTimeTokens OneTimeTokenRepository
	APIKeys       APIKeyRepository
	Outbox        OutboxWriter
	Transactor    Transactor

	Issuer   *Issuer
	Verifier *Verifier
	Digester *SecretDigester
	Clock    Clock

	PersonalOrgCreator PersonalOrgCreator

	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// NewService validates deps and returns a ready Service.
func NewService(deps Dependencies) (*Service, error) {
	switch {
	case deps.Users == nil, deps.RefreshTokens == nil, deps.OneTimeTokens == nil,
		deps.APIKeys == nil, deps.Outbox == nil, deps.Transactor == nil,
		deps.Issuer == nil, deps.Verifier == nil, deps.Digester == nil:
		return nil, errors.New("auth: all service dependencies are required")
	}
	if deps.AccessTokenTTL <= 0 || deps.RefreshTokenTTL <= 0 {
		return nil, errors.New("auth: access and refresh token ttl must be positive")
	}
	clock := deps.Clock
	if clock == nil {
		clock = SystemClock{}
	}
	return &Service{
		users:              deps.Users,
		refreshTokens:      deps.RefreshTokens,
		oneTimeTokens:      deps.OneTimeTokens,
		apiKeys:            deps.APIKeys,
		outbox:             deps.Outbox,
		transactor:         deps.Transactor,
		issuer:             deps.Issuer,
		verifier:           deps.Verifier,
		digester:           deps.Digester,
		clock:              clock,
		personalOrgCreator: deps.PersonalOrgCreator,
		accessTokenTTL:     deps.AccessTokenTTL,
		refreshTokenTTL:    deps.RefreshTokenTTL,
	}, nil
}

// SignUp creates a new account with a hashed password and, in the same
// transaction, writes an outbox event carrying an email-verification intent.
// It never sends SMTP inline (ADR-0005/0007).
func (service *Service) SignUp(ctx context.Context, input SignUpInput) (User, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return User{}, err
	}
	if err := validatePassword(input.Password); err != nil {
		return User{}, err
	}

	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	var created User
	err = service.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		user, err := service.users.Create(ctx, email, passwordHash)
		if err != nil {
			return err
		}
		created = user

		if service.personalOrgCreator != nil {
			if err := service.personalOrgCreator.CreatePersonalOrg(ctx, user); err != nil {
				return fmt.Errorf("create personal org: %w", err)
			}
		}

		if err := service.writeOneTimeTokenAndOutbox(ctx, user, PurposeEmailVerification, verificationTokenTTL, outboxTopicVerificationRequested); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return User{}, ErrEmailTaken
		}
		return User{}, fmt.Errorf("sign up: %w", err)
	}
	return created, nil
}

// PasswordSignIn verifies email/password and, on success, issues a fresh
// access token plus a new refresh session (family root). Unknown emails and
// wrong passwords return the same ErrInvalidCredentials after performing an
// equivalent-cost Argon2id comparison, to resist enumeration.
func (service *Service) PasswordSignIn(ctx context.Context, input PasswordSignInInput, metadata RefreshSessionMetadata) (IssuedTokens, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return IssuedTokens{}, ErrInvalidCredentials
	}

	user, err := service.users.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return IssuedTokens{}, fmt.Errorf("sign in: %w", err)
	}

	hash := ""
	if err == nil {
		hash = user.PasswordHash
	}
	if !VerifyPasswordOrDummy(hash, input.Password) {
		return IssuedTokens{}, ErrInvalidCredentials
	}

	familyID, err := service.refreshTokens.CreateFamily(ctx, user.ID)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("create refresh family: %w", err)
	}
	return service.issueSessionTokens(ctx, user.ID, familyID, metadata)
}

// RefreshAccessToken rotates the refresh session identified by
// rawRefreshToken. Reuse of an already-consumed or unknown-but-once-issued
// token revokes the entire family, per ADR-0003.
func (service *Service) RefreshAccessToken(ctx context.Context, rawRefreshToken string, metadata RefreshSessionMetadata) (IssuedTokens, error) {
	digest, err := decodeAndDigest(service.digester, rawRefreshToken)
	if err != nil {
		return IssuedTokens{}, ErrSessionNotFound
	}

	familyID, _, ok, err := service.refreshTokens.ConsumeSession(ctx, digest)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("consume refresh session: %w", err)
	}
	if !ok {
		userID, existingFamilyID, found, lookupErr := service.refreshTokens.FindByDigest(ctx, digest)
		if lookupErr != nil {
			return IssuedTokens{}, fmt.Errorf("lookup refresh session: %w", lookupErr)
		}
		if found {
			// The digest exists but could not be consumed (already used,
			// revoked, or expired): treat as replay and revoke the family.
			_ = service.refreshTokens.RevokeFamily(ctx, existingFamilyID, "refresh_token_reuse")
			_ = userID
			return IssuedTokens{}, ErrSessionReused
		}
		return IssuedTokens{}, ErrSessionNotFound
	}

	userID, _, found, err := service.refreshTokens.FindByDigest(ctx, digest)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("lookup refresh session: %w", err)
	}
	if !found {
		return IssuedTokens{}, ErrSessionNotFound
	}
	return service.issueSessionTokens(ctx, userID, familyID, metadata)
}

// SignOut revokes the entire refresh family that rawRefreshToken belongs to.
func (service *Service) SignOut(ctx context.Context, rawRefreshToken string) error {
	digest, err := decodeAndDigest(service.digester, rawRefreshToken)
	if err != nil {
		return nil
	}
	_, familyID, found, err := service.refreshTokens.FindByDigest(ctx, digest)
	if err != nil {
		return fmt.Errorf("sign out: %w", err)
	}
	if !found {
		return nil
	}
	return service.refreshTokens.RevokeFamily(ctx, familyID, "sign_out")
}

// CurrentIdentity returns the user addressed by verified access-token claims.
func (service *Service) CurrentIdentity(ctx context.Context, userID string) (User, error) {
	return service.users.FindByID(ctx, userID)
}

// VerifyAccessToken validates a bearer access token and returns its claims.
func (service *Service) VerifyAccessToken(token string) (Claims, error) {
	return service.verifier.Verify(token, audienceAPI)
}

// RequestEmailVerification issues a fresh verification token and writes an
// outbox event. It is anti-enumeration: it succeeds silently for unknown
// emails.
func (service *Service) RequestEmailVerification(ctx context.Context, email string) error {
	return service.requestOneTimeToken(ctx, email, PurposeEmailVerification, verificationTokenTTL, outboxTopicVerificationRequested)
}

// ConfirmEmailVerification consumes a verification token and marks the
// owning account verified.
func (service *Service) ConfirmEmailVerification(ctx context.Context, rawToken string) error {
	digest, err := decodeAndDigest(service.digester, rawToken)
	if err != nil {
		return ErrTokenNotFound
	}
	userID, ok, err := service.oneTimeTokens.Consume(ctx, digest, PurposeEmailVerification)
	if err != nil {
		return fmt.Errorf("confirm email verification: %w", err)
	}
	if !ok {
		return ErrTokenNotFound
	}
	if _, err := service.users.MarkEmailVerified(ctx, userID); err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	return nil
}

// RequestPasswordReset issues a fresh reset token and writes an outbox
// event. Anti-enumeration: succeeds silently for unknown emails.
func (service *Service) RequestPasswordReset(ctx context.Context, email string) error {
	return service.requestOneTimeToken(ctx, email, PurposeResetPassword, passwordResetTTL, outboxTopicPasswordResetRequested)
}

// ConfirmPasswordReset consumes a reset token, sets a new password, and
// revokes every refresh family for the account so existing sessions cannot
// outlive a credential compromise.
func (service *Service) ConfirmPasswordReset(ctx context.Context, rawToken, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	digest, err := decodeAndDigest(service.digester, rawToken)
	if err != nil {
		return ErrTokenNotFound
	}

	return service.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		userID, ok, err := service.oneTimeTokens.Consume(ctx, digest, PurposeResetPassword)
		if err != nil {
			return fmt.Errorf("confirm password reset: %w", err)
		}
		if !ok {
			return ErrTokenNotFound
		}
		passwordHash, err := HashPassword(newPassword)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		if err := service.users.UpdatePassword(ctx, userID, passwordHash); err != nil {
			return fmt.Errorf("update password: %w", err)
		}
		if err := service.refreshTokens.RevokeAllForUser(ctx, userID, "password_reset"); err != nil {
			return fmt.Errorf("revoke refresh families: %w", err)
		}
		return nil
	})
}

// ListSessions returns the caller's active refresh sessions.
func (service *Service) ListSessions(ctx context.Context, userID string) ([]RefreshSession, error) {
	return service.refreshTokens.ListSessionsForUser(ctx, userID)
}

// RevokeSession revokes one refresh session's family, scoped to userID so a
// caller cannot revoke another account's session.
func (service *Service) RevokeSession(ctx context.Context, sessionID, userID string) error {
	ok, err := service.refreshTokens.RevokeSessionForUser(ctx, sessionID, userID)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if !ok {
		return ErrSessionNotFound
	}
	return nil
}

// CreateAPIKey mints a new API key for userID. The raw key is returned only
// in this call's result and is never recoverable afterward.
func (service *Service) CreateAPIKey(ctx context.Context, userID, name string, expiresAt *time.Time) (CreatedAPIKey, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 {
		return CreatedAPIKey{}, ErrInvalidInput
	}

	rawKey, rawBytes, err := GenerateSecret()
	if err != nil {
		return CreatedAPIKey{}, fmt.Errorf("generate api key: %w", err)
	}
	prefix := rawKey
	if len(prefix) > apiKeyPrefixLength {
		prefix = prefix[:apiKeyPrefixLength]
	}
	digest := service.digester.Digest(rawBytes)

	record, err := service.apiKeys.Create(ctx, userID, name, prefix, digest, expiresAt)
	if err != nil {
		return CreatedAPIKey{}, fmt.Errorf("create api key: %w", err)
	}
	return CreatedAPIKey{APIKey: record, Key: rawKey}, nil
}

// ListAPIKeys returns metadata (never raw secrets) for userID's API keys.
func (service *Service) ListAPIKeys(ctx context.Context, userID string) ([]APIKey, error) {
	return service.apiKeys.ListForUser(ctx, userID)
}

// RevokeAPIKey revokes one API key, scoped to userID.
func (service *Service) RevokeAPIKey(ctx context.Context, id, userID string) error {
	ok, err := service.apiKeys.RevokeForUser(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	if !ok {
		return ErrAPIKeyNotFound
	}
	return nil
}

// AuthenticateAPIKey resolves rawKey to its owning API key record, or
// ErrAPIKeyNotFound if it is unknown, expired, or revoked. It produces a
// minimal Principal; authorization decisions are out of scope here.
func (service *Service) AuthenticateAPIKey(ctx context.Context, rawKey string) (APIKey, error) {
	raw, err := base64.RawURLEncoding.DecodeString(rawKey)
	if err != nil {
		return APIKey{}, ErrAPIKeyNotFound
	}
	digest := service.digester.Digest(raw)
	record, ok, err := service.apiKeys.FindActiveByDigest(ctx, digest)
	if err != nil {
		return APIKey{}, fmt.Errorf("authenticate api key: %w", err)
	}
	if !ok {
		return APIKey{}, ErrAPIKeyNotFound
	}
	return record, nil
}

func (service *Service) issueSessionTokens(ctx context.Context, userID, familyID string, metadata RefreshSessionMetadata) (IssuedTokens, error) {
	refreshRaw, refreshBytes, err := GenerateSecret()
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshDigest := service.digester.Digest(refreshBytes)
	refreshExpiresAt := service.clock.Now().UTC().Add(service.refreshTokenTTL)

	sessionID, err := service.refreshTokens.CreateSession(ctx, familyID, refreshDigest, refreshExpiresAt, metadata)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("create refresh session: %w", err)
	}

	accessToken, claims, err := service.issuer.IssueAccessToken(userID, sessionID, audienceAPI, service.accessTokenTTL)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("issue access token: %w", err)
	}

	return IssuedTokens{
		AccessToken:     accessToken,
		AccessExpiresAt: claims.ExpiresAt,
		RefreshToken:    refreshRaw,
		RefreshExpires:  refreshExpiresAt,
		SessionID:       sessionID,
		UserID:          userID,
	}, nil
}

func (service *Service) requestOneTimeToken(ctx context.Context, email string, purpose OneTimeTokenPurpose, ttl time.Duration, topic string) error {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return nil // anti-enumeration: never reveal validation failure differs from unknown-email
	}

	return service.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		user, err := service.users.FindByEmail(ctx, normalized)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				return nil // anti-enumeration
			}
			return fmt.Errorf("find user: %w", err)
		}
		return service.writeOneTimeTokenAndOutbox(ctx, user, purpose, ttl, topic)
	})
}

func (service *Service) writeOneTimeTokenAndOutbox(ctx context.Context, user User, purpose OneTimeTokenPurpose, ttl time.Duration, topic string) error {
	rawToken, rawBytes, err := GenerateSecret()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	digest := service.digester.Digest(rawBytes)
	expiresAt := service.clock.Now().UTC().Add(ttl)

	if err := service.oneTimeTokens.Create(ctx, user.ID, purpose, digest, expiresAt); err != nil {
		return fmt.Errorf("create one-time token: %w", err)
	}

	payload, err := json.Marshal(struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Token  string `json:"token"`
	}{UserID: user.ID, Email: user.Email, Token: rawToken})
	if err != nil {
		return fmt.Errorf("encode outbox payload: %w", err)
	}

	idempotencyKey := idempotencyKeyFor(topic, user.ID, digest)
	if err := service.outbox.WriteEvent(ctx, topic, "user", user.ID, payload, idempotencyKey); err != nil {
		return fmt.Errorf("write outbox event: %w", err)
	}
	return nil
}

func decodeAndDigest(digester *SecretDigester, raw string) ([]byte, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || len(bytes) == 0 {
		return nil, ErrInvalidInput
	}
	return digester.Digest(bytes), nil
}

func idempotencyKeyFor(topic, userID string, digest []byte) string {
	sum := sha256.Sum256(append([]byte(topic+"|"+userID+"|"), digest...))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func normalizeEmail(email string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(email))
	if trimmed == "" || len(trimmed) > 320 {
		return "", ErrInvalidInput
	}
	if _, err := mail.ParseAddress(trimmed); err != nil {
		return "", ErrInvalidInput
	}
	return trimmed, nil
}

func validatePassword(password string) error {
	if len(password) < MinPasswordLength || len(password) > MaxPasswordLength {
		return ErrInvalidInput
	}
	return nil
}

// ErrUserNotFound is the sentinel a UserRepository implementation must wrap
// or return so Service can distinguish "no such user" from a transport
// error without depending on sql/pgx directly.
var ErrUserNotFound = errors.New("auth: user not found")
