package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/store"
)

const (
	oauthStateTTL         = 10 * time.Minute
	oauthIntentSignIn     = "sign_in"
	oauthIntentLink       = "link"
	oauthLinkReauthWindow = 5 * time.Minute
)

var (
	// ErrOAuthProviderUnknown is returned for a provider with no configured
	// OAuthClient.
	ErrOAuthProviderUnknown = errors.New("auth: unknown oauth provider")
	// ErrOAuthStateInvalid covers an unknown, expired, already-consumed, or
	// intent-mismatched authorization state/PKCE round trip.
	ErrOAuthStateInvalid = errors.New("auth: invalid oauth state")
	// ErrOAuthAccountTaken is returned when linking would attach a
	// (provider, subject) identity that already belongs to a different
	// account. Account resolution never merges on email.
	ErrOAuthAccountTaken = errors.New("auth: oauth identity already linked to another account")
	// ErrReauthenticationRequired is returned when linking is attempted
	// with an access token that was not issued recently enough.
	ErrReauthenticationRequired = errors.New("auth: recent reauthentication is required")
)

// OAuthUserInfo is the normalized identity an OAuthClient resolves after
// exchanging an authorization code. Subject is the provider's stable,
// opaque user id — the only field account resolution keys on.
type OAuthUserInfo struct {
	Subject string
	Email   string
}

// OAuthClient performs one provider's authorization-code + PKCE exchange.
// Production wiring (oauth.go's NewOAuthClient, not shown here to keep
// golang.org/x/oauth2 usage isolated) implements this over
// golang.org/x/oauth2 plus the provider's userinfo endpoint; tests inject a
// fake so no network call is ever required to run `go test`.
type OAuthClient interface {
	// AuthCodeURL returns the provider's authorization URL for state and a
	// PKCE S256 codeChallenge.
	AuthCodeURL(state, codeChallenge string) string
	// Exchange trades an authorization code and its PKCE verifier for a
	// normalized identity.
	Exchange(ctx context.Context, code, codeVerifier string) (OAuthUserInfo, error)
}

// OAuthAccount is one linked external identity.
type OAuthAccount struct {
	Provider  string
	Subject   string
	Email     string
	CreatedAt time.Time
}

// OAuthAccountRepository persists the (provider, provider_subject) identity
// key. It never resolves or merges accounts by email.
type OAuthAccountRepository interface {
	Find(ctx context.Context, provider, subject string) (userID string, found bool, err error)
	Create(ctx context.Context, userID, provider, subject, email string) error
	ListForUser(ctx context.Context, userID string) ([]OAuthAccount, error)
}

// OAuthStateRecord is returned by OAuthStateRepository.Consume.
type OAuthStateRecord struct {
	Provider      string
	Nonce         string
	CodeVerifier  string
	Intent        string
	LinkingUserID string
}

// OAuthStateRepository persists one-time state/nonce/PKCE-verifier records.
type OAuthStateRepository interface {
	Create(ctx context.Context, provider string, stateDigest []byte, nonce, codeVerifier, intent, linkingUserID string, expiresAt time.Time) error
	// Consume atomically marks the state used-once.
	Consume(ctx context.Context, provider string, stateDigest []byte) (OAuthStateRecord, bool, error)
}

// BeginOAuthSignIn returns provider's authorization URL for an
// unauthenticated sign-in/sign-up attempt, persisting state/PKCE server-side.
func (service *Service) BeginOAuthSignIn(ctx context.Context, provider string) (string, error) {
	return service.beginOAuth(ctx, provider, oauthIntentSignIn, "")
}

// BeginOAuthLink returns provider's authorization URL to link the given
// authenticated account, requiring that the caller's access token was
// issued within oauthLinkReauthWindow — never auto-linking by email, and
// never linking without a recently-authenticated session.
func (service *Service) BeginOAuthLink(ctx context.Context, provider, userID string, accessTokenIssuedAt time.Time) (string, error) {
	if service.clock.Now().UTC().Sub(accessTokenIssuedAt) > oauthLinkReauthWindow {
		return "", ErrReauthenticationRequired
	}
	return service.beginOAuth(ctx, provider, oauthIntentLink, userID)
}

func (service *Service) beginOAuth(ctx context.Context, provider, intent, linkingUserID string) (string, error) {
	if service.oauthStates == nil {
		return "", ErrOAuthProviderUnknown
	}
	client, ok := service.oauthClients[provider]
	if !ok {
		return "", ErrOAuthProviderUnknown
	}

	state, stateBytes, err := GenerateSecret()
	if err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	nonce, _, err := GenerateSecret()
	if err != nil {
		return "", fmt.Errorf("generate oauth nonce: %w", err)
	}
	codeVerifier, _, err := GenerateSecret()
	if err != nil {
		return "", fmt.Errorf("generate pkce verifier: %w", err)
	}
	challengeSum := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(challengeSum[:])

	digest := service.digester.Digest(stateBytes)
	expiresAt := service.clock.Now().UTC().Add(oauthStateTTL)
	if err := service.oauthStates.Create(ctx, provider, digest, nonce, codeVerifier, intent, linkingUserID, expiresAt); err != nil {
		return "", fmt.Errorf("store oauth state: %w", err)
	}

	return client.AuthCodeURL(state, codeChallenge), nil
}

