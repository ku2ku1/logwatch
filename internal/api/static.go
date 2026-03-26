package api

import (
	"embed"
	"io/fs"
	"net/http"
	"time"
)

//go:embed dist
var staticFiles embed.FS

// spaHandler serves the SPA with fallback to index.html for client-side routing
type spaHandler struct {
	fs http.FileSystem
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Try to serve the requested file
	f, err := h.fs.Open(r.URL.Path)
	if err == nil {
		defer f.Close()
		
		// Check if it's a directory (has index.html)
		stat, err := f.Stat()
		if err == nil && !stat.IsDir() {
			// It's a file, serve it
			http.FileServer(h.fs).ServeHTTP(w, r)
			return
		}
	}
	
	// File not found or it's a directory
	// Serve index.html for client-side routing
	indexFile, _ := h.fs.Open("/index.html")
	if indexFile != nil {
		defer indexFile.Close()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "index.html", time.Time{}, indexFile)
		return
	}
	
	// Fallback error
	http.NotFound(w, r)
}

func staticHandler() http.Handler {
	dist, _ := fs.Sub(staticFiles, "dist")
	return &spaHandler{fs: http.FS(dist)}
}
