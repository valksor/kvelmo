package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// handleGetSpecifications returns specifications for a task.
func (s *Server) handleGetSpecifications(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Get task ID from path
	taskID := r.PathValue("id")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	specs, err := ws.ListSpecificationsWithStatus(taskID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list specifications: "+err.Error())

		return
	}

	specList := []map[string]any{}
	for _, spec := range specs {
		specList = append(specList, map[string]any{
			"number":            spec.Number,
			"name":              fmt.Sprintf("spec-%d", spec.Number),
			"title":             spec.Title,
			"description":       spec.Content,
			"component":         spec.Component,
			"status":            spec.Status,
			"created_at":        spec.CreatedAt,
			"completed_at":      spec.CompletedAt,
			"implemented_files": spec.ImplementedFiles,
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"specifications": specList,
		"count":          len(specList),
	})
}

// handleGetSessions returns sessions for a task.
func (s *Server) handleGetSessions(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Get task ID from path
	taskID := r.PathValue("id")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	sessions, err := ws.ListSessions(taskID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list sessions: "+err.Error())

		return
	}

	var sessionList []map[string]any
	for _, session := range sessions {
		sess := map[string]any{
			"kind":       session.Kind,
			"started_at": session.Metadata.StartedAt,
			"ended_at":   session.Metadata.EndedAt,
			"agent":      session.Metadata.Agent,
		}
		if session.Usage != nil {
			sess["usage"] = map[string]any{
				"input_tokens":  session.Usage.InputTokens,
				"output_tokens": session.Usage.OutputTokens,
				"cached_tokens": session.Usage.CachedTokens,
				"cost_usd":      session.Usage.CostUSD,
			}
		}
		sessionList = append(sessionList, sess)
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"sessions": sessionList,
		"count":    len(sessionList),
	})
}

// handleLogin processes login requests.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	// Handle both form and JSON submissions
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/x-www-form-urlencoded" {
		if err := r.ParseForm(); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid form data")

			return
		}
		req.Username = r.FormValue("username")
		req.Password = r.FormValue("password")
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body")

			return
		}
	}

	if req.Username == "" || req.Password == "" {
		s.handleLoginPage(w, r, "Username and password are required")

		return
	}

	// Validate credentials
	if s.config.AuthStore == nil || !s.config.AuthStore.ValidatePassword(req.Username, req.Password) {
		slog.Warn("login failed", "username", req.Username, "remote", r.RemoteAddr)
		s.handleLoginPage(w, r, "Invalid username or password")

		return
	}

	// Get user to retrieve their role
	user, userExists := s.config.AuthStore.GetUser(req.Username)
	if !userExists {
		s.writeError(w, http.StatusInternalServerError, "failed to get user")

		return
	}

	// Create session with user's role
	sess, err := s.sessions.create(req.Username, user.Role)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to create session")

		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionDuration.Seconds()),
	})

	slog.Info("login successful", "username", req.Username, "remote", r.RemoteAddr)

	// Redirect to home on success (for form submission)
	if contentType == "application/x-www-form-urlencoded" {
		http.Redirect(w, r, "/", http.StatusSeeOther)

		return
	}

	// JSON response for API — include CSRF token so clients can send it in X-CSRF-Token header
	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":     "ok",
		"csrf_token": sess.CSRFToken,
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

