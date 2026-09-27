//go:build integration

package authorization_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/hydrz/starter/internal/platform/testdb"
	"github.com/hydrz/starter/internal/store"
)

func TestCasbinRuleQueriesRoundTrip(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := testdb.Context(t)

	seeded, err := db.Queries.ListCasbinRules(ctx)
	if err != nil {
		t.Fatalf("list seeded casbin rules: %v", err)
	}
	if len(seeded) == 0 {
		t.Fatal("migration seed left no casbin rules")
	}

	if _, err := db.Queries.InsertCasbinRule(ctx, store.InsertCasbinRuleParams{
		Ptype: "p", V0: "member", V1: "org-1", V2: "announcements", V3: "read",
	}); err != nil {
		t.Fatalf("insert casbin rule: %v", err)
	}
	if err != nil {
		t.Fatalf("insert casbin rule: %v", err)
	}
	if duplicate, err := db.Queries.InsertCasbinRule(ctx, store.InsertCasbinRuleParams{
		Ptype: "p", V0: "member", V1: "org-1", V2: "announcements", V3: "read",
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("duplicate insert = %#v, %v; want pgx.ErrNoRows", duplicate, err)
	}

	deleted, err := db.Queries.DeleteCasbinRule(ctx, store.DeleteCasbinRuleParams{
		Ptype: "p", V0: "member", V1: "org-1", V2: "announcements", V3: "read",
	})
	if err != nil || deleted != 1 {
		t.Fatalf("delete casbin rule = %d, %v", deleted, err)
	}
	if _, err := db.Queries.InsertCasbinRule(ctx, store.InsertCasbinRuleParams{Ptype: "g", V0: "user-1", V1: "owner", V2: "org-1"}); err != nil {
		t.Fatalf("insert grouping rule: %v", err)
	}
	if byDomain, err := db.Queries.DeleteCasbinRulesByDomain(ctx, "org-1"); err != nil || byDomain != 1 {
		t.Fatalf("delete rules by domain = %d, %v", byDomain, err)
	}
}
