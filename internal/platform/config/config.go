// Package config loads and validates process configuration from environment variables.
package config

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAddress            = ":8080"
	defaultDatabaseURL        = "postgres://starter:starter@127.0.0.1:5432/starter?sslmode=disable"
	defaultAccessTokenTTL     = 15 * time.Minute
	defaultRefreshTokenTTL    = 30 * 24 * time.Hour
	defaultWorkerPollInterval = time.Second
	defaultWorkerBatchSize    = 50
)

// Config is the application configuration. Integrations whose Enabled field is
// false are deliberately not initialized by the application.
type Config struct {
	DatabaseURL string
	Address     string
	AutoMigrate bool

	Auth          AuthConfig
	OAuth         OAuthConfig
	WebAuthn      WebAuthnConfig
	SMTP          SMTPConfig
	Stripe        StripeConfig
	Worker        WorkerConfig
	Authorization AuthorizationConfig
}

// AuthConfig supplies Ed25519 access-token material. Future token issuers must
// sign with SigningPrivateKey using EdDSA and set the protected-header kid to
// ActiveKID. Verifiers must select VerificationPublicKeys[protected kid], reject
// an absent or unknown kid, and accept every configured key during rotation.
//
// SecretDigestPepper is independent, high-entropy secret material used only to
// key an HMAC digester for opaque bearer secrets (refresh tokens, one-time
// verification/reset tokens, API keys). It must never be derived from
// SigningPrivateKey: JWT signing keys and the secret-digest pepper rotate on
// different schedules for different reasons, and deriving one from the other
// would silently invalidate every stored digest whenever the JWT key rotates.
type AuthConfig struct {
	Enabled                bool
	JWTIssuer              string
	AccessTokenTTL         time.Duration
	RefreshTokenTTL        time.Duration
	ActiveKID              string
	SigningPrivateKey      ed25519.PrivateKey
	VerificationPublicKeys map[string]ed25519.PublicKey
	SecretDigestPepper     []byte

	// TOTPEncryptionKey is a dedicated AES-256 key (raw 32 bytes) used only
	// to encrypt TOTP secrets at rest (internal/auth.TOTPCipher). It must
	// never be derived from SigningPrivateKey or SecretDigestPepper: all
	// three rotate independently for different reasons.
	TOTPEncryptionKey []byte
}

const (
	authActiveKIDEnv          = "AUTH_JWT_ACTIVE_KID"
	authSigningPrivateKeyEnv  = "AUTH_JWT_SIGNING_PRIVATE_KEY"
	authVerificationKeysetEnv = "AUTH_JWT_VERIFICATION_KEYSET"
	authIssuerEnv             = "AUTH_JWT_ISSUER"
	authAccessTokenTTLEnv     = "AUTH_JWT_ACCESS_TTL"
	authRefreshTokenTTLEnv    = "AUTH_REFRESH_TOKEN_TTL"
	authSecretPepperEnv       = "AUTH_SECRET_PEPPER"
	authTOTPEncryptionKeyEnv  = "AUTH_TOTP_ENCRYPTION_KEY"

	minSecretDigestPepperLength = 16
	totpEncryptionKeyLength     = 32
)

type OAuthConfig struct {
	Google OAuthProviderConfig
	GitHub OAuthProviderConfig
}

