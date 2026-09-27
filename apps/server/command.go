package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	databaseMigrations "github.com/hydrz/starter/db"
	"github.com/hydrz/starter/internal/platform/config"
)

func command(ctx context.Context, cfg config.Config, logger *slog.Logger) bool {
	if len(os.Args) <= 1 {
		return false
	}
	var err error
	switch os.Args[1] {
	case "healthcheck":
		err = healthcheck(cfg.Address)
	case "migrate":
		err = databaseMigrations.Migrate(ctx, cfg.DatabaseURL)
	case "status":
		err = databaseMigrations.Status(ctx, cfg.DatabaseURL)
	default:
		return false
	}
	if err != nil {
		logger.Error("command failed", "command", os.Args[1], "error", err)
		os.Exit(1)
	}
	return true
}

func fatal(logger *slog.Logger, message string, err error) {
	logger.Error(message, "error", err)
	os.Exit(1)
}

func healthcheck(addr string) error {
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	response, err := (&http.Client{Timeout: 3 * time.Second}).Get("http://" + addr + "/api/readyz")
	if err != nil {
		return fmt.Errorf("request readiness endpoint: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("readiness endpoint returned %s", response.Status)
	}
	return nil
}
