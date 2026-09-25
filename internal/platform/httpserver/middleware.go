package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func AccessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				l := logger
				if l == nil {
					l = slog.Default()
				}
				status := ww.Status()
				duration := time.Since(start)
				reqID := middleware.GetReqID(r.Context())

				attrs := []slog.Attr{
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("remote_addr", r.RemoteAddr),
					slog.Int("status", status),
					slog.Int("bytes", ww.BytesWritten()),
					slog.Duration("duration", duration),
				}
				if reqID != "" {
					attrs = append(attrs, slog.String("request_id", reqID))
				}
				if ua := r.UserAgent(); ua != "" {
					attrs = append(attrs, slog.String("user_agent", ua))
				}

				level := slog.LevelInfo
				if status >= 500 {
					level = slog.LevelError
				} else if status >= 400 {
					level = slog.LevelWarn
				}

				l.LogAttrs(r.Context(), level, "http request", attrs...)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
