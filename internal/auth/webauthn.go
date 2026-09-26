package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/store"
)

const (
	webauthnChallengeTTL           = 5 * time.Minute
	webauthnCeremonyRegistration   = "registration"
	webauthnCeremonyAuthentication = "authentication"
)

var (
	// ErrWebAuthnUnauthenticated is returned by BeginWebAuthnRegistration
	// for a caller with no authenticated userID: passkey registration is
	// rejected outright at the service layer, never relying on HTTP
	// middleware alone.
	ErrWebAuthnUnauthenticated = errors.New("auth: webauthn registration requires an authenticated session")
	// ErrWebAuthnChallengeInvalid covers an unknown, expired, or
	// already-consumed ceremony challenge.
	ErrWebAuthnChallengeInvalid = errors.New("auth: invalid or expired webauthn challenge")
	// ErrWebAuthnVerificationFailed covers any ceremony verification
	// failure: origin, RP ID, challenge mismatch, credential ownership, or
	// a non-increasing signature counter (clone detection). It
	// intentionally does not distinguish the reason to callers.
	ErrWebAuthnVerificationFailed = errors.New("auth: webauthn verification failed")
)

// WebAuthnCeremonies wraps github.com/go-webauthn/webauthn's relying-party
// configuration. It is the only file in this package that imports that
// module, isolating origin/RPID/challenge/counter verification behind the
// narrow methods Service calls.
type WebAuthnCeremonies struct {
	rp *webauthn.WebAuthn
}

// NewWebAuthnCeremonies builds relying-party configuration from rpID,
// rpName, and the set of permitted origins.
func NewWebAuthnCeremonies(rpID, rpName string, origins []string) (*WebAuthnCeremonies, error) {
	rp, err := webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: rpName,
		RPOrigins:     origins,
	})
	if err != nil {
		return nil, fmt.Errorf("configure webauthn relying party: %w", err)
	}
	return &WebAuthnCeremonies{rp: rp}, nil
}

// webauthnUser adapts a user's stored credentials to webauthn.User.
type webauthnUser struct {
	id          []byte
	name        string
	credentials []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte                         { return u.id }
func (u *webauthnUser) WebAuthnName() string                       { return u.name }
func (u *webauthnUser) WebAuthnDisplayName() string                { return u.name }
func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// WebAuthnCredentialRecord is the narrow record Service/postgres store.
type WebAuthnCredentialRecord struct {
	ID           string
	UserID       string
	CredentialID []byte
	PublicKey    []byte
	SignCount    uint32
	UserHandle   []byte
}

// WebAuthnCredentialRepository persists registered passkeys.
type WebAuthnCredentialRepository interface {
	Create(ctx context.Context, userID string, credentialID, publicKey []byte, signCount uint32, userHandle []byte) error
	ListForUser(ctx context.Context, userID string) ([]WebAuthnCredentialRecord, error)
	FindByCredentialID(ctx context.Context, credentialID []byte) (WebAuthnCredentialRecord, bool, error)
	// UpdateSignCount atomically advances the stored counter, only when
	// newCount is strictly greater than the stored value (defense in depth
	// alongside the library's own clone-warning detection).
	UpdateSignCount(ctx context.Context, id string, newCount uint32) (bool, error)
}

// WebAuthnChallengeRepository persists server-side ceremony state.
type WebAuthnChallengeRepository interface {
	Create(ctx context.Context, ceremony, userID string, challenge, sessionData []byte, expiresAt time.Time) error
	// Consume atomically marks the challenge used-once, returning its
	// stored session data.
	Consume(ctx context.Context, ceremony string, challenge []byte) (sessionData []byte, userID string, ok bool, err error)
}

// BeginWebAuthnRegistration starts a registration ceremony for an
// authenticated userID. It is rejected outright — before touching any
// repository — for an empty userID, so an unauthenticated caller can never
// reach credential creation regardless of HTTP-layer wiring.
func (service *Service) BeginWebAuthnRegistration(ctx context.Context, userID string) (*protocol.CredentialCreation, error) {
	if userID == "" {
		return nil, ErrWebAuthnUnauthenticated
	}
	if service.webauthn == nil || service.webauthnChallenges == nil {
		return nil, ErrInvalidInput
	}
	user, err := service.users.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	existing, err := service.webauthnCredentials.ListForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list existing credentials: %w", err)
	}

	handle := make([]byte, 32)
	if _, err := rand.Read(handle); err != nil {
		return nil, fmt.Errorf("generate user handle: %w", err)
	}
	waUser := &webauthnUser{id: handle, name: user.Email, credentials: toWebAuthnCredentials(existing)}

	creation, session, err := service.webauthn.rp.BeginRegistration(waUser)
	if err != nil {
		return nil, fmt.Errorf("begin webauthn registration: %w", err)
	}
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("encode webauthn session: %w", err)
	}
	expiresAt := service.clock.Now().UTC().Add(webauthnChallengeTTL)
	if err := service.webauthnChallenges.Create(ctx, webauthnCeremonyRegistration, userID, []byte(session.Challenge), sessionJSON, expiresAt); err != nil {
		return nil, fmt.Errorf("store webauthn challenge: %w", err)
	}
	// The user handle bound to this ceremony must be persisted alongside
	// the credential at finish time; stash it in the creation response's
	// user ID field, which the client already round-trips verbatim.
	creation.Response.User.ID = protocol.URLEncodedBase64(handle)
	return creation, nil
}

