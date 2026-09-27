package clientipcontext_test

import (
	"context"
	"testing"

	"github.com/hydrz/starter/internal/platform/httpserver/clientipcontext"
)

func TestFromContextReturnsStoredIP(t *testing.T) {
	t.Parallel()

	if got := clientipcontext.FromContext(clientipcontext.WithClientIP(context.Background(), "198.51.100.7")); got != "198.51.100.7" {
		t.Errorf("FromContext() = %q, want %q", got, "198.51.100.7")
	}
}

func TestFromContextEmptyWhenAbsent(t *testing.T) {
	t.Parallel()

	if got := clientipcontext.FromContext(context.Background()); got != "" {
		t.Errorf("FromContext() = %q, want empty", got)
	}
}
