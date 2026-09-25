package webui

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

func NewHandler() (http.Handler, error) {
	public, err := assets()
	if err != nil {
		return nil, err
	}
	files := http.FileServer(http.FS(public))

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			http.NotFound(response, request)
			return
		}
		if strings.HasPrefix(request.URL.Path, "/api/") {
			http.NotFound(response, request)
			return
		}

		requestedPath := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
		if requestedPath == "." || requestedPath == "" {
			requestedPath = "index.html"
		}
		if _, err := fs.Stat(public, requestedPath); err == nil {
			setCacheHeader(response, requestedPath)
			files.ServeHTTP(response, request)
			return
		}
		if path.Ext(requestedPath) != "" {
			http.NotFound(response, request)
			return
		}

		response.Header().Set("Cache-Control", "no-cache")
		fallback := request.Clone(request.Context())
		fallback.URL.Path = "/"
		files.ServeHTTP(response, fallback)
	}), nil
}

func setCacheHeader(response http.ResponseWriter, requestedPath string) {
	if strings.HasPrefix(requestedPath, "assets/") {
		response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	response.Header().Set("Cache-Control", "no-cache")
}
