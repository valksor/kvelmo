// Package server provides an HTTP server for the Mehrhof web UI.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/eventbus"
)

// Mode represents the server operating mode.
type Mode int

const (
	// ModeProject runs the server for a single project.
	ModeProject Mode = iota
	// ModeGlobal runs the server showing all discovered projects.
	ModeGlobal
)

// Config holds server configuration.
type Config struct {
	// Port specifies the port to listen on (0 = random available port).
	Port int
	// Host specifies the host to bind to (default: "localhost").
	Host string
	// Mode specifies whether to run in project or global mode.
	Mode Mode
	// Conductor is the conductor instance for project mode (nil for global mode).
	Conductor *conductor.Conductor
	// EventBus is the event bus for real-time updates.
	EventBus *eventbus.Bus
	// WorkspaceRoot is the root directory of the workspace (for project mode).
	WorkspaceRoot string
	// AuthStore is the authentication store (nil means no auth required).
	AuthStore *storage.AuthStore
}

// Server is the Mehrhof web UI HTTP server.
type Server struct {
	config     Config
	httpServer *http.Server
	listener   net.Listener
	router     http.Handler
	sessions   *sessionStore
	templates  *Templates

	mu                  sync.RWMutex
	running             bool
	actualPort          int
	startedInGlobalMode bool // Tracks if server originally started in global mode
}

// New creates a new server with the given configuration.
func New(cfg Config) (*Server, error) {
	// Create EventBus if not provided (for global mode)
	// In project mode, the conductor provides the EventBus, but in global mode
	// we need our own for SSE connections to work properly.
	if cfg.EventBus == nil {
		cfg.EventBus = eventbus.NewBus()
	}

	s := &Server{
		config:              cfg,
		sessions:            newSessionStore(),
		startedInGlobalMode: cfg.Mode == ModeGlobal,
	}

	// Load templates
	templates, err := LoadTemplates()
	if err != nil {
		slog.Warn("failed to load templates, using fallback UI", "error", err)
	} else {
		s.templates = templates
	}

	// Create router
	s.router = s.setupRouter()

	return s, nil
}

// Start starts the server and blocks until the context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	// Create listener
	host := s.config.Host
	if host == "" {
		host = "localhost"
	}
	addr := fmt.Sprintf("%s:%d", host, s.config.Port)
	lc := net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	s.listener = listener

	// Extract actual port (important for random port allocation)
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return fmt.Errorf("unexpected listener address type: %T", listener.Addr())
	}

	// Create HTTP server
	s.httpServer = &http.Server{
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.mu.Lock()
	s.actualPort = tcpAddr.Port
	s.running = true
	s.mu.Unlock()

	// Log server start
	slog.Info("server started", "port", s.actualPort, "mode", s.modeString())

	// Handle graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpServer.Serve(listener)
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: %w", err)
		}
	case <-ctx.Done():
		// Graceful shutdown - intentionally use Background() since parent context is cancelled
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil { //nolint:contextcheck // shutdownCtx is derived from Background, intentionally
			slog.Warn("server shutdown error", "error", err)
		}
	}

	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	slog.Info("server stopped")

	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()

	if !s.running {
		s.mu.Unlock()

		return nil
	}

	if s.httpServer != nil {
		hs := s.httpServer
		s.mu.Unlock()

		return hs.Shutdown(ctx)
	}

	s.mu.Unlock()

	return nil
}

// Port returns the actual port the server is listening on.
// Returns 0 if the server is not running.
func (s *Server) Port() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.actualPort
}

// URL returns the full URL to access the server.
func (s *Server) URL() string {
	host := s.config.Host
	if host == "" || host == "0.0.0.0" {
		host = "localhost"
	}

	return fmt.Sprintf("http://%s:%d", host, s.Port())
}

// IsRunning returns true if the server is running.
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.running
}

// modeString returns a human-readable string for the server mode.
func (s *Server) modeString() string {
	switch s.config.Mode {
	case ModeProject:
		return "project"
	case ModeGlobal:
		return "global"
	default:
		return "unknown"
	}
}

// isLocalRequest returns true if the request originates from localhost.
// Used to determine whether to show sensitive data like API tokens.
func isLocalRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If we can't parse, assume it's the host without port
		host = r.RemoteAddr
	}

	// Check for loopback addresses
	if host == "127.0.0.1" || host == "::1" || host == "localhost" {
		return true
	}

	// Also check if the IP is a loopback
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return true
	}

	return false
}

// switchToProject switches the server from global mode to project mode.
// This updates the config, creates a conductor for the project, and rebuilds the router.
func (s *Server) switchToProject(projectPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a new conductor for the project
	cond, err := conductor.New(conductor.WithWorkDir(projectPath))
	if err != nil {
		return fmt.Errorf("create conductor: %w", err)
	}

	// Update server config
	s.config.Mode = ModeProject
	s.config.WorkspaceRoot = projectPath
	s.config.Conductor = cond

	// Rebuild router with new mode
	s.router = s.setupRouter()

	// Update the http server's handler
	if s.httpServer != nil {
		s.httpServer.Handler = s.router
	}

	return nil
}

// switchToGlobal switches the server back to global mode.
func (s *Server) switchToGlobal() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update server config
	s.config.Mode = ModeGlobal
	s.config.WorkspaceRoot = ""
	s.config.Conductor = nil

	// Rebuild router with global mode
	s.router = s.setupRouter()

	// Update the http server's handler
	if s.httpServer != nil {
		s.httpServer.Handler = s.router
	}
}

// canSwitchProject returns true if the server can switch between projects.
func (s *Server) canSwitchProject() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.startedInGlobalMode
}
