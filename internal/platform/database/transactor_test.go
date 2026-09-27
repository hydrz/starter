package database_test

import (
	"context"
	"testing"

	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/store"
)

func TestTxFromContext_Empty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tx, ok := database.TxFromContext(ctx)
	if ok || tx != nil {
		t.Fatalf("expected no tx in empty context, got ok=%v, tx=%v", ok, tx)
	}
}

func TestQueries_WithoutTransaction_ReturnsOriginalQueries(t *testing.T) {
	t.Parallel()

	queries := store.New(nil)
	result := database.Queries(context.Background(), queries)

	if result != queries {
		t.Fatal("expected queries without a transaction to be returned unchanged")
	}
}
