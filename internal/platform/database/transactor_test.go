package database_test

import (
	"context"
	"testing"

	"github.com/hydrz/starter/internal/platform/database"
)

func TestTxFromContext_Empty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tx, ok := database.TxFromContext(ctx)
	if ok || tx != nil {
		t.Fatalf("expected no tx in empty context, got ok=%v, tx=%v", ok, tx)
	}
}