// FinishWebAuthnRegistration completes a registration ceremony: it consumes
// the stored challenge (rejecting any without a matching, unexpired,
// unconsumed record), verifies the client/authenticator response against
// it, and persists the new credential tied to userID.
func (service *Service) FinishWebAuthnRegistration(ctx context.Context, userID string, credentialJSON []byte) error {
	if userID == "" {
		return ErrWebAuthnUnauthenticated
	}
	if service.webauthn == nil || service.webauthnChallenges == nil || service.webauthnCredentials == nil {
		return ErrInvalidInput
	}

	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(credentialJSON))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrWebAuthnVerificationFailed, err)
	}

	sessionData, sessionUserID, ok, err := service.webauthnChallenges.Consume(ctx, webauthnCeremonyRegistration, []byte(parsedResponse.Response.CollectedClientData.Challenge))
	if err != nil {
		return fmt.Errorf("consume webauthn challenge: %w", err)
	}
	if !ok || sessionUserID != userID {
		return ErrWebAuthnChallengeInvalid
	}
	var session webauthn.SessionData
	if err := json.Unmarshal(sessionData, &session); err != nil {
		return fmt.Errorf("decode webauthn session: %w", err)
	}

	waUser := &webauthnUser{id: session.UserID, name: userID}
	credential, err := service.webauthn.rp.CreateCredential(waUser, session, parsedResponse)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrWebAuthnVerificationFailed, err)
	}

	if err := service.webauthnCredentials.Create(ctx, userID, credential.ID, credential.PublicKey, uint32(credential.Authenticator.SignCount), session.UserID); err != nil {
		return fmt.Errorf("store webauthn credential: %w", err)
	}
	return nil
}

// BeginWebAuthnAuthentication starts a discoverable (usernameless) login
// ceremony: the RP does not know the caller's identity until the assertion
// arrives, so no userID is required or accepted here.
func (service *Service) BeginWebAuthnAuthentication(ctx context.Context) (*protocol.CredentialAssertion, error) {
	if service.webauthn == nil || service.webauthnChallenges == nil {
		return nil, ErrInvalidInput
	}
	assertion, session, err := service.webauthn.rp.BeginDiscoverableLogin()
	if err != nil {
		return nil, fmt.Errorf("begin webauthn login: %w", err)
	}
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("encode webauthn session: %w", err)
	}
	expiresAt := service.clock.Now().UTC().Add(webauthnChallengeTTL)
	if err := service.webauthnChallenges.Create(ctx, webauthnCeremonyAuthentication, "", []byte(session.Challenge), sessionJSON, expiresAt); err != nil {
		return nil, fmt.Errorf("store webauthn challenge: %w", err)
	}
	return assertion, nil
}

