package authorization

import (
	"context"
	"errors"

	"github.com/casbin/casbin/v2"
)

// Authorizer defines the narrow authorization boundary for three-part decisions:
// subject (sub), domain (dom), and object/action (obj, act).
type Authorizer interface {
	Authorize(ctx context.Context, sub, dom, obj, act string) (bool, error)
}

// Service implements the Authorizer interface using Casbin.
type Service struct {
	enforcer *casbin.Enforcer
}

var _ Authorizer = (*Service)(nil)

// NewService creates a new authorization service.
func NewService(enforcer *casbin.Enforcer) *Service {
	return &Service{enforcer: enforcer}
}

// Authorize evaluates whether sub is permitted to perform act on obj within domain dom.
func (s *Service) Authorize(ctx context.Context, sub, dom, obj, act string) (bool, error) {
	if s.enforcer == nil {
		return false, errors.New("authorization: enforcer is nil")
	}
	return s.enforcer.Enforce(sub, dom, obj, act)
}

// Enforcer exposes the underlying Casbin enforcer for administrative tasks (adding/removing roles).
func (s *Service) Enforcer() *casbin.Enforcer {
	return s.enforcer
}

// Enforce satisfies the PermissionEnforcer interface by delegating to the underlying enforcer.
func (s *Service) Enforce(rvals ...interface{}) (bool, error) {
	if s.enforcer == nil {
		return false, errors.New("authorization: enforcer is nil")
	}
	return s.enforcer.Enforce(rvals...)
}

// GetRolesForUserInDomain satisfies the PermissionEnforcer interface by delegating to the underlying enforcer.
func (s *Service) GetRolesForUserInDomain(name string, domain string) []string {
	if s.enforcer == nil {
		return nil
	}
	return s.enforcer.GetRolesForUserInDomain(name, domain)
}
