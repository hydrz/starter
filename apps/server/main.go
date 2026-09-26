package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	databaseMigrations "github.com/hydrz/starter/db"
	"github.com/hydrz/starter/internal/announcement"
	"github.com/hydrz/starter/internal/auth"
	"github.com/hydrz/starter/internal/authorization"
	"github.com/hydrz/starter/internal/billing"
	"github.com/hydrz/starter/internal/delivery"
	"github.com/hydrz/starter/internal/notification"
	"github.com/hydrz/starter/internal/organization"
	"github.com/hydrz/starter/internal/platform/config"
	"github.com/hydrz/starter/internal/platform/database"
	apphttp "github.com/hydrz/starter/internal/platform/httpserver"
	"github.com/hydrz/starter/internal/store"
)

const shutdownTimeout = 10 * time.Second

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	_ = godotenv.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		logger.Error("configuration invalid", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			if err := healthcheck(cfg.Address); err != nil {
				logger.Error("healthcheck failed", "error", err)
				os.Exit(1)
			}
			return
		case "migrate":
			if err := databaseMigrations.Migrate(ctx, cfg.DatabaseURL); err != nil {
				logger.Error("database migration failed", "error", err)
				os.Exit(1)
			}
			logger.Info("database migrations completed")
			return
		case "status":
			if err := databaseMigrations.Status(ctx, cfg.DatabaseURL); err != nil {
				logger.Error("database migration status failed", "error", err)
				os.Exit(1)
			}
			return
		}
	}

	if cfg.AutoMigrate {
		if err := databaseMigrations.Migrate(ctx, cfg.DatabaseURL); err != nil {
			logger.Error("automatic database migration failed", "error", err)
			os.Exit(1)
		}
		logger.Info("database migrations applied automatically")
	}

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := store.New(pool)
	announcementService := announcement.NewService(announcement.NewPostgresRepository(queries))

	adapter := authorization.NewDatabaseAdapter(queries)
	enforcer, err := authorization.NewEnforcer(adapter)
	if err != nil {
		logger.Error("authorization enforcer initialization failed", "error", err)
		os.Exit(1)
	}
	authzService := authorization.NewService(enforcer)

	orgService, err := organization.NewService(organization.Dependencies{
		Repository: organization.NewPostgresRepository(queries),
		Transactor: database.NewTransactor(pool),
		Enforcer:   enforcer,
	})
	if err != nil {
		logger.Error("organization service initialization failed", "error", err)
		os.Exit(1)
	}

	var identityService *auth.Service
	if cfg.Auth.Enabled {
		identityService, err = buildIdentityService(cfg, queries, pool, orgService)
		if err != nil {
			logger.Error("identity service initialization failed", "error", err)
			os.Exit(1)
		}
	}

	var billingService *billing.Service
	if cfg.Stripe.Enabled {
		billingService, err = buildBillingService(cfg.Stripe, queries, pool)
		if err != nil {
			logger.Error("billing service initialization failed", "error", err)
			os.Exit(1)
		}
	}

	var outboxWorker *delivery.Worker
	if cfg.Worker.Enabled || cfg.SMTP.Enabled {
		outboxWorker, err = buildOutboxWorker(cfg, store.New(pool), logger)
		if err != nil {
			logger.Error("outbox worker initialization failed", "error", err)
			os.Exit(1)
		}
		if err := outboxWorker.Start(ctx); err != nil {
			logger.Error("outbox worker start failed", "error", err)
			os.Exit(1)
		}
	}

	handler, err := apphttp.NewHandler(announcementService, pool, identityService, orgService, authzService, billingService)
	if err != nil {
		logger.Error("http handler initialization failed", "error", err)
		os.Exit(1)
	}
	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http server started", "address", server.Addr, "version", version, "commit", commit, "build_date", buildDate)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("shutdown requested")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if outboxWorker != nil {
		if err := outboxWorker.Stop(shutdownCtx); err != nil {
			logger.Error("outbox worker graceful shutdown failed", "error", err)
		} else {
			logger.Info("outbox worker stopped")
		}
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("http server stopped")
}

