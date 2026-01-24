package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// handleWorkflowContinue resumes work on the active task with optional auto-execution.
func (s *Server) handleWorkflowContinue(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req continueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to no auto if body is empty
		req = continueRequest{}
	}

	// Check for active task
	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		s.writeJSON(w, http.StatusOK, continueResponse{
			Success:     false,
			Message:     "no active task",
			NextActions: []string{"POST /api/v1/workflow/start"},
		})

		return
	}

	// Get status
	//nolint:contextcheck // Status() doesn't accept context; internal implementation issue
	status, err := s.config.Conductor.Status()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to get status: "+err.Error())

		return
	}

	state := workflow.State(status.State)
	nextActions := getNextActionsForState(state, status.Specifications)

	// Auto-execute next step if requested
	if req.Auto {
		action, err := executeNextStep(r.Context(), s.config.Conductor, status)
		if err != nil {
			s.writeJSON(w, http.StatusOK, continueResponse{
				Success:     false,
				State:       status.State,
				Message:     err.Error(),
				NextActions: nextActions,
			})

			return
		}

		// Get updated status after action
		//nolint:contextcheck // Status() doesn't accept context; internal implementation issue
		updatedStatus, _ := s.config.Conductor.Status()
		if updatedStatus != nil {
			state = workflow.State(updatedStatus.State)
			nextActions = getNextActionsForState(state, updatedStatus.Specifications)
		}

		s.writeJSON(w, http.StatusOK, continueResponse{
			Success:     true,
			State:       string(state),
			Action:      action,
			NextActions: nextActions,
			Message:     "auto-executed: " + action,
		})

		return
	}

	// Return status and suggestions without auto-execution
	s.writeJSON(w, http.StatusOK, continueResponse{
		Success:     true,
		State:       status.State,
		NextActions: nextActions,
		Message:     "task resumed",
	})
}

// executeNextStep determines and executes the next logical workflow step.
// Returns the action taken or an error.
func executeNextStep(ctx context.Context, cond *conductor.Conductor, status *conductor.TaskStatus) (string, error) {
	switch workflow.State(status.State) {
	case workflow.StateIdle:
		if status.Specifications == 0 {
			if err := cond.Plan(ctx); err != nil {
				return "", err
			}

			return "plan", nil
		}
		if err := cond.Implement(ctx); err != nil {
			return "", err
		}

		return "implement", nil

	case workflow.StatePlanning:
		if err := cond.Implement(ctx); err != nil {
			return "", err
		}

		return "implement", nil

	case workflow.StateImplementing, workflow.StateReviewing:
		return "none", nil // Cannot auto-continue, user should finish

	case workflow.StateDone:
		return "none", nil

	case workflow.StateFailed, workflow.StateWaiting:
		return "none", nil // User intervention required

	case workflow.StateCheckpointing, workflow.StateReverting, workflow.StateRestoring:
		return "none", nil // Internal states, wait for completion
	}

	return "none", nil
}

// getNextActionsForState returns suggested next actions based on workflow state.
func getNextActionsForState(state workflow.State, specifications int) []string {
	switch state {
	case workflow.StateIdle:
		if specifications == 0 {
			return []string{
				"POST /api/v1/workflow/plan",
				"POST /api/v1/tasks/{id}/notes",
			}
		}

		return []string{
			"POST /api/v1/workflow/implement",
			"POST /api/v1/workflow/plan",
			"POST /api/v1/tasks/{id}/notes",
		}

	case workflow.StatePlanning:
		return []string{
			"POST /api/v1/workflow/implement",
			"POST /api/v1/tasks/{id}/notes",
		}

	case workflow.StateImplementing:
		return []string{
			"POST /api/v1/workflow/implement",
			"POST /api/v1/workflow/undo",
			"POST /api/v1/workflow/finish",
			"POST /api/v1/tasks/{id}/notes",
		}

	case workflow.StateReviewing:
		return []string{
			"POST /api/v1/workflow/finish",
			"POST /api/v1/workflow/implement",
		}

	case workflow.StateFailed:
		return []string{
			"GET /api/v1/task",
			"POST /api/v1/workflow/implement",
			"POST /api/v1/tasks/{id}/notes",
		}

	case workflow.StateWaiting:
		return []string{
			"POST /api/v1/workflow/answer",
		}

	case workflow.StateDone:
		return []string{
			"POST /api/v1/workflow/start",
		}

	case workflow.StateCheckpointing, workflow.StateReverting, workflow.StateRestoring:
		return []string{
			"GET /api/v1/task",
		}
	}

	return []string{
		"GET /api/v1/task",
		"POST /api/v1/tasks/{id}/notes",
	}
}

// handleWorkflowAuto runs a complete automation cycle.
func (s *Server) handleWorkflowAuto(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req autoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.Ref == "" {
		s.writeError(w, http.StatusBadRequest, "ref is required")

		return
	}

	// Check for existing active task
	if s.config.Conductor.GetActiveTask() != nil {
		s.writeError(w, http.StatusConflict, "task already active; use abandon first or status for details")

		return
	}

	// Set defaults
	maxRetries := req.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}
	qualityTarget := req.QualityTarget
	if qualityTarget == "" {
		qualityTarget = "quality"
	}

	// Build auto options
	opts := conductor.AutoOptions{
		QualityTarget: qualityTarget,
		MaxRetries:    maxRetries,
		SquashMerge:   !req.NoSquash,
		DeleteBranch:  !req.NoDelete,
		TargetBranch:  req.TargetBranch,
		Push:          !req.NoPush,
	}

	// Skip quality if requested
	if req.NoQuality {
		opts.MaxRetries = 0
	}

	// Run the full auto cycle
	result, err := s.config.Conductor.RunAuto(r.Context(), req.Ref, opts)

	resp := autoResponse{
		Success:         err == nil,
		PlanningDone:    result.PlanningDone,
		ImplementDone:   result.ImplementDone,
		QualityAttempts: result.QualityAttempts,
		QualityPassed:   result.QualityPassed,
		FinishDone:      result.FinishDone,
		FailedAt:        result.FailedAt,
	}

	if err != nil {
		resp.Error = err.Error()
	}

	s.writeJSON(w, http.StatusOK, resp)
}
