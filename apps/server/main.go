package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	databaseMigrations "github.com/hydrz/starter/db"
	"github.com/hydrz/starter/internal/delivery"
	"github.com/hydrz/starter/internal/platform/config"
	"github.com/hydrz/starter/internal/platform/database"
	apphttp "github.com/hydrz/starter/internal/platform/httpserver"
	platformmodule "github.com/hydrz/starter/internal/platform/module"
)

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
		fatal(logger, "configuration invalid", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if command(ctx, cfg, logger) {
		return
	}

	if cfg.AutoMigrate {
		if err := databaseMigrations.Migrate(ctx, cfg.DatabaseURL); err != nil {
			fatal(logger, "automatic database migration failed", err)
		}
		logger.Info("database migrations applied automatically")
	}
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal(logger, "database connection failed", err)
	}
	defer pool.Close()

	modules, deps, err := initializeModules(ctx, cfg, pool, logger)
	if err != nil {
		fatal(logger, "module initialization failed", err)
	}
	worker, err := startWorker(ctx, cfg, deps, modules)
	if err != nil {
		shutdownModules(cfg.ShutdownTimeout, modules, logger)
		fatal(logger, "outbox worker initialization failed", err)
	}

	handler, err := apphttp.NewHandler(modules, pool, cfg.TrustedProxies)
	if err != nil {
		shutdownModules(cfg.ShutdownTimeout, modules, logger)
		fatal(logger, "http handler initialization failed", err)
	}
	server := &http.Server{Addr: cfg.Address, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	serve(ctx, server, cfg.ShutdownTimeout, worker, modules, logger)
}

func startWorker(ctx context.Context, cfg config.Config, deps platformmodule.Deps, modules []platformmodule.Module) (*delivery.Worker, error) {
	if !(cfg.Worker.Enabled || cfg.SMTP.Enabled) {
		return nil, nil
	}
	dispatcher, err := platformmodule.NewDispatcher(modules)
	if err != nil {
		return nil, err
	}
	worker := delivery.NewWorker(delivery.NewPostgresStore(deps.Queries), dispatcher, delivery.WorkerOptions{
		PollInterval: cfg.Worker.PollInterval,
		BatchSize:    cfg.Worker.BatchSize,
		MaxAttempts:  cfg.Worker.MaxAttempts,
		Logger:       deps.Logger,
	})
	if err := worker.Start(ctx); err != nil {
		return nil, err
	}
	return worker, nil
}

func serve(ctx context.Context, server *http.Server, timeout time.Duration, worker *delivery.Worker, modules []platformmodule.Module, logger *slog.Logger) {
	errorsCh := make(chan error, 1)
	go func() {
		logger.Info("http server started", "address", server.Addr, "version", version, "commit", commit, "build_date", buildDate)
		errorsCh <- server.ListenAndServe()
	}()
	select {
	case err := <-errorsCh:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server stopped unexpectedly", "error", err)
		}
	case <-ctx.Done():
		logger.Info("shutdown requested")
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server graceful shutdown failed", "error", err)
	}
	if worker != nil {
		if err := worker.Stop(shutdownCtx); err != nil {
			logger.Error("outbox worker graceful shutdown failed", "error", err)
		}
	}
	shutdownModules(timeout, modules, logger)
}
