package httpserver

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/hydrz/starter/internal/platform/httpserver/clientipcontext"
)

// ClientIP resolves the client address for every request. Forwarded
// addresses are considered only when the direct peer belongs to one of the
// configured trusted proxy networks; with no trusted proxies configured all
// forwarding headers are ignored and the direct peer is used as-is. The
// resolved address is written to the request context for downstream
// consumers (session metadata, rate limiting).
func ClientIP(trustedProxies []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := remoteHost(r.RemoteAddr)
			if peer, err := netip.ParseAddr(host); err == nil && isTrustedProxy(peer, trustedProxies) {
				host = firstUntrustedForwarded(r.Header.Get("X-Forwarded-For"), peer, trustedProxies)
			}
			ctx := clientipcontext.WithClientIP(r.Context(), host)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClientIPFromContext returns the client address resolved by the ClientIP
// middleware, or an empty string when it has not run for this request.
func ClientIPFromContext(ctx context.Context) string {
	return clientipcontext.FromContext(ctx)
}

// firstUntrustedForwarded walks X-Forwarded-For from right to left, skipping
// trusted proxies, and returns the first address that is not itself a
// trusted proxy. When every entry is trusted (or none parses), the direct
// peer address is kept: an all-trusted chain means the client is the proxy
// itself as far as this deployment can tell.
func firstUntrustedForwarded(header string, peer netip.Addr, trustedProxies []netip.Prefix) string {
	items := strings.Split(header, ",")
	for index := len(items) - 1; index >= 0; index-- {
		if ip, err := netip.ParseAddr(strings.TrimSpace(items[index])); err == nil && !isTrustedProxy(ip, trustedProxies) {
			return ip.String()
		}
	}
	return peer.String()
}

func remoteHost(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

func isTrustedProxy(ip netip.Addr, trustedProxies []netip.Prefix) bool {
	for _, prefix := range trustedProxies {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}
