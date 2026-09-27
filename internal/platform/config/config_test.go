package config

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLoadUsesCurrentApplicationDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := Load(mapLookup(nil))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseURL != defaultDatabaseURL {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, defaultDatabaseURL)
	}
	if cfg.Address != defaultAddress {
		t.Errorf("Address = %q, want %q", cfg.Address, defaultAddress)
	}
	if !cfg.AutoMigrate {
		t.Error("AutoMigrate = false, want true")
	}
	if cfg.App.Name != defaultAppName || cfg.App.BaseURL != defaultAppBaseURL {
		t.Errorf("App = %#v, want default application identity", cfg.App)
	}
	if cfg.Billing.SuccessURL != defaultAppBaseURL+"/billing/success" || cfg.Billing.CancelURL != defaultAppBaseURL+"/billing/cancel" || cfg.Billing.PortalReturnURL != defaultAppBaseURL+"/billing" {
		t.Errorf("Billing = %#v, want URLs derived from default app base URL", cfg.Billing)
	}
	if cfg.Auth.TOTPIssuer != defaultAppName {
		t.Errorf("Auth.TOTPIssuer = %q, want %q", cfg.Auth.TOTPIssuer, defaultAppName)
	}
	if cfg.Auth.Enabled || cfg.OAuth.Google.Enabled || cfg.WebAuthn.Enabled || cfg.SMTP.Enabled || cfg.Stripe.Enabled || cfg.Worker.Enabled {
		t.Error("optional integrations must be disabled when their variables are absent")
	}
}

func TestLoadReadsServerSettings(t *testing.T) {
	t.Parallel()

	cfg, err := Load(mapLookup(map[string]string{
		"DATABASE_URL": "postgres://user:password@example.test:5432/app?sslmode=require",
		"PORT":         "9090",
		"AUTO_MIGRATE": "false",
	}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Address != ":9090" || cfg.AutoMigrate {
		t.Errorf("server config = %#v, want address :9090 and auto migrate false", cfg)
	}
}

func TestLoadRejectsPartialAuthConfiguration(t *testing.T) {
	t.Parallel()

	_, err := Load(mapLookup(map[string]string{authActiveKIDEnv: "2026-09"}))
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
	for _, name := range []string{authSigningPrivateKeyEnv, authVerificationKeysetEnv, authIssuerEnv, authAccessTokenTTLEnv, authRefreshTokenTTLEnv, authSecretPepperEnv} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("validation error %q does not name %s", err, name)
		}
	}
}

func TestLoadRejectsInvalidAndPartialConfigurationWithoutSecrets(t *testing.T) {
	t.Parallel()

	const secret = "must-not-appear-in-errors"
	_, err := Load(mapLookup(map[string]string{
		authSigningPrivateKeyEnv: secret,
		"SMTP_HOST":              "smtp.example.test",
		"STRIPE_SECRET_KEY":      secret,
		"WORKER_BATCH_SIZE":      "zero",
	}))
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Load() error type = %T, want *ValidationError", err)
	}
	message := err.Error()
	if strings.Contains(message, secret) {
		t.Fatalf("validation error leaked secret: %q", message)
	}
	for _, name := range []string{"SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_FROM_ADDRESS", "STRIPE_WEBHOOK_SECRET", "STRIPE_PUBLISHABLE_KEY", "WORKER_BATCH_SIZE"} {
		if !strings.Contains(message, name) {
			t.Errorf("validation error %q does not name %s", message, name)
		}
	}
}