type OAuthProviderConfig struct {
	Enabled      bool
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type WebAuthnConfig struct {
	Enabled   bool
	RPID      string
	RPName    string
	RPOrigins []string
}

type SMTPConfig struct {
	Enabled     bool
	Host        string
	Port        int
	Username    string
	Password    string
	FromAddress string
}

type StripeConfig struct {
	Enabled        bool
	SecretKey      string
	WebhookSecret  string
	PublishableKey string
}

type WorkerConfig struct {
	Enabled      bool
	PollInterval time.Duration
	BatchSize    int
}

type AuthorizationConfig struct {
	Enabled    bool
	ModelPath  string
	PolicyPath string
}

// ValidationError reports configuration variable names only. It intentionally
// excludes values so callers can log it without leaking credentials.
type ValidationError struct {
	Names []string
}

func (e *ValidationError) Error() string {
	return "invalid configuration: " + strings.Join(e.Names, ", ")
}

// Load reads the process environment through lookup and returns a fully
// validated configuration. Passing os.LookupEnv is the normal production use;
// injecting a lookup function keeps tests hermetic.
func Load(lookup func(string) (string, bool)) (Config, error) {
	if lookup == nil {
		return Config{}, errors.New("configuration lookup is required")
	}

	cfg := Config{
		DatabaseURL: valueOr(lookup, "DATABASE_URL", defaultDatabaseURL),
		Address:     address(lookup),
		AutoMigrate: valueOr(lookup, "AUTO_MIGRATE", "true") != "false",
		Auth: AuthConfig{
			JWTIssuer:       valueOr(lookup, authIssuerEnv, "starter"),
			AccessTokenTTL:  defaultAccessTokenTTL,
			RefreshTokenTTL: defaultRefreshTokenTTL,
		},
		Worker: WorkerConfig{
			PollInterval: defaultWorkerPollInterval,
			BatchSize:    defaultWorkerBatchSize,
		},
	}

	var invalid []string
	if parsed, ok := parseDatabaseURL(cfg.DatabaseURL); !ok {
		invalid = append(invalid, "DATABASE_URL")
	} else {
		cfg.DatabaseURL = parsed
	}
	if !validAddress(cfg.Address) {
		invalid = append(invalid, "HTTP_ADDRESS")
	}

	cfg.Auth = loadAuth(lookup, &invalid, cfg.Auth)

	cfg.OAuth.Google = loadOAuthProvider(lookup, "OAUTH_GOOGLE", &invalid)
	cfg.OAuth.GitHub = loadOAuthProvider(lookup, "OAUTH_GITHUB", &invalid)
	cfg.WebAuthn = loadWebAuthn(lookup, &invalid)
	cfg.SMTP = loadSMTP(lookup, &invalid)
	cfg.Stripe = loadStripe(lookup, &invalid)
	cfg.Authorization = loadAuthorization(lookup, &invalid)

	if raw, present := lookup("WORKER_ENABLED"); present {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			invalid = append(invalid, "WORKER_ENABLED")
		} else {
			cfg.Worker.Enabled = parsed
		}
	}
	if raw, present := lookup("WORKER_POLL_INTERVAL"); present {
		cfg.Worker.PollInterval = parseDuration(raw, "WORKER_POLL_INTERVAL", &invalid)
	}
	if raw, present := lookup("WORKER_BATCH_SIZE"); present {
		cfg.Worker.BatchSize = parsePositiveInt(raw, "WORKER_BATCH_SIZE", &invalid)
	}

	if len(invalid) > 0 {
		return Config{}, &ValidationError{Names: unique(invalid)}
	}
	return cfg, nil
}

func address(lookup func(string) (string, bool)) string {
	if value, present := lookup("HTTP_ADDRESS"); present && value != "" {
		return value
	}
	if port, present := lookup("PORT"); present && port != "" {
		if strings.HasPrefix(port, ":") {
			return port
		}
		return ":" + port
	}
	return defaultAddress
}

func parseDatabaseURL(raw string) (string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	return raw, true
}

func validAddress(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}
	_, _, err := net.SplitHostPort(raw)
	return err == nil
}

