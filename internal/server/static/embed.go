// Package static provides embedded static assets for the web UI.
package static

import (
	"embed"
	"io/fs"
)

//go:embed fonts/* licenses.json
var FS embed.FS

//go:embed all:app
var ReactFS embed.FS

// Public returns a sub-filesystem with only the files that should be served via HTTP.
func Public() fs.FS {
	fsys, err := fs.Sub(FS, ".")
	if err != nil {
		panic(err)
	}

	return fsys
}

// ReactApp returns a sub-filesystem for the React SPA.
func ReactApp() fs.FS {
	fsys, err := fs.Sub(ReactFS, "app")
	if err != nil {
		panic(err)
	}

	return fsys
}
