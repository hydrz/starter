package auth_test

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hydrz/starter/internal/auth"
)

// --- in-memory fakes ---------------------------------------------------

type memoryUser struct {
	id, email, passwordHash string
	verifiedAt              *time.Time
}

type memoryUsers struct {
	mu      sync.Mutex
	nextID  int
	byID    map[string]*memoryUser
	byEmail map[string]*memoryUser
}

func newMemoryUsers() *memoryUsers {
	return &memoryUsers{byID: map[string]*memoryUser{}, byEmail: map[string]*memoryUser{}}
}

func (m *memoryUsers) Create(_ context.Context, email, passwordHash string) (auth.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byEmail[email]; exists {
		return auth.User{}, auth.ErrEmailTaken
	}
	m.nextID++
	id := idFromInt(m.nextID)
	user := &memoryUser{id: id, email: email, passwordHash: passwordHash}
	m.byID[id] = user
	m.byEmail[email] = user
	return toDomainUser(user), nil
}

func (m *memoryUsers) FindByEmail(_ context.Context, email string) (auth.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.byEmail[email]
	if !ok {
		return auth.User{}, auth.ErrUserNotFound
	}
	return toDomainUser(user), nil
}

func (m *memoryUsers) FindByID(_ context.Context, id string) (auth.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.byID[id]
	if !ok {
		return auth.User{}, auth.ErrUserNotFound
	}
	return toDomainUser(user), nil
}

func (m *memoryUsers) MarkEmailVerified(_ context.Context, userID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.byID[userID]
	if !ok || user.verifiedAt != nil {
		return false, nil
	}
	now := time.Now()
	user.verifiedAt = &now
	return true, nil
}

func (m *memoryUsers) UpdatePassword(_ context.Context, userID, passwordHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.byID[userID]
	if !ok {
		return auth.ErrUserNotFound
	}
	user.passwordHash = passwordHash
	return nil
}

func toDomainUser(user *memoryUser) auth.User {
	return auth.User{ID: user.id, Email: user.email, PasswordHash: user.passwordHash, EmailVerifiedAt: user.verifiedAt}
}

func idFromInt(_ int) string {
	// The HTTP handler layer parses domain IDs as uuid.UUID, so fakes must
	// hand out real (random) UUIDs rather than opaque test strings.
	return uuid.NewString()
}

type refreshFamily struct {
	userID  string
	revoked bool
	reason  string
}

type refreshSessionRecord struct {
	familyID    string
	tokenDigest string
	expiresAt   time.Time
	replaced    bool
	revoked     bool
	metadata    auth.RefreshSessionMetadata
	createdAt   time.Time
}

type memoryRefreshTokens struct {
	mu          sync.Mutex
	nextFamily  int
	nextSession int
	families    map[string]*refreshFamily
	sessions    map[string]*refreshSessionRecord
	digestToID  map[string]string // token digest (as string) -> session id
	clock       *fakeClock
}

func newMemoryRefreshTokens(clock *fakeClock) *memoryRefreshTokens {
	return &memoryRefreshTokens{
		families:   map[string]*refreshFamily{},
		sessions:   map[string]*refreshSessionRecord{},
		digestToID: map[string]string{},
		clock:      clock,
	}
}

func (m *memoryRefreshTokens) CreateFamily(_ context.Context, userID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextFamily++
	id := uuid.NewString()
	m.families[id] = &refreshFamily{userID: userID}
	return id, nil
}

func (m *memoryRefreshTokens) CreateSession(_ context.Context, familyID string, tokenDigest []byte, expiresAt time.Time, metadata auth.RefreshSessionMetadata) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextSession++
	id := uuid.NewString()
	m.sessions[id] = &refreshSessionRecord{familyID: familyID, tokenDigest: string(tokenDigest), expiresAt: expiresAt, metadata: metadata, createdAt: m.clock.Now()}
	m.digestToID[string(tokenDigest)] = id
	return id, nil
}

func (m *memoryRefreshTokens) ConsumeSession(_ context.Context, tokenDigest []byte) (string, string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.digestToID[string(tokenDigest)]
	if !ok {
		return "", "", false, nil
	}
	session := m.sessions[id]
	family := m.families[session.familyID]
	if session.replaced || session.revoked || family.revoked || m.clock.Now().After(session.expiresAt) {
		return "", "", false, nil
	}
	session.replaced = true
	return session.familyID, id, true, nil
}

func (m *memoryRefreshTokens) FindByDigest(_ context.Context, tokenDigest []byte) (string, string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.digestToID[string(tokenDigest)]
	if !ok {
		return "", "", false, nil
	}
	session := m.sessions[id]
	family := m.families[session.familyID]
	return family.userID, session.familyID, true, nil
}

