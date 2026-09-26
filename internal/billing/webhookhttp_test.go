package billing_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hydrz/starter/internal/billing"
)

func TestWebhookHTTPHandler_ValidEventReturns200(t *testing.T) {
	t.Parallel()
	catalog := billing.Catalog{"lifetime": {StripePriceID: "price_lifetime", FeatureKey: "premium", Mode: billing.ModePayment}}
	h := newHarness(t, catalog)

	account, err := h.billingAccounts.GetOrCreate(t.Context(), "org-1", "cus_1")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if _, err := h.checkoutSessions.Create(t.Context(), account.ID, "cs_1", "lifetime", billing.ModePayment); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	now := time.Now()
	payload := checkoutCompletedPayload("evt_http_1", "cs_1", "cus_1", 1000, "usd", "payment", now.Unix())
	header := signPayload(payload, testWebhookSecret, now)

	handler := billing.NewWebhookHTTPHandler(h.service)
	req := httptest.NewRequest(http.MethodPost, "/api/billing/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", header)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestWebhookHTTPHandler_InvalidSignatureReturns400(t *testing.T) {
	t.Parallel()
	h := newHarness(t, billing.Catalog{})

	payload := checkoutCompletedPayload("evt_bad_sig", "cs_1", "cus_1", 1000, "usd", "payment", time.Now().Unix())
	handler := billing.NewWebhookHTTPHandler(h.service)
	req := httptest.NewRequest(http.MethodPost, "/api/billing/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", "t=1,v1=not-a-real-signature")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestWebhookHTTPHandler_MissingSignatureReturns400(t *testing.T) {
	t.Parallel()
	h := newHarness(t, billing.Catalog{})

	payload := checkoutCompletedPayload("evt_no_sig", "cs_1", "cus_1", 1000, "usd", "payment", time.Now().Unix())
	handler := billing.NewWebhookHTTPHandler(h.service)
	req := httptest.NewRequest(http.MethodPost, "/api/billing/webhooks/stripe", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestWebhookHTTPHandler_OversizedBodyRejected(t *testing.T) {
	t.Parallel()
	h := newHarness(t, billing.Catalog{})

	oversized := bytes.Repeat([]byte("a"), (1<<20)+1)
	handler := billing.NewWebhookHTTPHandler(h.service)
	req := httptest.NewRequest(http.MethodPost, "/api/billing/webhooks/stripe", bytes.NewReader(oversized))
	req.Header.Set("Stripe-Signature", "t=1,v1=irrelevant")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestWebhookHTTPHandler_GetNotAllowed(t *testing.T) {
	t.Parallel()
	h := newHarness(t, billing.Catalog{})

	handler := billing.NewWebhookHTTPHandler(h.service)
	req := httptest.NewRequest(http.MethodGet, "/api/billing/webhooks/stripe", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestWebhookHTTPHandler_RateLimitsPerSourceIP exercises the per-IP webhook
// rate limiter (internal/billing/webhook_ratelimit.go, ceiling
// webhookRateLimitPerWindow requests per webhookRateLimitWindow) added so
// an attacker cannot cheaply force repeated HMAC verification work by
// hammering this unauthenticated endpoint. Requests carry no signature so
// they are fast (rejected at 400) until the limiter itself starts
// rejecting at 429 — before any body is read for the over-limit request.
func TestWebhookHTTPHandler_RateLimitsPerSourceIP(t *testing.T) {
	t.Parallel()
	h := newHarness(t, billing.Catalog{})
	handler := billing.NewWebhookHTTPHandler(h.service)

	// Matches internal/billing/webhook_ratelimit.go's
	// webhookRateLimitPerWindow; kept in sync by this test's comment rather
	// than an export, since the constant is intentionally unexported.
	const limit = 300

	newRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/api/billing/webhooks/stripe", bytes.NewBufferString(`{}`))
		// httptest.NewRequest defaults RemoteAddr to a fixed value, so every
		// call in this test shares one source IP key.
		req.RemoteAddr = "203.0.113.7:54321"
		return req
	}

	sawTooManyRequests := false
	for i := 0; i < limit+10; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, newRequest())
		if rec.Code == http.StatusTooManyRequests {
			sawTooManyRequests = true
			if body := rec.Body.String(); body != "" {
				t.Fatalf("429 response body = %q, want empty (no diagnostic detail)", body)
			}
			break
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("request %d: status = %d, want %d (missing signature) or eventually %d", i, rec.Code, http.StatusBadRequest, http.StatusTooManyRequests)
		}
	}

	if !sawTooManyRequests {
		t.Fatalf("expected a 429 within %d requests from one source IP, never saw one", limit+10)
	}
}

// TestWebhookHTTPHandler_RateLimitIsPerIP confirms the limiter keys by
// source IP: a different IP is unaffected by another IP's requests within
// the same window.
func TestWebhookHTTPHandler_RateLimitIsPerIP(t *testing.T) {
	t.Parallel()
	h := newHarness(t, billing.Catalog{})
	handler := billing.NewWebhookHTTPHandler(h.service)

	const limit = 300
	exhaust := func(ip string) {
		for i := 0; i < limit; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/billing/webhooks/stripe", bytes.NewBufferString(`{}`))
			req.RemoteAddr = ip + ":1"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	}
	exhaust("198.51.100.1")

	req := httptest.NewRequest(http.MethodPost, "/api/billing/webhooks/stripe", bytes.NewBufferString(`{}`))
	req.RemoteAddr = "198.51.100.2:1"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("a different source IP's request status = %d, want %d (not rate limited by another IP's requests)", rec.Code, http.StatusBadRequest)
	}
}