func healthcheck(addr string) error {
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	url := "http://" + addr + "/api/readyz"
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("request readiness endpoint: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("readiness endpoint returned %s", response.Status)
	}
	return nil
}

// buildIdentityService wires internal/auth.Service from the process
// AuthConfig and a pgxpool-backed store. SecretDigestPepper is independent,
// high-entropy configuration (AUTH_SECRET_PEPPER) rather than being derived
// from the JWT signing key, so rotating one never invalidates the other.
func buildIdentityService(cfg config.Config, queries *store.Queries, pool *pgxpool.Pool, orgCreator auth.PersonalOrgCreator) (*auth.Service, error) {
	authConfig := cfg.Auth
	issuer, err := auth.NewIssuer(authConfig.ActiveKID, authConfig.SigningPrivateKey, authConfig.JWTIssuer, auth.SystemClock{})
	if err != nil {
		return nil, fmt.Errorf("create token issuer: %w", err)
	}
	verifier, err := auth.NewVerifier(authConfig.VerificationPublicKeys, authConfig.JWTIssuer, auth.SystemClock{})
	if err != nil {
		return nil, fmt.Errorf("create token verifier: %w", err)
	}
	digester, err := auth.NewSecretDigester(1, authConfig.SecretDigestPepper)
	if err != nil {
		return nil, fmt.Errorf("create secret digester: %w", err)
	}
	totpCipher, err := auth.NewTOTPCipher(authConfig.TOTPEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create totp cipher: %w", err)
	}

	deps := auth.Dependencies{
		Users:              auth.NewPostgresUserRepository(queries),
		RefreshTokens:      auth.NewPostgresRefreshTokenRepository(queries),
		OneTimeTokens:      auth.NewPostgresOneTimeTokenRepository(queries),
		APIKeys:            auth.NewPostgresAPIKeyRepository(queries),
		Outbox:             auth.NewPostgresOutboxWriter(queries),
		Transactor:         database.NewTransactor(pool),
		Issuer:             issuer,
		Verifier:           verifier,
		Digester:           digester,
		Clock:              auth.SystemClock{},
		PersonalOrgCreator: orgCreator,
		AccessTokenTTL:     authConfig.AccessTokenTTL,
		RefreshTokenTTL:    authConfig.RefreshTokenTTL,

		EmailOTP:      auth.NewPostgresEmailOTPRepository(queries),
		TOTPFactors:   auth.NewPostgresTOTPFactorRepository(queries),
		RecoveryCodes: auth.NewPostgresTOTPRecoveryCodeRepository(queries),
		MFAChallenges: auth.NewPostgresMFAChallengeRepository(queries),
		TOTPCipher:    totpCipher,
		TOTPIssuer:    envOrDefault("AUTH_TOTP_ISSUER", "Starter"),

		OAuthAccounts: auth.NewPostgresOAuthAccountRepository(queries),
		OAuthStates:   auth.NewPostgresOAuthStateRepository(queries),
		OAuthClients:  buildOAuthClients(cfg.OAuth),

		WebAuthnCredentials: auth.NewPostgresWebAuthnCredentialRepository(queries),
		WebAuthnChallenges:  auth.NewPostgresWebAuthnChallengeRepository(queries),
	}

	if cfg.WebAuthn.Enabled {
		ceremonies, err := auth.NewWebAuthnCeremonies(cfg.WebAuthn.RPID, cfg.WebAuthn.RPName, cfg.WebAuthn.RPOrigins)
		if err != nil {
			return nil, fmt.Errorf("create webauthn ceremonies: %w", err)
		}
		deps.WebAuthn = ceremonies
	}

	return auth.NewService(deps)
}

// buildOAuthClients wires internal/auth.OAuthClient implementations from
// config.OAuthConfig, one per enabled provider. A provider absent from the
// map cannot be used to begin a flow (Service.beginOAuth returns
// ErrOAuthProviderUnknown), so OAuth sign-in stays entirely optional.
func buildOAuthClients(oauthConfig config.OAuthConfig) map[string]auth.OAuthClient {
	clients := map[string]auth.OAuthClient{}
	if oauthConfig.Google.Enabled {
		clients["google"] = auth.NewGoogleOAuthClient(oauthConfig.Google.ClientID, oauthConfig.Google.ClientSecret, oauthConfig.Google.RedirectURL)
	}
	if oauthConfig.GitHub.Enabled {
		clients["github"] = auth.NewGitHubOAuthClient(oauthConfig.GitHub.ClientID, oauthConfig.GitHub.ClientSecret, oauthConfig.GitHub.RedirectURL)
	}
	return clients
}

