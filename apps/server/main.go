package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	databaseMigrations "github.com/hydrz/starter/db"
	"github.com/hydrz/starter/internal/announcement"
	"github.com/hydrz/starter/internal/platform/database"
	apphttp "github.com/hydrz/starter/internal/platform/httpserver"
	"github.com/hydrz/starter/internal/store"
)

const (
	defaultAddress     = ":8080"
	defaultDatabaseURL = "postgres://starter:starter@127.0.0.1:5432/starter?sslmode=disable"
	shutdownTimeout    = 10 * time.Second
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			if err := healthcheck(); err != nil {
				logger.Error("healthcheck failed", "error", err)
				os.Exit(1)
			}
			return
		case "migrate":
			if err := databaseMigrations.Migrate(ctx, databaseURL()); err != nil {
				logger.Error("database migration failed", "error", err)
				os.Exit(1)
			}
			logger.Info("database migrations completed")
			return
		}
	}

	pool, err := database.Open(ctx, databaseURL())
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	announcementService := announcement.NewService(announcement.NewPostgresRepository(store.New(pool)))
	handler, err := apphttp.NewHandler(announcementService, pool)
	if err != nil {
		logger.Error("http handler initialization failed", "error", err)
		os.Exit(1)
	}
	server := &http.Server{
		Addr:              address(),
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

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("http server stopped")
}

func healthcheck() error {
	url := os.Getenv("HEALTHCHECK_URL")
	if url == "" {
		url = "http://127.0.0.1:8080/api/readyz"
	}
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

func databaseURL() string {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}
	return defaultDatabaseURL
}

func address() string {
	if value := os.Getenv("HTTP_ADDRESS"); value != "" {
		return value
	}
	return defaultAddress
}
