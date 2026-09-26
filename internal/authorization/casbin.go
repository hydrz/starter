package authorization

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
)

// DefaultDomainRBACModel is the standard Casbin configuration for RBAC with domains.
// Request and policy tuples are (sub, dom, obj, act). Role grouping tuple is (user, role, dom).
const DefaultDomainRBACModel = `
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (r.sub == p.sub || g(r.sub, p.sub, r.dom)) && (p.dom == "*" || r.dom == p.dom) && (p.obj == "*" || r.obj == p.obj) && (p.act == "*" || r.act == p.act)
`

// NewEnforcer initializes a casbin.Enforcer with the DefaultDomainRBACModel and the provided adapter.
// If adapter is nil, an in-memory enforcer is returned.
func NewEnforcer(adapter persist.Adapter) (*casbin.Enforcer, error) {
	return NewEnforcerWithModel(DefaultDomainRBACModel, adapter)
}

// NewEnforcerWithModel initializes a casbin.Enforcer with a custom model string and adapter.
func NewEnforcerWithModel(modelText string, adapter persist.Adapter) (*casbin.Enforcer, error) {
	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("authorization: parse casbin model: %w", err)
	}

	var enforcer *casbin.Enforcer
	if adapter != nil {
		enforcer, err = casbin.NewEnforcer(m, adapter)
	} else {
		enforcer, err = casbin.NewEnforcer(m)
	}
	if err != nil {
		return nil, fmt.Errorf("authorization: create casbin enforcer: %w", err)
	}

	return enforcer, nil
}