// buildBillingService wires internal/billing.Service from the process
// StripeConfig and a pgxpool-backed store. It never changes StripeConfig's
// shape; it only consumes cfg.Stripe.{SecretKey,WebhookSecret}. The price
// catalog and redirect URLs are billing-owned configuration (never
// client-supplied): BILLING_PRICE_CATALOG is an optional JSON map from
// server-controlled price key to Stripe price ID/feature key/mode (see
// billing.ParseCatalogJSON); BILLING_SUCCESS_URL, BILLING_CANCEL_URL, and
// BILLING_PORTAL_RETURN_URL fall back to same-origin defaults when unset.
func buildBillingService(stripeConfig config.StripeConfig, queries *store.Queries, pool *pgxpool.Pool) (*billing.Service, error) {
	gateway, err := billing.NewLiveGateway(stripeConfig.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("create stripe gateway: %w", err)
	}

	catalog, err := billing.ParseCatalogJSON(os.Getenv("BILLING_PRICE_CATALOG"))
	if err != nil {
		return nil, err
	}

	return billing.NewService(billing.Dependencies{
		BillingAccounts:  billing.NewPostgresBillingAccountRepository(queries),
		CheckoutSessions: billing.NewPostgresCheckoutSessionRepository(queries),
		Subscriptions:    billing.NewPostgresSubscriptionRepository(queries),
		OneTimePurchases: billing.NewPostgresOneTimePurchaseRepository(queries),
		WebhookEvents:    billing.NewPostgresWebhookEventRepository(queries),
		Entitlements:     billing.NewPostgresEntitlementRepository(queries),
		Transactor:       database.NewTransactor(pool),
		Outbox:           billing.NewPostgresOutboxWriter(queries),
		Gateway:          gateway,
		Catalog:          catalog,
		Clock:            billing.SystemClock{},
		WebhookSecret:    stripeConfig.WebhookSecret,
		SuccessURL:       envOrDefault("BILLING_SUCCESS_URL", "http://localhost:8080/billing/success"),
		CancelURL:        envOrDefault("BILLING_CANCEL_URL", "http://localhost:8080/billing/cancel"),
		PortalReturnURL:  envOrDefault("BILLING_PORTAL_RETURN_URL", "http://localhost:8080/billing"),
	})
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

// buildOutboxWorker wires internal/delivery.Worker with an embedded template renderer,
// an SMTP (or noop) delivery channel, and the notification routing service.
func buildOutboxWorker(cfg config.Config, queries *store.Queries, logger *slog.Logger) (*delivery.Worker, error) {
	renderer, err := delivery.NewTemplateRenderer("Starter")
	if err != nil {
		return nil, fmt.Errorf("create template renderer: %w", err)
	}

	channels := make(map[string]delivery.Channel)
	if cfg.SMTP.Enabled {
		channels[notification.ChannelSMTP] = delivery.NewSMTPChannel(cfg.SMTP)
	} else {
		channels[notification.ChannelSMTP] = delivery.NewNoopChannel("smtp-noop", logger)
	}

	notificationRepo := notification.NewPostgresRepository(queries)
	notificationService, err := notification.NewService(notification.Dependencies{
		Repository: notificationRepo,
		Renderer:   renderer,
		Channels:   channels,
		Clock:      notification.SystemClock{},
		Logger:     logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create notification service: %w", err)
	}

	claimer := delivery.NewPostgresClaimer(queries)
	worker := delivery.NewWorker(claimer, notificationService, delivery.WorkerOptions{
		PollInterval: cfg.Worker.PollInterval,
		BatchSize:    cfg.Worker.BatchSize,
		Logger:       logger,
	})

	return worker, nil
}