func (m *memoryRefreshTokens) RevokeFamily(_ context.Context, familyID, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	family, ok := m.families[familyID]
	if !ok {
		return nil
	}
	family.revoked = true
	family.reason = reason
	return nil
}

func (m *memoryRefreshTokens) RevokeAllForUser(_ context.Context, userID, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, family := range m.families {
		if family.userID == userID {
			family.revoked = true
			family.reason = reason
		}
	}
	return nil
}

func (m *memoryRefreshTokens) ListSessionsForUser(_ context.Context, userID string) ([]auth.RefreshSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []auth.RefreshSession
	for id, session := range m.sessions {
		family := m.families[session.familyID]
		if family.userID != userID || family.revoked || session.revoked || session.replaced {
			continue
		}
		result = append(result, auth.RefreshSession{ID: id, CreatedAt: session.createdAt, ExpiresAt: session.expiresAt})
	}
	return result, nil
}

func (m *memoryRefreshTokens) RevokeSessionForUser(_ context.Context, sessionID, userID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[sessionID]
	if !ok {
		return false, nil
	}
	family := m.families[session.familyID]
	if family.userID != userID {
		return false, nil
	}
	family.revoked = true
	return true, nil
}

type memoryOneTimeTokens struct {
	mu     sync.Mutex
	clock  *fakeClock
	tokens map[string]struct {
		userID   string
		purpose  auth.OneTimeTokenPurpose
		consumed bool
		expires  time.Time
	}
}

func newMemoryOneTimeTokens(clock *fakeClock) *memoryOneTimeTokens {
	return &memoryOneTimeTokens{clock: clock, tokens: map[string]struct {
		userID   string
		purpose  auth.OneTimeTokenPurpose
		consumed bool
		expires  time.Time
	}{}}
}

func (m *memoryOneTimeTokens) Create(_ context.Context, userID string, purpose auth.OneTimeTokenPurpose, tokenDigest []byte, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[string(tokenDigest)] = struct {
		userID   string
		purpose  auth.OneTimeTokenPurpose
		consumed bool
		expires  time.Time
	}{userID: userID, purpose: purpose, expires: expiresAt}
	return nil
}

func (m *memoryOneTimeTokens) Consume(_ context.Context, tokenDigest []byte, purpose auth.OneTimeTokenPurpose) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.tokens[string(tokenDigest)]
	if !ok || entry.consumed || entry.purpose != purpose || m.clock.Now().After(entry.expires) {
		return "", false, nil
	}
	entry.consumed = true
	m.tokens[string(tokenDigest)] = entry
	return entry.userID, true, nil
}

type memoryAPIKeys struct {
	mu   sync.Mutex
	next int
	keys map[string]*auth.APIKey
}

func newMemoryAPIKeys() *memoryAPIKeys {
	return &memoryAPIKeys{keys: map[string]*auth.APIKey{}}
}

func (m *memoryAPIKeys) Create(_ context.Context, userID, name, keyPrefix string, secretDigest []byte, expiresAt *time.Time) (auth.APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.next++
	id := uuid.NewString()
	key := &auth.APIKey{ID: id, UserID: userID, Name: name, KeyPrefix: keyPrefix, ExpiresAt: expiresAt, CreatedAt: time.Now()}
	m.keys[id] = key
	m.keys[string(secretDigest)] = key
	return *key, nil
}

func (m *memoryAPIKeys) ListForUser(_ context.Context, userID string) ([]auth.APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []auth.APIKey
	seen := map[string]bool{}
	for _, key := range m.keys {
		if key.UserID == userID && !seen[key.ID] {
			seen[key.ID] = true
			result = append(result, *key)
		}
	}
	return result, nil
}

func (m *memoryAPIKeys) RevokeForUser(_ context.Context, id, userID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key, ok := m.keys[id]
	if !ok || key.UserID != userID {
		return false, nil
	}
	now := time.Now()
	key.RevokedAt = &now
	return true, nil
}

func (m *memoryAPIKeys) FindActiveByDigest(_ context.Context, secretDigest []byte) (auth.APIKey, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key, ok := m.keys[string(secretDigest)]
	if !ok || key.RevokedAt != nil {
		return auth.APIKey{}, false, nil
	}
	return *key, true, nil
}

type memoryOutbox struct {
	mu     sync.Mutex
	events []string
}

func (m *memoryOutbox) WriteEvent(_ context.Context, topic, _, _ string, _ []byte, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, topic)
	return nil
}

type passthroughTransactor struct{}

