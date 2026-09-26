package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
)

// secretByteLength is the amount of random entropy used for opaque bearer
// secrets (refresh tokens, one-time tokens, API key material).
const secretByteLength = 32

// ErrEmptyPepper is returned by NewSecretDigester when constructed without a
// pepper. A pepper is required so stored digests cannot be recomputed from a
// database leak alone.
var ErrEmptyPepper = errors.New("auth: secret digester pepper must not be empty")

// GenerateSecret returns a new random opaque secret encoded as unpadded
// base64url, along with its raw bytes. The encoded form is safe to hand to
// clients; the raw bytes are not persisted.
func GenerateSecret() (encoded string, raw []byte, err error) {
	raw = make([]byte, secretByteLength)
	if _, err = rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), raw, nil
}

// SecretDigester computes versioned HMAC-SHA256 digests of opaque secrets so
// that only the digest, never the secret itself, is stored. The version
// allows the pepper to rotate without invalidating comparison logic; a
// digest carries its version so verification always uses the pepper that
// produced it. Only the current, active pepper is supported by this type;
// operators wishing to rotate a pepper must invalidate outstanding secrets
// of that version (refresh tokens, reset/verification tokens, API keys).
type SecretDigester struct {
	version byte
	pepper  []byte
}

// NewSecretDigester returns a SecretDigester keyed by pepper and tagged with
// version. pepper must be non-empty; it is normally sourced from process
// configuration and never logged.
func NewSecretDigester(version byte, pepper []byte) (*SecretDigester, error) {
	if len(pepper) == 0 {
		return nil, ErrEmptyPepper
	}
	cloned := make([]byte, len(pepper))
	copy(cloned, pepper)
	return &SecretDigester{version: version, pepper: cloned}, nil
}

// Digest returns the versioned HMAC-SHA256 digest of secret: one version
// byte followed by the 32-byte HMAC output.
func (digester *SecretDigester) Digest(secret []byte) []byte {
	mac := hmac.New(sha256.New, digester.pepper)
	mac.Write(secret)
	sum := mac.Sum(nil)
	digest := make([]byte, 0, 1+len(sum))
	digest = append(digest, digester.version)
	digest = append(digest, sum...)
	return digest
}

// Verify reports whether secret matches digest in constant time. Digests
// produced under a different version never match.
func (digester *SecretDigester) Verify(digest, secret []byte) bool {
	if len(digest) == 0 || digest[0] != digester.version {
		return false
	}
	expected := digester.Digest(secret)
	return subtle.ConstantTimeCompare(expected, digest) == 1
}
