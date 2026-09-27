package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/hydrz/starter/internal/platform/httpserver"
)

func TestClientIPIgnoresForwardedHeaderFromUntrustedPeer(t *testing.T) {
	t.Parallel()

	handler := clientIPHandler(t, []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "198.51.100.10:1234"
	request.Header.Set("X-Forwarded-For", "203.0.113.42")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if got, want := recorder.Body.String(), "198.51.100.10"; got != want {
		t.Errorf("resolved client IP = %q, want %q", got, want)
	}
}

func TestClientIPResolvesFirstUntrustedAddressFromTrustedProxy(t *testing.T) {
	t.Parallel()

	handler := clientIPHandler(t, []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("2001:db8:feed::/48"),
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "10.0.0.9:443"
	request.Header.Set("X-Forwarded-For", "198.51.100.7, 10.1.2.3, 2001:db8:feed::1")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if got, want := recorder.Body.String(), "198.51.100.7"; got != want {
		t.Errorf("resolved client IP = %q, want %q", got, want)
	}
}

func TestClientIPDoesNotTrustForwardedHeaderWithoutTrustedProxies(t *testing.T) {
	t.Parallel()

	handler := clientIPHandler(t, nil)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "10.0.0.9:443"
	request.Header.Set("X-Forwarded-For", "198.51.100.7")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if got, want := recorder.Body.String(), "10.0.0.9"; got != want {
		t.Errorf("resolved client IP = %q, want %q", got, want)
	}
}

func clientIPHandler(t *testing.T, trustedProxies []netip.Prefix) http.Handler {
	t.Helper()
	return httpserver.ClientIP(trustedProxies)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(httpserver.ClientIPFromContext(r.Context())))
	}))
}