// handleLogout clears the session.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Get and delete session
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		s.sessions.delete(cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Redirect to login
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// handleLoginPage renders the login page.
func (s *Server) handleLoginPage(w http.ResponseWriter, _ *http.Request, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	errorHTML := ""
	if errMsg != "" {
		errorHTML = `<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">` + errMsg + `</div>`
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <title>Login - Mehrhof</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 min-h-screen flex items-center justify-center">
    <div class="bg-white p-8 rounded-lg shadow-md w-96">
        <h1 class="text-2xl font-bold mb-6 text-center">Mehrhof Login</h1>
        ` + errorHTML + `
        <form action="/api/v1/auth/login" method="POST">
            <div class="mb-4">
                <label class="block text-gray-700 text-sm font-bold mb-2" for="username">
                    Username
                </label>
                <input name="username" id="username" type="text" placeholder="Username"
                       class="w-full p-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                       required autofocus>
            </div>
            <div class="mb-6">
                <label class="block text-gray-700 text-sm font-bold mb-2" for="password">
                    Password
                </label>
                <input name="password" id="password" type="password" placeholder="Password"
                       class="w-full p-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                       required>
            </div>
            <button type="submit"
                    class="w-full bg-blue-500 hover:bg-blue-600 text-white font-bold py-2 px-4 rounded focus:outline-none focus:ring-2 focus:ring-blue-500">
                Login
            </button>
        </form>
    </div>
</body>
</html>`

	if _, err := w.Write([]byte(html)); err != nil {
		slog.Error("failed to write login page", "error", err)
	}
}

// selectProjectRequest is the JSON request body for project selection.
type selectProjectRequest struct {
	Path string `json:"path"`
}

// handleSelectProject switches from global mode to project mode.
func (s *Server) handleSelectProject(w http.ResponseWriter, r *http.Request) {
	var projectPath string

	// Check if JSON request (API client) or form (HTMX)
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		var req selectProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())

			return
		}

		projectPath = req.Path
	} else {
		// Form data (HTMX)
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

	// Return JSON for API clients, redirect for HTMX
	if contentType == "application/json" {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"path":    projectPath,
		})

		return
	}

	// Redirect for HTMX/browser
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleSwitchProject switches back to global mode to pick another project.
func (s *Server) handleSwitchProject(w http.ResponseWriter, r *http.Request) {
	s.switchToGlobal()

	slog.Info("switched back to global mode")

	s.writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

// labelRequest represents a label modification request.
type labelRequest struct {
	Action string   `json:"action"` // add, remove, set
	Labels []string `json:"labels"`
}

// handleTaskLabels handles GET/POST for task labels.
// GET: Returns labels for the active task.
// POST: Modifies labels (add/remove/set) on the active task.
func (s *Server) handleTaskLabels(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		s.writeError(w, http.StatusBadRequest, "no active task")

		return
	}

	taskID := activeTask.ID

	switch r.Method {
	case http.MethodGet:
		labels, err := ws.GetLabels(taskID)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to get labels: "+err.Error())

			return
		}

		s.writeJSON(w, http.StatusOK, map[string]any{
			"task_id": taskID,
			"labels":  labels,
		})

	case http.MethodPost:
		var req labelRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

			return
		}

		switch req.Action {
		case "add":
			for _, label := range req.Labels {
				if err := ws.AddLabel(taskID, label); err != nil {
					s.writeError(w, http.StatusInternalServerError, "failed to add label: "+err.Error())

					return
				}
			}

		case "remove":
			for _, label := range req.Labels {
				if err := ws.RemoveLabel(taskID, label); err != nil {
					s.writeError(w, http.StatusInternalServerError, "failed to remove label: "+err.Error())

					return
				}
			}

		case "set":
			if err := ws.SetLabels(taskID, req.Labels); err != nil {
				s.writeError(w, http.StatusInternalServerError, "failed to set labels: "+err.Error())

				return
			}

		default:
			s.writeError(w, http.StatusBadRequest, "invalid action: "+req.Action)

			return
		}

		// Get updated labels
		labels, _ := ws.GetLabels(taskID)

		s.writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"action":  req.Action,
			"labels":  labels,
		})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleListLabels returns all unique labels across all tasks with counts.
func (s *Server) handleListLabels(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	taskIDs, err := ws.ListWorks()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list tasks: "+err.Error())

		return
	}

	// Count labels across all tasks
	labelCounts := make(map[string]int)
	for _, taskID := range taskIDs {
		work, err := ws.LoadWork(taskID)
		if err != nil {
			continue
		}

		for _, label := range work.Metadata.Labels {
			labelCounts[label]++
		}
	}

	// Convert to sorted slice
	type labelInfo struct {
		Label string `json:"label"`
		Count int    `json:"count"`
	}

	var labels []labelInfo
	for label, count := range labelCounts {
		labels = append(labels, labelInfo{Label: label, Count: count})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"labels": labels,
		"count":  len(labels),
	})
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
		return
	}

	// Compute progress phase for context-aware state display
	// This allows the frontend to show "Planned" when state is "idle" but specs exist
	var progressPhase string
	if ws := s.config.Conductor.GetWorkspace(); ws != nil && status.TaskID != "" {
		phase := computeProgressPhase(ws, status.TaskID)
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
