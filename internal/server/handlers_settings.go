package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// loadProjectConfig loads config for a specific project by ID.
// Returns the config and workspace, or an error message.
func loadProjectConfig(ctx context.Context, projectID string) (*storage.WorkspaceConfig, *storage.Workspace, string) {
	// Load project registry to get the project's repo path
	registry, err := storage.LoadRegistry()
	if err != nil {
		return nil, nil, "failed to load project registry: " + err.Error()
	}

	project, ok := registry.Projects[projectID]
	if !ok {
		return nil, nil, "project not found in registry"
	}

	// Open workspace using the project's repo path
	ws, err := storage.OpenWorkspace(ctx, project.Path, nil)
	if err != nil {
		return nil, nil, "failed to open workspace: " + err.Error()
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		return nil, nil, "failed to load config: " + err.Error()
	}

	return cfg, ws, ""
}

// handleSaveSettings saves settings from JSON submission.
func (s *Server) handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	selectedProject := r.URL.Query().Get("project")

	var ws *storage.Workspace
	var cfg *storage.WorkspaceConfig
	var err error

	// Global mode with project selection
	if s.config.Mode == ModeGlobal && selectedProject != "" {
		var loadErr string
		_, ws, loadErr = loadProjectConfig(r.Context(), selectedProject)
		if ws == nil {
			s.writeError(w, http.StatusNotFound, "project not found: "+loadErr)

			return
		}
	} else if s.config.Conductor != nil {
		ws = s.config.Conductor.GetWorkspace()
	}

	if ws == nil {
		if s.config.Mode == ModeGlobal {
			s.writeError(w, http.StatusBadRequest, "select a project first")

			return
		}
		// Project mode: open workspace directly using WorkspaceRoot
		// This allows saving settings even when workspace isn't initialized yet
		if s.config.WorkspaceRoot == "" {
			s.writeError(w, http.StatusServiceUnavailable, "workspace root not configured")

			return
		}
		ws, err = storage.OpenWorkspace(r.Context(), s.config.WorkspaceRoot, nil)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to open workspace: "+err.Error())

			return
		}
	}

	// Load existing config to merge with, or use defaults if not initialized
	cfg, err = ws.LoadConfig()
	if err != nil {
		// If config doesn't exist, use defaults (expected for uninitialized workspace)
		cfg = storage.NewDefaultWorkspaceConfig()
	}

	// JSON submission only
	if err := json.NewDecoder(r.Body).Decode(cfg); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())

		return
	}

	// Save config
	if err := ws.SaveConfig(cfg); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())

		return
	}

	// Reinitialize conductor to pick up the newly created/updated workspace
	if s.config.Conductor != nil {
		if err := s.config.Conductor.Initialize(r.Context()); err != nil {
			// Log but don't fail - settings were saved, conductor refresh is secondary
			slog.Warn("failed to reinitialize conductor after saving settings", "error", err)
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "Settings saved successfully",
	})
}
