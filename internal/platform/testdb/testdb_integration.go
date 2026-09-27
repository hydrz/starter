//go:build integration

// Package testdb provisions isolated PostgreSQL databases for integration tests.
package testdb

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hydrz/starter/db"
	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/store"
)

const startupTimeout = 90 * time.Second

// Database is an isolated, fully migrated PostgreSQL database for a test.
type Database struct {
	Pool    *pgxpool.Pool
	Queries *store.Queries
}

// New starts PostgreSQL 17, applies every production migration, and returns a
// pinged connection pool. Docker provider failures skip the test; all other
// setup failures fail it because they indicate an invalid integration fixture.
func New(t *testing.T) *Database {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	t.Cleanup(cancel)

	container, err := postgres.Run(ctx, "postgres:17",
		postgres.WithDatabase("starter"),
		postgres.WithUsername("starter"),
		postgres.WithPassword("starter"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2)),
	)
	if err != nil {
		t.Fatalf("start PostgreSQL container: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := container.Terminate(cleanupCtx); err != nil {
			t.Errorf("terminate PostgreSQL container: %v", err)
		}
	})

	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("build PostgreSQL connection string: %v", err)
	}
	if err := db.Migrate(ctx, databaseURL); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open PostgreSQL pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return &Database{Pool: pool, Queries: store.New(pool)}
}

// Context returns a test-bounded context for individual database operations.
func Context(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// Now returns the current wall-clock time in UTC for timestamp fixtures.
func Now() time.Time { return time.Now().UTC() }

// RequireExec executes fixture SQL and fails the calling test on error.
func RequireExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("execute fixture SQL: %v\nSQL: %s", err, sql)
	}
}
