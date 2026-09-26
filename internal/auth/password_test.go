package auth_test

import (
	"strings"
	"testing"

	"github.com/hydrz/starter/internal/auth"
)

func TestHashPasswordAndVerify(t *testing.T) {
	t.Parallel()

	hash, err := auth.HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("hash = %q, want PHC argon2id prefix", hash)
	}

	ok, err := auth.VerifyPassword(hash, "correct horse battery staple")
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !ok {
		t.Error("VerifyPassword() = false, want true for correct password")
	}
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	t.Parallel()

	hash, err := auth.HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	ok, err := auth.VerifyPassword(hash, "wrong password")
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if ok {
		t.Error("VerifyPassword() = true, want false for wrong password")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	t.Parallel()

	if _, err := auth.VerifyPassword("not-a-phc-hash", "anything"); err == nil {
		t.Fatal("VerifyPassword() error = nil, want ErrInvalidPasswordHash")
	}
}

func TestVerifyPasswordOrDummyHandlesUnknownAccount(t *testing.T) {
	t.Parallel()

	if auth.VerifyPasswordOrDummy("", "any-password") {
		t.Error("VerifyPasswordOrDummy(\"\") = true, want false for unknown account")
	}
}

func TestHashPasswordProducesUniqueSalts(t *testing.T) {
	t.Parallel()

	first, err := auth.HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	second, err := auth.HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if first == second {
		t.Error("HashPassword() produced identical hashes for two calls; salts must differ")
	}
}
