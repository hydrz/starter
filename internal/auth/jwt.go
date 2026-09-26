// Package auth implements the identity domain: accounts, password
// credentials, JWT access tokens, opaque refresh sessions, one-time
// verification/reset tokens, and API keys. It depends only on narrow ports
// (Clock, SecretGenerator, TokenIssuer, Mailer) so the domain logic is
// testable without a database, SMTP server, or wall clock.
package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// TokenType is the required "typ" claim on every access token this package
// issues. It exists so an access token can never be confused with an opaque
// refresh token or another JWT typ used elsewhere in the platform.
const TokenType = "access"

const jwtAlgorithm = "EdDSA"

// Claims are the minimum stable fields carried by an access token. It
// intentionally excludes roles, organizations, or entitlements: those are
// authorization concerns evaluated per-request against current state, not
// facts to freeze into a bearer token.
type Claims struct {
	Subject   string    `json:"sub"`
	SessionID string    `json:"sid"`
	TokenID   string    `json:"jti"`
	Issuer    string    `json:"iss"`
	Audience  string    `json:"aud"`
	Type      string    `json:"typ"`
	IssuedAt  time.Time `json:"-"`
	NotBefore time.Time `json:"-"`
	ExpiresAt time.Time `json:"-"`
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	Type      string `json:"typ"`
}

type jwtClaims struct {
	Subject   string `json:"sub"`
	SessionID string `json:"sid"`
	TokenID   string `json:"jti"`
	Issuer    string `json:"iss"`
	Audience  string `json:"aud"`
	Type      string `json:"typ"`
	IssuedAt  int64  `json:"iat"`
	NotBefore int64  `json:"nbf"`
	ExpiresAt int64  `json:"exp"`
}

var (
	// ErrInvalidToken is returned for any structurally invalid, mismatched,
	// or expired JWT. It intentionally does not distinguish the reason so
	// callers cannot be used as a decoding oracle.
	ErrInvalidToken = errors.New("auth: invalid access token")
)

// Issuer signs access tokens with a single active Ed25519 key. It matches
// config.AuthConfig: exactly one active kid/private key pair per process.
type Issuer struct {
	activeKID  string
	privateKey ed25519.PrivateKey
	jwtIssuer  string
	clock      Clock
}

// NewIssuer returns an Issuer that signs with privateKey under activeKID.
// privateKey must be a valid Ed25519 private key.
func NewIssuer(activeKID string, privateKey ed25519.PrivateKey, jwtIssuer string, clock Clock) (*Issuer, error) {
	if activeKID == "" {
		return nil, errors.New("auth: active kid must not be empty")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("auth: signing private key must be a valid Ed25519 key")
	}
	if clock == nil {
		clock = SystemClock{}
	}
	return &Issuer{activeKID: activeKID, privateKey: privateKey, jwtIssuer: jwtIssuer, clock: clock}, nil
}

