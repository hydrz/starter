package auth

import (
	"context"
	"net/http"
)

// principalContextKey holds the authenticated subject's user ID, set by
// AccessTokenMiddleware after verifying the bearer JWT. This is
// authentication only: it establishes "who", not "what they may do". Later
// authorization work (organizations/Casbin, workstream C) reads this
// principal as its subject input; it is not itself an authorization
// decision, and no other business route is protected by it yet (per the
// integration gate, global middleware wiring is out of scope for B).
type principalContextKey struct{}

func principalFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(principalContextKey{}).(string)
	return userID, ok && userID != ""
}

func contextWithPrincipal(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, principalContextKey{}, userID)
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
	return contextWithPrincipal(ctx, claims.Subject), true
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