func loadAuth(lookup func(string) (string, bool), invalid *[]string, defaults AuthConfig) AuthConfig {
	names := []string{authActiveKIDEnv, authSigningPrivateKeyEnv, authVerificationKeysetEnv, authIssuerEnv, authAccessTokenTTLEnv, authRefreshTokenTTLEnv, authSecretPepperEnv, authTOTPEncryptionKeyEnv}
	var authInvalid []string
	if !requireGroup(lookup, &authInvalid, names...) {
		*invalid = append(*invalid, authInvalid...)
		return defaults
	}

	activeKID := valueOr(lookup, authActiveKIDEnv, "")
	privateKey, ok := decodeEd25519PrivateKey(valueOr(lookup, authSigningPrivateKeyEnv, ""))
	if !ok {
		authInvalid = append(authInvalid, authSigningPrivateKeyEnv)
	}
	publicKeys, ok := decodeVerificationKeyset(valueOr(lookup, authVerificationKeysetEnv, ""))
	if !ok || publicKeys[activeKID] == nil {
		authInvalid = append(authInvalid, authVerificationKeysetEnv)
	}
	accessTokenTTL := parseDuration(valueOr(lookup, authAccessTokenTTLEnv, ""), authAccessTokenTTLEnv, &authInvalid)
	refreshTokenTTL := parseDuration(valueOr(lookup, authRefreshTokenTTLEnv, ""), authRefreshTokenTTLEnv, &authInvalid)
	if len(privateKey) == ed25519.PrivateKeySize && len(publicKeys[activeKID]) == ed25519.PublicKeySize && !bytes.Equal(privateKey.Public().(ed25519.PublicKey), publicKeys[activeKID]) {
		authInvalid = append(authInvalid, authVerificationKeysetEnv)
	}
	secretDigestPepper := valueOr(lookup, authSecretPepperEnv, "")
	if len(secretDigestPepper) < minSecretDigestPepperLength {
		authInvalid = append(authInvalid, authSecretPepperEnv)
	}
	totpEncryptionKey, ok := decodeRawBase64(valueOr(lookup, authTOTPEncryptionKeyEnv, ""), totpEncryptionKeyLength)
	if !ok {
		authInvalid = append(authInvalid, authTOTPEncryptionKeyEnv)
	}
	if len(authInvalid) > 0 {
		*invalid = append(*invalid, authInvalid...)
		return defaults
	}

	return AuthConfig{
		Enabled:                true,
		JWTIssuer:              valueOr(lookup, authIssuerEnv, ""),
		AccessTokenTTL:         accessTokenTTL,
		RefreshTokenTTL:        refreshTokenTTL,
		ActiveKID:              activeKID,
		SigningPrivateKey:      privateKey,
		VerificationPublicKeys: publicKeys,
		SecretDigestPepper:     []byte(secretDigestPepper),
		TOTPEncryptionKey:      totpEncryptionKey,
	}
}

func decodeRawBase64(value string, wantLength int) ([]byte, bool) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) != wantLength {
		return nil, false
	}
	return decoded, true
}

func decodeEd25519PrivateKey(value string) (ed25519.PrivateKey, bool) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) != ed25519.PrivateKeySize {
		return nil, false
	}
	return ed25519.PrivateKey(decoded), true
}

func decodeVerificationKeyset(value string) (map[string]ed25519.PublicKey, bool) {
	encoded := map[string]string{}
	if err := json.Unmarshal([]byte(value), &encoded); err != nil || len(encoded) == 0 {
		return nil, false
	}
	keys := make(map[string]ed25519.PublicKey, len(encoded))
	for kid, encodedKey := range encoded {
		decoded, err := base64.RawURLEncoding.DecodeString(encodedKey)
		if kid == "" || err != nil || len(decoded) != ed25519.PublicKeySize {
			return nil, false
		}
		keys[kid] = ed25519.PublicKey(decoded)
	}
	return keys, true
}

func loadOAuthProvider(lookup func(string) (string, bool), prefix string, invalid *[]string) OAuthProviderConfig {
	names := []string{prefix + "_CLIENT_ID", prefix + "_CLIENT_SECRET", prefix + "_REDIRECT_URL"}
	if !requireGroup(lookup, invalid, names...) {
		return OAuthProviderConfig{}
	}
	redirectURL := valueOr(lookup, names[2], "")
	if parsed, err := url.ParseRequestURI(redirectURL); err != nil || parsed.Scheme == "" || parsed.Host == "" {
		*invalid = append(*invalid, names[2])
	}
	return OAuthProviderConfig{
		Enabled:      true,
		ClientID:     valueOr(lookup, names[0], ""),
		ClientSecret: valueOr(lookup, names[1], ""),
		RedirectURL:  redirectURL,
	}
}

