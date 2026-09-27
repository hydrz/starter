package webui

import (
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
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
		} else if path.Ext(requestedPath) == "" {
			if _, err := fs.Stat(public, path.Join(requestedPath, "index.html")); err == nil {
				requestedPath = path.Join(requestedPath, "index.html")
			}
		}
		if file, err := public.Open(requestedPath); err == nil {
			defer file.Close()
			setCacheHeader(response, requestedPath)
			serveFile(response, request, requestedPath, file)
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

func serveFile(response http.ResponseWriter, request *http.Request, requestedPath string, file fs.File) {
	if reader, ok := file.(io.ReadSeeker); ok {
		http.ServeContent(response, request, path.Base(requestedPath), fileInfoModTime(file), reader)
		return
	}
	_, _ = io.Copy(response, file)
}

func fileInfoModTime(file fs.File) (modTime time.Time) {
	info, err := file.Stat()
	if err != nil {
		return modTime
	}
	return info.ModTime()
}

func setCacheHeader(response http.ResponseWriter, requestedPath string) {
	if strings.HasPrefix(requestedPath, "assets/") {
		response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	response.Header().Set("Cache-Control", "no-cache")
}