// CompleteOAuthSignIn handles a provider callback for an unauthenticated
// sign-in attempt: it verifies state/PKCE, exchanges the code, resolves an
// existing (provider, subject) account or creates a brand-new one (never by
// matching email), and completes sign-in through issueSessionTokens like
// every other flow.
func (service *Service) CompleteOAuthSignIn(ctx context.Context, provider, state, code string, metadata RefreshSessionMetadata) (IssuedTokens, error) {
	record, err := service.consumeOAuthState(ctx, provider, state, oauthIntentSignIn)
	if err != nil {
		return IssuedTokens{}, err
	}

	client, ok := service.oauthClients[provider]
	if !ok {
		return IssuedTokens{}, ErrOAuthProviderUnknown
	}
	info, err := client.Exchange(ctx, code, record.CodeVerifier)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("exchange oauth code: %w", err)
	}
	if info.Subject == "" {
		return IssuedTokens{}, ErrOAuthStateInvalid
	}

	var userID string
	err = service.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		existingUserID, found, err := service.oauthAccounts.Find(ctx, provider, info.Subject)
		if err != nil {
			return fmt.Errorf("find oauth account: %w", err)
		}
		if found {
			userID = existingUserID
			return nil
		}

		// No account yet for this (provider, subject) identity: create a
		// brand-new user. This never looks at, matches, or merges on
		// info.Email — a different provider_subject always gets its own
		// account even if the provider reports the same email address.
		randomPassword, _, err := GenerateSecret()
		if err != nil {
			return fmt.Errorf("generate placeholder credential: %w", err)
		}
		passwordHash, err := HashPassword(randomPassword)
		if err != nil {
			return fmt.Errorf("hash placeholder credential: %w", err)
		}
		// The users.email column is unique, but the provider's reported
		// email is not: two different (provider, subject) identities can
		// legitimately report the same address (a shared mailbox, or one
		// provider's stale data). Account resolution keys strictly on
		// (provider, subject), so the login email is always a synthetic,
		// per-identity value; the provider's email is kept only on the
		// oauth_accounts row (info.Email below) for display.
		syntheticEmail, err := normalizeEmail(provider + "+" + uuid.NewString() + "@oauth.invalid")
		if err != nil {
			return fmt.Errorf("build synthetic oauth email: %w", err)
		}
		user, err := service.users.Create(ctx, syntheticEmail, passwordHash)
		if err != nil {
			return fmt.Errorf("create user for oauth identity: %w", err)
		}
		if service.personalOrgCreator != nil {
			if err := service.personalOrgCreator.CreatePersonalOrg(ctx, user); err != nil {
				return fmt.Errorf("create personal org: %w", err)
			}
		}
		if err := service.oauthAccounts.Create(ctx, user.ID, provider, info.Subject, info.Email); err != nil {
			return fmt.Errorf("create oauth account: %w", err)
		}
		userID = user.ID
		return nil
	})
	if err != nil {
		return IssuedTokens{}, err
	}

	if challengeID, err := service.checkMFARequired(ctx, userID); err != nil {
		return IssuedTokens{}, fmt.Errorf("check mfa: %w", err)
	} else if challengeID != "" {
		return IssuedTokens{MFAChallengeID: challengeID}, ErrMFARequired
	}

	familyID, err := service.refreshTokens.CreateFamily(ctx, userID)
	if err != nil {
		return IssuedTokens{}, fmt.Errorf("create refresh family: %w", err)
	}
	return service.issueSessionTokens(ctx, userID, familyID, metadata)
}

// LinkOAuthAccount handles a provider callback for an authenticated
// "link this provider to my account" attempt. userID must match the state's
// linking_user_id (bound at BeginOAuthLink time): an unauthenticated or
// mismatched-session callback can never attach a provider identity to an
// account it did not itself begin the flow from.
func (service *Service) LinkOAuthAccount(ctx context.Context, provider, state, code, userID string) error {
	record, err := service.consumeOAuthState(ctx, provider, state, oauthIntentLink)
	if err != nil {
		return err
	}
	if record.LinkingUserID != userID {
		return ErrOAuthStateInvalid
	}

	client, ok := service.oauthClients[provider]
	if !ok {
		return ErrOAuthProviderUnknown
	}
	info, err := client.Exchange(ctx, code, record.CodeVerifier)
	if err != nil {
		return fmt.Errorf("exchange oauth code: %w", err)
	}

	existingUserID, found, err := service.oauthAccounts.Find(ctx, provider, info.Subject)
	if err != nil {
		return fmt.Errorf("find oauth account: %w", err)
	}
	if found {
		if existingUserID == userID {
			return nil // idempotent: already linked to this account
		}
		return ErrOAuthAccountTaken
	}

	return service.oauthAccounts.Create(ctx, userID, provider, info.Subject, info.Email)
}

