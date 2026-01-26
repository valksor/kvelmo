// Package static provides embedded static assets for the web UI.
package static

import (
	"embed"
	"io/fs"
)

//go:embed js/* css/* fonts/* licenses.json
var FS embed.FS

// Public returns a sub-filesystem with only the files that should be served via HTTP.
func Public() fs.FS {
	fsys, err := fs.Sub(FS, ".")
	if err != nil {
		panic(err)
	}

	return fsys
}
