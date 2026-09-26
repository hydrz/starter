package auth_test

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"

	"github.com/hydrz/starter/internal/auth"
)

// --- in-memory fakes for workstream F ports --------------------------------

type memoryEmailOTP struct {
	mu         sync.Mutex
	nextID     int
	challenges map[string]*emailOTPRecord
	byEmail    map[string][]string // email|purpose -> ids, most recent last
}

type emailOTPRecord struct {
	email        string
	purpose      string
	codeDigest   []byte
	ip           string
	attemptCount int
	maxAttempts  int
	expiresAt    time.Time
	consumed     bool
	createdAt    time.Time
}

func newMemoryEmailOTP() *memoryEmailOTP {
	return &memoryEmailOTP{challenges: map[string]*emailOTPRecord{}, byEmail: map[string][]string{}}
}

func (m *memoryEmailOTP) CreateChallenge(_ context.Context, email, purpose string, codeDigest []byte, ip string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := uuid.NewString()
	m.challenges[id] = &emailOTPRecord{email: email, purpose: purpose, codeDigest: codeDigest, ip: ip, maxAttempts: 5, expiresAt: expiresAt, createdAt: time.Now()}
	key := email + "|" + purpose
	m.byEmail[key] = append(m.byEmail[key], id)
	return nil
}

func (m *memoryEmailOTP) CountRecentForEmail(_ context.Context, email string, since time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, record := range m.challenges {
		if record.email == email && record.createdAt.After(since) {
			count++
		}
	}
	return count, nil
}

func (m *memoryEmailOTP) CountRecentForIP(_ context.Context, ip string, since time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, record := range m.challenges {
		if record.ip == ip && record.createdAt.After(since) {
			count++
		}
	}
	return count, nil
}

func (m *memoryEmailOTP) FindActive(_ context.Context, email, purpose string) (auth.EmailOTPChallenge, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := m.byEmail[email+"|"+purpose]
	for i := len(ids) - 1; i >= 0; i-- {
		record := m.challenges[ids[i]]
		if !record.consumed {
			return auth.EmailOTPChallenge{
				ID: ids[i], Email: record.email, CodeDigest: record.codeDigest,
				AttemptCount: record.attemptCount, MaxAttempts: record.maxAttempts, ExpiresAt: record.expiresAt,
			}, true, nil
		}
	}
	return auth.EmailOTPChallenge{}, false, nil
}

func (m *memoryEmailOTP) IncrementAttempt(_ context.Context, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.challenges[id]
	if !ok || record.consumed || record.attemptCount >= record.maxAttempts {
		return false, nil
	}
	record.attemptCount++
	return true, nil
}

func (m *memoryEmailOTP) Consume(_ context.Context, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.challenges[id]
	if !ok || record.consumed {
		return false, nil
	}
	record.consumed = true
	return true, nil
}

type memoryTOTPFactors struct {
	mu      sync.Mutex
	factors map[string]*auth.TOTPFactorRecord
}

func newMemoryTOTPFactors() *memoryTOTPFactors {
	return &memoryTOTPFactors{factors: map[string]*auth.TOTPFactorRecord{}}
}

func (m *memoryTOTPFactors) CreatePending(_ context.Context, userID string, ciphertext, nonce []byte) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.NewString()
	m.factors[id] = &auth.TOTPFactorRecord{ID: id, UserID: userID, SecretCiphertext: ciphertext, SecretNonce: nonce, Status: "pending"}
	return id, nil
}

func (m *memoryTOTPFactors) GetByID(_ context.Context, factorID string) (auth.TOTPFactorRecord, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.factors[factorID]
	if !ok {
		return auth.TOTPFactorRecord{}, false, nil
	}
	return *record, true, nil
}

func (m *memoryTOTPFactors) GetVerifiedForUser(_ context.Context, userID string) (auth.TOTPFactorRecord, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, record := range m.factors {
		if record.UserID == userID && record.Status == "verified" {
			return *record, true, nil
		}
	}
	return auth.TOTPFactorRecord{}, false, nil
}

func (m *memoryTOTPFactors) Activate(_ context.Context, factorID, userID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.factors[factorID]
	if !ok || record.UserID != userID || record.Status != "pending" {
		return false, nil
	}
	record.Status = "verified"
	return true, nil
}

func (m *memoryTOTPFactors) DeletePendingForUser(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, record := range m.factors {
		if record.UserID == userID && record.Status == "pending" {
			delete(m.factors, id)
		}
	}
	return nil
}

