package httpserver

import "net/http"

// SecurityHeaders is global middleware setting baseline security response
// headers on every response. It is additive to, and never touches, the
// route-specific Content-Security-Policy header /api/docs sets for the
// Scalar reference UI (CSP is meaningful only for HTML-rendering routes;
// these headers apply uniformly to the whole API surface).
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		header.Set("X-Frame-Options", "DENY")

		// docs/deployment.md: the app binds loopback-only behind a managed
		// TLS reverse proxy, so a plain local/dev request over HTTP never
		// carries TLS itself. Only claim HSTS when the request demonstrably
		// arrived over TLS (direct TLS termination in this process) or a
		// trusted reverse proxy says it terminated TLS upstream
		// (X-Forwarded-Proto: https, the standard signal that proxy sets).
		// Unconditionally sending HSTS would incorrectly tell a plain-HTTP
		// local-dev browser to force HTTPS on this host.
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			header.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}
