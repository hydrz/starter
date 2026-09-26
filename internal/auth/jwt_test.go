package auth_test

import (
	"crypto/ed25519"
	"strings"
	"testing"
	"time"

	"github.com/hydrz/starter/internal/auth"
)

func generateKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}
	return public, private
}

func TestIssueAndVerifyAccessToken(t *testing.T) {
	t.Parallel()

	public, private := generateKeyPair(t)
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	issuer, err := auth.NewIssuer("kid-1", private, "starter", clock)
	if err != nil {
		t.Fatalf("NewIssuer() error = %v", err)
	}
	verifier, err := auth.NewVerifier(map[string]ed25519.PublicKey{"kid-1": public}, "starter", clock)
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}

	token, claims, err := issuer.IssueAccessToken("user-1", "session-1", "starter-api", 15*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}
	if claims.Type != auth.TokenType {
		t.Errorf("claims.Type = %q, want %q", claims.Type, auth.TokenType)
	}

	verified, err := verifier.Verify(token, "starter-api")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if verified.Subject != "user-1" || verified.SessionID != "session-1" {
		t.Errorf("verified claims = %+v, want subject=user-1 session=session-1", verified)
	}
}

func TestVerifyRejectsWrongAudience(t *testing.T) {
	t.Parallel()

	public, private := generateKeyPair(t)
	clock := &fakeClock{now: time.Now()}
	issuer, _ := auth.NewIssuer("kid-1", private, "starter", clock)
	verifier, _ := auth.NewVerifier(map[string]ed25519.PublicKey{"kid-1": public}, "starter", clock)

	token, _, err := issuer.IssueAccessToken("user-1", "session-1", "starter-api", 15*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	if _, err := verifier.Verify(token, "some-other-audience"); err == nil {
		t.Fatal("Verify() error = nil, want ErrInvalidToken for wrong audience")
	}
}

func TestVerifyRejectsUnknownKID(t *testing.T) {
	t.Parallel()

	_, private := generateKeyPair(t)
	otherPublic, _ := generateKeyPair(t)
	clock := &fakeClock{now: time.Now()}
	issuer, _ := auth.NewIssuer("kid-1", private, "starter", clock)
	verifier, _ := auth.NewVerifier(map[string]ed25519.PublicKey{"kid-2": otherPublic}, "starter", clock)

	token, _, err := issuer.IssueAccessToken("user-1", "session-1", "starter-api", 15*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	if _, err := verifier.Verify(token, "starter-api"); err == nil {
		t.Fatal("Verify() error = nil, want ErrInvalidToken for unknown kid")
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	t.Parallel()

	public, private := generateKeyPair(t)
	issuerClock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	issuer, _ := auth.NewIssuer("kid-1", private, "starter", issuerClock)

	token, _, err := issuer.IssueAccessToken("user-1", "session-1", "starter-api", time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	verifierClock := &fakeClock{now: issuerClock.now.Add(2 * time.Minute)}
	verifier, _ := auth.NewVerifier(map[string]ed25519.PublicKey{"kid-1": public}, "starter", verifierClock)

	if _, err := verifier.Verify(token, "starter-api"); err == nil {
		t.Fatal("Verify() error = nil, want ErrInvalidToken for expired token")
	}
}

func TestVerifyRejectsTamperedSignature(t *testing.T) {
	t.Parallel()

	public, private := generateKeyPair(t)
	clock := &fakeClock{now: time.Now()}
	issuer, _ := auth.NewIssuer("kid-1", private, "starter", clock)
	verifier, _ := auth.NewVerifier(map[string]ed25519.PublicKey{"kid-1": public}, "starter", clock)

	token, _, err := issuer.IssueAccessToken("user-1", "session-1", "starter-api", 15*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	parts := strings.Split(token, ".")
	// Flip the first character of the signature segment.
	sig := []byte(parts[2])
	if sig[0] == 'A' {
		sig[0] = 'B'
	} else {
		sig[0] = 'A'
	}
	tampered := parts[0] + "." + parts[1] + "." + string(sig)

	if _, err := verifier.Verify(tampered, "starter-api"); err == nil {
		t.Fatal("Verify() error = nil, want ErrInvalidToken for tampered signature")
	}
}

func TestVerifyRejectsMalformedToken(t *testing.T) {
	t.Parallel()

	public, _ := generateKeyPair(t)
	clock := &fakeClock{now: time.Now()}
	verifier, _ := auth.NewVerifier(map[string]ed25519.PublicKey{"kid-1": public}, "starter", clock)

	for _, token := range []string{"", "not-a-jwt", "a.b", "a.b.c.d"} {
		if _, err := verifier.Verify(token, "starter-api"); err == nil {
			t.Errorf("Verify(%q) error = nil, want ErrInvalidToken", token)
		}
	}
}

func TestNewIssuerRejectsInvalidKey(t *testing.T) {
	t.Parallel()

	if _, err := auth.NewIssuer("kid-1", []byte("too-short"), "starter", nil); err == nil {
		t.Fatal("NewIssuer() error = nil, want error for invalid private key size")
	}
}

func TestNewVerifierRejectsEmptyKeyset(t *testing.T) {
	t.Parallel()

	if _, err := auth.NewVerifier(nil, "starter", nil); err == nil {
		t.Fatal("NewVerifier() error = nil, want error for empty keyset")
	}
}
