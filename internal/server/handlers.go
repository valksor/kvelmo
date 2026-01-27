package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
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
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

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
		// Redirect to dashboard for browser requests
		http.Redirect(w, r, "/", http.StatusSeeOther)

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "task started",
	})
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

// handleWorkflowPlan triggers planning phase.
func (s *Server) handleWorkflowPlan(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Enter planning phase
	if err := s.config.Conductor.Plan(r.Context()); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to enter planning: "+err.Error())

		return
	}

	// Run planning
	if err := s.config.Conductor.RunPlanning(r.Context()); err != nil {
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

	// Run implementation
	if err := s.config.Conductor.RunImplementation(r.Context()); err != nil {
		s.writeError(w, http.StatusInternalServerError, "implementation failed: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "implementation completed",
	})
}

// handleWorkflowReview triggers review phase.
func (s *Server) handleWorkflowReview(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Enter review phase
	if err := s.config.Conductor.Review(r.Context()); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to enter review: "+err.Error())

		return
	}

	// Run review
	if err := s.config.Conductor.RunReview(r.Context()); err != nil {
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
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req struct {
		Answer string `json:"answer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.Answer == "" {
		s.writeError(w, http.StatusBadRequest, "answer is required")

		return
	}

	// Get current task
	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		s.writeError(w, http.StatusBadRequest, "no active task")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	taskID := activeTask.ID

	// Check if there's a pending question
	if !ws.HasPendingQuestion(taskID) {
		s.writeError(w, http.StatusBadRequest, "no pending question")

		return
	}

	// Load the pending question to format the answer
	q, err := ws.LoadPendingQuestion(taskID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to load question: "+err.Error())

		return
	}

	// Save as a Q&A pair in notes
	note := "**Q:** " + q.Question + "\n\n**A:** " + req.Answer
	if err := ws.AppendNote(taskID, note, "answer"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to save answer: "+err.Error())

		return
	}

	// Clear the pending question
	if err := ws.ClearPendingQuestion(taskID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to clear question: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "answer submitted",
	})
}

// handleWorkflowAbandon abandons the current task.
func (s *Server) handleWorkflowAbandon(w http.ResponseWriter, r *http.Request) {
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

	var specList []map[string]any
	for _, spec := range specs {
		specList = append(specList, map[string]any{
			"number":       spec.Number,
			"title":        spec.Title,
			"status":       spec.Status,
			"created_at":   spec.CreatedAt,
			"completed_at": spec.CompletedAt,
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

	// Create session
	sess, err := s.sessions.create(req.Username)
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

	// JSON response for API
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

// handleSelectProject switches from global mode to project mode.
func (s *Server) handleSelectProject(w http.ResponseWriter, r *http.Request) {
	// Parse request
	if err := r.ParseForm(); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid form data")

		return
	}

	projectPath := r.FormValue("path")
	if projectPath == "" {
		s.writeError(w, http.StatusBadRequest, "project path is required")

		return
	}

	// Switch to project mode
	if err := s.switchToProject(projectPath); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to switch project: "+err.Error())

		return
	}

	slog.Info("switched to project mode", "path", projectPath)

	// Check if this is an HTMX request (keep URL, return HTML)
	// HTMX sets this header when making requests
	isHTMX := r.Header.Get("Hx-Request") == "true"

	if isHTMX {
		// Render and return the full dashboard HTML
		// HTMX will swap the body content, URL stays the same
		s.handleDashboard(w, r)

		return
	}

	// For non-HTMX requests, redirect to dashboard
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleSwitchProject switches back to global mode to pick another project.
func (s *Server) handleSwitchProject(w http.ResponseWriter, r *http.Request) {
	s.switchToGlobal()

	slog.Info("switched back to global mode")

	// Redirect to dashboard
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