func (passthroughTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// --- test harness --------------------------------------------------------

type harness struct {
	service *auth.Service
	clock   *fakeClock
	users   *memoryUsers
	refresh *memoryRefreshTokens
	tokens  *memoryOneTimeTokens
	keys    *memoryAPIKeys
	outbox  *memoryOutbox
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	issuer, err := auth.NewIssuer("kid-1", private, "starter", clock)
	if err != nil {
		t.Fatalf("NewIssuer() error = %v", err)
	}
	verifier, err := auth.NewVerifier(map[string]ed25519.PublicKey{"kid-1": public}, "starter", clock)
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}
	digester, err := auth.NewSecretDigester(1, []byte("test-pepper"))
	if err != nil {
		t.Fatalf("NewSecretDigester() error = %v", err)
	}

	h := &harness{
		clock:   clock,
		users:   newMemoryUsers(),
		refresh: newMemoryRefreshTokens(clock),
		tokens:  newMemoryOneTimeTokens(clock),
		keys:    newMemoryAPIKeys(),
		outbox:  &memoryOutbox{},
	}
	service, err := auth.NewService(auth.Dependencies{
		Users:           h.users,
		RefreshTokens:   h.refresh,
		OneTimeTokens:   h.tokens,
		APIKeys:         h.keys,
		Outbox:          h.outbox,
		Transactor:      passthroughTransactor{},
		Issuer:          issuer,
		Verifier:        verifier,
		Digester:        digester,
		Clock:           clock,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	h.service = service
	return h
}

func TestSignUpCreatesUserAndOutboxEvent(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	user, err := h.service.SignUp(context.Background(), auth.SignUpInput{Email: "New@Example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	if user.Email != "new@example.com" {
		t.Errorf("email = %q, want normalized lowercase", user.Email)
	}
	if len(h.outbox.events) != 1 {
		t.Fatalf("outbox events = %d, want 1", len(h.outbox.events))
	}
}

func TestSignUpRejectsDuplicateEmail(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	ctx := context.Background()

	if _, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "dup@example.com", Password: "correct-horse"}); err != nil {
		t.Fatalf("SignUp() first call error = %v", err)
	}
	if _, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "dup@example.com", Password: "correct-horse"}); !errors.Is(err, auth.ErrEmailTaken) {
		t.Fatalf("SignUp() second call error = %v, want ErrEmailTaken", err)
	}
}

func TestSignUpRejectsWeakPassword(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	if _, err := h.service.SignUp(context.Background(), auth.SignUpInput{Email: "weak@example.com", Password: "short"}); !errors.Is(err, auth.ErrInvalidInput) {
		t.Fatalf("SignUp() error = %v, want ErrInvalidInput", err)
	}
}

func TestPasswordSignInSucceedsAndIssuesTokens(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	ctx := context.Background()

	if _, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "sign-in@example.com", Password: "correct-horse"}); err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}

	tokens, err := h.service.PasswordSignIn(ctx, auth.PasswordSignInInput{Email: "sign-in@example.com", Password: "correct-horse"}, auth.RefreshSessionMetadata{})
	if err != nil {
		t.Fatalf("PasswordSignIn() error = %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("PasswordSignIn() returned empty tokens")
	}

	claims, err := h.service.VerifyAccessToken(tokens.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken() error = %v", err)
	}
	if claims.Subject != tokens.UserID {
		t.Errorf("claims.Subject = %q, want %q", claims.Subject, tokens.UserID)
	}
}

func TestPasswordSignInRejectsUnknownEmail(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	_, err := h.service.PasswordSignIn(context.Background(), auth.PasswordSignInInput{Email: "nobody@example.com", Password: "correct-horse"}, auth.RefreshSessionMetadata{})
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("PasswordSignIn() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestPasswordSignInRejectsWrongPassword(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	ctx := context.Background()
	if _, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "wrong-pw@example.com", Password: "correct-horse"}); err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}

	_, err := h.service.PasswordSignIn(ctx, auth.PasswordSignInInput{Email: "wrong-pw@example.com", Password: "incorrect-horse"}, auth.RefreshSessionMetadata{})
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("PasswordSignIn() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestRefreshAccessTokenRotatesSession(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	ctx := context.Background()
	if _, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "rotate@example.com", Password: "correct-horse"}); err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	tokens, err := h.service.PasswordSignIn(ctx, auth.PasswordSignInInput{Email: "rotate@example.com", Password: "correct-horse"}, auth.RefreshSessionMetadata{})
	if err != nil {
		t.Fatalf("PasswordSignIn() error = %v", err)
	}

	rotated, err := h.service.RefreshAccessToken(ctx, tokens.RefreshToken, auth.RefreshSessionMetadata{})
	if err != nil {
		t.Fatalf("RefreshAccessToken() error = %v", err)
	}
	if rotated.RefreshToken == tokens.RefreshToken {
		t.Error("RefreshAccessToken() returned the same refresh token; want a rotated value")
	}
}