type memoryRecoveryCodes struct {
	mu    sync.Mutex
	codes map[string]bool // factorID|digest -> consumed
}

func newMemoryRecoveryCodes() *memoryRecoveryCodes {
	return &memoryRecoveryCodes{codes: map[string]bool{}}
}

func (m *memoryRecoveryCodes) CreateMany(_ context.Context, factorID string, digests [][]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, digest := range digests {
		m.codes[factorID+"|"+string(digest)] = false
	}
	return nil
}

func (m *memoryRecoveryCodes) Consume(_ context.Context, factorID string, digest []byte) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := factorID + "|" + string(digest)
	consumed, exists := m.codes[key]
	if !exists || consumed {
		return false, nil
	}
	m.codes[key] = true
	return true, nil
}

type memoryMFAChallenges struct {
	mu         sync.Mutex
	challenges map[string]*mfaChallengeRecord
}

type mfaChallengeRecord struct {
	userID, factorID string
	expiresAt        time.Time
	consumed         bool
}

func newMemoryMFAChallenges() *memoryMFAChallenges {
	return &memoryMFAChallenges{challenges: map[string]*mfaChallengeRecord{}}
}

func (m *memoryMFAChallenges) Create(_ context.Context, userID, factorID string, expiresAt time.Time) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.NewString()
	m.challenges[id] = &mfaChallengeRecord{userID: userID, factorID: factorID, expiresAt: expiresAt}
	return id, nil
}

func (m *memoryMFAChallenges) Consume(_ context.Context, challengeID string) (auth.MFAChallengeRecord, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.challenges[challengeID]
	if !ok || record.consumed {
		return auth.MFAChallengeRecord{}, false, nil
	}
	record.consumed = true
	return auth.MFAChallengeRecord{ID: challengeID, UserID: record.userID, FactorID: record.factorID}, true, nil
}

type memoryOAuthAccounts struct {
	mu     sync.Mutex
	byKey  map[string]string // provider|subject -> userID
	byUser map[string][]auth.OAuthAccount
}

func newMemoryOAuthAccounts() *memoryOAuthAccounts {
	return &memoryOAuthAccounts{byKey: map[string]string{}, byUser: map[string][]auth.OAuthAccount{}}
}

func (m *memoryOAuthAccounts) Find(_ context.Context, provider, subject string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	userID, ok := m.byKey[provider+"|"+subject]
	return userID, ok, nil
}

func (m *memoryOAuthAccounts) Create(_ context.Context, userID, provider, subject, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := provider + "|" + subject
	if _, exists := m.byKey[key]; exists {
		return auth.ErrOAuthAccountTaken
	}
	m.byKey[key] = userID
	m.byUser[userID] = append(m.byUser[userID], auth.OAuthAccount{Provider: provider, Subject: subject, Email: email, CreatedAt: time.Now()})
	return nil
}

func (m *memoryOAuthAccounts) ListForUser(_ context.Context, userID string) ([]auth.OAuthAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.byUser[userID], nil
}

type memoryOAuthStates struct {
	mu     sync.Mutex
	states map[string]*oauthStateRecord
}

type oauthStateRecord struct {
	provider                          string
	nonce, codeVerifier, intent, link string
	expiresAt                         time.Time
	consumed                          bool
}

func newMemoryOAuthStates() *memoryOAuthStates {
	return &memoryOAuthStates{states: map[string]*oauthStateRecord{}}
}

func (m *memoryOAuthStates) Create(_ context.Context, provider string, stateDigest []byte, nonce, codeVerifier, intent, linkingUserID string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[provider+"|"+string(stateDigest)] = &oauthStateRecord{
		provider: provider, nonce: nonce, codeVerifier: codeVerifier, intent: intent, link: linkingUserID, expiresAt: expiresAt,
	}
	return nil
}

func (m *memoryOAuthStates) Consume(_ context.Context, provider string, stateDigest []byte) (auth.OAuthStateRecord, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := provider + "|" + string(stateDigest)
	record, ok := m.states[key]
	if !ok || record.consumed {
		return auth.OAuthStateRecord{}, false, nil
	}
	record.consumed = true
	return auth.OAuthStateRecord{Provider: record.provider, Nonce: record.nonce, CodeVerifier: record.codeVerifier, Intent: record.intent, LinkingUserID: record.link}, true, nil
}

type fakeOAuthClient struct {
	info map[string]auth.OAuthUserInfo
	err  error
}

