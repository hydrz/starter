package auth_test

// This file closes the gap left by f_test.go's WebAuthn coverage, which
// only exercised the unauthenticated-caller rejection: it did not exercise
// a real registration+authentication ceremony end to end, so
// origin/RPID/challenge/credential-ownership/sign-counter verification —
// github.com/go-webauthn/webauthn's actual security value — was untested
// in this repository (only covered indirectly by that library's own test
// suite). It builds synthetic "packed" self-attestation (ES256/ECDSA P-256)
// ceremony data the same way that library's own
// webauthn/es256k_e2e_test.go builds ES256K ceremony data for its unit
// tests: by hand-assembling authenticatorData/clientDataJSON/CBOR
// attestationObject and signing with a locally generated key, rather than
// a real browser/authenticator, then driving it through
// internal/auth.Service's actual Begin/Finish methods and its real
// in-memory WebAuthnCredentialRepository/WebAuthnChallengeRepository fakes
// (wired into fHarness in f_test.go) exactly like every other test in this
// package.

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"

	"github.com/hydrz/starter/internal/auth"
)

const (
	testWebAuthnRPID   = "example.com"
	testWebAuthnOrigin = "https://example.com"
)

// --- in-memory WebAuthnCredentialRepository/WebAuthnChallengeRepository ---

type webauthnCredentialRecord struct {
	id           string
	userID       string
	credentialID []byte
	publicKey    []byte
	signCount    uint32
	userHandle   []byte
}

type memoryWebAuthnCredentials struct {
	mu      sync.Mutex
	nextID  int
	records map[string]*webauthnCredentialRecord // keyed by id
}

func newMemoryWebAuthnCredentials() *memoryWebAuthnCredentials {
	return &memoryWebAuthnCredentials{records: map[string]*webauthnCredentialRecord{}}
}

func (m *memoryWebAuthnCredentials) Create(_ context.Context, userID string, credentialID, publicKey []byte, signCount uint32, userHandle []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := fmt.Sprintf("cred-%d", m.nextID)
	m.records[id] = &webauthnCredentialRecord{
		id: id, userID: userID,
		credentialID: append([]byte(nil), credentialID...),
		publicKey:    append([]byte(nil), publicKey...),
		signCount:    signCount,
		userHandle:   append([]byte(nil), userHandle...),
	}
	return nil
}

func (m *memoryWebAuthnCredentials) ListForUser(_ context.Context, userID string) ([]auth.WebAuthnCredentialRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []auth.WebAuthnCredentialRecord
	for _, r := range m.records {
		if r.userID == userID {
			out = append(out, toRecord(r))
		}
	}
	return out, nil
}

func (m *memoryWebAuthnCredentials) FindByCredentialID(_ context.Context, credentialID []byte) (auth.WebAuthnCredentialRecord, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.records {
		if string(r.credentialID) == string(credentialID) {
			return toRecord(r), true, nil
		}
	}
	return auth.WebAuthnCredentialRecord{}, false, nil
}

func (m *memoryWebAuthnCredentials) UpdateSignCount(_ context.Context, id string, newCount uint32) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.records[id]
	if !ok || newCount <= r.signCount {
		return false, nil
	}
	r.signCount = newCount
	return true, nil
}

func toRecord(r *webauthnCredentialRecord) auth.WebAuthnCredentialRecord {
	return auth.WebAuthnCredentialRecord{
		ID: r.id, UserID: r.userID, CredentialID: r.credentialID,
		PublicKey: r.publicKey, SignCount: r.signCount, UserHandle: r.userHandle,
	}
}

type webauthnChallengeRecord struct {
	ceremony    string
	userID      string
	challenge   []byte
	sessionData []byte
	expiresAt   time.Time
	consumed    bool
}

type memoryWebAuthnChallenges struct {
	mu      sync.Mutex
	records []*webauthnChallengeRecord
}

func newMemoryWebAuthnChallenges() *memoryWebAuthnChallenges {
	return &memoryWebAuthnChallenges{}
}

func (m *memoryWebAuthnChallenges) Create(_ context.Context, ceremony, userID string, challenge, sessionData []byte, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, &webauthnChallengeRecord{
		ceremony: ceremony, userID: userID,
		challenge:   append([]byte(nil), challenge...),
		sessionData: append([]byte(nil), sessionData...),
		expiresAt:   expiresAt,
	})
	return nil
}

