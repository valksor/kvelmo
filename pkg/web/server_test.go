package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewServer(t *testing.T) {
	srv, err := NewServer("", 0)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
	_ = srv.Shutdown(context.Background())
}

func TestServerStaticFallback(t *testing.T) {
	// With no static dir, root should not be handled
	srv, err := NewServer("", 0)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer func() { _ = srv.Shutdown(context.Background()) }()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.httpServer.Handler.ServeHTTP(w, req)

	// Without static dir, should get 404
	if w.Code != http.StatusNotFound {
		t.Errorf("/ without static dir status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestServerWorktreeWSPathParsing(t *testing.T) {
	srv, err := NewServer("", 0)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer func() { _ = srv.Shutdown(context.Background()) }()

	// Missing worktree ID should fail
	req := httptest.NewRequest(http.MethodGet, "/ws/worktree/", nil)
	w := httptest.NewRecorder()

	srv.httpServer.Handler.ServeHTTP(w, req)

	// Should fail because it's not a WebSocket upgrade
	if w.Code == http.StatusOK {
		t.Error("/ws/worktree/ should not return 200 without WebSocket upgrade")
	}
}

func TestServerPort(t *testing.T) {
	srv, err := NewServer("", 0)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer func() { _ = srv.Shutdown(context.Background()) }()

	port := srv.Port()
	if port <= 0 {
		t.Errorf("Port() = %d, want > 0", port)
	}
}

func TestServerURL(t *testing.T) {
	srv, err := NewServer("", 0)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer func() { _ = srv.Shutdown(context.Background()) }()

	url := srv.URL()
	if url == "" {
		t.Error("URL() should not be empty")
	}
	if !strings.HasPrefix(url, "http://localhost:") {
		t.Errorf("URL() = %q, want prefix http://localhost:", url)
	}
}

func TestServerWorktreeWS_MissingID(t *testing.T) {
	srv, err := NewServer("", 0)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer func() { _ = srv.Shutdown(context.Background()) }()

	// Call handler directly with a short path that produces < 4 split parts
	req := httptest.NewRequest(http.MethodGet, "/ws/worktree", nil)
	w := httptest.NewRecorder()

	srv.handleWorktreeWS(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("handleWorktreeWS short path status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestServerStatic(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a simple index.html in the temp dir
	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte("<html>test</html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv, err := NewServer(tmpDir, 0)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer func() { _ = srv.Shutdown(context.Background()) }()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("/ with static dir status = %d, want %d", w.Code, http.StatusOK)
	}
}
