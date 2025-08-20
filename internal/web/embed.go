package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

func Static() http.FileSystem {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return http.FS(staticFS)
	}
	return http.FS(sub)
}

func ReadIndex() ([]byte, error) {
	return staticFS.ReadFile("static/index.html")
}
