package auth_test

import (
	"testing"

	"github.com/hydrz/starter/internal/auth"
)

func TestGenerateSecretIsUnique(t *testing.T) {
	t.Parallel()

	firstEncoded, firstRaw, err := auth.GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret() error = %v", err)
	}
	secondEncoded, secondRaw, err := auth.GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret() error = %v", err)
	}
	if firstEncoded == secondEncoded {
		t.Error("GenerateSecret() produced identical encoded secrets across calls")
	}
	if string(firstRaw) == string(secondRaw) {
		t.Error("GenerateSecret() produced identical raw secrets across calls")
	}
}

func TestSecretDigesterVerify(t *testing.T) {
	t.Parallel()

	digester, err := auth.NewSecretDigester(1, []byte("test-pepper"))
	if err != nil {
		t.Fatalf("NewSecretDigester() error = %v", err)
	}

	secret := []byte("a-raw-secret")
	digest := digester.Digest(secret)

	if !digester.Verify(digest, secret) {
		t.Error("Verify() = false, want true for matching secret")
	}
	if digester.Verify(digest, []byte("a-different-secret")) {
		t.Error("Verify() = true, want false for mismatched secret")
	}
}

func TestSecretDigesterRejectsDifferentVersion(t *testing.T) {
	t.Parallel()

	v1, err := auth.NewSecretDigester(1, []byte("pepper"))
	if err != nil {
		t.Fatalf("NewSecretDigester() error = %v", err)
	}
	v2, err := auth.NewSecretDigester(2, []byte("pepper"))
	if err != nil {
		t.Fatalf("NewSecretDigester() error = %v", err)
	}

	secret := []byte("a-raw-secret")
	digest := v1.Digest(secret)

	if v2.Verify(digest, secret) {
		t.Error("Verify() = true across versions, want false")
	}
}

func TestNewSecretDigesterRejectsEmptyPepper(t *testing.T) {
	t.Parallel()

	if _, err := auth.NewSecretDigester(1, nil); err == nil {
		t.Fatal("NewSecretDigester() error = nil, want ErrEmptyPepper")
	}
}