// ListOAuthAccounts returns userID's linked provider identities.
func (service *Service) ListOAuthAccounts(ctx context.Context, userID string) ([]OAuthAccount, error) {
	if service.oauthAccounts == nil {
		return nil, nil
	}
	return service.oauthAccounts.ListForUser(ctx, userID)
}

func (service *Service) consumeOAuthState(ctx context.Context, provider, state, wantIntent string) (OAuthStateRecord, error) {
	if service.oauthStates == nil {
		return OAuthStateRecord{}, ErrOAuthProviderUnknown
	}
	digest, err := decodeAndDigest(service.digester, state)
	if err != nil {
		return OAuthStateRecord{}, ErrOAuthStateInvalid
	}
	record, ok, err := service.oauthStates.Consume(ctx, provider, digest)
	if err != nil {
		return OAuthStateRecord{}, fmt.Errorf("consume oauth state: %w", err)
	}
	if !ok || record.Intent != wantIntent {
		return OAuthStateRecord{}, ErrOAuthStateInvalid
	}
	return record, nil
}

// --- PostgreSQL adapters ---------------------------------------------------

type PostgresOAuthAccountRepository struct{ queries *store.Queries }

func NewPostgresOAuthAccountRepository(queries *store.Queries) *PostgresOAuthAccountRepository {
	return &PostgresOAuthAccountRepository{queries: queries}
}

func (repository *PostgresOAuthAccountRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return repository.queries.WithTx(tx)
	}
	return repository.queries
}

func (repository *PostgresOAuthAccountRepository) Find(ctx context.Context, provider, subject string) (string, bool, error) {
	row, err := repository.q(ctx).FindOAuthAccount(ctx, store.FindOAuthAccountParams{Provider: provider, ProviderSubject: subject})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return uuid.UUID(row.UserID.Bytes).String(), true, nil
}

func (repository *PostgresOAuthAccountRepository) Create(ctx context.Context, userID, provider, subject, email string) error {
	parsed, err := parseUUID(userID)
	if err != nil {
		return ErrUserNotFound
	}
	var emailPtr *string
	if email != "" {
		emailPtr = &email
	}
	_, err = repository.q(ctx).CreateOAuthAccount(ctx, store.CreateOAuthAccountParams{
		UserID:          parsed,
		Provider:        provider,
		ProviderSubject: subject,
		Email:           emailPtr,
	})
	if isUniqueViolationError(err) {
		return ErrOAuthAccountTaken
	}
	return err
}

func (repository *PostgresOAuthAccountRepository) ListForUser(ctx context.Context, userID string) ([]OAuthAccount, error) {
	parsed, err := parseUUID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	rows, err := repository.q(ctx).ListOAuthAccountsForUser(ctx, parsed)
	if err != nil {
		return nil, err
	}
	accounts := make([]OAuthAccount, 0, len(rows))
	for _, row := range rows {
		account := OAuthAccount{Provider: row.Provider, Subject: row.ProviderSubject, CreatedAt: row.CreatedAt.Time}
		if row.Email != nil {
			account.Email = *row.Email
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

type PostgresOAuthStateRepository struct{ queries *store.Queries }

func NewPostgresOAuthStateRepository(queries *store.Queries) *PostgresOAuthStateRepository {
	return &PostgresOAuthStateRepository{queries: queries}
}

func (repository *PostgresOAuthStateRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return repository.queries.WithTx(tx)
	}
	return repository.queries
}

func (repository *PostgresOAuthStateRepository) Create(ctx context.Context, provider string, stateDigest []byte, nonce, codeVerifier, intent, linkingUserID string, expiresAt time.Time) error {
	var linkingUser pgtype.UUID
	if linkingUserID != "" {
		parsed, err := parseUUID(linkingUserID)
		if err != nil {
			return ErrUserNotFound
		}
		linkingUser = parsed
	}
	_, err := repository.q(ctx).CreateOAuthAuthorizationState(ctx, store.CreateOAuthAuthorizationStateParams{
		Provider:      provider,
		StateDigest:   stateDigest,
		Nonce:         nonce,
		CodeVerifier:  codeVerifier,
		Intent:        intent,
		LinkingUserID: linkingUser,
		ExpiresAt:     pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	return err
}

func (repository *PostgresOAuthStateRepository) Consume(ctx context.Context, provider string, stateDigest []byte) (OAuthStateRecord, bool, error) {
	row, err := repository.q(ctx).ConsumeOAuthAuthorizationState(ctx, store.ConsumeOAuthAuthorizationStateParams{StateDigest: stateDigest, Provider: provider})
	if errors.Is(err, pgx.ErrNoRows) {
		return OAuthStateRecord{}, false, nil
	}
	if err != nil {
		return OAuthStateRecord{}, false, err
	}
	record := OAuthStateRecord{
		Provider:     row.Provider,
		Nonce:        row.Nonce,
		CodeVerifier: row.CodeVerifier,
		Intent:       row.Intent,
	}
	if row.LinkingUserID.Valid {
		record.LinkingUserID = uuid.UUID(row.LinkingUserID.Bytes).String()
	}
	return record, true, nil
}