func loadWebAuthn(lookup func(string) (string, bool), invalid *[]string) WebAuthnConfig {
	names := []string{"WEBAUTHN_RP_ID", "WEBAUTHN_RP_NAME", "WEBAUTHN_RP_ORIGINS"}
	if !requireGroup(lookup, invalid, names...) {
		return WebAuthnConfig{}
	}
	origins := strings.Split(valueOr(lookup, "WEBAUTHN_RP_ORIGINS", ""), ",")
	for index := range origins {
		origins[index] = strings.TrimSpace(origins[index])
		parsed, err := url.ParseRequestURI(origins[index])
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			*invalid = append(*invalid, "WEBAUTHN_RP_ORIGINS")
			break
		}
	}
	return WebAuthnConfig{Enabled: true, RPID: valueOr(lookup, "WEBAUTHN_RP_ID", ""), RPName: valueOr(lookup, "WEBAUTHN_RP_NAME", ""), RPOrigins: origins}
}

func loadSMTP(lookup func(string) (string, bool), invalid *[]string) SMTPConfig {
	names := []string{"SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_FROM_ADDRESS"}
	if !requireGroup(lookup, invalid, names...) {
		return SMTPConfig{}
	}
	from := valueOr(lookup, "SMTP_FROM_ADDRESS", "")
	if _, err := mail.ParseAddress(from); err != nil {
		*invalid = append(*invalid, "SMTP_FROM_ADDRESS")
	}
	return SMTPConfig{
		Enabled:     true,
		Host:        valueOr(lookup, "SMTP_HOST", ""),
		Port:        parsePort(valueOr(lookup, "SMTP_PORT", ""), invalid),
		Username:    valueOr(lookup, "SMTP_USERNAME", ""),
		Password:    valueOr(lookup, "SMTP_PASSWORD", ""),
		FromAddress: from,
	}
}

func loadStripe(lookup func(string) (string, bool), invalid *[]string) StripeConfig {
	names := []string{"STRIPE_SECRET_KEY", "STRIPE_WEBHOOK_SECRET", "STRIPE_PUBLISHABLE_KEY"}
	if !requireGroup(lookup, invalid, names...) {
		return StripeConfig{}
	}
	return StripeConfig{Enabled: true, SecretKey: valueOr(lookup, names[0], ""), WebhookSecret: valueOr(lookup, names[1], ""), PublishableKey: valueOr(lookup, names[2], "")}
}

func loadAuthorization(lookup func(string) (string, bool), invalid *[]string) AuthorizationConfig {
	names := []string{"AUTHORIZATION_MODEL_PATH", "AUTHORIZATION_POLICY_PATH"}
	if !requireGroup(lookup, invalid, names...) {
		return AuthorizationConfig{}
	}
	return AuthorizationConfig{Enabled: true, ModelPath: valueOr(lookup, names[0], ""), PolicyPath: valueOr(lookup, names[1], "")}
}

func requireGroup(lookup func(string) (string, bool), invalid *[]string, names ...string) bool {
	present := 0
	for _, name := range names {
		if value, exists := lookup(name); exists && strings.TrimSpace(value) != "" {
			present++
		}
	}
	if present == 0 {
		return false
	}
	if present != len(names) {
		for _, name := range names {
			if value, exists := lookup(name); !exists || strings.TrimSpace(value) == "" {
				*invalid = append(*invalid, name)
			}
		}
		return false
	}
	return true
}

func parseDuration(raw, name string, invalid *[]string) time.Duration {
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		*invalid = append(*invalid, name)
		return 0
	}
	return parsed
}

func parsePositiveInt(raw, name string, invalid *[]string) int {
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		*invalid = append(*invalid, name)
		return 0
	}
	return parsed
}

func parsePort(raw string, invalid *[]string) int {
	port := parsePositiveInt(raw, "SMTP_PORT", invalid)
	if port > 65535 {
		*invalid = append(*invalid, "SMTP_PORT")
	}
	return port
}

func valueOr(lookup func(string) (string, bool), name, fallback string) string {
	if value, present := lookup(name); present && value != "" {
		return value
	}
	return fallback
}

func unique(names []string) []string {
	seen := make(map[string]struct{}, len(names))
	result := make([]string, 0, len(names))
	for _, name := range names {
		if _, exists := seen[name]; !exists {
			seen[name] = struct{}{}
			result = append(result, name)
		}
	}
	return result
}

var _ error = (*ValidationError)(nil)
