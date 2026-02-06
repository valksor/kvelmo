package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/server/views"
	"github.com/valksor/go-toolkit/eventbus"
)

// handleWorkflowFinish completes the task.
func (s *Server) handleWorkflowFinish(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req finishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to no merge/PR if body is empty
		req = finishRequest{}
	}

	opts := conductor.FinishOptions{
		SquashMerge:  req.SquashMerge,
		DeleteBranch: req.DeleteBranch,
		TargetBranch: req.TargetBranch,
		PushAfter:    req.PushAfter,
		ForceMerge:   req.ForceMerge,
		DraftPR:      req.DraftPR,
		PRTitle:      req.PRTitle,
		PRBody:       req.PRBody,
	}

	if err := s.config.Conductor.Finish(r.Context(), opts); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to finish task: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "task finished",
	})
}

// handleWorkflowUndo undoes to the previous checkpoint.
func (s *Server) handleWorkflowUndo(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	if err := s.config.Conductor.Undo(r.Context()); err != nil {
		s.writeError(w, http.StatusInternalServerError, "undo failed: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "undo completed",
	})
}

// handleWorkflowRedo redoes to the next checkpoint.
func (s *Server) handleWorkflowRedo(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	if err := s.config.Conductor.Redo(r.Context()); err != nil {
		s.writeError(w, http.StatusInternalServerError, "redo failed: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "redo completed",
	})
}

// handleWorkflowAnswer submits an answer to a pending agent question.
// This saves the answer to notes and clears the pending question,
// allowing the next plan/implement call to continue with the answer.
func (s *Server) handleWorkflowAnswer(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Handle both form and JSON submissions (HTMX forms send form-urlencoded)
	var answer string
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid form data: "+err.Error())

			return
		}
		answer = r.FormValue("answer")
	} else {
		var req struct {
			Answer string `json:"answer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

			return
		}
		answer = req.Answer
	}

	if answer == "" {
		s.writeError(w, http.StatusBadRequest, "answer is required")

		return
	}

	// Use conductor method to answer and transition state machine
	if err := s.config.Conductor.AnswerQuestion(r.Context(), answer); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to answer: "+err.Error())

		return
	}

	// Publish state change event to trigger UI refresh (actions, task card)
	if s.config.EventBus != nil {
		s.config.EventBus.PublishRaw(eventbus.Event{
			Type: views.EventWorkflowStateChanged,
			Data: map[string]any{
				"state":   "idle",
				"action":  "answer_submitted",
				"message": "Question answered",
			},
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "answer submitted",
	})
}

// handleWorkflowAbandon abandons the current task.
func (s *Server) handleWorkflowAbandon(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	opts := conductor.DeleteOptions{
		Force: true, // Skip confirmation in API context
	}

	if err := s.config.Conductor.Delete(r.Context(), opts); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to abandon task: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "task abandoned",
	})
}

// handleWorkflowResume resumes a task paused due to budget limits.
func (s *Server) handleWorkflowResume(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	if err := s.config.Conductor.ResumePaused(r.Context()); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