func TestLoadEnablesAndValidatesOptionalGroups(t *testing.T) {
	t.Parallel()

	privateKey, publicKey := testEd25519KeyPair(t)
	cfg, err := Load(mapLookup(map[string]string{
		authActiveKIDEnv:             "2026-09",
		authSigningPrivateKeyEnv:     encodeBase64URL(privateKey),
		authVerificationKeysetEnv:    encodeKeyset(t, map[string]ed25519.PublicKey{"2026-09": publicKey, "2026-06": makeTestPublicKey(t)}),
		authIssuerEnv:                "example.test",
		authAccessTokenTTLEnv:        "20m",
		authRefreshTokenTTLEnv:       "720h",
		authSecretPepperEnv:          "at-least-sixteen-bytes-of-entropy",
		authTOTPEncryptionKeyEnv:     encodeBase64URL(make([]byte, totpEncryptionKeyLength)),
		"OAUTH_GOOGLE_CLIENT_ID":     "google-client",
		"OAUTH_GOOGLE_CLIENT_SECRET": "google-secret",
		"OAUTH_GOOGLE_REDIRECT_URL":  "https://app.example.test/api/auth/google/callback",
		"OAUTH_GITHUB_CLIENT_ID":     "github-client",
		"OAUTH_GITHUB_CLIENT_SECRET": "github-secret",
		"OAUTH_GITHUB_REDIRECT_URL":  "https://app.example.test/api/auth/github/callback",
		"WEBAUTHN_RP_ID":             "app.example.test",
		"WEBAUTHN_RP_NAME":           "Example App",
		"WEBAUTHN_RP_ORIGINS":        "https://app.example.test,https://admin.example.test",
		"SMTP_HOST":                  "smtp.example.test",
		"SMTP_PORT":                  "587",
		"SMTP_USERNAME":              "mailer",
		"SMTP_PASSWORD":              "smtp-secret",
		"SMTP_FROM_ADDRESS":          "Example <noreply@example.test>",
		"STRIPE_SECRET_KEY":          "sk_test_example",
		"STRIPE_WEBHOOK_SECRET":      "whsec_example",
		"STRIPE_PUBLISHABLE_KEY":     "pk_test_example",
		"APP_NAME":                   "Example App",
		"APP_BASE_URL":               "https://app.example.test",
		"BILLING_PRICE_CATALOG":      `{"pro_monthly":{"stripePriceId":"price_123","featureKey":"pro","mode":"subscription"}}`,
		"BILLING_SUCCESS_URL":        "https://app.example.test/checkout/success",
		"BILLING_CANCEL_URL":         "https://app.example.test/checkout/cancel",
		"BILLING_PORTAL_RETURN_URL":  "https://app.example.test/settings/billing",
		authTOTPIssuerEnv:            "Example Authenticator",
		"WORKER_ENABLED":             "true",
		"WORKER_POLL_INTERVAL":       "2s",
		"WORKER_BATCH_SIZE":          "25",
	}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Auth.Enabled || cfg.Auth.AccessTokenTTL != 20*time.Minute || cfg.Auth.RefreshTokenTTL != 720*time.Hour {
		t.Errorf("Auth = %#v, want enabled with configured TTLs", cfg.Auth)
	}
	if cfg.Auth.ActiveKID != "2026-09" || !cfg.Auth.SigningPrivateKey.Public().(ed25519.PublicKey).Equal(cfg.Auth.VerificationPublicKeys["2026-09"]) || len(cfg.Auth.VerificationPublicKeys) != 2 {
		t.Errorf("Auth keyset = %#v, want active kid and both rotation keys", cfg.Auth)
	}
	if !cfg.OAuth.Google.Enabled || !cfg.OAuth.GitHub.Enabled || !cfg.WebAuthn.Enabled || !cfg.SMTP.Enabled || !cfg.Stripe.Enabled || !cfg.Worker.Enabled {
		t.Errorf("one or more complete integrations were not enabled: %#v", cfg)
	}
	if cfg.Worker.PollInterval != 2*time.Second || cfg.Worker.BatchSize != 25 {
		t.Errorf("Worker = %#v, want poll interval 2s and batch size 25", cfg.Worker)
	}
	if cfg.App.Name != "Example App" || cfg.App.BaseURL != "https://app.example.test" {
		t.Errorf("App = %#v, want configured application identity", cfg.App)
	}
	if cfg.Auth.TOTPIssuer != "Example Authenticator" {
		t.Errorf("Auth.TOTPIssuer = %q, want configured issuer", cfg.Auth.TOTPIssuer)
	}
	if cfg.Billing.PriceCatalog == "" || cfg.Billing.SuccessURL != "https://app.example.test/checkout/success" || cfg.Billing.CancelURL != "https://app.example.test/checkout/cancel" || cfg.Billing.PortalReturnURL != "https://app.example.test/settings/billing" {
		t.Errorf("Billing = %#v, want configured billing values", cfg.Billing)
	}
}

func TestLoadRejectsInvalidEd25519KeyMaterial(t *testing.T) {
	t.Parallel()

	privateKey, publicKey := testEd25519KeyPair(t)
	for name, value := range map[string]string{
		authSigningPrivateKeyEnv:  "not-base64",
		authVerificationKeysetEnv: `{"2026-09":"not-base64"}`,
	} {
		values := validAuthEnvironment(t, privateKey, publicKey)
		values[name] = value
		_, err := Load(mapLookup(values))
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("Load() error = %v, want validation error naming %s", err, name)
		}
	}
}

func TestLoadRejectsKeysetWithoutMatchingActiveKey(t *testing.T) {
	t.Parallel()

	privateKey, publicKey := testEd25519KeyPair(t)
	values := validAuthEnvironment(t, privateKey, publicKey)
	values[authVerificationKeysetEnv] = encodeKeyset(t, map[string]ed25519.PublicKey{"retired": publicKey})
	_, err := Load(mapLookup(values))
	if err == nil || !strings.Contains(err.Error(), authVerificationKeysetEnv) {
		t.Errorf("Load() error = %v, want validation error naming %s", err, authVerificationKeysetEnv)
	}
}

func TestLoadRejectsActiveSigningKeyThatDoesNotMatchKeyset(t *testing.T) {
	t.Parallel()

	privateKey, _ := testEd25519KeyPair(t)
	values := validAuthEnvironment(t, privateKey, makeTestPublicKey(t))
	_, err := Load(mapLookup(values))
	if err == nil || !strings.Contains(err.Error(), authVerificationKeysetEnv) {
		t.Errorf("Load() error = %v, want validation error naming %s", err, authVerificationKeysetEnv)
	}
}

func TestLoadRejectsShortSecretPepper(t *testing.T) {
	t.Parallel()

	privateKey, publicKey := testEd25519KeyPair(t)
	values := validAuthEnvironment(t, privateKey, publicKey)
	values[authSecretPepperEnv] = "too-short"
	_, err := Load(mapLookup(values))
	if err == nil || !strings.Contains(err.Error(), authSecretPepperEnv) {
		t.Errorf("Load() error = %v, want validation error naming %s", err, authSecretPepperEnv)
	}
}

func TestLoadDerivesIndependentSecretDigestPepper(t *testing.T) {
	t.Parallel()

	privateKey, publicKey := testEd25519KeyPair(t)
	values := validAuthEnvironment(t, privateKey, publicKey)
	cfg, err := Load(mapLookup(values))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if string(cfg.Auth.SecretDigestPepper) != values[authSecretPepperEnv] {
		t.Errorf("SecretDigestPepper = %q, want the configured pepper value", cfg.Auth.SecretDigestPepper)
	}
	if bytes.Equal(cfg.Auth.SecretDigestPepper, cfg.Auth.SigningPrivateKey) {
		t.Error("SecretDigestPepper must not equal or be derived from SigningPrivateKey")
	}
}

func TestLoadDerivesBillingURLsAndTOTPIssuerFromApplicationConfig(t *testing.T) {
	t.Parallel()

	cfg, err := Load(mapLookup(map[string]string{
		"APP_NAME":     "Acme",
		"APP_BASE_URL": "https://app.acme.test",
	}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Auth.TOTPIssuer != "Acme" {
		t.Errorf("Auth.TOTPIssuer = %q, want Acme", cfg.Auth.TOTPIssuer)
	}
	if cfg.Billing.SuccessURL != "https://app.acme.test/billing/success" || cfg.Billing.CancelURL != "https://app.acme.test/billing/cancel" || cfg.Billing.PortalReturnURL != "https://app.acme.test/billing" {
		t.Errorf("Billing = %#v, want URLs derived from APP_BASE_URL", cfg.Billing)
	}
}

func TestLoadRejectsInvalidApplicationAndBillingConfiguration(t *testing.T) {
	t.Parallel()

	_, err := Load(mapLookup(map[string]string{
		"APP_BASE_URL":              "not-a-url",
		"BILLING_PRICE_CATALOG":     "not-json",
		"BILLING_SUCCESS_URL":       "/relative-success",
		"BILLING_CANCEL_URL":        "not-a-url",
		"BILLING_PORTAL_RETURN_URL": "relative",
	}))
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
	for _, name := range []string{"APP_BASE_URL", "BILLING_PRICE_CATALOG", "BILLING_SUCCESS_URL", "BILLING_CANCEL_URL", "BILLING_PORTAL_RETURN_URL"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("validation error %q does not name %s", err, name)
		}
	}
}

func TestLoadRejectsInvalidOptionalValues(t *testing.T) {
	t.Parallel()

	_, err := Load(mapLookup(map[string]string{
		"OAUTH_GOOGLE_CLIENT_ID":     "client",
		"OAUTH_GOOGLE_CLIENT_SECRET": "secret",
		"OAUTH_GOOGLE_REDIRECT_URL":  "not a url",
		"WEBAUTHN_RP_ID":             "example.test",
		"WEBAUTHN_RP_NAME":           "Example",
		"WEBAUTHN_RP_ORIGINS":        "not-a-url",
		"SMTP_HOST":                  "smtp.example.test",
		"SMTP_PORT":                  "70000",
		"SMTP_USERNAME":              "mailer",
		"SMTP_PASSWORD":              "secret",
		"SMTP_FROM_ADDRESS":          "not-an-address",
	}))
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
	for _, name := range []string{"OAUTH_GOOGLE_REDIRECT_URL", "WEBAUTHN_RP_ORIGINS", "SMTP_PORT", "SMTP_FROM_ADDRESS"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("validation error %q does not name %s", err, name)
		}
	}
}

func validAuthEnvironment(t *testing.T, privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey) map[string]string {
	t.Helper()
	return map[string]string{
		authActiveKIDEnv:          "2026-09",
		authSigningPrivateKeyEnv:  encodeBase64URL(privateKey),
		authVerificationKeysetEnv: encodeKeyset(t, map[string]ed25519.PublicKey{"2026-09": publicKey}),
		authIssuerEnv:             "example.test",
		authAccessTokenTTLEnv:     "15m",
		authRefreshTokenTTLEnv:    "720h",
		authSecretPepperEnv:       "at-least-sixteen-bytes-of-entropy",
		authTOTPEncryptionKeyEnv:  encodeBase64URL(make([]byte, totpEncryptionKeyLength)),
	}
}

func testEd25519KeyPair(t *testing.T) (ed25519.PrivateKey, ed25519.PublicKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	return privateKey, publicKey
}

func makeTestPublicKey(t *testing.T) ed25519.PublicKey {
	t.Helper()
	_, publicKey := testEd25519KeyPair(t)
	return publicKey
}

func encodeKeyset(t *testing.T, keys map[string]ed25519.PublicKey) string {
	t.Helper()
	encoded := make(map[string]string, len(keys))
	for kid, key := range keys {
		encoded[kid] = encodeBase64URL(key)
	}
	value, err := json.Marshal(encoded)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return string(value)
}

func encodeBase64URL(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}
