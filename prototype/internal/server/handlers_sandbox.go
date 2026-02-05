package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"

	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/sandbox"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// handleSandboxStatus returns the current sandbox status.
// GET /api/v1/sandbox/status.
func (s *Server) handleSandboxStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var cfg *storage.WorkspaceConfig
	var loadErr string

	// Load workspace config
	if s.config.Conductor != nil {
		ws := s.config.Conductor.GetWorkspace()
		if ws != nil {
			var err error
			cfg, err = ws.LoadConfig()
			if err != nil {
				loadErr = "failed to load config: " + err.Error()
			}
		}
	}

	if cfg == nil {
		cfg = storage.NewDefaultWorkspaceConfig()
	}

	// Build status response
	status := sandbox.Status{
		Enabled:   cfg.Sandbox != nil && cfg.Sandbox.Enabled,
		Platform:  runtime.GOOS,
		Active:    s.isSandboxActive(),
		Network:   cfg.Sandbox != nil && cfg.Sandbox.Network,
		Supported: sandbox.Supported(),
	}

	// Add load error if any (non-blocking)
	if loadErr != "" {
		status.Profile = loadErr // Using Profile field to convey error info
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		slog.Error("failed to encode sandbox status", "error", err)
	}
}

// handleSandboxEnable enables sandbox for the current session.
// POST /api/v1/sandbox/enable.
func (s *Server) handleSandboxEnable(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")

		return
	}

	// Get workspace
	var ws *storage.Workspace
	if s.config.Conductor != nil {
		ws = s.config.Conductor.GetWorkspace()
	}
	if ws == nil {
		s.writeError(w, http.StatusInternalServerError, "workspace not available")

		return
	}

	// Load and update config
	cfg, err := ws.LoadConfig()
	if err != nil {
		cfg = storage.NewDefaultWorkspaceConfig()
	}

	// Enable sandbox
	if cfg.Sandbox == nil {
		cfg.Sandbox = &storage.SandboxSettings{}
	}
	cfg.Sandbox.Enabled = true

	// Save config
	if err := ws.SaveConfig(cfg); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())

		return
	}

	// Publish SSE event
	s.publishSandboxStatus(true)

	// Return updated status
	status := sandbox.Status{
		Enabled:  true,
		Platform: runtime.GOOS,
		Active:   false, // Will be active when next task runs
		Network:  cfg.Sandbox.Network,
	}
	if err := json.NewEncoder(w).Encode(status); err != nil {
		slog.Error("failed to encode sandbox status", "error", err)
	}
}

// handleSandboxDisable disables sandbox for the current session.
// POST /api/v1/sandbox/disable.
func (s *Server) handleSandboxDisable(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")

		return
	}

	// Get workspace
	var ws *storage.Workspace
	if s.config.Conductor != nil {
		ws = s.config.Conductor.GetWorkspace()
	}
	if ws == nil {
		s.writeError(w, http.StatusInternalServerError, "workspace not available")

		return
	}

	// Load and update config
	cfg, err := ws.LoadConfig()
	if err != nil {
		cfg = storage.NewDefaultWorkspaceConfig()
	}

	// Disable sandbox
	if cfg.Sandbox != nil {
		cfg.Sandbox.Enabled = false
	}

	// Save config
	if err := ws.SaveConfig(cfg); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())

		return
	}

	// Publish SSE event
	s.publishSandboxStatus(false)

	// Return updated status
	status := sandbox.Status{
		Enabled:  false,
		Platform: runtime.GOOS,
		Active:   false,
		Network:  true,
	}
	if err := json.NewEncoder(w).Encode(status); err != nil {
		slog.Error("failed to encode sandbox status", "error", err)
	}
}

// isSandboxActive checks if sandbox is currently active for a running task.
func (s *Server) isSandboxActive() bool {
	// Check if there's an active task with sandbox enabled
	// This is a simple check - in production, you'd track this in the task state
	if s.config.Conductor != nil {
		activeTask := s.config.Conductor.GetActiveTask()
		if activeTask != nil && activeTask.State == "implementing" {
			// Check if sandbox is enabled in config
			ws := s.config.Conductor.GetWorkspace()
			if ws != nil {
				if cfg, err := ws.LoadConfig(); err == nil && cfg.Sandbox != nil && cfg.Sandbox.Enabled {
					return true
				}
			}
		}
	}

	return false
}

// publishSandboxStatus publishes a sandbox status change event via SSE.
func (s *Server) publishSandboxStatus(enabled bool) {
	if s.config.EventBus != nil {
		s.config.EventBus.Publish(events.SandboxStatusChangedEvent{
			Enabled:  enabled,
			Active:   false, // Will be set when task runs
			Platform: runtime.GOOS,
		})
	}
}

// isSandboxEnabled returns whether sandbox is enabled in the config.
func (s *Server) isSandboxEnabled() bool {
	if s.config.Conductor != nil {
		ws := s.config.Conductor.GetWorkspace()
		if ws != nil {
			if cfg, err := ws.LoadConfig(); err == nil && cfg.Sandbox != nil {
				return cfg.Sandbox.Enabled
			}
		}
	}

	return false
}
