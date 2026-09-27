package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const migrationAdvisoryLockID int64 = 849_930_413_472_528_071

var gooseMu sync.Mutex

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate serializes schema changes across every application replica that uses
// the same PostgreSQL database. The advisory lock belongs to the dedicated
// migration connection, so it remains held while Goose opens its own
// transactions to apply pending migrations. A competing process waits, then
// sees the schema version updated by the first process and has nothing left to
// apply.
func Migrate(ctx context.Context, databaseURL string) error {
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open migration database: %w", err)
	}
	defer database.Close()

	lockConnection, err := database.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration lock connection: %w", err)
	}
	defer lockConnection.Close()

	if _, err := lockConnection.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrationAdvisoryLockID); err != nil {
		return fmt.Errorf("acquire migration advisory lock: %w", err)
	}
	defer func() {
		_, _ = lockConnection.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", migrationAdvisoryLockID)
	}()

	gooseMu.Lock()
	defer gooseMu.Unlock()
	if err := configureGoose(); err != nil {
		return err
	}
	if err := goose.UpContext(ctx, database, "migrations"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func Status(ctx context.Context, databaseURL string) error {
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open migration database: %w", err)
	}
	defer database.Close()

	gooseMu.Lock()
	defer gooseMu.Unlock()
	if err := configureGoose(); err != nil {
		return err
	}
	if err := goose.StatusContext(ctx, database, "migrations"); err != nil {
		return fmt.Errorf("check migration status: %w", err)
	}
	return nil
}

func configureGoose() error {
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}
	return nil
}