func (f *fakeOAuthClient) AuthCodeURL(state, codeChallenge string) string {
	return "https://provider.example/authorize?state=" + state + "&code_challenge=" + codeChallenge
}

func (f *fakeOAuthClient) Exchange(_ context.Context, code, _ string) (auth.OAuthUserInfo, error) {
	if f.err != nil {
		return auth.OAuthUserInfo{}, f.err
	}
	info, ok := f.info[code]
	if !ok {
		return auth.OAuthUserInfo{}, errors.New("unknown code")
	}
	return info, nil
}

// payloadOutbox records full event payloads (unlike service_test.go's
// memoryOutbox, which only records topics) so F's tests can read the code
// an OTP outbox event carries, the same way a real delivery worker would.
type payloadOutbox struct {
	mu     sync.Mutex
	events []payloadEvent
}

type payloadEvent struct {
	topic   string
	payload []byte
}

func (o *payloadOutbox) WriteEvent(_ context.Context, topic, _, _ string, payload []byte, _ string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, payloadEvent{topic: topic, payload: payload})
	return nil
}

func (o *payloadOutbox) latestCode(t *testing.T) string {
	t.Helper()
	o.mu.Lock()
	defer o.mu.Unlock()
	for i := len(o.events) - 1; i >= 0; i-- {
		if o.events[i].topic != "auth.email_otp_requested" {
			continue
		}
		var decoded struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal(o.events[i].payload, &decoded); err != nil {
			t.Fatalf("decode otp outbox payload: %v", err)
		}
		return decoded.Code
	}
	t.Fatal("no email_otp_requested outbox event was recorded")
	return ""
}

// --- harness ----------------------------------------------------------

type fHarness struct {
	service *auth.Service
	clock   *fakeClock
	users   *memoryUsers
	refresh *memoryRefreshTokens

	outbox        *payloadOutbox
	emailOTP      *memoryEmailOTP
	totpFactors   *memoryTOTPFactors
	recoveryCodes *memoryRecoveryCodes
	mfaChallenges *memoryMFAChallenges
	oauthAccounts *memoryOAuthAccounts
	oauthStates   *memoryOAuthStates
	oauthClients  map[string]auth.OAuthClient
}