// IssueAccessToken returns a signed EdDSA JWT for subject/sessionID valid for
// ttl, along with the claims that were embedded (useful for logging jti).
func (issuer *Issuer) IssueAccessToken(subject, sessionID, audience string, ttl time.Duration) (string, Claims, error) {
	tokenID, _, err := GenerateSecret()
	if err != nil {
		return "", Claims{}, fmt.Errorf("generate token id: %w", err)
	}

	now := issuer.clock.Now().UTC()
	claims := Claims{
		Subject:   subject,
		SessionID: sessionID,
		TokenID:   tokenID,
		Issuer:    issuer.jwtIssuer,
		Audience:  audience,
		Type:      TokenType,
		IssuedAt:  now,
		NotBefore: now,
		ExpiresAt: now.Add(ttl),
	}

	header := jwtHeader{Algorithm: jwtAlgorithm, KeyID: issuer.activeKID, Type: "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", Claims{}, fmt.Errorf("encode header: %w", err)
	}
	payload := jwtClaims{
		Subject:   claims.Subject,
		SessionID: claims.SessionID,
		TokenID:   claims.TokenID,
		Issuer:    claims.Issuer,
		Audience:  claims.Audience,
		Type:      claims.Type,
		IssuedAt:  claims.IssuedAt.Unix(),
		NotBefore: claims.NotBefore.Unix(),
		ExpiresAt: claims.ExpiresAt.Unix(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", Claims{}, fmt.Errorf("encode claims: %w", err)
	}

	signingInput := encodeSegment(headerJSON) + "." + encodeSegment(payloadJSON)
	signature := ed25519.Sign(issuer.privateKey, []byte(signingInput))
	return signingInput + "." + encodeSegment(signature), claims, nil
}

// Verifier validates EdDSA JWTs against a keyset of currently trusted
// public keys, selected strictly by the token's protected kid.
type Verifier struct {
	publicKeys map[string]ed25519.PublicKey
	jwtIssuer  string
	clock      Clock
}

// NewVerifier returns a Verifier that accepts only tokens signed by one of
// publicKeys, addressed by kid, matching issuer and audience.
func NewVerifier(publicKeys map[string]ed25519.PublicKey, jwtIssuer string, clock Clock) (*Verifier, error) {
	if len(publicKeys) == 0 {
		return nil, errors.New("auth: verification keyset must not be empty")
	}
	if clock == nil {
		clock = SystemClock{}
	}
	cloned := make(map[string]ed25519.PublicKey, len(publicKeys))
	for kid, key := range publicKeys {
		if kid == "" || len(key) != ed25519.PublicKeySize {
			return nil, errors.New("auth: verification keyset contains an invalid entry")
		}
		cloned[kid] = key
	}
	return &Verifier{publicKeys: cloned, jwtIssuer: jwtIssuer, clock: clock}, nil
}

// Verify parses and validates token, rejecting any algorithm other than
// EdDSA, a missing or unknown kid, a bad signature, a mismatched
// issuer/audience/typ, or a token outside its nbf/exp window.
func (verifier *Verifier) Verify(token, expectedAudience string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	headerJSON, err := decodeSegment(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var header jwtHeader
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if header.Algorithm != jwtAlgorithm {
		return Claims{}, ErrInvalidToken
	}
	publicKey, ok := verifier.publicKeys[header.KeyID]
	if !ok {
		return Claims{}, ErrInvalidToken
	}

	signature, err := decodeSegment(parts[2])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	signingInput := parts[0] + "." + parts[1]
	if !ed25519.Verify(publicKey, []byte(signingInput), signature) {
		return Claims{}, ErrInvalidToken
	}

	payloadJSON, err := decodeSegment(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var payload jwtClaims
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if payload.Type != TokenType {
		return Claims{}, ErrInvalidToken
	}
	if verifier.jwtIssuer != "" && payload.Issuer != verifier.jwtIssuer {
		return Claims{}, ErrInvalidToken
	}
	if expectedAudience != "" && payload.Audience != expectedAudience {
		return Claims{}, ErrInvalidToken
	}

	now := verifier.clock.Now().UTC()
	if now.Before(time.Unix(payload.NotBefore, 0)) {
		return Claims{}, ErrInvalidToken
	}
	if !now.Before(time.Unix(payload.ExpiresAt, 0)) {
		return Claims{}, ErrInvalidToken
	}

	return Claims{
		Subject:   payload.Subject,
		SessionID: payload.SessionID,
		TokenID:   payload.TokenID,
		Issuer:    payload.Issuer,
		Audience:  payload.Audience,
		Type:      payload.Type,
		IssuedAt:  time.Unix(payload.IssuedAt, 0).UTC(),
		NotBefore: time.Unix(payload.NotBefore, 0).UTC(),
		ExpiresAt: time.Unix(payload.ExpiresAt, 0).UTC(),
	}, nil
}

func encodeSegment(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeSegment(segment string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(segment)
}
