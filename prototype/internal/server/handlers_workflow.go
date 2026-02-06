package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/eventbus"
)

// handleWorkflowContinue resumes work on the active task with optional auto-execution.
func (s *Server) handleWorkflowContinue(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

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

	status, err := s.config.Conductor.Status(r.Context())
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

		updatedStatus, _ := s.config.Conductor.Status(r.Context())
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

	case workflow.StateFailed, workflow.StateWaiting, workflow.StatePaused:
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
			"POST /api/v1/workflow/question",
			"POST /api/v1/tasks/{id}/notes",
		}

	case workflow.StateImplementing:
		return []string{
			"POST /api/v1/workflow/implement",
			"POST /api/v1/workflow/question",
			"POST /api/v1/workflow/undo",
			"POST /api/v1/workflow/finish",
			"POST /api/v1/tasks/{id}/notes",
		}

	case workflow.StateReviewing:
		return []string{
			"POST /api/v1/workflow/finish",
			"POST /api/v1/workflow/implement",
			"POST /api/v1/workflow/question",
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
	case workflow.StatePaused:
		return []string{
			"POST /api/v1/workflow/resume",
			"GET /api/v1/costs",
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
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

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

// handleWorkflowDiagram returns an SVG workflow diagram showing current state.
func (s *Server) handleWorkflowDiagram(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Get the workflow machine from conductor
	machine := s.config.Conductor.GetMachine()
	if machine == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workflow machine not available")

		return
	}

	// Get current state from machine directly
	currentState := machine.State()

	// Generate SVG diagram with current state highlighted
	opts := workflow.DiagramOptions{
		CurrentState: currentState,
		ShowEvents:   true,
		Compact:      false,
	}

	svg := workflow.SVGDiagram(machine, opts)

	// Set SVG content type
	w.Header().Set("Content-Type", "image/svg+xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(svg))
}

// handleWorkflowQuestion asks the agent a question during planning/implementing/reviewing.
// Streams the agent's response via SSE.
func (s *Server) handleWorkflowQuestion(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Parse request - handle both form and JSON submissions
	var question string
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid form data: "+err.Error())

			return
		}
		question = r.FormValue("question")
	} else {
		var req questionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

			return
		}
		question = req.Question
	}

	if question == "" {
		s.writeError(w, http.StatusBadRequest, "question is required")

		return
	}

	// Validate state allows questions BEFORE starting SSE stream.
	// Once SSE headers are set, we're committed to HTTP 200.
	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		s.writeError(w, http.StatusBadRequest, "no active task")

		return
	}

	allowedStates := map[string]bool{
		string(workflow.StatePlanning):     true,
		string(workflow.StateImplementing): true,
		string(workflow.StateReviewing):    true,
	}
	if !allowedStates[activeTask.State] {
		s.writeError(w, http.StatusConflict, fmt.Sprintf(
			"cannot ask questions in state '%s'; use during planning, implementing, or reviewing",
			activeTask.State,
		))

		return
	}

	// Set SSE headers AFTER validation passes
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Subscribe to agent message events for streaming
	eventBus := s.config.Conductor.GetEventBus()
	if eventBus == nil {
		s.writeErrorSSE(w, "event bus not available")

		return
	}

	// Set up event subscription for this request
	eventCh := make(chan eventbus.Event, 100)
	unsubscribeID := eventBus.SubscribeAll(func(e eventbus.Event) {
		if e.Type == events.TypeAgentMessage {
			select {
			case eventCh <- e:
			default:
				// Channel full, drop event
			}
		}
	})
	defer eventBus.Unsubscribe(unsubscribeID)
	defer close(eventCh)

	// Start streaming in background
	ctx := r.Context()
	go func() {
		if err := s.config.Conductor.AskQuestion(ctx, question); err != nil {
			// Check if agent asked a back-question
			if errors.Is(err, conductor.ErrPendingQuestion) {
				sendSSE(w, "", "{\"event\":\"back_question\",\"question\":\"Agent has a follow-up question\"}")
			} else {
				sendSSE(w, "", "{\"event\":\"error\",\"error\":\""+escapeJSON(err.Error())+"\"}")
			}
		}
		sendSSE(w, "", "{\"event\":\"done\"}")
	}()

	// Stream events to client
	for {
		select {
		case <-r.Context().Done():
			return
		case e, ok := <-eventCh:
			if !ok {
				return
			}
			if eventType, ok := e.Data["type"].(string); ok && eventType == "text" {
				if text, ok := e.Data["content"].(string); ok && text != "" {
					sendSSE(w, "", "{\"event\":\"content\",\"text\":\""+escapeJSON(text)+"\"}")
				}
			}
		}
	}
}

// sendSSE sends a Server-Sent Event to the client.
func sendSSE(w http.ResponseWriter, _ string, data string) {
	_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
	flusher, _ := w.(http.Flusher)
	if flusher != nil {
		flusher.Flush()
	}
}

// escapeJSON escapes a string for safe inclusion in JSON.
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")

	return s
}

// writeErrorSSE writes an error via SSE and closes the connection.
func (s *Server) writeErrorSSE(w http.ResponseWriter, message string) {
	sendSSE(w, "", "{\"event\":\"error\",\"error\":\""+escapeJSON(message)+"\"}")
}

// handleWorkflowReset resets the workflow state to idle without losing work.
// Use this to recover from hung agent sessions.
func (s *Server) handleWorkflowReset(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Check for active task
	if s.config.Conductor.GetActiveTask() == nil {
		s.writeError(w, http.StatusBadRequest, "no active task")

		return
	}

	// Reset state
	if err := s.config.Conductor.ResetState(r.Context()); err != nil {
		s.writeError(w, http.StatusInternalServerError, "reset state: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "reset",
		"state":   "idle",
		"message": "workflow state reset to idle",
	})
}