// FinishWebAuthnAuthentication completes a discoverable login ceremony. It
// verifies origin, RP ID, and challenge match via the webauthn library,
// resolves the credential owner strictly from the authenticator-reported
// user handle (never a client-supplied identity claim), rejects a
// non-increasing signature counter as a clone signal, and on success
// completes sign-in through issueSessionTokens like every other flow.
func (service *Service) FinishWebAuthnAuthentication(ctx context.Context, assertionJSON []byte, metadata RefreshSessionMetadata) (IssuedTokens, error) {
	if service.webauthn == nil || service.webauthnChallenges == nil || service.webauthnCredentials == nil {
		return IssuedTokens{}, ErrInvalidInput
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(assertionJSON))
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("%w: %v", ErrWebAuthnVerificationFailed, err)
	}

	sessionData, _, ok, err := service.webauthnChallenges.Consume(ctx, webauthnCeremonyAuthentication, []byte(parsedResponse.Response.CollectedClientData.Challenge))
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("consume webauthn challenge: %w", err)
	}
	if !ok {
		return IssuedTokens{}, ErrWebAuthnChallengeInvalid
	}
	var session webauthn.SessionData
	if err := json.Unmarshal(sessionData, &session); err != nil {
		return IssuedTokens{}, fmt.Errorf("decode webauthn session: %w", err)
	}

	var resolvedRecord WebAuthnCredentialRecord
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		record, ok, err := service.webauthnCredentials.FindByCredentialID(ctx, rawID)
		if err != nil {
			return nil, err
		}
		// Credential ownership check: the stored credential's user handle
		// must match exactly what the authenticator asserted.
		if !ok || !bytes.Equal(record.UserHandle, userHandle) {
			return nil, protocol.ErrBadRequest.WithDetails("unknown credential or user handle mismatch")
		}
		resolvedRecord = record
		return &webauthnUser{id: userHandle, name: record.UserID, credentials: toWebAuthnCredentials([]WebAuthnCredentialRecord{record})}, nil
	}

	credential, err := service.webauthn.rp.ValidateDiscoverableLogin(handler, session, parsedResponse)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("%w: %v", ErrWebAuthnVerificationFailed, err)
	}
	if credential.Authenticator.CloneWarning {
		// A non-increasing signature counter: reject as a possible cloned
		// authenticator rather than accepting and merely flagging it.
		return IssuedTokens{}, ErrWebAuthnVerificationFailed
	}
	if updated, err := service.webauthnCredentials.UpdateSignCount(ctx, resolvedRecord.ID, uint32(credential.Authenticator.SignCount)); err != nil {
		return IssuedTokens{}, fmt.Errorf("update sign count: %w", err)
	} else if !updated && credential.Authenticator.SignCount != 0 {
		return IssuedTokens{}, ErrWebAuthnVerificationFailed
	}

	if challengeID, err := service.checkMFARequired(ctx, resolvedRecord.UserID); err != nil {
		return IssuedTokens{}, fmt.Errorf("check mfa: %w", err)
	} else if challengeID != "" {
		return IssuedTokens{MFAChallengeID: challengeID}, ErrMFARequired
	}

	familyID, err := service.refreshTokens.CreateFamily(ctx, resolvedRecord.UserID)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("create refresh family: %w", err)
	}
	return service.issueSessionTokens(ctx, resolvedRecord.UserID, familyID, metadata)
}

func toWebAuthnCredentials(records []WebAuthnCredentialRecord) []webauthn.Credential {
	result := make([]webauthn.Credential, 0, len(records))
	for _, record := range records {
		result = append(result, webauthn.Credential{
			ID:        record.CredentialID,
			PublicKey: record.PublicKey,
			Authenticator: webauthn.Authenticator{
				SignCount: record.SignCount,
			},
		})
	}
	return result
}

// --- PostgreSQL adapters ---------------------------------------------------

type PostgresWebAuthnCredentialRepository struct{ queries *store.Queries }

func NewPostgresWebAuthnCredentialRepository(queries *store.Queries) *PostgresWebAuthnCredentialRepository {
	return &PostgresWebAuthnCredentialRepository{queries: queries}
}

func (repository *PostgresWebAuthnCredentialRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return repository.queries.WithTx(tx)
	}
	return repository.queries
}

