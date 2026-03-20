// cmd/openvas-tracker/frontend.go
package main

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
)

//go:embed all:static
var frontendFS embed.FS

func serveFrontend(e *echo.Echo) {
	distFS, _ := fs.Sub(frontendFS, "static")
	fileServer := http.FileServer(http.FS(distFS))

	// Serve static files, fall back to index.html for SPA routing
	e.GET("/*", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		path := r.URL.Path[1:] // strip leading /
		if path == "" {
			path = "index.html"
		}
		f, err := distFS.Open(path)
		if err != nil {
			// Fall back to index.html for SPA client-side routing
			r.URL.Path = "/index.html"
		} else {
			f.Close()
		}
		fileServer.ServeHTTP(w, r)
	})))
}
