// Package web provides the HTTP server for the kvelmo web UI.
package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/valksor/kvelmo/pkg/socket"
)

// WorktreeCreator creates worktree sockets on-demand.
// This allows the web server to ensure sockets exist before proxying.
type WorktreeCreator interface {
	// GetOrCreateWorktreeSocket returns an existing or new worktree socket.
	// The socket is started automatically if created.
	GetOrCreateWorktreeSocket(projectPath string) (interface{}, error)
}

// Server serves the web UI and proxies WebSocket connections to Unix sockets.
type Server struct {
	httpServer      *http.Server
	listener        net.Listener
	upgrader        websocket.Upgrader
	staticDir       string
	port            int
	allowedOrigins  []string // If empty, only localhost is allowed
	worktreeCreator WorktreeCreator
}

// ServerOption configures the web server.
type ServerOption func(*Server)

// WithAllowedOrigins sets allowed CORS origins.
// If origins is empty or nil, only localhost is allowed (secure default).
// Use []string{"*"} to allow all origins (development only).
func WithAllowedOrigins(origins []string) ServerOption {
	return func(s *Server) {
		s.allowedOrigins = origins
	}
}

// WithWorktreeCreator sets the worktree creator for on-demand socket creation.
// When set, the server will ensure worktree sockets exist before proxying.
func WithWorktreeCreator(creator WorktreeCreator) ServerOption {
	return func(s *Server) {
		s.worktreeCreator = creator
	}
}

// NewServer creates a new web server with the given port.
// Use port 0 for a random available port.
func NewServer(staticDir string, port int, opts ...ServerOption) (*Server, error) {
	addr := fmt.Sprintf("localhost:%d", port)
	ln, err := net.Listen("tcp", addr) //nolint:noctx // Context cancellation handled via server shutdown
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	// Get actual port (important when port=0)
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()

		return nil, errors.New("unexpected listener address type")
	}
	actualPort := tcpAddr.Port

	s := &Server{
		staticDir: staticDir,
		listener:  ln,
		port:      actualPort,
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Configure CORS check
	s.upgrader = websocket.Upgrader{
		CheckOrigin: s.checkOrigin,
	}

	mux := http.NewServeMux()

	// WebSocket proxy endpoints
	mux.HandleFunc("/ws/global", s.handleGlobalWS)
	mux.HandleFunc("/ws/worktree/", s.handleWorktreeWS)

	// Static file serving (SPA)
	if staticDir != "" {
		mux.HandleFunc("/", s.handleStatic)
	}

	s.httpServer = &http.Server{
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second, // generous for WebSocket upgrades
		IdleTimeout:       120 * time.Second,
	}

	return s, nil
}

// checkOrigin validates WebSocket connection origins.
// By default, only localhost is allowed (secure default).
func (s *Server) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // No origin header = same-origin request
	}

	// Check if all origins are allowed
	for _, allowed := range s.allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
	}

	// Default: allow localhost only (http or https, parse URL to prevent subdomain bypass)
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()

	return (u.Scheme == "http" || u.Scheme == "https") && (host == "localhost" || host == "127.0.0.1" || host == "::1")
}

// securityHeaders wraps a handler with security headers.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// Start starts the HTTP server using the pre-bound listener.
func (s *Server) Start() error {
	return s.httpServer.Serve(s.listener)
}

// Port returns the actual port the server is listening on.
func (s *Server) Port() int {
	return s.port
}

// URL returns the full URL to access the server.
func (s *Server) URL() string {
	return fmt.Sprintf("http://localhost:%d", s.port)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	fullPath := filepath.Join(s.staticDir, path)

	// Check if file exists
	http.ServeFile(w, r, fullPath)
}

