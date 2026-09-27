package billing

import (
	"errors"
	"io"
	"net/http"

	"github.com/hydrz/starter/internal/platform/httpserver/clientipcontext"
)

// maxWebhookBodyBytes bounds how much of a webhook request body is read
// before any parsing or signature verification happens, so an oversized
// request cannot exhaust memory.
const maxWebhookBodyBytes = 1 << 20 // 1 MiB, well above Stripe's typical event size.

// WebhookHTTPHandler is a raw (non-ogen) http.Handler for Stripe webhook
// deliveries. It is mounted directly in internal/platform/httpserver,
// outside the TypeSpec/ogen JSON router, because it must control raw-body
// size limiting and HMAC signature verification before any JSON parsing —
// the opposite order from a normal decoded-JSON operation.
type WebhookHTTPHandler struct {
	service *Service
	limiter *ipWindowLimiter
}

func NewWebhookHTTPHandler(service *Service) *WebhookHTTPHandler {
	return &WebhookHTTPHandler{
		service: service,
		limiter: newIPWindowLimiter(service.clock, webhookRateLimitWindow, webhookRateLimitPerWindow),
	}
}

func (h *WebhookHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Rate limit by source IP before any body read or HMAC verification
	// work: the endpoint is unauthenticated, so this is the only cheap
	// signal available to bound the cost an attacker can force. 429 is a
	// status Stripe's own retry logic treats as retryable, so a legitimate
	// burst above the (generous) ceiling still eventually gets through on
	// redelivery. No diagnostic detail is ever included in the response
	// body, consistent with never leaking internal detail to a caller.
	if !h.limiter.Allow(clientipcontext.FromContext(r.Context())) {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes))
	if err != nil {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	signature := r.Header.Get("Stripe-Signature")
	err = h.service.HandleWebhook(r.Context(), body, signature)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, ErrInvalidSignature):
		// Not verified: reject outright. This is never treated as proof of
		// payment, and the body is never parsed further.
		w.WriteHeader(http.StatusBadRequest)
	default:
		// Any other (persistence/transient) failure must surface as 5xx so
		// Stripe retries delivery; it is never swallowed as a 2xx.
		w.WriteHeader(http.StatusInternalServerError)
	}
}