func (m *memoryWebAuthnChallenges) Consume(_ context.Context, ceremony string, challenge []byte) ([]byte, string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.records {
		if r.consumed || r.ceremony != ceremony || string(r.challenge) != string(challenge) {
			continue
		}
		r.consumed = true
		return r.sessionData, r.userID, true, nil
	}
	return nil, "", false, nil
}

// --- synthetic ceremony construction ---------------------------------------
//
// These helpers mirror github.com/go-webauthn/webauthn's own
// webauthn/es256k_e2e_test.go pattern (testES256K*) for building a
// synthetic authenticator response for a "packed" self ("basic_surrogate")
// attestation, using ES256 (ECDSA P-256) rather than ES256K, since ES256 is
// this library's default accepted algorithm (CredentialParametersDefault,
// registration_credential_parameters.go) and needs only the standard
// library's crypto/ecdsa rather than a third-party curve.

func generateWebAuthnKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey() error = %v", err)
	}
	return key
}

func coseECDSAPublicKey(t *testing.T, pub *ecdsa.PublicKey) []byte {
	t.Helper()
	data, err := webauthncbor.Marshal(map[int64]any{
		1:  int64(webauthncose.EllipticKey),
		3:  int64(webauthncose.AlgES256),
		-1: int64(webauthncose.P256),
		-2: pub.X.FillBytes(make([]byte, 32)),
		-3: pub.Y.FillBytes(make([]byte, 32)),
	})
	if err != nil {
		t.Fatalf("marshal COSE public key: %v", err)
	}
	return data
}

func attestedCredentialData(t *testing.T, credentialID, coseKey []byte) []byte {
	t.Helper()
	data := make([]byte, 16, 16+2+len(credentialID)+len(coseKey)) // 16-byte zero AAGUID
	data = binary.BigEndian.AppendUint16(data, uint16(len(credentialID)))
	data = append(data, credentialID...)
	return append(data, coseKey...)
}

// authenticatorData builds the raw authenticatorData bytes for rpID,
// hashing rpID itself the same way an authenticator/browser would — this
// is what lets the "wrong RPID" test forge a mismatch without touching
// clientDataJSON, which carries only the origin, not the RP ID.
func authenticatorData(rpID string, flags protocol.AuthenticatorFlags, signCount uint32, attestedData []byte) []byte {
	hash := sha256.Sum256([]byte(rpID))
	data := make([]byte, 0, 37+len(attestedData))
	data = append(data, hash[:]...)
	data = append(data, byte(flags))
	data = binary.BigEndian.AppendUint32(data, signCount)
	return append(data, attestedData...)
}

func clientDataJSON(t *testing.T, ceremonyType, challenge, origin string) []byte {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"type":        ceremonyType,
		"challenge":   challenge,
		"origin":      origin,
		"crossOrigin": false,
	})
	if err != nil {
		t.Fatalf("marshal clientDataJSON: %v", err)
	}
	return data
}

func signCeremony(t *testing.T, key *ecdsa.PrivateKey, authData, clientData []byte) []byte {
	t.Helper()
	clientDataHash := sha256.Sum256(clientData)
	signed := make([]byte, 0, len(authData)+len(clientDataHash))
	signed = append(signed, authData...)
	signed = append(signed, clientDataHash[:]...)
	digest := sha256.Sum256(signed)
	sig, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
	if err != nil {
		t.Fatalf("ecdsa.SignASN1() error = %v", err)
	}
	return sig
}

// packedAttestationObject builds a "packed" attestation with no x5c
// (self/"basic_surrogate" attestation): the signature is produced by the
// credential's own private key, which is exactly what BeginRegistration's
// default AttestationConveyancePreference tolerates.
func packedAttestationObject(t *testing.T, authData, sig []byte) []byte {
	t.Helper()
	data, err := webauthncbor.Marshal(map[string]any{
		"fmt":      "packed",
		"attStmt":  map[string]any{"alg": int64(webauthncose.AlgES256), "sig": sig},
		"authData": authData,
	})
	if err != nil {
		t.Fatalf("marshal attestationObject: %v", err)
	}
	return data
}

