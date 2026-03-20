// cmd/openvas-tracker/frontend.go
package main

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

//go:embed all:static
var frontendFS embed.FS

func serveFrontend(e *echo.Echo) {
	distFS, _ := fs.Sub(frontendFS, "static")

	// Read index.html once at startup
	indexHTML, _ := fs.ReadFile(distFS, "index.html")

	fileServer := http.FileServer(http.FS(distFS))

	e.GET("/*", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Try to serve the actual file (JS, CSS, images, etc.)
		clean := strings.TrimPrefix(path, "/")
		if f, err := distFS.Open(clean); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for all unknown routes
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(indexHTML)
	})))
}
