package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/server/views"
	"github.com/valksor/go-toolkit/eventbus"
)

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
