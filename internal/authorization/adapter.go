package authorization

import (
	"context"
	"errors"
	"fmt"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"

	"github.com/hydrz/starter/internal/store"
)

// DatabaseAdapter implements persist.Adapter on top of store.Queries.
type DatabaseAdapter struct {
	queries *store.Queries
}

var _ persist.Adapter = (*DatabaseAdapter)(nil)

// NewDatabaseAdapter creates a new Casbin database adapter backed by sqlc queries.
func NewDatabaseAdapter(queries *store.Queries) *DatabaseAdapter {
	return &DatabaseAdapter{queries: queries}
}

// LoadPolicy loads all policy rules from the casbin_rules table into the model.
func (a *DatabaseAdapter) LoadPolicy(m model.Model) error {
	ctx := context.Background()
	rows, err := a.queries.ListCasbinRules(ctx)
	if err != nil {
		return fmt.Errorf("authorization: list casbin rules: %w", err)
	}

	for _, row := range rows {
		rule := []string{row.Ptype}
		for _, v := range []string{row.V0, row.V1, row.V2, row.V3, row.V4, row.V5} {
			if v != "" {
				rule = append(rule, v)
			}
		}
		if err := persist.LoadPolicyArray(rule, m); err != nil {
			return fmt.Errorf("authorization: load policy line: %w", err)
		}
	}
	return nil
}

// SavePolicy saves all policy rules from the model into the storage.
func (a *DatabaseAdapter) SavePolicy(m model.Model) error {
	ctx := context.Background()

	// Clear existing rules and rewrite all current rules
	if _, err := a.queries.DeleteCasbinRulesByPtype(ctx, "p"); err != nil {
		return fmt.Errorf("authorization: clear p rules: %w", err)
	}
	if _, err := a.queries.DeleteCasbinRulesByPtype(ctx, "g"); err != nil {
		return fmt.Errorf("authorization: clear g rules: %w", err)
	}

	for ptype, ast := range m["p"] {
		for _, rule := range ast.Policy {
			if err := a.AddPolicy("p", ptype, rule); err != nil {
				return err
			}
		}
	}
	for ptype, ast := range m["g"] {
		for _, rule := range ast.Policy {
			if err := a.AddPolicy("g", ptype, rule); err != nil {
				return err
			}
		}
	}
	return nil
}

// AddPolicy adds a policy rule to the casbin_rules table.
func (a *DatabaseAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	params := store.InsertCasbinRuleParams{
		Ptype: ptype,
	}
	if len(rule) > 0 {
		params.V0 = rule[0]
	}
	if len(rule) > 1 {
		params.V1 = rule[1]
	}
	if len(rule) > 2 {
		params.V2 = rule[2]
	}
	if len(rule) > 3 {
		params.V3 = rule[3]
	}
	if len(rule) > 4 {
		params.V4 = rule[4]
	}
	if len(rule) > 5 {
		params.V5 = rule[5]
	}

	ctx := context.Background()
	if _, err := a.queries.InsertCasbinRule(ctx, params); err != nil {
		return fmt.Errorf("authorization: insert casbin rule: %w", err)
	}
	return nil
}

