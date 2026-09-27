//go:build integration

package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/platform/testdb"
	"github.com/hydrz/starter/internal/store"
)

func TestPgxTransactorCommitsAndRollsBack(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := testdb.Context(t)
	transactor := database.NewTransactor(db.Pool)

	var committedID pgtype.UUID
	if err := transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		user, err := database.Queries(txCtx, db.Queries).CreateUser(txCtx, store.CreateUserParams{
			Email:        "committed@example.com",
			PasswordHash: "hash",
		})
		if err != nil {
			return err
		}
		committedID = user.ID
		return nil
	}); err != nil {
		t.Fatalf("commit transaction: %v", err)
	}
	if _, err := db.Queries.GetUserByID(ctx, committedID); err != nil {
		t.Fatalf("query committed user outside transaction: %v", err)
	}

	rollbackErr := errors.New("force rollback")
	if err := transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		_, err := database.Queries(txCtx, db.Queries).CreateUser(txCtx, store.CreateUserParams{
			Email:        "rolled-back@example.com",
			PasswordHash: "hash",
		})
		if err != nil {
			return err
		}
		return rollbackErr
	}); !errors.Is(err, rollbackErr) {
		t.Fatalf("transaction error = %v, want %v", err, rollbackErr)
	}
	if _, err := db.Queries.GetUserByEmail(ctx, "rolled-back@example.com"); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("rolled-back user lookup error = %v, want pgx.ErrNoRows", err)
	}
}
