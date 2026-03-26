package api

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var staticFiles embed.FS

func staticHandler() http.Handler {
	dist, _ := fs.Sub(staticFiles, "dist")
	return http.FileServer(http.FS(dist))
}