// handleGlobalWS proxies WebSocket connections to the global Unix socket.
func (s *Server) handleGlobalWS(w http.ResponseWriter, r *http.Request) {
	wsConn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = wsConn.Close() }()

	// Connect to global Unix socket
	sockPath := socket.GlobalSocketPath()
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	unixConn, err := dialer.DialContext(r.Context(), "unix", sockPath)
	if err != nil {
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"error":"failed to connect to global socket: %v"}`, err)))

		return
	}
	defer func() { _ = unixConn.Close() }()

	// Proxy bidirectionally
	s.proxyConnections(wsConn, unixConn)
}

// handleWorktreeWS proxies WebSocket connections to a worktree Unix socket.
func (s *Server) handleWorktreeWS(w http.ResponseWriter, r *http.Request) {
	// Extract worktree ID from path: /ws/worktree/{id}
	// The ID is a URL-encoded filesystem path (e.g., %2Fprivate%2Fvar%2F...).
	// We use RawPath to get the encoded version, then decode it ourselves,
	// because URL.Path auto-decodes %2F to / which breaks our parsing.
	const prefix = "/ws/worktree/"
	rawPath := r.URL.RawPath
	if rawPath == "" {
		rawPath = r.URL.Path // Fallback if RawPath not set
	}
	if !strings.HasPrefix(rawPath, prefix) {
		http.Error(w, "invalid path", http.StatusBadRequest)

		return
	}
	encodedID := strings.TrimPrefix(rawPath, prefix)
	if encodedID == "" {
		http.Error(w, "missing worktree id", http.StatusBadRequest)

		return
	}
	worktreeID, err := url.PathUnescape(encodedID)
	if err != nil {
		http.Error(w, "invalid worktree id encoding", http.StatusBadRequest)

		return
	}

	// Ensure worktree socket exists (create on-demand if creator is configured)
	sockPath := socket.WorktreeSocketPath(worktreeID)
	if s.worktreeCreator != nil {
		if _, err := s.worktreeCreator.GetOrCreateWorktreeSocket(worktreeID); err != nil {
			slog.Error("failed to create worktree socket", "worktree_id", worktreeID, "error", err)
			http.Error(w, "failed to create worktree socket", http.StatusInternalServerError)

			return
		}
	}

	wsConn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = wsConn.Close() }()

	// Connect to worktree Unix socket
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	unixConn, err := dialer.DialContext(r.Context(), "unix", sockPath)
	if err != nil {
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"error":"failed to connect to worktree socket: %v"}`, err)))

		return
	}
	defer func() { _ = unixConn.Close() }()

	// Proxy bidirectionally
	s.proxyConnections(wsConn, unixConn)
}

// proxyConnections handles bidirectional proxying between WebSocket and Unix socket.
// When either connection fails, both are closed to prevent goroutine leaks.
func (s *Server) proxyConnections(wsConn *websocket.Conn, unixConn net.Conn) {
	var wg sync.WaitGroup
	wg.Add(3)

	// closeOnce ensures both connections are closed exactly once when either side fails
	var closeOnce sync.Once
	closeAll := func() {
		closeOnce.Do(func() {
			_ = wsConn.Close()
			_ = unixConn.Close()
		})
	}

	// Keepalive: send a ping every 30s and close if pong not received within 60s.
	// WriteControl is concurrent-safe per gorilla/websocket docs.
	_ = wsConn.SetReadDeadline(time.Now().Add(60 * time.Second))
	wsConn.SetPongHandler(func(string) error {
		return wsConn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})
	go func() {
		defer func() { closeAll(); wg.Done() }()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			deadline := time.Now().Add(10 * time.Second)
			if err := wsConn.WriteControl(websocket.PingMessage, nil, deadline); err != nil {
				return
			}
		}
	}()

	// WebSocket -> Unix socket
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "ws->unix proxy panic: %v\n", r)
			}
			closeAll()
			wg.Done()
		}()
		for {
			_, msg, err := wsConn.ReadMessage()
			if err != nil {
				return
			}
			_, err = unixConn.Write(msg)
			if err != nil {
				return
			}
		}
	}()

	// Unix socket -> WebSocket
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "unix->ws proxy panic: %v\n", r)
			}
			closeAll()
			wg.Done()
		}()
		buf := make([]byte, 4096)
		for {
			n, err := unixConn.Read(buf)
			if err != nil {
				return
			}
			err = wsConn.WriteMessage(websocket.TextMessage, buf[:n])
			if err != nil {
				return
			}
		}
	}()

	wg.Wait()
}
