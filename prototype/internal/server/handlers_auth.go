package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/conductor/commands"
	"github.com/valksor/go-mehrhof/internal/server/views"
	"github.com/valksor/go-toolkit/eventbus"
)

// selectProjectRequest is the JSON request body for project selection.
type selectProjectRequest struct {
	Path string `json:"path"`
}

// handleSelectProject switches from global mode to project mode.
func (s *Server) handleSelectProject(w http.ResponseWriter, r *http.Request) {
	var projectPath string

	// Check if JSON request or form-encoded
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		var req selectProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())

			return
		}

		projectPath = req.Path
	} else {
		// Form-encoded
		if err := r.ParseForm(); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid form data")

			return
		}

		projectPath = r.FormValue("path")
	}

	if projectPath == "" {
		s.writeError(w, http.StatusBadRequest, "project path is required")

		return
	}

	// Switch to project mode
	if err := s.switchToProject(r.Context(), projectPath); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to switch project: "+err.Error())

		return
	}

	slog.Info("switched to project mode", "path", projectPath)

	// Return JSON for API clients, redirect for browser
	if contentType == "application/json" {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"path":    projectPath,
		})

		return
	}

	// Redirect for browser
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleSwitchProject switches back to global mode to pick another project.
func (s *Server) handleSwitchProject(w http.ResponseWriter, r *http.Request) {
	s.switchToGlobal()

	slog.Info("switched back to global mode")

	s.writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

// publishStateChangeEvent publishes an SSE event for workflow state changes.
// This ensures the UI updates immediately when entering a new workflow phase.
// Includes progress_phase for context-aware display (e.g., "Planned" instead of "idle").
func (s *Server) publishStateChangeEvent(ctx context.Context) {
	if s.config.EventBus == nil || s.config.Conductor == nil {
		return
	}

	status, err := s.config.Conductor.Status(ctx)
	if err != nil {
		// No active task (e.g., after finish/abandon) — publish idle transition
		// so the frontend's SSE handler refreshes queries immediately.
		s.config.EventBus.PublishRaw(eventbus.Event{
			Type: views.EventWorkflowStateChanged,
			Data: map[string]any{
				"state":          "idle",
				"task_id":        "",
				"progress_phase": "",
			},
		})

		return
	}

	// Compute progress phase for context-aware state display
	// This allows the frontend to show "Planned" when state is "idle" but specs exist
	var progressPhase string
	if ws := s.config.Conductor.GetWorkspace(); ws != nil && status.TaskID != "" {
		phase := commands.ComputeProgressPhase(ws, status.TaskID)
		progressPhase = string(phase)
	}

	s.config.EventBus.PublishRaw(eventbus.Event{
		Type: views.EventWorkflowStateChanged,
		Data: map[string]any{
			"state":          status.State,
			"task_id":        status.TaskID,
			"progress_phase": progressPhase,
		},
	})
}
