package httpserver

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gopkg.in/yaml.v3"

	"github.com/hydrz/starter/internal/announcement"
	"github.com/hydrz/starter/internal/api/announcementsapi"
	"github.com/hydrz/starter/internal/api/systemapi"
	"github.com/hydrz/starter/internal/platform/webui"
)

type HealthChecker interface {
	Ping(context.Context) error
}

//go:embed assets/*
var docsAssets embed.FS

type SystemHandler struct {
	readiness HealthChecker
}

var _ systemapi.Handler = (*SystemHandler)(nil)

func (h *SystemHandler) GetHealth(_ context.Context) (*systemapi.HealthResponse, error) {
	return &systemapi.HealthResponse{Status: systemapi.HealthResponseStatusOk}, nil
}

func (h *SystemHandler) GetReadiness(ctx context.Context) (systemapi.GetReadinessRes, error) {
	if h.readiness == nil || h.readiness.Ping(ctx) != nil {
		return &systemapi.ApiError{
			Code:    "not_ready",
			Message: "Service dependencies are not ready.",
		}, nil
	}
	return &systemapi.HealthResponse{Status: systemapi.HealthResponseStatusOk}, nil
}

func NewHandler(announcements *announcement.Service, readiness HealthChecker) (http.Handler, error) {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(AccessLog(slog.Default()))
	router.Use(middleware.Recoverer)

	router.Get("/api/openapi.yaml", openAPISpecYAML)
	router.Get("/api/openapi.json", openAPISpecJSON)
	router.Get("/api/docs", scalarReference)
	router.Get("/api/docs/scalar.js", scalarScript)

	systemServer, err := systemapi.NewServer(&SystemHandler{readiness: readiness})
	if err != nil {
		return nil, fmt.Errorf("initialize system api server: %w", err)
	}
	router.Handle("/api/healthz", systemServer)
	router.Handle("/api/readyz", systemServer)

	if announcements != nil {
		announcementServer, err := announcementsapi.NewServer(announcement.NewHTTPHandler(announcements))
		if err != nil {
			return nil, fmt.Errorf("initialize announcements api server: %w", err)
		}
		router.Handle("/api/announcements", announcementServer)
		router.Handle("/api/announcements/*", announcementServer)
	}

	webHandler, err := webui.NewHandler()
	if err != nil {
		return nil, fmt.Errorf("initialize web assets: %w", err)
	}
	router.NotFound(webHandler.ServeHTTP)

	return router, nil
}

var (
	openAPIYAML []byte
	openAPIJSON []byte
	openAPIOnce sync.Once
	openAPIErr  error
)

func loadOpenAPISpecs() error {
	openAPIOnce.Do(func() {
		openAPIYAML, openAPIErr = docsAssets.ReadFile("assets/openapi.yaml")
		if openAPIErr != nil {
			return
		}
		var raw any
		if openAPIErr = yaml.Unmarshal(openAPIYAML, &raw); openAPIErr != nil {
			return
		}
		openAPIJSON, openAPIErr = json.Marshal(raw)
	})
	return openAPIErr
}

func openAPISpecYAML(response http.ResponseWriter, _ *http.Request) {
	if err := loadOpenAPISpecs(); err != nil {
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = response.Write(openAPIYAML)
}

func openAPISpecJSON(response http.ResponseWriter, _ *http.Request) {
	if err := loadOpenAPISpecs(); err != nil {
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = response.Write(openAPIJSON)
}

func scalarReference(response http.ResponseWriter, _ *http.Request) {
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src data: https:; font-src data:; connect-src 'self'")
	_, _ = response.Write([]byte(scalarHTML))
}

func scalarScript(response http.ResponseWriter, _ *http.Request) {
	script, err := docsAssets.ReadFile("assets/scalar.js")
	if err != nil {
		http.Error(response, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	response.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = response.Write(script)
}

const scalarHTML = `<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Starter API</title>
  </head>
  <body>
    <script id="api-reference" data-url="/api/openapi.yaml"></script>
    <script src="/api/docs/scalar.js"></script>
  </body>
</html>`
