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
