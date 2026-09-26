package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"
)

// apiError mirrors the shared ApiError shape declared once in
// packages/contracts/common/responses.tsp ({code, message}) and reused by
// every generated *api package's own ApiError type. It is declared locally,
// rather than importing one feature package's copy, so this file has no
// dependency on any single feature API being wired up.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewAPIErrorHandler returns a single ogenerrors.ErrorHandler shared by
// every generated *api.NewServer(...) call in NewHandler (see
// systemapi/authapi/organizationsapi/announcementsapi/billingapi's
// WithErrorHandler server option). It replaces ogen's DefaultErrorHandler,
// which serializes err.Error() verbatim as {"error_message": "..."} with a
// 500 — this was reproduced live as a Stripe API failure leaking the
// outbound URL and vendor error text to an API client, and violates
// docs/implementation/identity-platform-contract-map.md's rule that
// internal diagnostics, tokens, and vendor error text never enter an
// ApiError's message.
//
// This handler only ever runs on the fallback path: a handler method that
// returns (nil, err) for an error that is not one of the operation's own
// declared typed responses (those are already-correct api.XxxResponse
// values, encoded normally, and never reach here), or a genuine ogen
// transport-decode error (malformed params/body). Every already-typed
// 400/401/403/404/409/412/429/503 ApiError response a handler returns
// explicitly is unaffected.
func NewAPIErrorHandler(logger *slog.Logger) ogenerrors.ErrorHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
		code := ogenerrors.ErrorCode(err)

		if code == http.StatusInternalServerError {
			// Full diagnostic detail (which may include database/driver
			// detail or, for billing, raw Stripe SDK error text) is logged
			// server-side only, at ERROR level, with request context but not
			// the request body.
			logger.ErrorContext(ctx, "unhandled api error",
				"method", r.Method,
				"path", r.URL.Path,
				"error", err,
			)
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal error occurred.")
			return
		}

		// A genuine ogen transport error (malformed params/body, unsupported
		// media type, and similar) rather than a business-logic error. The
		// raw error text can embed client-supplied bytes, so it is logged,
		// not echoed; the client gets a fixed, generic message for the
		// resolved status code.
		logger.WarnContext(ctx, "rejected api request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", code,
			"error", err,
		)
		writeAPIError(w, code, "request_error", http.StatusText(code))
	}
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiError{Code: code, Message: message})
}