func (repository *PostgresWebAuthnCredentialRepository) Create(ctx context.Context, userID string, credentialID, publicKey []byte, signCount uint32, userHandle []byte) error {
	parsed, err := parseUUID(userID)
	if err != nil {
		return ErrUserNotFound
	}
	_, err = repository.q(ctx).CreateWebAuthnCredential(ctx, store.CreateWebAuthnCredentialParams{
		UserID:       parsed,
		CredentialID: credentialID,
		PublicKey:    publicKey,
		SignCount:    int64(signCount),
		UserHandle:   userHandle,
	})
	return err
}

func (repository *PostgresWebAuthnCredentialRepository) ListForUser(ctx context.Context, userID string) ([]WebAuthnCredentialRecord, error) {
	parsed, err := parseUUID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	rows, err := repository.q(ctx).ListWebAuthnCredentialsForUser(ctx, parsed)
	if err != nil {
		return nil, err
	}
	records := make([]WebAuthnCredentialRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, webauthnRecordFromStore(row))
	}
	return records, nil
}

func (repository *PostgresWebAuthnCredentialRepository) FindByCredentialID(ctx context.Context, credentialID []byte) (WebAuthnCredentialRecord, bool, error) {
	row, err := repository.q(ctx).GetWebAuthnCredentialByCredentialID(ctx, credentialID)
	if errors.Is(err, pgx.ErrNoRows) {
		return WebAuthnCredentialRecord{}, false, nil
	}
	if err != nil {
		return WebAuthnCredentialRecord{}, false, err
	}
	return webauthnRecordFromStore(row), true, nil
}

func (repository *PostgresWebAuthnCredentialRepository) UpdateSignCount(ctx context.Context, id string, newCount uint32) (bool, error) {
	parsed, err := parseUUID(id)
	if err != nil {
		return false, nil
	}
	count, err := repository.q(ctx).UpdateWebAuthnCredentialSignCount(ctx, store.UpdateWebAuthnCredentialSignCountParams{ID: parsed, SignCount: int64(newCount)})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func webauthnRecordFromStore(row store.WebauthnCredential) WebAuthnCredentialRecord {
	return WebAuthnCredentialRecord{
		ID:           uuid.UUID(row.ID.Bytes).String(),
		UserID:       uuid.UUID(row.UserID.Bytes).String(),
		CredentialID: row.CredentialID,
		PublicKey:    row.PublicKey,
		SignCount:    uint32(row.SignCount),
		UserHandle:   row.UserHandle,
	}
}

type PostgresWebAuthnChallengeRepository struct{ queries *store.Queries }

func NewPostgresWebAuthnChallengeRepository(queries *store.Queries) *PostgresWebAuthnChallengeRepository {
	return &PostgresWebAuthnChallengeRepository{queries: queries}
}

func (repository *PostgresWebAuthnChallengeRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return repository.queries.WithTx(tx)
	}
	return repository.queries
}

func (repository *PostgresWebAuthnChallengeRepository) Create(ctx context.Context, ceremony, userID string, challenge, sessionData []byte, expiresAt time.Time) error {
	var userIDValue pgtype.UUID
	if userID != "" {
		parsed, err := parseUUID(userID)
		if err != nil {
			return ErrUserNotFound
		}
		userIDValue = parsed
	}
	_, err := repository.q(ctx).CreateWebAuthnChallenge(ctx, store.CreateWebAuthnChallengeParams{
		Ceremony:    ceremony,
		UserID:      userIDValue,
		Challenge:   challenge,
		SessionData: sessionData,
		ExpiresAt:   pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	return err
}

func (repository *PostgresWebAuthnChallengeRepository) Consume(ctx context.Context, ceremony string, challenge []byte) ([]byte, string, bool, error) {
	row, err := repository.q(ctx).ConsumeWebAuthnChallenge(ctx, store.ConsumeWebAuthnChallengeParams{Challenge: challenge, Ceremony: ceremony})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", false, nil
	}
	if err != nil {
		return nil, "", false, err
	}
	userID := ""
	if row.UserID.Valid {
		userID = uuid.UUID(row.UserID.Bytes).String()
	}
	return row.SessionData, userID, true, nil
}
