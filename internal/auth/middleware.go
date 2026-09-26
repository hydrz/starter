package auth

import (
	"context"
	"net/http"
	"time"
)

// principalContextKey holds the authenticated subject's user ID, set by
// AccessTokenMiddleware after verifying the bearer JWT. This is
// authentication only: it establishes "who", not "what they may do". Later
// authorization work (organizations/Casbin, workstream C) reads this
// principal as its subject input; it is not itself an authorization
// decision, and no other business route is protected by it yet (per the
// integration gate, global middleware wiring is out of scope for B).
type principalContextKey struct{}

// PrincipalFromContext extracts the authenticated user ID from context.
func PrincipalFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(principalContextKey{}).(string)
	return userID, ok && userID != ""
}

func principalFromContext(ctx context.Context) (string, bool) {
	return PrincipalFromContext(ctx)
}

// ContextWithPrincipal returns a new context carrying the authenticated user ID.
func ContextWithPrincipal(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, principalContextKey{}, userID)
}

func contextWithPrincipal(ctx context.Context, userID string) context.Context {
	return ContextWithPrincipal(ctx, userID)
}

// authenticatedAtContextKey holds the verified access token's issued-at
// time, set by AuthenticateRequest alongside the principal. It powers a
// "recently reauthenticated" check (e.g. before linking an OAuth provider
// to an existing account) without requiring a database round trip: since
// access tokens are short-lived, a fresh one is itself evidence of a recent
// credential check.
type authenticatedAtContextKey struct{}

// AuthenticatedAtFromContext extracts the issued-at time of the access
// token that authenticated the current request, if any.
func AuthenticatedAtFromContext(ctx context.Context) (time.Time, bool) {
	issuedAt, ok := ctx.Value(authenticatedAtContextKey{}).(time.Time)
	return issuedAt, ok
}

// AuthenticateRequest verifies the bearer access token found via
// BearerTokenFromContext(ctx) and, on success, returns a context carrying
// the resolved principal for principalFromContext / GetCurrentIdentity and
// friends to read. It performs no authorization: callers protected by a
// missing/invalid token must reject the request themselves (see
// HTTPHandler, which returns 401 ApiError bodies per operation).
func (service *Service) AuthenticateRequest(ctx context.Context) (context.Context, bool) {
	token, ok := BearerTokenFromContext(ctx)
	if !ok {
		return ctx, false
	}
	claims, err := service.VerifyAccessToken(token)
	if err != nil {
		return ctx, false
	}
	ctx = contextWithPrincipal(ctx, claims.Subject)
	ctx = context.WithValue(ctx, authenticatedAtContextKey{}, claims.IssuedAt)
	return ctx, true
}

// AuthenticationMiddleware resolves the caller's principal, if any, from
// the request's bearer token before delegating to next. It never rejects a
// request itself: operations that require authentication check
// principalFromContext (via HTTPHandler) and return their own 401 response
// when absent, keeping response shapes owned by the OpenAPI contract.
func (service *Service) AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := service.AuthenticateRequest(r.Context())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
