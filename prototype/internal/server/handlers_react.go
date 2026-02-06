package server

import (
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/valksor/go-mehrhof/internal/server/static"
)

// handleReactApp serves the React SPA.
// It serves static files if they exist in the React app bundle (CSS, JS, images),
// otherwise serves index.html for client-side routing.
// Public routes (/login, /logout) are served without auth check.
func (s *Server) handleReactApp(w http.ResponseWriter, r *http.Request) {
	// API routes should never fall through to the SPA - return 404
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)

		return
	}

	// Public routes that don't require authentication
	// publicPaths := map[string]bool{
	//	// DISABLED: remote serve temporarily unavailable
	//	// "/login":  true,
	//	//"/logout": true,
	//}

	// Check auth if auth store is configured (non-localhost mode)
	// Skip auth check for public paths and static assets
	// isStaticAsset := strings.HasPrefix(r.URL.Path, "/assets/") ||
	//	strings.HasSuffix(r.URL.Path, ".svg") ||
	//	strings.HasSuffix(r.URL.Path, ".ico") ||
	//	strings.HasSuffix(r.URL.Path, ".png")

	// if s.config.AuthStore != nil && !publicPaths[r.URL.Path] && !isStaticAsset {
	//	session := s.getSessionFromRequest(r)
	//	if session == nil {
	//		// Redirect to login with return URL
	//		redirectURL := "/login?next=" + url.QueryEscape(r.URL.Path)
	//		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	//
	//		return
	//	}
	//}

	// Try to serve static file from React app bundle
	reactFS := static.ReactApp()
	reqPath := strings.TrimPrefix(r.URL.Path, "/")
	if reqPath == "" {
		reqPath = "index.html"
	}

	// Check if file exists and is not a directory
	if f, err := reactFS.Open(reqPath); err == nil {
		if info, statErr := f.Stat(); statErr == nil && !info.IsDir() {
			defer func() { _ = f.Close() }()
			s.serveFile(w, reqPath, f)

			return
		}
		_ = f.Close()
	}

	// Fall back to index.html for SPA routing
	f, err := reactFS.Open("index.html")
	if err != nil {
		http.Error(w, "React app not found", http.StatusNotFound)

		return
	}
	defer func() { _ = f.Close() }()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if _, err := io.Copy(w, f); err != nil {
		http.Error(w, "Failed to serve React app", http.StatusInternalServerError)
	}
}

// serveFile serves a file with appropriate content type.
func (s *Server) serveFile(w http.ResponseWriter, name string, f fs.File) {
	// Determine content type from extension
	ext := strings.ToLower(path.Ext(name))
	contentType := "application/octet-stream"

	switch ext {
	case ".html":
		contentType = "text/html; charset=utf-8"
	case ".css":
		contentType = "text/css; charset=utf-8"
	case ".js":
		contentType = "application/javascript; charset=utf-8"
	case ".json":
		contentType = "application/json; charset=utf-8"
	case ".svg":
		contentType = "image/svg+xml"
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".ico":
		contentType = "image/x-icon"
	case ".woff":
		contentType = "font/woff"
	case ".woff2":
		contentType = "font/woff2"
	case ".ttf":
		contentType = "font/ttf"
	}

	w.Header().Set("Content-Type", contentType)

	// Set cache headers for immutable assets (hashed filenames)
	if strings.Contains(name, "-") && (ext == ".js" || ext == ".css") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}

	if _, err := io.Copy(w, f); err != nil {
		http.Error(w, "Failed to serve file", http.StatusInternalServerError)
	}
}
