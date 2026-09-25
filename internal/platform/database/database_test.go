package database_test

import (
	"context"
	"testing"

	"github.com/hydrz/starter/internal/platform/database"
)

func TestOpenRejectsInvalidURL(t *testing.T) {
	t.Parallel()

	pool, err := database.Open(context.Background(), "://invalid")
	if err == nil {
		pool.Close()
		t.Fatal("Open() error = nil, want an invalid configuration error")
	}
}
