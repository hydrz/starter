package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// TOTPCipher encrypts/decrypts TOTP secrets at rest with AES-256-GCM, keyed
// by a dedicated symmetric key (config.AuthConfig.TOTPEncryptionKey /
// AUTH_TOTP_ENCRYPTION_KEY) that is never derived from or shared with the
// JWT signing key or the secret-digest pepper: each rotates independently.
// The plaintext secret only ever exists in memory inside this package,
// never logged and never returned across the Service boundary.
type TOTPCipher struct {
	gcm cipher.AEAD
}

// NewTOTPCipher returns a TOTPCipher keyed by key, which must be exactly 32
// bytes (AES-256).
func NewTOTPCipher(key []byte) (*TOTPCipher, error) {
	if len(key) != 32 {
		return nil, errors.New("auth: totp encryption key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}
	return &TOTPCipher{gcm: gcm}, nil
}

// Encrypt seals plaintext, returning ciphertext and the random nonce used.
// Both must be persisted together; the nonce is not secret.
func (c *TOTPCipher) Encrypt(plaintext []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, c.gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext = c.gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt opens ciphertext using nonce, returning the plaintext secret. The
// caller must not log or otherwise persist the returned bytes.
func (c *TOTPCipher) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt totp secret: %w", err)
	}
	return plaintext, nil
}