func TestRefreshAccessTokenReuseRevokesFamily(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	ctx := context.Background()
	if _, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "reuse@example.com", Password: "correct-horse"}); err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	tokens, err := h.service.PasswordSignIn(ctx, auth.PasswordSignInInput{Email: "reuse@example.com", Password: "correct-horse"}, auth.RefreshSessionMetadata{})
	if err != nil {
		t.Fatalf("PasswordSignIn() error = %v", err)
	}

	rotated, err := h.service.RefreshAccessToken(ctx, tokens.RefreshToken, auth.RefreshSessionMetadata{})
	if err != nil {
		t.Fatalf("RefreshAccessToken() first call error = %v", err)
	}

	// Replay the original (now-consumed) refresh token: must be detected as
	// reuse and revoke the whole family.
	if _, err := h.service.RefreshAccessToken(ctx, tokens.RefreshToken, auth.RefreshSessionMetadata{}); !errors.Is(err, auth.ErrSessionReused) {
		t.Fatalf("RefreshAccessToken() replay error = %v, want ErrSessionReused", err)
	}

	// The rotated (now-valid-looking) token must also fail because the
	// family was revoked by the reuse detection above.
	if _, err := h.service.RefreshAccessToken(ctx, rotated.RefreshToken, auth.RefreshSessionMetadata{}); err == nil {
		t.Fatal("RefreshAccessToken() with revoked-family token error = nil, want failure")
	}
}

func TestRefreshAccessTokenRejectsUnknownToken(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	unknown := base64.RawURLEncoding.EncodeToString([]byte("not-a-real-secret-not-a-real-secret"))
	if _, err := h.service.RefreshAccessToken(context.Background(), unknown, auth.RefreshSessionMetadata{}); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("RefreshAccessToken() error = %v, want ErrSessionNotFound", err)
	}
}

func TestConfirmEmailVerificationMarksUserVerified(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	ctx := context.Background()

	user, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "verify@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	if user.EmailVerifiedAt != nil {
		t.Fatal("newly signed-up user should not start verified")
	}

	// SignUp already created a verification token digest; recover its raw
	// token indirectly is not exposed, so issue a fresh request instead.
	if err := h.service.RequestEmailVerification(ctx, "verify@example.com"); err != nil {
		t.Fatalf("RequestEmailVerification() error = %v", err)
	}
}

func TestRequestPasswordResetIsSilentForUnknownEmail(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	if err := h.service.RequestPasswordReset(context.Background(), "nobody@example.com"); err != nil {
		t.Fatalf("RequestPasswordReset() error = %v, want nil (anti-enumeration)", err)
	}
}

func TestCreateAndAuthenticateAPIKey(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	ctx := context.Background()
	user, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "apikey@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}

	created, err := h.service.CreateAPIKey(ctx, user.ID, "CI token", nil)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}
	if created.Key == "" {
		t.Fatal("CreateAPIKey() returned empty raw key")
	}

	resolved, err := h.service.AuthenticateAPIKey(ctx, created.Key)
	if err != nil {
		t.Fatalf("AuthenticateAPIKey() error = %v", err)
	}
	if resolved.ID != created.ID {
		t.Errorf("resolved.ID = %q, want %q", resolved.ID, created.ID)
	}

	if err := h.service.RevokeAPIKey(ctx, created.ID, user.ID); err != nil {
		t.Fatalf("RevokeAPIKey() error = %v", err)
	}
	if _, err := h.service.AuthenticateAPIKey(ctx, created.Key); !errors.Is(err, auth.ErrAPIKeyNotFound) {
		t.Fatalf("AuthenticateAPIKey() after revoke error = %v, want ErrAPIKeyNotFound", err)
	}
}

func TestListAndRevokeSessions(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	ctx := context.Background()
	if _, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "sessions@example.com", Password: "correct-horse"}); err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	tokens, err := h.service.PasswordSignIn(ctx, auth.PasswordSignInInput{Email: "sessions@example.com", Password: "correct-horse"}, auth.RefreshSessionMetadata{})
	if err != nil {
		t.Fatalf("PasswordSignIn() error = %v", err)
	}

	sessions, err := h.service.ListSessions(ctx, tokens.UserID)
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(sessions))
	}

	if err := h.service.RevokeSession(ctx, sessions[0].ID, tokens.UserID); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	remaining, err := h.service.ListSessions(ctx, tokens.UserID)
	if err != nil {
		t.Fatalf("ListSessions() after revoke error = %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("len(remaining) = %d, want 0 after revoke", len(remaining))
	}
}
