//go:build integration

package db

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	integrationDatabaseURLEnv = "D4_TEST_DATABASE_URL"
	helperDoneToken           = "HELPER-MIGRATIONS-COMPLETE"
)

// TestMigrateConcurrentProcessesApplyMigrationsOnce starts two independent
// OS processes that run Migrate against the same empty database at the same
// time and asserts both succeed while the schema is applied exactly once.
// Requires a disposable PostgreSQL instance; set D4_TEST_DATABASE_URL to
// enable it (skipped otherwise, so plain `go test ./...` stays hermetic).
func TestMigrateConcurrentProcessesApplyMigrationsOnce(t *testing.T) {
	databaseURL := os.Getenv(integrationDatabaseURLEnv)
	if databaseURL == "" {
		t.Skipf("%s is not configured", integrationDatabaseURLEnv)
	}

	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	// The database must be disposable: this resets the public schema so both
	// helpers race on a fresh, unmigrated state.
	if _, err := database.Exec(`DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatalf("reset test schema: %v", err)
	}

	type helper struct {
		command *exec.Cmd
		output  bytes.Buffer
	}
	helpers := make([]helper, 2)
	for index := range helpers {
		helpers[index].command = exec.Command(os.Args[0], "-test.run=TestMigrateConcurrentProcessesApplyMigrationsOnceHelper", "-test.v")
		helpers[index].command.Env = append(os.Environ(), integrationDatabaseURLEnv+"="+databaseURL)
		helpers[index].command.Stdout = &helpers[index].output
		helpers[index].command.Stderr = &helpers[index].output
		if err := helpers[index].command.Start(); err != nil {
			t.Fatalf("start migration helper %d: %v", index, err)
		}
	}

	for index := range helpers {
		if err := helpers[index].command.Wait(); err != nil {
			t.Fatalf("run migration helper %d: %v\n%s", index, err, helpers[index].output.String())
		}
	}

	for index := range helpers {
		if got := strings.Count(helpers[index].output.String(), helperDoneToken); got != 1 {
			t.Errorf("helper %d completed-marker count = %d, want 1\n%s", index, got, helpers[index].output.String())
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// goose_db_version holds one row per applied migration plus one row for
	// version 0. A duplicate application would add rows or violate the
	// primary key, so six rows means each migration ran exactly once.
	var applied int
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM goose_db_version WHERE is_applied`).Scan(&applied); err != nil {
		t.Fatalf("count applied migrations: %v", err)
	}
	const (
		versionZeroRows = 1
		migrationCount  = 5
		wantApplied     = versionZeroRows + migrationCount
	)
	if applied != wantApplied {
		t.Errorf("applied migration rows = %d, want %d (a duplicate application would exceed it)", applied, wantApplied)
	}
	var maxVersion int64
	if err := database.QueryRowContext(ctx, `SELECT coalesce(max(version_id), 0) FROM goose_db_version`).Scan(&maxVersion); err != nil {
		t.Fatalf("query max migration version: %v", err)
	}
	if maxVersion != migrationCount {
		t.Errorf("max applied version = %d, want %d", maxVersion, migrationCount)
	}
}

// TestMigrateConcurrentProcessesApplyMigrationsOnceHelper is only executed by
// TestMigrateConcurrentProcessesApplyMigrationsOnce as a separate process; it
// exits without running when the test database is not configured.
func TestMigrateConcurrentProcessesApplyMigrationsOnceHelper(t *testing.T) {
	databaseURL := os.Getenv(integrationDatabaseURLEnv)
	if databaseURL == "" {
		t.Skip("helper requires a configured test database")
	}
	if err := Migrate(context.Background(), databaseURL); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	fmt.Println(helperDoneToken)
}
