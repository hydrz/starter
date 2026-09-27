//go:build production

package webui_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hydrz/starter/internal/platform/webui"
)

func TestHandlerServesPrerenderedPublicRoutes(t *testing.T) {
	t.Parallel()

	handler, err := webui.NewHandler()
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	for _, testCase := range []struct {
		target string
		want   string
	}{
		{target: "/", want: "端到端全栈极速交付"},
		{target: "/sign-in", want: "<title>登录 Starter</title>"},
		{target: "/sign-up", want: "<title>注册 Starter</title>"},
		{target: "/sitemap.xml", want: "<urlset"},
		{target: "/robots.txt", want: "Sitemap: /sitemap.xml"},
	} {
		request := httptest.NewRequest(http.MethodGet, testCase.target, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", testCase.target, response.Code, http.StatusOK)
		}
		if !strings.Contains(response.Body.String(), testCase.want) {
			t.Errorf("GET %s body does not contain %q", testCase.target, testCase.want)
		}
	}
}

func TestHandlerServesPrerenderedMeta(t *testing.T) {
	t.Parallel()

	handler, err := webui.NewHandler()
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	body := response.Body.String()
	for _, want := range []string{
		"<meta name=\"description\"",
		"<meta property=\"og:title\"",
		"<meta property=\"og:description\"",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GET / body does not contain %q", want)
		}
	}
}
