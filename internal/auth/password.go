package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters. These are conservative interactive defaults; they can
// be tuned later without breaking existing hashes because parameters are
// encoded in the PHC string itself.
const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // KiB
	argon2Threads = 2
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

// ErrInvalidPasswordHash is returned when a stored password hash cannot be
// parsed as a PHC-formatted Argon2id hash.
var ErrInvalidPasswordHash = errors.New("auth: invalid password hash")

// dummyPasswordHash is a valid Argon2id PHC hash computed for a fixed
// password. It is used to perform a constant-time-shaped comparison when no
// account exists for a given email, preventing account enumeration through
// timing differences between "unknown email" and "wrong password".
var dummyPasswordHash = mustHashPassword("dummy-password-for-timing-parity")

// HashPassword returns a PHC-formatted Argon2id hash of password.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	return encodePasswordHash(salt, password), nil
}

func mustHashPassword(password string) string {
	hash, err := HashPassword(password)
	if err != nil {
		panic(err)
	}
	return hash
}

func encodePasswordHash(salt []byte, password string) string {
	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Memory,
		argon2Time,
		argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
}

// VerifyPassword reports whether password matches the PHC-formatted Argon2id
// hash. Comparison of the derived key is constant-time.
func VerifyPassword(hash, password string) (bool, error) {
	version, memory, time_, threads, salt, key, err := decodePasswordHash(hash)
	if err != nil {
		return false, err
	}
	if version != argon2.Version {
		return false, ErrInvalidPasswordHash
	}
	candidate := argon2.IDKey([]byte(password), salt, time_, memory, threads, uint32(len(key)))
	return subtle.ConstantTimeCompare(candidate, key) == 1, nil
}

// VerifyPasswordOrDummy behaves like VerifyPassword but performs an
// equivalent Argon2id computation against a fixed dummy hash when hash is
// empty, so callers can avoid branching on account existence before running
// password verification.
func VerifyPasswordOrDummy(hash, password string) bool {
	if hash == "" {
		_, _ = VerifyPassword(dummyPasswordHash, password)
		return false
	}
	ok, err := VerifyPassword(hash, password)
	return err == nil && ok
}

func decodePasswordHash(hash string) (version, memory, time_ uint32, threads uint8, salt, key []byte, err error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return 0, 0, 0, 0, nil, nil, ErrInvalidPasswordHash
	}
	if _, err = fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return 0, 0, 0, 0, nil, nil, ErrInvalidPasswordHash
	}
	var parsedThreads uint32
	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time_, &parsedThreads); err != nil {
		return 0, 0, 0, 0, nil, nil, ErrInvalidPasswordHash
	}
	threads = uint8(parsedThreads)
	if salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil {
		return 0, 0, 0, 0, nil, nil, ErrInvalidPasswordHash
	}
	if key, err = base64.RawStdEncoding.DecodeString(parts[5]); err != nil {
		return 0, 0, 0, 0, nil, nil, ErrInvalidPasswordHash
	}
	return version, memory, time_, threads, salt, key, nil
}