func registrationResponseJSON(t *testing.T, credentialID, attestationObject, clientData []byte) []byte {
	t.Helper()
	id := base64.RawURLEncoding.EncodeToString(credentialID)
	data, err := json.Marshal(map[string]any{
		"id": id, "rawId": id, "type": "public-key",
		"response": map[string]any{
			"attestationObject": base64.RawURLEncoding.EncodeToString(attestationObject),
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString(clientData),
		},
	})
	if err != nil {
		t.Fatalf("marshal registration response: %v", err)
	}
	return data
}

func assertionResponseJSON(t *testing.T, credentialID, authData, clientData, sig, userHandle []byte) []byte {
	t.Helper()
	id := base64.RawURLEncoding.EncodeToString(credentialID)
	data, err := json.Marshal(map[string]any{
		"id": id, "rawId": id, "type": "public-key",
		"response": map[string]any{
			"authenticatorData": base64.RawURLEncoding.EncodeToString(authData),
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString(clientData),
			"signature":         base64.RawURLEncoding.EncodeToString(sig),
			"userHandle":        base64.RawURLEncoding.EncodeToString(userHandle),
		},
	})
	if err != nil {
		t.Fatalf("marshal assertion response: %v", err)
	}
	return data
}

// registerWebAuthnCredential drives a full, real registration ceremony
// through auth.Service (Begin then Finish), returning the credential ID,
// its private key, and the server-generated user handle the service bound
// this credential to (auth.Service.BeginWebAuthnRegistration mints a fresh
// random handle per registration; it is not the account's user ID).
func registerWebAuthnCredential(t *testing.T, ctx context.Context, h *fHarness, userID string) (credentialID []byte, key *ecdsa.PrivateKey, userHandle []byte) {
	t.Helper()

	creation, err := h.service.BeginWebAuthnRegistration(ctx, userID)
	if err != nil {
		t.Fatalf("BeginWebAuthnRegistration() error = %v", err)
	}
	handle, ok := creation.Response.User.ID.(protocol.URLEncodedBase64)
	if !ok {
		t.Fatalf("creation.Response.User.ID has unexpected type %T", creation.Response.User.ID)
	}

	credentialID = []byte("synthetic-credential-0000000001")
	key = generateWebAuthnKey(t)
	coseKey := coseECDSAPublicKey(t, &key.PublicKey)
	attested := attestedCredentialData(t, credentialID, coseKey)
	authData := authenticatorData(testWebAuthnRPID, protocol.FlagUserPresent|protocol.FlagAttestedCredentialData, 0, attested)
	clientData := clientDataJSON(t, "webauthn.create", creation.Response.Challenge.String(), testWebAuthnOrigin)
	sig := signCeremony(t, key, authData, clientData)
	attestationObject := packedAttestationObject(t, authData, sig)
	body := registrationResponseJSON(t, credentialID, attestationObject, clientData)

	if err := h.service.FinishWebAuthnRegistration(ctx, userID, body); err != nil {
		t.Fatalf("FinishWebAuthnRegistration() error = %v", err)
	}
	return credentialID, key, []byte(handle)
}

// --- tests -------------------------------------------------------------

