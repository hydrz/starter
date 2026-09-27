package auth

import (
	"context"
	"fmt"

	"github.com/go-chi/chi/v5"

	"github.com/hydrz/starter/internal/api/authapi"
	"github.com/hydrz/starter/internal/delivery"
	"github.com/hydrz/starter/internal/platform/config"
	platformmodule "github.com/hydrz/starter/internal/platform/module"
)

const (
	serviceName = "auth"

	// organizationServiceName is the registry key under which the
	// organization module publishes its service. auth needs it to provision
	// each new user's personal organization at signup.
	organizationServiceName = "organization"
)

type Module struct {
	service *Service
}

var _ platformmodule.Module = (*Module)(nil)

func NewModule() *Module { return &Module{} }

func (*Module) Name() string { return serviceName }

func (*Module) Enabled(cfg config.Config) bool { return cfg.Auth.Enabled }

func (m *Module) Init(_ context.Context, deps platformmodule.Deps) error {
	orgService, ok := deps.Services.Get(organizationServiceName)
	if !ok {
		return fmt.Errorf("auth: organization service must be registered first")
	}
	orgCreator, ok := orgService.(PersonalOrgCreator)
	if !ok {
		return fmt.Errorf("auth: registered organization service does not implement PersonalOrgCreator")
	}
	service, err := newService(deps, orgCreator)
	if err != nil {
		return err
	}
	m.service = service
	deps.Services.Set(serviceName, service)
	return nil
}

// RouteMiddlewares exposes the auth service's optional-principal middleware to
// the composition root without making platform packages depend on auth.
func (m *Module) RouteMiddlewares() platformmodule.Middlewares {
	if m.service == nil {
		return platformmodule.Middlewares{}
	}
	return platformmodule.Middlewares{Authenticate: m.service.AuthenticationMiddleware}
}

func (m *Module) Routes(router chi.Router, mw platformmodule.Middlewares) {
	if m.service == nil {
		return
	}
	server, err := authapi.NewServer(NewHTTPHandler(m.service), authapi.WithErrorHandler(mw.APIErrorHandler))
	if err != nil {
		panic(err)
	}
	router.Handle("/api/auth/*", m.service.AuthenticationMiddleware(server))
}

func (*Module) OutboxHandlers() map[string]delivery.Handler { return nil }

func (*Module) Shutdown(context.Context) error { return nil }

func newService(deps platformmodule.Deps, orgCreator PersonalOrgCreator) (*Service, error) {
	cfg := deps.Config
	issuer, err := NewIssuer(cfg.Auth.ActiveKID, cfg.Auth.SigningPrivateKey, cfg.Auth.JWTIssuer, SystemClock{})
	if err != nil {
		return nil, fmt.Errorf("create token issuer: %w", err)
	}
	verifier, err := NewVerifier(cfg.Auth.VerificationPublicKeys, cfg.Auth.JWTIssuer, SystemClock{})
	if err != nil {
		return nil, fmt.Errorf("create token verifier: %w", err)
	}
	digester, err := NewSecretDigester(1, cfg.Auth.SecretDigestPepper)
	if err != nil {
		return nil, fmt.Errorf("create secret digester: %w", err)
	}
	totpCipher, err := NewTOTPCipher(cfg.Auth.TOTPEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create totp cipher: %w", err)
	}

	serviceDeps := Dependencies{
		Users:               NewPostgresUserRepository(deps.Queries),
		RefreshTokens:       NewPostgresRefreshTokenRepository(deps.Queries),
		OneTimeTokens:       NewPostgresOneTimeTokenRepository(deps.Queries),
		APIKeys:             NewPostgresAPIKeyRepository(deps.Queries),
		Outbox:              NewPostgresOutboxWriter(deps.Queries),
		Transactor:          deps.Transactor,
		Issuer:              issuer,
		Verifier:            verifier,
		Digester:            digester,
		Clock:               SystemClock{},
		PersonalOrgCreator:  orgCreator,
		AccessTokenTTL:      cfg.Auth.AccessTokenTTL,
		RefreshTokenTTL:     cfg.Auth.RefreshTokenTTL,
		EmailOTP:            NewPostgresEmailOTPRepository(deps.Queries),
		TOTPFactors:         NewPostgresTOTPFactorRepository(deps.Queries),
		RecoveryCodes:       NewPostgresTOTPRecoveryCodeRepository(deps.Queries),
		MFAChallenges:       NewPostgresMFAChallengeRepository(deps.Queries),
		TOTPCipher:          totpCipher,
		TOTPIssuer:          cfg.Auth.TOTPIssuer,
		OAuthAccounts:       NewPostgresOAuthAccountRepository(deps.Queries),
		OAuthStates:         NewPostgresOAuthStateRepository(deps.Queries),
		OAuthClients:        oauthClients(cfg.OAuth),
		WebAuthnCredentials: NewPostgresWebAuthnCredentialRepository(deps.Queries),
		WebAuthnChallenges:  NewPostgresWebAuthnChallengeRepository(deps.Queries),
	}
	if cfg.WebAuthn.Enabled {
		ceremonies, err := NewWebAuthnCeremonies(cfg.WebAuthn.RPID, cfg.WebAuthn.RPName, cfg.WebAuthn.RPOrigins)
		if err != nil {
			return nil, fmt.Errorf("create webauthn ceremonies: %w", err)
		}
		serviceDeps.WebAuthn = ceremonies
	}
	return NewService(serviceDeps)
}

func oauthClients(oauthConfig config.OAuthConfig) map[string]OAuthClient {
	clients := map[string]OAuthClient{}
	if oauthConfig.Google.Enabled {
		clients["google"] = NewGoogleOAuthClient(oauthConfig.Google.ClientID, oauthConfig.Google.ClientSecret, oauthConfig.Google.RedirectURL)
	}
	if oauthConfig.GitHub.Enabled {
		clients["github"] = NewGitHubOAuthClient(oauthConfig.GitHub.ClientID, oauthConfig.GitHub.ClientSecret, oauthConfig.GitHub.RedirectURL)
	}
	return clients
}
