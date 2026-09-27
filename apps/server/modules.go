package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hydrz/starter/internal/announcement"
	"github.com/hydrz/starter/internal/auth"
	"github.com/hydrz/starter/internal/authorization"
	"github.com/hydrz/starter/internal/billing"
	"github.com/hydrz/starter/internal/notification"
	"github.com/hydrz/starter/internal/organization"
	"github.com/hydrz/starter/internal/platform/config"
	"github.com/hydrz/starter/internal/platform/database"
	platformmodule "github.com/hydrz/starter/internal/platform/module"
	"github.com/hydrz/starter/internal/store"
)

func initializeModules(ctx context.Context, cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) ([]platformmodule.Module, platformmodule.Deps, error) {
	queries := store.New(pool)
	adapter := authorization.NewDatabaseAdapter(queries)
	enforcer, err := authorization.NewEnforcer(adapter)
	if err != nil {
		return nil, platformmodule.Deps{}, fmt.Errorf("initialize authorization enforcer: %w", err)
	}
	authorizer := authorization.NewService(enforcer)
	platformmodule.SetPrincipalResolver(auth.PrincipalFromContext)
	deps := platformmodule.Deps{Pool: pool, Queries: queries, Transactor: database.NewTransactor(pool), Logger: logger, Clock: platformmodule.SystemClock(), Config: cfg, Authorizer: authorizer, Enforcer: enforcer, Services: platformmodule.NewServices()}
	// Organization must precede auth: auth signup uses the registered
	// PersonalOrgCreator to provision each user's initial organization.
	all := []platformmodule.Module{platformmodule.NewMiddlewareModule(authorizer), organization.NewModule(), auth.NewModule(), announcement.NewModule(), billing.NewModule(), notification.NewModule()}
	modules := make([]platformmodule.Module, 0, len(all))
	for _, m := range all {
		if !m.Enabled(cfg) {
			continue
		}
		if err := m.Init(ctx, deps); err != nil {
			shutdownModules(cfg.ShutdownTimeout, modules, logger)
			return nil, deps, fmt.Errorf("initialize %s module: %w", m.Name(), err)
		}
		modules = append(modules, m)
	}
	return modules, deps, nil
}

func shutdownModules(timeout time.Duration, modules []platformmodule.Module, logger *slog.Logger) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for i := len(modules) - 1; i >= 0; i-- {
		if err := modules[i].Shutdown(shutdownCtx); err != nil {
			logger.Error("module graceful shutdown failed", "module", modules[i].Name(), "error", err)
		}
	}
}