// TestWebAuthnRegistrationAndAuthenticationCeremonySucceeds is case 1: a
// full successful registration followed by a full successful
// authentication ceremony, ending in issueSessionTokens (confirmed here via
// the fake refresh-token repository's family/session counters, reinforcing
// the "no parallel token issuance" invariant from workstream F).
func TestWebAuthnRegistrationAndAuthenticationCeremonySucceeds(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()
	user, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "passkey@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}

	credentialID, key, userHandle := registerWebAuthnCredential(t, ctx, h, user.ID)

	assertion, err := h.service.BeginWebAuthnAuthentication(ctx)
	if err != nil {
		t.Fatalf("BeginWebAuthnAuthentication() error = %v", err)
	}
	authData := authenticatorData(testWebAuthnRPID, protocol.FlagUserPresent, 1, nil)
	clientData := clientDataJSON(t, "webauthn.get", assertion.Response.Challenge.String(), testWebAuthnOrigin)
	sig := signCeremony(t, key, authData, clientData)
	body := assertionResponseJSON(t, credentialID, authData, clientData, sig, userHandle)

	tokens, err := h.service.FinishWebAuthnAuthentication(ctx, body, auth.RefreshSessionMetadata{})
	if err != nil {
		t.Fatalf("FinishWebAuthnAuthentication() error = %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("FinishWebAuthnAuthentication() did not issue access/refresh tokens")
	}

	h.refresh.mu.Lock()
	families, sessions := h.refresh.nextFamily, h.refresh.nextSession
	h.refresh.mu.Unlock()
	if families != 1 || sessions != 1 {
		t.Fatalf("refresh family/session creations = %d/%d, want exactly 1/1 (a single issueSessionTokens call, no parallel issuance path)", families, sessions)
	}
}

// TestWebAuthnAuthenticationRejectsWrongOrigin is case 2.
func TestWebAuthnAuthenticationRejectsWrongOrigin(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()
	user, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "wrong-origin@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	credentialID, key, userHandle := registerWebAuthnCredential(t, ctx, h, user.ID)

	assertion, err := h.service.BeginWebAuthnAuthentication(ctx)
	if err != nil {
		t.Fatalf("BeginWebAuthnAuthentication() error = %v", err)
	}
	authData := authenticatorData(testWebAuthnRPID, protocol.FlagUserPresent, 1, nil)
	// The RP origin is configured as testWebAuthnOrigin; a client claiming a
	// different origin in clientDataJSON must be rejected.
	clientData := clientDataJSON(t, "webauthn.get", assertion.Response.Challenge.String(), "https://evil.example")
	sig := signCeremony(t, key, authData, clientData)
	body := assertionResponseJSON(t, credentialID, authData, clientData, sig, userHandle)

	if _, err := h.service.FinishWebAuthnAuthentication(ctx, body, auth.RefreshSessionMetadata{}); !errors.Is(err, auth.ErrWebAuthnVerificationFailed) {
		t.Fatalf("FinishWebAuthnAuthentication() with wrong origin error = %v, want ErrWebAuthnVerificationFailed", err)
	}
}

// TestWebAuthnAuthenticationRejectsWrongRPID is case 3. The RP ID is never
// sent directly by a client; it is only verifiable via the SHA-256 hash
// embedded in authenticatorData, so this forges that hash while keeping
// clientDataJSON's origin correct.
func TestWebAuthnAuthenticationRejectsWrongRPID(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()
	user, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "wrong-rpid@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	credentialID, key, userHandle := registerWebAuthnCredential(t, ctx, h, user.ID)

	assertion, err := h.service.BeginWebAuthnAuthentication(ctx)
	if err != nil {
		t.Fatalf("BeginWebAuthnAuthentication() error = %v", err)
	}
	// Configured RP ID is testWebAuthnRPID ("example.com"); hash a different
	// RP ID into authenticatorData instead.
	authData := authenticatorData("attacker.example", protocol.FlagUserPresent, 1, nil)
	clientData := clientDataJSON(t, "webauthn.get", assertion.Response.Challenge.String(), testWebAuthnOrigin)
	sig := signCeremony(t, key, authData, clientData)
	body := assertionResponseJSON(t, credentialID, authData, clientData, sig, userHandle)

	if _, err := h.service.FinishWebAuthnAuthentication(ctx, body, auth.RefreshSessionMetadata{}); !errors.Is(err, auth.ErrWebAuthnVerificationFailed) {
		t.Fatalf("FinishWebAuthnAuthentication() with wrong RP ID error = %v, want ErrWebAuthnVerificationFailed", err)
	}
}

// TestWebAuthnAuthenticationRejectsChallengeMismatch is case 4: the
// client-reported challenge never matches any persisted, server-issued
// challenge, so the single-use challenge lookup itself must fail.
func TestWebAuthnAuthenticationRejectsChallengeMismatch(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()
	user, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "bad-challenge@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	credentialID, key, userHandle := registerWebAuthnCredential(t, ctx, h, user.ID)

	if _, err := h.service.BeginWebAuthnAuthentication(ctx); err != nil {
		t.Fatalf("BeginWebAuthnAuthentication() error = %v", err)
	}
	authData := authenticatorData(testWebAuthnRPID, protocol.FlagUserPresent, 1, nil)
	forgedChallenge := base64.RawURLEncoding.EncodeToString([]byte("this-challenge-was-never-issued"))
	clientData := clientDataJSON(t, "webauthn.get", forgedChallenge, testWebAuthnOrigin)
	sig := signCeremony(t, key, authData, clientData)
	body := assertionResponseJSON(t, credentialID, authData, clientData, sig, userHandle)

	if _, err := h.service.FinishWebAuthnAuthentication(ctx, body, auth.RefreshSessionMetadata{}); !errors.Is(err, auth.ErrWebAuthnChallengeInvalid) {
		t.Fatalf("FinishWebAuthnAuthentication() with a forged challenge error = %v, want ErrWebAuthnChallengeInvalid", err)
	}
}

// TestWebAuthnAuthenticationRejectsNonIncreasingSignCounter is case 5:
// clone/replay detection. A second assertion reusing a stored (not
// strictly increasing) sign counter after a first successful ceremony must
// be rejected.
func TestWebAuthnAuthenticationRejectsNonIncreasingSignCounter(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()
	user, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "clone@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	credentialID, key, userHandle := registerWebAuthnCredential(t, ctx, h, user.ID)

	authenticateOnce := func(signCount uint32) (auth.IssuedTokens, error) {
		assertion, err := h.service.BeginWebAuthnAuthentication(ctx)
		if err != nil {
			t.Fatalf("BeginWebAuthnAuthentication() error = %v", err)
		}
		authData := authenticatorData(testWebAuthnRPID, protocol.FlagUserPresent, signCount, nil)
		clientData := clientDataJSON(t, "webauthn.get", assertion.Response.Challenge.String(), testWebAuthnOrigin)
		sig := signCeremony(t, key, authData, clientData)
		body := assertionResponseJSON(t, credentialID, authData, clientData, sig, userHandle)
		return h.service.FinishWebAuthnAuthentication(ctx, body, auth.RefreshSessionMetadata{})
	}

	// First ceremony: sign count 1, greater than the stored 0 from
	// registration, so it succeeds and the stored counter advances to 1.
	if _, err := authenticateOnce(1); err != nil {
		t.Fatalf("first FinishWebAuthnAuthentication() error = %v", err)
	}

	// Second ceremony (a fresh, otherwise-valid challenge) reuses the same
	// sign count (1), which is not strictly greater than the now-stored
	// value of 1 — the library's own CloneWarning signal, reinforced by
	// this repository's WHERE sign_count < $2 guard on the real store.
	if _, err := authenticateOnce(1); !errors.Is(err, auth.ErrWebAuthnVerificationFailed) {
		t.Fatalf("second FinishWebAuthnAuthentication() with a non-increasing sign counter error = %v, want ErrWebAuthnVerificationFailed", err)
	}
}

// TestWebAuthnAuthenticationRejectsCredentialUserHandleMismatch is case 6:
// the asserting credential ID is real and registered, but the user handle
// the authenticator reports does not match the handle that credential was
// actually registered under — ownership must be resolved strictly from the
// stored record, never trusted from the assertion alone.
func TestWebAuthnAuthenticationRejectsCredentialUserHandleMismatch(t *testing.T) {
	t.Parallel()
	h := newFHarness(t)
	ctx := context.Background()
	user, err := h.service.SignUp(ctx, auth.SignUpInput{Email: "mismatch@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	credentialID, key, _ := registerWebAuthnCredential(t, ctx, h, user.ID)

	assertion, err := h.service.BeginWebAuthnAuthentication(ctx)
	if err != nil {
		t.Fatalf("BeginWebAuthnAuthentication() error = %v", err)
	}
	authData := authenticatorData(testWebAuthnRPID, protocol.FlagUserPresent, 1, nil)
	clientData := clientDataJSON(t, "webauthn.get", assertion.Response.Challenge.String(), testWebAuthnOrigin)
	sig := signCeremony(t, key, authData, clientData)
	// A real, registered credential ID, but a user handle that belongs to
	// no one — never the handle this credential was actually created with.
	claimedHandle := []byte("not-the-real-user-handle-000000")
	body := assertionResponseJSON(t, credentialID, authData, clientData, sig, claimedHandle)

	if _, err := h.service.FinishWebAuthnAuthentication(ctx, body, auth.RefreshSessionMetadata{}); !errors.Is(err, auth.ErrWebAuthnVerificationFailed) {
		t.Fatalf("FinishWebAuthnAuthentication() with a mismatched user handle error = %v, want ErrWebAuthnVerificationFailed", err)
	}
}