func newFHarness(t *testing.T) *fHarness {
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
	totpCipher, err := auth.NewTOTPCipher(make([]byte, 32))
	if err != nil {
		t.Fatalf("NewTOTPCipher() error = %v", err)
	}

	fh := &fHarness{
		clock:   clock,
		users:   newMemoryUsers(),
		refresh: newMemoryRefreshTokens(clock),

		outbox:        &payloadOutbox{},
		emailOTP:      newMemoryEmailOTP(),
		totpFactors:   newMemoryTOTPFactors(),
		recoveryCodes: newMemoryRecoveryCodes(),
		mfaChallenges: newMemoryMFAChallenges(),
		oauthAccounts: newMemoryOAuthAccounts(),
		oauthStates:   newMemoryOAuthStates(),
		oauthClients:  map[string]auth.OAuthClient{},
	}

	service, err := auth.NewService(auth.Dependencies{
		Users:           fh.users,
		RefreshTokens:   fh.refresh,
		OneTimeTokens:   newMemoryOneTimeTokens(clock),
		APIKeys:         newMemoryAPIKeys(),
		Outbox:          fh.outbox,
		Transactor:      passthroughTransactor{},
		Issuer:          issuer,
		Verifier:        verifier,
		Digester:        digester,
		Clock:           clock,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,

		EmailOTP:      fh.emailOTP,
		TOTPFactors:   fh.totpFactors,
		RecoveryCodes: fh.recoveryCodes,
		MFAChallenges: fh.mfaChallenges,
		TOTPCipher:    totpCipher,
		TOTPIssuer:    "Starter Test",

		OAuthAccounts: fh.oauthAccounts,
		OAuthStates:   fh.oauthStates,
		OAuthClients:  fh.oauthClients,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	fh.service = service
	return fh
}

// --- Email OTP tests -----------------------------------------------------

func TestEmailOTPVerifySucceedsThenRejectsReuse(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()
	user, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "otp@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	_ = user

	if err := h.service.RequestEmailOTP(ctx, "otp@example.com", "203.0.113.5"); err != nil {
		t.Fatalf("RequestEmailOTP() error = %v", err)
	}
	code := h.outbox.latestCode(t)

	tokens, err := h.service.VerifyEmailOTP(ctx, "otp@example.com", code, auth.RefreshSessionMetadata{})
	if err != nil {
		t.Fatalf("VerifyEmailOTP() error = %v", err)
	}
	if tokens.AccessToken == "" {
		t.Fatal("VerifyEmailOTP() did not issue an access token")
	}

	if _, err := h.service.VerifyEmailOTP(ctx, "otp@example.com", code, auth.RefreshSessionMetadata{}); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("second VerifyEmailOTP() error = %v, want ErrInvalidCredentials (single-use)", err)
	}
}

func TestEmailOTPRequestIsIndistinguishableForUnknownEmail(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()

	if err := h.service.RequestEmailOTP(ctx, "nobody@example.com", "203.0.113.6"); err != nil {
		t.Fatalf("RequestEmailOTP() for unknown email error = %v, want nil (anti-enumeration)", err)
	}
	if len(h.outbox.events) != 0 {
		t.Fatalf("outbox events = %d, want 0 for an unknown email", len(h.outbox.events))
	}
}

func TestEmailOTPRequestRateLimitedByIP(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()

	// Use a distinct email per request so the per-email limit (which
	// silently no-ops rather than erroring, to avoid an enumeration
	// signal) never masks the IP limit under test.
	var lastErr error
	for i := range 15 {
		email := fmt.Sprintf("rate-%d@example.com", i)
		if _, err := h.service.SignUp(ctx, auth.SignUpInput{Email: email, Password: "correct-horse"}); err != nil {
			t.Fatalf("SignUp() error = %v", err)
		}
		lastErr = h.service.RequestEmailOTP(ctx, email, "203.0.113.7")
	}
	if !errors.Is(lastErr, auth.ErrRateLimited) {
		t.Fatalf("RequestEmailOTP() after repeated requests from one IP error = %v, want ErrRateLimited", lastErr)
	}
}

// --- TOTP / MFA tests ------------------------------------------------

func TestTOTPCipherRoundTrips(t *testing.T) {
	t.Parallel()
	cipher, err := auth.NewTOTPCipher(make([]byte, 32))
	if err != nil {
		t.Fatalf("NewTOTPCipher() error = %v", err)
	}
	ciphertext, nonce, err := cipher.Encrypt([]byte("JBSWY3DPEHPK3PXP"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if string(ciphertext) == "JBSWY3DPEHPK3PXP" {
		t.Fatal("Encrypt() returned the plaintext unchanged")
	}
	plaintext, err := cipher.Decrypt(ciphertext, nonce)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if string(plaintext) != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("Decrypt() = %q, want original secret", plaintext)
	}
}

func TestTOTPEnrollmentAndMFASignInFlow(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()
	user, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "mfa@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}

	provisioning, err := h.service.BeginTOTPEnrollment(ctx, user.ID)
	if err != nil {
		t.Fatalf("BeginTOTPEnrollment() error = %v", err)
	}
	code, err := totpCodeFor(t, h, provisioning.FactorID)
	if err != nil {
		t.Fatalf("compute totp code: %v", err)
	}
	recoveryCodes, err := h.service.ConfirmTOTPEnrollment(ctx, user.ID, provisioning.FactorID, code)
	if err != nil {
		t.Fatalf("ConfirmTOTPEnrollment() error = %v", err)
	}
	if len(recoveryCodes) == 0 {
		t.Fatal("ConfirmTOTPEnrollment() returned no recovery codes")
	}

	// A subsequent primary sign-in must now pause for MFA instead of
	// issuing tokens directly (forward-only enforcement).
	tokens, err := h.service.PasswordSignIn(ctx, auth.PasswordSignInInput{Email: "mfa@example.com", Password: "correct-horse"}, auth.RefreshSessionMetadata{})
	if !errors.Is(err, auth.ErrMFARequired) {
		t.Fatalf("PasswordSignIn() error = %v, want ErrMFARequired", err)
	}
	if tokens.MFAChallengeID == "" {
		t.Fatal("PasswordSignIn() did not return an MFA challenge id")
	}
	if tokens.AccessToken != "" {
		t.Fatal("PasswordSignIn() returned an access token despite MFA being required")
	}

	secondCode, err := totpCodeFor(t, h, provisioning.FactorID)
	if err != nil {
		t.Fatalf("compute totp code: %v", err)
	}
	completed, err := h.service.VerifyMFAChallenge(ctx, tokens.MFAChallengeID, secondCode, auth.RefreshSessionMetadata{})
	if err != nil {
		t.Fatalf("VerifyMFAChallenge() error = %v", err)
	}
	if completed.AccessToken == "" {
		t.Fatal("VerifyMFAChallenge() did not issue an access token")
	}

	// The challenge is single-use: replaying it must fail even with a
	// freshly valid code.
	if _, err := h.service.VerifyMFAChallenge(ctx, tokens.MFAChallengeID, secondCode, auth.RefreshSessionMetadata{}); !errors.Is(err, auth.ErrMFAChallengeNotFound) {
		t.Fatalf("second VerifyMFAChallenge() error = %v, want ErrMFAChallengeNotFound (single-use)", err)
	}
}

func TestTOTPRecoveryCodeIsSingleUse(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()
	user, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "recovery@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	provisioning, err := h.service.BeginTOTPEnrollment(ctx, user.ID)
	if err != nil {
		t.Fatalf("BeginTOTPEnrollment() error = %v", err)
	}
	code, err := totpCodeFor(t, h, provisioning.FactorID)
	if err != nil {
		t.Fatalf("compute totp code: %v", err)
	}
	recoveryCodes, err := h.service.ConfirmTOTPEnrollment(ctx, user.ID, provisioning.FactorID, code)
	if err != nil {
		t.Fatalf("ConfirmTOTPEnrollment() error = %v", err)
	}

	tokens, err := h.service.PasswordSignIn(ctx, auth.PasswordSignInInput{Email: "recovery@example.com", Password: "correct-horse"}, auth.RefreshSessionMetadata{})
	if !errors.Is(err, auth.ErrMFARequired) {
		t.Fatalf("PasswordSignIn() error = %v, want ErrMFARequired", err)
	}

	recoveryCode := recoveryCodes[0]
	if _, err := h.service.VerifyMFAChallenge(ctx, tokens.MFAChallengeID, recoveryCode, auth.RefreshSessionMetadata{}); err != nil {
		t.Fatalf("VerifyMFAChallenge() with recovery code error = %v", err)
	}

	// Consuming the same recovery code again, against a fresh challenge,
	// must fail: recovery codes are single-use.
	tokens2, err := h.service.PasswordSignIn(ctx, auth.PasswordSignInInput{Email: "recovery@example.com", Password: "correct-horse"}, auth.RefreshSessionMetadata{})
	if !errors.Is(err, auth.ErrMFARequired) {
		t.Fatalf("second PasswordSignIn() error = %v, want ErrMFARequired", err)
	}
	if _, err := h.service.VerifyMFAChallenge(ctx, tokens2.MFAChallengeID, recoveryCode, auth.RefreshSessionMetadata{}); !errors.Is(err, auth.ErrTOTPInvalidCode) {
		t.Fatalf("reused recovery code error = %v, want ErrTOTPInvalidCode", err)
	}
}

// totpCodeFor decrypts the pending/verified factor's secret directly from
// the fake repository (as the service itself would) to compute a valid
// code without depending on any provisioning URL parsing.
func totpCodeFor(t *testing.T, h *fHarness, factorID string) (string, error) {
	t.Helper()
	record, ok, err := h.totpFactors.GetByID(context.Background(), factorID)
	if err != nil || !ok {
		t.Fatalf("GetByID(%q) ok=%v err=%v", factorID, ok, err)
	}
	cipher, err := auth.NewTOTPCipher(make([]byte, 32))
	if err != nil {
		t.Fatalf("NewTOTPCipher() error = %v", err)
	}
	secret, err := cipher.Decrypt(record.SecretCiphertext, record.SecretNonce)
	if err != nil {
		return "", err
	}
	return totp.GenerateCode(string(secret), h.clock.Now())
}

// --- OAuth tests -------------------------------------------------------

func TestOAuthDifferentSubjectsSharingEmailNeverCollide(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()
	client := &fakeOAuthClient{info: map[string]auth.OAuthUserInfo{
		"code-a": {Subject: "subject-a", Email: "shared@example.com"},
		"code-b": {Subject: "subject-b", Email: "shared@example.com"},
	}}
	h.oauthClients["google"] = client

	urlA, err := h.service.BeginOAuthSignIn(ctx, "google")
	if err != nil {
		t.Fatalf("BeginOAuthSignIn() error = %v", err)
	}
	stateA := extractState(t, urlA)
	tokensA, err := h.service.CompleteOAuthSignIn(ctx, "google", stateA, "code-a", auth.RefreshSessionMetadata{})
	if err != nil {
		t.Fatalf("CompleteOAuthSignIn() (a) error = %v", err)
	}

	urlB, err := h.service.BeginOAuthSignIn(ctx, "google")
	if err != nil {
		t.Fatalf("BeginOAuthSignIn() error = %v", err)
	}
	stateB := extractState(t, urlB)
	tokensB, err := h.service.CompleteOAuthSignIn(ctx, "google", stateB, "code-b", auth.RefreshSessionMetadata{})
	if err != nil {
		t.Fatalf("CompleteOAuthSignIn() (b) error = %v", err)
	}

	if tokensA.UserID == tokensB.UserID {
		t.Fatalf("two different (provider, subject) identities sharing an email resolved to the same user %q", tokensA.UserID)
	}
}

func TestOAuthStateIsSingleUse(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()
	h.oauthClients["google"] = &fakeOAuthClient{info: map[string]auth.OAuthUserInfo{"code": {Subject: "subject-1", Email: "once@example.com"}}}

	authURL, err := h.service.BeginOAuthSignIn(ctx, "google")
	if err != nil {
		t.Fatalf("BeginOAuthSignIn() error = %v", err)
	}
	state := extractState(t, authURL)

	if _, err := h.service.CompleteOAuthSignIn(ctx, "google", state, "code", auth.RefreshSessionMetadata{}); err != nil {
		t.Fatalf("first CompleteOAuthSignIn() error = %v", err)
	}
	if _, err := h.service.CompleteOAuthSignIn(ctx, "google", state, "code", auth.RefreshSessionMetadata{}); !errors.Is(err, auth.ErrOAuthStateInvalid) {
		t.Fatalf("second CompleteOAuthSignIn() (state reuse) error = %v, want ErrOAuthStateInvalid", err)
	}
}

func TestOAuthLinkRequiresMatchingAuthenticatedSession(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()
	victim, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "victim@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	attacker, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "attacker@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	h.oauthClients["github"] = &fakeOAuthClient{info: map[string]auth.OAuthUserInfo{"link-code": {Subject: "gh-subject", Email: "victim@example.com"}}}

	// The victim begins a link flow (binding the state to their user ID)...
	authURL, err := h.service.BeginOAuthLink(ctx, "github", victim.ID, h.clock.Now())
	if err != nil {
		t.Fatalf("BeginOAuthLink() error = %v", err)
	}
	state := extractState(t, authURL)

	// ...but the callback is completed while authenticated as a different
	// account. It must never attach the provider identity to the attacker.
	if err := h.service.LinkOAuthAccount(ctx, "github", state, "link-code", attacker.ID); !errors.Is(err, auth.ErrOAuthStateInvalid) {
		t.Fatalf("LinkOAuthAccount() with mismatched session error = %v, want ErrOAuthStateInvalid", err)
	}

	accounts, err := h.service.ListOAuthAccounts(ctx, attacker.ID)
	if err != nil {
		t.Fatalf("ListOAuthAccounts() error = %v", err)
	}
	if len(accounts) != 0 {
		t.Fatalf("attacker account has %d linked oauth accounts, want 0", len(accounts))
	}
}

func TestOAuthLinkRequiresRecentReauthentication(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()
	user, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "stale@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	staleIssuedAt := h.clock.Now().Add(-time.Hour)
	if _, err := h.service.BeginOAuthLink(ctx, "github", user.ID, staleIssuedAt); !errors.Is(err, auth.ErrReauthenticationRequired) {
		t.Fatalf("BeginOAuthLink() with stale access token error = %v, want ErrReauthenticationRequired", err)
	}
}

func extractState(t *testing.T, authorizationURL string) string {
	t.Helper()
	parsed, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", authorizationURL, err)
	}
	state := parsed.Query().Get("state")
	if state == "" {
		t.Fatalf("authorization url %q has no state parameter", authorizationURL)
	}
	return state
}

// --- WebAuthn tests ------------------------------------------------------

func TestWebAuthnRegistrationRejectedForUnauthenticatedCaller(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()

	// No userID at all: the service layer itself must reject this, not
	// merely rely on HTTP middleware to have blocked the request earlier.
	if _, err := h.service.BeginWebAuthnRegistration(ctx, ""); !errors.Is(err, auth.ErrWebAuthnUnauthenticated) {
		t.Fatalf("BeginWebAuthnRegistration(\"\") error = %v, want ErrWebAuthnUnauthenticated", err)
	}
	if err := h.service.FinishWebAuthnRegistration(ctx, "", []byte(`{}`)); !errors.Is(err, auth.ErrWebAuthnUnauthenticated) {
		t.Fatalf("FinishWebAuthnRegistration(\"\") error = %v, want ErrWebAuthnUnauthenticated", err)
	}
}
