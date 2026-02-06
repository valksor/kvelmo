package server

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/server/views"
	"github.com/valksor/go-toolkit/eventbus"
)

// Workflow action request/response types.

type startTaskRequest struct {
	Ref     string `json:"ref"`
	Content string `json:"content"`
}

type finishRequest struct {
	SquashMerge  bool   `json:"squash_merge"`
	DeleteBranch bool   `json:"delete_branch"`
	TargetBranch string `json:"target_branch"`
	PushAfter    bool   `json:"push_after"`
	ForceMerge   bool   `json:"force_merge"`
	DraftPR      bool   `json:"draft_pr"`
	PRTitle      string `json:"pr_title"`
	PRBody       string `json:"pr_body"`
}

// handleWorkflowStart starts a new task.
// Accepts three input methods:
// 1. multipart/form-data with "file" field - file upload
// 2. application/x-www-form-urlencoded with "content" or "ref" field - form submission
// 3. application/json with "content" or "ref" field - API call.
func (s *Server) handleWorkflowStart(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Check for active task conflict (non-worktree mode only allows one active task)
	if conflict := s.config.Conductor.CheckActiveTaskConflict(r.Context()); conflict != nil {
		response := map[string]any{
			"success":       false,
			"conflict_type": "active_task",
			"active_task": map[string]any{
				"id":             conflict.ActiveTaskID,
				"title":          conflict.ActiveTaskTitle,
				"branch":         conflict.ActiveBranch,
				"using_worktree": conflict.UsingWorktree,
			},
			"message": "Another task is already active. Use worktree mode for parallel tasks, or finish/abandon current task first.",
		}
		s.writeJSON(w, http.StatusConflict, response)

		return
	}

	var ref string
	contentType := r.Header.Get("Content-Type")

	// Handle multipart/form-data (file upload)
	if strings.HasPrefix(contentType, "multipart/form-data") {
		taskRef, err := s.handleFileUpload(r)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, err.Error())

			return
		}
		ref = taskRef
	} else if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		// Handle form submission (from HTMX)
		if err := r.ParseForm(); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid form data: "+err.Error())

			return
		}

		content := r.FormValue("content")
		refVal := r.FormValue("ref")

		if content != "" {
			// Direct content - save to temp file
			taskRef, err := s.saveContentToFile(content)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, "failed to save content: "+err.Error())

				return
			}
			ref = taskRef
		} else if refVal != "" {
			ref = refVal
		} else {
			s.writeError(w, http.StatusBadRequest, "ref or content is required")

			return
		}
	} else {
		// Handle JSON body
		var req startTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

			return
		}

		if req.Content != "" {
			// Direct content - save to temp file
			taskRef, err := s.saveContentToFile(req.Content)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, "failed to save content: "+err.Error())

				return
			}
			ref = taskRef
		} else if req.Ref != "" {
			// External reference
			ref = req.Ref
		} else {
			s.writeError(w, http.StatusBadRequest, "ref or content is required")

			return
		}
	}

	if err := s.config.Conductor.Start(r.Context(), ref); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to start task: "+err.Error())

		return
	}

	// Check if this is a browser request (wants HTML) or API request (wants JSON)
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/html") {
		// Redirect to dashboard for browser form submissions
		http.Redirect(w, r, "/", http.StatusSeeOther)

		return
	}

	response := map[string]any{
		"success": true,
		"message": "task started",
	}
	if activeTask := s.config.Conductor.GetActiveTask(); activeTask != nil {
		response["task_id"] = activeTask.ID
	}
	s.writeJSON(w, http.StatusOK, response)
}

// handleFileUpload processes file upload and returns a file: ref.
func (s *Server) handleFileUpload(r *http.Request) (string, error) {
	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return "", err
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".md" && ext != ".txt" && ext != ".markdown" {
		return "", &invalidFileError{ext: ext}
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return s.saveContentToFile(string(content))
}

// saveContentToFile saves content to a temp file in .mehrhof/tasks/ and returns a file: ref.
func (s *Server) saveContentToFile(content string) (string, error) {
	// Create tasks directory in workspace
	tasksDir := filepath.Join(s.config.WorkspaceRoot, ".mehrhof", "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		return "", err
	}

	// Create temp file
	f, err := os.CreateTemp(tasksDir, "task-*.md")
	if err != nil {
		return "", err
	}

	name := f.Name()

	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()

		return "", err
	}

	if err := f.Close(); err != nil {
		return "", err
	}

	return "file:" + name, nil
}

// invalidFileError represents an invalid file extension error.
type invalidFileError struct {
	ext string
}

func (e *invalidFileError) Error() string {
	return "invalid file type: " + e.ext + " (expected .md, .txt, or .markdown)"
}

