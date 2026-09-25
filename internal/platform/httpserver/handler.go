package httpserver

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/hydrz/starter/internal/announcement"
	"github.com/hydrz/starter/internal/api"
	"github.com/hydrz/starter/internal/platform/webui"
)

const maxRequestBodySize = 1 << 20

type Handler struct {
	announcements *announcement.Service
	readiness     HealthChecker
}

var _ api.ServerInterface = (*Handler)(nil)

type HealthChecker interface {
	Ping(context.Context) error
}

//go:embed assets/*
var docsAssets embed.FS

func NewHandler(announcements *announcement.Service, readiness HealthChecker) (http.Handler, error) {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(AccessLog(slog.Default()))
	router.Use(middleware.Recoverer)

	handler := &Handler{announcements: announcements, readiness: readiness}
	router.Get("/api/openapi.json", openAPISpec)
	router.Get("/api/docs", scalarReference)
	router.Get("/api/docs/scalar.js", scalarScript)

	apiHandler := api.HandlerWithOptions(handler, api.ChiServerOptions{
		BaseRouter: router,
		ErrorHandlerFunc: func(response http.ResponseWriter, _ *http.Request, _ error) {
			writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request parameters are invalid.")
		},
	})
	webHandler, err := webui.NewHandler()
	if err != nil {
		return nil, fmt.Errorf("initialize web assets: %w", err)
	}
	router.NotFound(webHandler.ServeHTTP)
	return apiHandler, nil
}

func (*Handler) GetHealth(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (handler *Handler) GetReadiness(response http.ResponseWriter, request *http.Request) {
	if handler.readiness == nil || handler.readiness.Ping(request.Context()) != nil {
		writeAPIError(response, http.StatusServiceUnavailable, "not_ready", "Service dependencies are not ready.")
		return
	}
	writeJSON(response, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (handler *Handler) ListAnnouncements(response http.ResponseWriter, request *http.Request, params api.ListAnnouncementsParams) {
	filter := announcement.Filter{Limit: valueOr(params.Limit, announcement.DefaultPageSize), Offset: valueOr(params.Offset, 0)}
	if params.Status != nil {
		status := announcement.Status(*params.Status)
		filter.Status = &status
	}

	page, err := handler.announcements.List(request.Context(), filter)
	if err != nil {
		handleServiceError(response, err)
		return
	}

	items := make([]api.Announcement, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, toAPIAnnouncement(item))
	}
	writeJSON(response, http.StatusOK, api.PageAnnouncement{
		Items: items, Total: page.Total, Limit: page.Limit, Offset: page.Offset,
	})
}

func (handler *Handler) CreateAnnouncement(response http.ResponseWriter, request *http.Request) {
	input, ok := decodeAnnouncementInput(response, request)
	if !ok {
		return
	}
	item, err := handler.announcements.Create(request.Context(), input)
	if err != nil {
		handleServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, toAPIAnnouncement(item))
}

func (handler *Handler) GetAnnouncement(response http.ResponseWriter, request *http.Request, id openapi_types.UUID) {
	item, err := handler.announcements.Get(request.Context(), id.String())
	if err != nil {
		handleServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, toAPIAnnouncement(item))
}

func (handler *Handler) UpdateAnnouncement(response http.ResponseWriter, request *http.Request, id openapi_types.UUID) {
	input, ok := decodeAnnouncementInput(response, request)
	if !ok {
		return
	}
	item, err := handler.announcements.Update(request.Context(), id.String(), input)
	if err != nil {
		handleServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, toAPIAnnouncement(item))
}

func (handler *Handler) DeleteAnnouncement(response http.ResponseWriter, request *http.Request, id openapi_types.UUID) {
	if err := handler.announcements.Delete(request.Context(), id.String()); err != nil {
		handleServiceError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func openAPISpec(response http.ResponseWriter, _ *http.Request) {
	spec, err := api.GetSpecJSON()
	if err != nil {
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = response.Write(spec)
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

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	if err := json.NewEncoder(response).Encode(value); err != nil {
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func decodeAnnouncementInput(response http.ResponseWriter, request *http.Request) (announcement.Input, bool) {
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBodySize)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	var input api.AnnouncementInput
	if err := decoder.Decode(&input); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return announcement.Input{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request body must contain one JSON object.")
		return announcement.Input{}, false
	}
	return announcement.Input{Title: input.Title, Content: input.Content, Status: announcement.Status(input.Status)}, true
}

func handleServiceError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, announcement.ErrInvalidInput):
		writeAPIError(response, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, announcement.ErrNotFound):
		writeAPIError(response, http.StatusNotFound, "not_found", "Announcement was not found.")
	default:
		writeAPIError(response, http.StatusInternalServerError, "internal_error", "The request could not be completed.")
	}
}

func writeAPIError(response http.ResponseWriter, status int, code, message string) {
	writeJSON(response, status, api.ApiError{Code: code, Message: message})
}

func toAPIAnnouncement(item announcement.Announcement) api.Announcement {
	return api.Announcement{
		Id: openapi_types.UUID(uuid.MustParse(item.ID)), Title: item.Title, Content: item.Content,
		Status: api.AnnouncementStatus(item.Status), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func valueOr[T any](value *T, fallback T) T {
	if value == nil {
		return fallback
	}
	return *value
}

const scalarHTML = `<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Starter API</title>
  </head>
  <body>
    <script id="api-reference" data-url="/api/openapi.json"></script>
    <script src="/api/docs/scalar.js"></script>
  </body>
</html>`
