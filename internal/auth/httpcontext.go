package auth

import (
	"context"
	"net"
	"net/http"
	"time"
)

// RefreshCookieName is the HttpOnly cookie carrying the opaque refresh
// token. It is scoped to RefreshCookiePath so the browser only attaches it
// to the auth endpoints that need it.
const RefreshCookieName = "starter_refresh"

// RefreshCookiePath restricts the refresh cookie to the auth API surface,
// per the frozen contract: the browser access token stays in memory and the
// refresh token never becomes reachable to arbitrary page script or routes.
const RefreshCookiePath = "/api/auth"

type httpContextKey struct{}

type httpContext struct {
	request        *http.Request
	responseWriter http.ResponseWriter
}

// WithHTTPContext makes the raw *http.Request and http.ResponseWriter
// available to ogen Handler methods via context, so the handler can read
// the refresh cookie and set Set-Cookie response headers. Header
// modifications are safe as long as they happen before the response body is
// written, which holds here because the handler method always returns
// before ogen's generated encoder calls WriteHeader.
func WithHTTPContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), httpContextKey{}, &httpContext{request: r, responseWriter: w})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func fromContext(ctx context.Context) (*httpContext, bool) {
	value, ok := ctx.Value(httpContextKey{}).(*httpContext)
	return value, ok
}

// RequestMetadataFromContext extracts a best-effort User-Agent/IP pair for
// refresh session bookkeeping. It never fails: an empty RefreshSessionMetadata
// is returned when no request is present in ctx.
func RequestMetadataFromContext(ctx context.Context) RefreshSessionMetadata {
	httpCtx, ok := fromContext(ctx)
	if !ok || httpCtx.request == nil {
		return RefreshSessionMetadata{}
	}
	ip := httpCtx.request.RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	return RefreshSessionMetadata{
		UserAgent: httpCtx.request.UserAgent(),
		IPAddress: ip,
	}
}

// RefreshTokenFromContext reads the raw refresh token from its HttpOnly
// cookie, if present.
func RefreshTokenFromContext(ctx context.Context) (string, bool) {
	httpCtx, ok := fromContext(ctx)
	if !ok || httpCtx.request == nil {
		return "", false
	}
	cookie, err := httpCtx.request.Cookie(RefreshCookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}

// BearerTokenFromContext reads the raw access token from the Authorization
// header, if present and well-formed.
func BearerTokenFromContext(ctx context.Context) (string, bool) {
	httpCtx, ok := fromContext(ctx)
	if !ok || httpCtx.request == nil {
		return "", false
	}
	header := httpCtx.request.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) <= len(prefix) || header[:len(prefix)] != prefix {
		return "", false
	}
	return header[len(prefix):], true
}

// SetRefreshCookie writes the HttpOnly/Secure/SameSite=Lax refresh cookie on
// the response captured in ctx, scoped to RefreshCookiePath.
func SetRefreshCookie(ctx context.Context, value string, expiresAt time.Time) {
	httpCtx, ok := fromContext(ctx)
	if !ok || httpCtx.responseWriter == nil {
		return
	}
	http.SetCookie(httpCtx.responseWriter, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    value,
		Path:     RefreshCookiePath,
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearRefreshCookie expires the refresh cookie immediately, used on sign-out
// and on detected refresh-token reuse.
func ClearRefreshCookie(ctx context.Context) {
	httpCtx, ok := fromContext(ctx)
	if !ok || httpCtx.responseWriter == nil {
		return
	}
	http.SetCookie(httpCtx.responseWriter, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    "",
		Path:     RefreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
