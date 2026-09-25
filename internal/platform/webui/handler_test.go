package webui_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hydrz/starter/internal/platform/webui"
)

func TestHandlerServesIndexAndSPAFallback(t *testing.T) {
	t.Parallel()

	handler, err := webui.NewHandler()
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	for _, target := range []string{"/", "/announcements/123"} {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", target, response.Code, http.StatusOK)
		}
		if !strings.Contains(strings.ToLower(response.Body.String()), "<!doctype html>") {
			t.Errorf("GET %s did not serve index", target)
		}
	}
}

func TestHandlerDoesNotFallbackForAPIOrMissingAsset(t *testing.T) {
	t.Parallel()

	handler, err := webui.NewHandler()
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	for _, target := range []string{"/api/unknown", "/assets/missing.js"} {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want %d", target, response.Code, http.StatusNotFound)
		}
	}
}