// handleNonFatalWorkflowError checks for non-fatal workflow errors and returns
// an appropriate success response. Returns true if the error was handled.
// Non-fatal errors include: pending questions, budget paused, budget stopped.
func (s *Server) handleNonFatalWorkflowError(w http.ResponseWriter, err error, phase string) bool {
	if err == nil {
		return false
	}

	// Handle pending question - agent needs user input
	if errors.Is(err, conductor.ErrPendingQuestion) {
		response := map[string]any{
			"success": true,
			"status":  "waiting",
			"message": "Agent has a question",
			"phase":   phase,
		}
		// Add task details if conductor is available
		if cond := s.config.Conductor; cond != nil {
			task := cond.GetActiveTask()
			if task != nil {
				response["task_id"] = task.ID
				if q, loadErr := cond.GetWorkspace().LoadPendingQuestion(task.ID); loadErr == nil && q != nil {
					response["question"] = q.Question
					response["options"] = q.Options
				} else if loadErr != nil {
					slog.Warn("failed to load pending question", "task_id", task.ID, "error", loadErr)
				}
			}
		}
		// Publish SSE event to trigger question partial refresh in Web UI
		if s.config.EventBus != nil {
			s.config.EventBus.PublishRaw(eventbus.Event{
				Type: views.EventQuestionAsked,
				Data: response,
			})
		}
		s.writeJSON(w, http.StatusOK, response)

		return true
	}

	// Handle budget paused - task paused due to budget limits
	if errors.Is(err, conductor.ErrBudgetPaused) {
		response := map[string]any{
			"success": true,
			"status":  "paused",
			"message": "Task paused due to budget limit",
			"phase":   phase,
		}
		// Add task_id for consistency with pending question response
		if cond := s.config.Conductor; cond != nil {
			if task := cond.GetActiveTask(); task != nil {
				response["task_id"] = task.ID
			}
		}
		s.writeJSON(w, http.StatusOK, response)

		return true
	}

	// Handle budget stopped - task stopped due to budget limits
	if errors.Is(err, conductor.ErrBudgetStopped) {
		response := map[string]any{
			"success": false,
			"status":  "stopped",
			"message": "Task stopped due to budget limit",
			"phase":   phase,
		}
		// Add task_id for consistency with pending question response
		if cond := s.config.Conductor; cond != nil {
			if task := cond.GetActiveTask(); task != nil {
				response["task_id"] = task.ID
			}
		}
		s.writeJSON(w, http.StatusOK, response)

		return true
	}

	return false
}

// handleWorkflowPlan triggers planning phase.
func (s *Server) handleWorkflowPlan(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Enter planning phase
	if err := s.config.Conductor.Plan(r.Context()); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to enter planning: "+err.Error())

		return
	}

	// Publish state change event immediately so UI updates
	s.publishStateChangeEvent(r.Context())

	// Run planning
	if err := s.config.Conductor.RunPlanning(r.Context()); err != nil {
		if s.handleNonFatalWorkflowError(w, err, "planning") {
			return
		}
		s.writeError(w, http.StatusInternalServerError, "planning failed: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "planning completed",
	})
}

// handleWorkflowImplement triggers implementation phase.
func (s *Server) handleWorkflowImplement(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Parse query parameters for implementation options
	component := r.URL.Query().Get("component")
	parallel := r.URL.Query().Get("parallel")

	// Apply temporary options if specified
	if component != "" || parallel != "" {
		s.config.Conductor.SetImplementationOptions(component, parallel)
		defer s.config.Conductor.ClearImplementationOptions()
	}

	// Enter implementing phase
	if err := s.config.Conductor.Implement(r.Context()); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to enter implementing: "+err.Error())

		return
	}

	// Publish state change event immediately so UI updates
	s.publishStateChangeEvent(r.Context())

	// Run implementation
	if err := s.config.Conductor.RunImplementation(r.Context()); err != nil {
		if s.handleNonFatalWorkflowError(w, err, "implementing") {
			return
		}
		s.writeError(w, http.StatusInternalServerError, "implementation failed: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "implementation completed",
	})
}

// handleWorkflowImplementReview triggers implementation fixes for a specific review.
func (s *Server) handleWorkflowImplementReview(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Parse review number from path
	nStr := r.PathValue("n")
	if nStr == "" {
		s.writeError(w, http.StatusBadRequest, "review number is required")

		return
	}

	reviewNumber, err := strconv.Atoi(nStr)
	if err != nil || reviewNumber <= 0 {
		s.writeError(w, http.StatusBadRequest, "invalid review number: must be a positive integer")

		return
	}

	// Enter implementing phase for review fixes
	if err := s.config.Conductor.ImplementReview(r.Context(), reviewNumber); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to enter implementing for review: "+err.Error())

		return
	}

	// Publish state change event immediately so UI updates
	s.publishStateChangeEvent(r.Context())

	// Run review implementation
	if err := s.config.Conductor.RunReviewImplementation(r.Context(), reviewNumber); err != nil {
		if s.handleNonFatalWorkflowError(w, err, "implementing review") {
			return
		}
		s.writeError(w, http.StatusInternalServerError, "review implementation failed: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"message":       "review implementation completed",
		"review_number": reviewNumber,
	})
}

// handleWorkflowReview triggers review phase.
func (s *Server) handleWorkflowReview(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Enter review phase
	if err := s.config.Conductor.Review(r.Context()); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to enter review: "+err.Error())

		return
	}

	// Publish state change event immediately so UI updates
	s.publishStateChangeEvent(r.Context())

	// Run review
	if err := s.config.Conductor.RunReview(r.Context()); err != nil {
		if s.handleNonFatalWorkflowError(w, err, "reviewing") {
			return
		}
		s.writeError(w, http.StatusInternalServerError, "review failed: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "review completed",
	})
}