// RemovePolicy removes a policy rule from the casbin_rules table.
func (a *DatabaseAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	params := store.DeleteCasbinRuleParams{
		Ptype: ptype,
	}
	if len(rule) > 0 {
		params.V0 = rule[0]
	}
	if len(rule) > 1 {
		params.V1 = rule[1]
	}
	if len(rule) > 2 {
		params.V2 = rule[2]
	}
	if len(rule) > 3 {
		params.V3 = rule[3]
	}
	if len(rule) > 4 {
		params.V4 = rule[4]
	}
	if len(rule) > 5 {
		params.V5 = rule[5]
	}

	ctx := context.Background()
	if _, err := a.queries.DeleteCasbinRule(ctx, params); err != nil {
		return fmt.Errorf("authorization: delete casbin rule: %w", err)
	}
	return nil
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
func (a *DatabaseAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	if len(fieldValues) == 0 {
		return errors.New("authorization: fieldValues cannot be empty")
	}

	ctx := context.Background()

	// Handle domain filtering: for p rules, domain is v1 (fieldIndex 1).
	// For g rules, domain is v2 (fieldIndex 2).
	if (ptype == "p" && fieldIndex == 1) || (ptype == "g" && fieldIndex == 2) {
		domain := fieldValues[0]
		if _, err := a.queries.DeleteCasbinRulesByDomain(ctx, domain); err != nil {
			return fmt.Errorf("authorization: delete rules by domain: %w", err)
		}
		return nil
	}

	// If filtering only by ptype (fieldIndex -1 or similar)
	if fieldIndex < 0 {
		if _, err := a.queries.DeleteCasbinRulesByPtype(ctx, ptype); err != nil {
			return fmt.Errorf("authorization: delete rules by ptype: %w", err)
		}
		return nil
	}

	return nil
}

// MemoryAdapter stores casbin rules in-memory for testing.
type MemoryAdapter struct {
	rules [][]string
}

var _ persist.Adapter = (*MemoryAdapter)(nil)

// NewMemoryAdapter initializes an in-memory adapter seeded with rules.
func NewMemoryAdapter(rules [][]string) *MemoryAdapter {
	copied := make([][]string, len(rules))
	for i, r := range rules {
		c := make([]string, len(r))
		copy(c, r)
		copied[i] = c
	}
	return &MemoryAdapter{rules: copied}
}

func (m *MemoryAdapter) LoadPolicy(model model.Model) error {
	for _, rule := range m.rules {
		if err := persist.LoadPolicyArray(rule, model); err != nil {
			return err
		}
	}
	return nil
}

func (m *MemoryAdapter) SavePolicy(model model.Model) error {
	return nil
}

func (m *MemoryAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	m.rules = append(m.rules, append([]string{ptype}, rule...))
	return nil
}

func (m *MemoryAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	target := append([]string{ptype}, rule...)
	filtered := make([][]string, 0, len(m.rules))
	for _, r := range m.rules {
		if !equalSlices(r, target) {
			filtered = append(filtered, r)
		}
	}
	m.rules = filtered
	return nil
}

func (m *MemoryAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	filtered := make([][]string, 0, len(m.rules))
	for _, r := range m.rules {
		if r[0] != ptype {
			filtered = append(filtered, r)
			continue
		}
		match := true
		for i, val := range fieldValues {
			idx := fieldIndex + i + 1 // +1 because r[0] is ptype
			if idx >= len(r) || r[idx] != val {
				match = false
				break
			}
		}
		if !match {
			filtered = append(filtered, r)
		}
	}
	m.rules = filtered
	return nil
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// DefaultSeedRules returns standard role-based policies matching migration 00003.
func DefaultSeedRules() [][]string {
	return [][]string{
		{"p", "owner", "*", "organizations", "read"},
		{"p", "owner", "*", "organizations", "update"},
		{"p", "owner", "*", "organizations", "delete"},
		{"p", "owner", "*", "members", "read"},
		{"p", "owner", "*", "members", "create"},
		{"p", "owner", "*", "members", "update"},
		{"p", "owner", "*", "members", "delete"},
		{"p", "owner", "*", "invitations", "read"},
		{"p", "owner", "*", "invitations", "create"},
		{"p", "owner", "*", "invitations", "delete"},
		{"p", "owner", "*", "announcements", "read"},
		{"p", "owner", "*", "announcements", "create"},
		{"p", "owner", "*", "announcements", "update"},
		{"p", "owner", "*", "announcements", "delete"},

		{"p", "admin", "*", "organizations", "read"},
		{"p", "admin", "*", "organizations", "update"},
		{"p", "admin", "*", "members", "read"},
		{"p", "admin", "*", "members", "create"},
		{"p", "admin", "*", "members", "update"},
		{"p", "admin", "*", "members", "delete"},
		{"p", "admin", "*", "invitations", "read"},
		{"p", "admin", "*", "invitations", "create"},
		{"p", "admin", "*", "invitations", "delete"},
		{"p", "admin", "*", "announcements", "read"},
		{"p", "admin", "*", "announcements", "create"},
		{"p", "admin", "*", "announcements", "update"},
		{"p", "admin", "*", "announcements", "delete"},

		{"p", "member", "*", "organizations", "read"},
		{"p", "member", "*", "members", "read"},
		{"p", "member", "*", "announcements", "read"},
		{"p", "member", "*", "announcements", "create"},
		{"p", "member", "*", "announcements", "update"},

		{"p", "viewer", "*", "organizations", "read"},
		{"p", "viewer", "*", "members", "read"},
		{"p", "viewer", "*", "announcements", "read"},
	}
}
