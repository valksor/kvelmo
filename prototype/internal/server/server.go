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

	"github.com/valksor/go-mehrhof/internal/automation"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/library"
	"github.com/valksor/go-mehrhof/internal/registration"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/taskrunner"
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

// activeOperation tracks a cancellable operation for a session.
type activeOperation struct {
	cancel    context.CancelFunc
	operation string // Command name being executed
	startedAt time.Time
}

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
	// APIOnly specifies whether to run in API-only mode (no web UI).
	APIOnly bool
}

// Server is the Mehrhof web UI HTTP server.
type Server struct {
	config     Config
	httpServer *http.Server
	listener   net.Listener
	router     http.Handler
	sessions   *sessionStore

	mu                  sync.RWMutex
	running             bool
	actualPort          int
	startedInGlobalMode bool // Tracks if server originally started in global mode

	// Task registry for parallel task tracking
	taskRegistry *taskrunner.Registry

	// Operation tracking for interactive mode cancellation
	opMu      sync.RWMutex
	activeOps map[string]*activeOperation // sessionID -> active operation

	// Automation for webhook processing
	automation       *automation.Automation
	automationConfig *storage.AutomationSettings

	// Shared-only library for global mode (when no project selected)
	sharedLibrary *library.Manager
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
		activeOps:           make(map[string]*activeOperation),
	}

	// Initialize shared-only library for global mode (allows accessing shared collections
	// even when no project is selected and conductor is nil)
	if cfg.Mode == ModeGlobal {
		sharedLib, err := library.NewManager(context.Background(), "")
		if err != nil {
			slog.Warn("failed to initialize shared library", "error", err)
		} else {
			s.sharedLibrary = sharedLib
		}
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

	// Stop the session cleanup goroutine
	if s.sessions != nil {
		s.sessions.stop()
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

// switchToProject switches the server from global mode to project mode.
// This updates the config, creates a conductor for the project, and rebuilds the router.
func (s *Server) switchToProject(ctx context.Context, projectPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a new conductor for the project
	cond, err := conductor.New(conductor.WithWorkDir(projectPath))
	if err != nil {
		return fmt.Errorf("create conductor: %w", err)
	}

	// Register standard providers
	registration.RegisterStandardProviders(cond)

	// Register standard agents
	if err := registration.RegisterStandardAgents(cond); err != nil {
		return fmt.Errorf("register agents: %w", err)
	}

	// Initialize the conductor to open the workspace
	if err := cond.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize conductor: %w", err)
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

// Operation tracking helpers for interactive mode cancellation.

// getSessionID extracts a session identifier from the request.
// Returns session cookie value if present, falls back to X-Request-ID header.
func (s *Server) getSessionID(r *http.Request) string {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		return cookie.Value
	}
	// Fall back to X-Request-ID header for API clients
	if reqID := r.Header.Get("X-Request-ID"); reqID != "" {
		return reqID
	}

	return ""
}

// registerOperation tracks an active operation for a session, enabling cancellation.
func (s *Server) registerOperation(sessionID string, cancel context.CancelFunc, op string) {
	if sessionID == "" {
		return
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()
	s.activeOps[sessionID] = &activeOperation{
		cancel:    cancel,
		operation: op,
		startedAt: time.Now(),
	}
}

// unregisterOperation removes operation tracking for a session.
func (s *Server) unregisterOperation(sessionID string) {
	if sessionID == "" {
		return
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()
	delete(s.activeOps, sessionID)
}

// cancelOperation cancels the active operation for a session if one exists.
// Returns the operation name and true if an operation was cancelled.
func (s *Server) cancelOperation(sessionID string) (string, bool) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	op, exists := s.activeOps[sessionID]
	if !exists {
		return "", false
	}
	op.cancel()

	return op.operation, true
}
