package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/valksor/go-mehrhof/internal/events"
)

// setupRouter creates and configures the HTTP router.
func (s *Server) setupRouter() http.Handler {
	mux := http.NewServeMux()

	// Health check (public)
	mux.HandleFunc("GET /health", s.handleHealth)

	// Auth routes (public)
	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		s.handleLoginPageUI(w, r, "")
	})
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("GET /logout", s.handleLogout)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)

	// API routes
	mux.HandleFunc("GET /api/v1/status", s.handleStatus)
	mux.HandleFunc("GET /api/v1/context", s.handleContext)

	// Project mode routes
	if s.config.Mode == ModeProject {
		// Task endpoints
		mux.HandleFunc("GET /api/v1/task", s.handleGetTask)
		mux.HandleFunc("GET /api/v1/tasks", s.handleListTasks)
		mux.HandleFunc("GET /api/v1/tasks/{id}/specs", s.handleGetSpecifications)
		mux.HandleFunc("GET /api/v1/tasks/{id}/sessions", s.handleGetSessions)

		// Workflow action endpoints
		mux.HandleFunc("POST /api/v1/workflow/start", s.handleWorkflowStart)
		mux.HandleFunc("POST /api/v1/workflow/plan", s.handleWorkflowPlan)
		mux.HandleFunc("POST /api/v1/workflow/implement", s.handleWorkflowImplement)
		mux.HandleFunc("POST /api/v1/workflow/review", s.handleWorkflowReview)
		mux.HandleFunc("POST /api/v1/workflow/finish", s.handleWorkflowFinish)
		mux.HandleFunc("POST /api/v1/workflow/undo", s.handleWorkflowUndo)
		mux.HandleFunc("POST /api/v1/workflow/redo", s.handleWorkflowRedo)
		mux.HandleFunc("POST /api/v1/workflow/answer", s.handleWorkflowAnswer)
		mux.HandleFunc("POST /api/v1/workflow/abandon", s.handleWorkflowAbandon)
		mux.HandleFunc("POST /api/v1/workflow/continue", s.handleWorkflowContinue)
		mux.HandleFunc("POST /api/v1/workflow/auto", s.handleWorkflowAuto)

		// Notes endpoints
		mux.HandleFunc("POST /api/v1/tasks/{id}/notes", s.handleAddNote)
		mux.HandleFunc("GET /api/v1/tasks/{id}/notes", s.handleGetNotes)

		// Cost tracking endpoints
		mux.HandleFunc("GET /api/v1/tasks/{id}/costs", s.handleGetTaskCosts)
		mux.HandleFunc("GET /api/v1/costs", s.handleGetAllCosts)

		// Guide endpoint
		mux.HandleFunc("GET /api/v1/guide", s.handleGuide)

		// Info endpoints
		mux.HandleFunc("GET /api/v1/agents", s.handleListAgents)
		mux.HandleFunc("GET /api/v1/providers", s.handleListProviders)

		// Browser automation endpoints
		mux.HandleFunc("GET /api/v1/browser/status", s.handleBrowserStatus)
		mux.HandleFunc("GET /api/v1/browser/tabs", s.handleBrowserTabs)
		mux.HandleFunc("POST /api/v1/browser/goto", s.handleBrowserGoto)
		mux.HandleFunc("POST /api/v1/browser/navigate", s.handleBrowserNavigate)
		mux.HandleFunc("POST /api/v1/browser/screenshot", s.handleBrowserScreenshot)
		mux.HandleFunc("POST /api/v1/browser/click", s.handleBrowserClick)
		mux.HandleFunc("POST /api/v1/browser/type", s.handleBrowserType)
		mux.HandleFunc("POST /api/v1/browser/eval", s.handleBrowserEval)
		mux.HandleFunc("POST /api/v1/browser/dom", s.handleBrowserDOM)
		mux.HandleFunc("POST /api/v1/browser/reload", s.handleBrowserReload)
		mux.HandleFunc("POST /api/v1/browser/close", s.handleBrowserClose)

		// Security scan endpoint
		mux.HandleFunc("POST /api/v1/scan", s.handleSecurityScan)

		// Memory endpoints
		mux.HandleFunc("GET /api/v1/memory/search", s.handleMemorySearch)
		mux.HandleFunc("POST /api/v1/memory/index", s.handleMemoryIndex)
		mux.HandleFunc("GET /api/v1/memory/stats", s.handleMemoryStats)

		// Sync and simplify endpoints
		mux.HandleFunc("POST /api/v1/workflow/sync", s.handleWorkflowSync)
		mux.HandleFunc("POST /api/v1/workflow/simplify", s.handleWorkflowSimplify)

		// Templates endpoints
		mux.HandleFunc("GET /api/v1/templates", s.handleListTemplates)
		mux.HandleFunc("GET /api/v1/templates/{name}", s.handleGetTemplate)
		mux.HandleFunc("POST /api/v1/templates/apply", s.handleApplyTemplate)

		// Settings endpoints
		mux.HandleFunc("GET /settings", s.handleSettingsPage)
		mux.HandleFunc("GET /api/v1/settings", s.handleGetSettings)
		mux.HandleFunc("POST /api/v1/settings", s.handleSaveSettings)
		mux.HandleFunc("GET /api/v1/settings/explain", s.handleConfigExplain)
		mux.HandleFunc("GET /api/v1/settings/provider-health", s.handleProviderHealth)

		// Project planning UI
		mux.HandleFunc("GET /project", s.handleProjectUI)

		// Browser control panel UI
		mux.HandleFunc("GET /browser", s.handleBrowserUI)

		// Task history UI
		mux.HandleFunc("GET /history", s.handleHistoryUI)

		// Project workflow endpoints
		mux.HandleFunc("POST /api/v1/project/upload", s.handleProjectUpload)
		mux.HandleFunc("POST /api/v1/project/source", s.handleProjectSource)
		mux.HandleFunc("POST /api/v1/project/plan", s.handleProjectPlan)
		mux.HandleFunc("GET /api/v1/project/queues", s.handleProjectQueues)
		mux.HandleFunc("GET /api/v1/project/queue/", s.handleProjectQueueRoute)
		mux.HandleFunc("DELETE /api/v1/project/queue/", s.handleProjectQueueDeleteRoute)
		mux.HandleFunc("GET /api/v1/project/tasks", s.handleProjectTasks)
		mux.HandleFunc("PUT /api/v1/project/tasks/", s.handleProjectTaskEditRoute)
		mux.HandleFunc("POST /api/v1/project/reorder", s.handleProjectReorder)
		mux.HandleFunc("POST /api/v1/project/submit", s.handleProjectSubmit)
		mux.HandleFunc("POST /api/v1/project/start", s.handleProjectStart)
	}

	// Global mode routes
	if s.config.Mode == ModeGlobal {
		mux.HandleFunc("GET /api/v1/projects", s.handleListProjects)
		mux.HandleFunc("POST /api/v1/projects/select", s.handleSelectProject)

		// Settings endpoints (also available in global mode for viewing/editing config)
		mux.HandleFunc("GET /settings", s.handleSettingsPage)
		mux.HandleFunc("GET /api/v1/settings", s.handleGetSettings)
		mux.HandleFunc("POST /api/v1/settings", s.handleSaveSettings)
		mux.HandleFunc("GET /api/v1/settings/explain", s.handleConfigExplain)
		mux.HandleFunc("GET /api/v1/settings/provider-health", s.handleProviderHealth)
	}

	// Switch project route (available when started in global mode)
	if s.startedInGlobalMode {
		mux.HandleFunc("POST /api/v1/projects/switch", s.handleSwitchProject)
	}

	// SSE events endpoint
	mux.HandleFunc("GET /api/v1/events", s.handleEvents)

	// Agent logs streaming endpoints
	mux.HandleFunc("GET /api/v1/agent/logs/stream", s.handleAgentLogs)
	mux.HandleFunc("GET /api/v1/agent/logs/history", s.handleAgentLogsHistory)

	// UI partial routes (for HTMX updates)
	mux.HandleFunc("GET /ui/partials/task", s.handleTaskPartial)
	mux.HandleFunc("GET /ui/partials/actions", s.handleActionsPartial)
	mux.HandleFunc("GET /ui/partials/specs", s.handleSpecsPartial)
	mux.HandleFunc("GET /ui/partials/question", s.handleQuestionPartial)
	mux.HandleFunc("GET /ui/partials/costs", s.handleCostsPartial)
	mux.HandleFunc("GET /ui/partials/recent-tasks", s.handleRecentTasksPartial)

	// Main dashboard
	mux.HandleFunc("GET /", s.handleDashboard)

	// Wrap with middleware (logging, then auth)
	handler := s.withMiddleware(mux)
	handler = s.authMiddleware(handler)

	return handler
}

// withMiddleware wraps the handler with common middleware.
func (s *Server) withMiddleware(h http.Handler) http.Handler {
	// Logging middleware
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		h.ServeHTTP(rw, r)

		// Log request
		slog.Debug("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration", time.Since(start).String(),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter

	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Health check handler.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"mode":   s.modeString(),
	})
}

// Status handler returns server and workspace status.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"mode":    s.modeString(),
		"running": s.IsRunning(),
		"port":    s.Port(),
	}

	if s.config.Mode == ModeProject && s.config.Conductor != nil {
		machine := s.config.Conductor.GetMachine()
		if machine != nil {
			response["state"] = string(machine.State())
		}
	}

	s.writeJSON(w, http.StatusOK, response)
}

// Context handler returns server context (worktree status, current task).
func (s *Server) handleContext(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"mode":           s.modeString(),
		"workspace_root": s.config.WorkspaceRoot,
	}

	if s.config.Mode == ModeProject && s.config.Conductor != nil {
		activeTask := s.config.Conductor.GetActiveTask()
		if activeTask != nil {
			response["current_task"] = map[string]any{
				"id":            activeTask.ID,
				"state":         activeTask.State,
				"ref":           activeTask.Ref,
				"branch":        activeTask.Branch,
				"worktree_path": activeTask.WorktreePath,
				"started":       activeTask.Started,
			}
		}
	}

	s.writeJSON(w, http.StatusOK, response)
}

// GetTask handler returns the active task details.
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"active": false,
		})

		return
	}

	taskWork := s.config.Conductor.GetTaskWork()

	response := map[string]any{
		"active": true,
		"task": map[string]any{
			"id":            activeTask.ID,
			"state":         activeTask.State,
			"ref":           activeTask.Ref,
			"branch":        activeTask.Branch,
			"worktree_path": activeTask.WorktreePath,
			"started":       activeTask.Started,
		},
	}

	if taskWork != nil {
		response["work"] = map[string]any{
			"title":        taskWork.Metadata.Title,
			"external_key": taskWork.Metadata.ExternalKey,
			"created_at":   taskWork.Metadata.CreatedAt,
			"updated_at":   taskWork.Metadata.UpdatedAt,
			"costs":        taskWork.Costs,
		}
	}

	// Add pending question if present
	ws := s.config.Conductor.GetWorkspace()
	if ws != nil {
		enhanceTaskResponseWithPendingQuestion(response, ws, activeTask.ID)
	}

	s.writeJSON(w, http.StatusOK, response)
}

// ListTasks handler returns all tasks in the workspace.
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
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

	var tasks []map[string]any
	for _, id := range taskIDs {
		work, err := ws.LoadWork(id)
		if err != nil {
			continue
		}

		task := map[string]any{
			"id":         id,
			"title":      work.Metadata.Title,
			"created_at": work.Metadata.CreatedAt,
		}

		// Check for worktree
		if work.Git.WorktreePath != "" {
			task["worktree_path"] = work.Git.WorktreePath
		}

		tasks = append(tasks, task)
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"tasks": tasks,
		"count": len(tasks),
	})
}

// ListProjects handler returns all discovered projects (global mode).
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := DiscoverProjects()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to discover projects: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"projects": projects,
		"count":    len(projects),
	})
}

// Events handler provides SSE stream of events.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming not supported")

		return
	}

	// If no event bus, just keep connection alive
	if s.config.EventBus == nil {
		// Send initial connection event
		s.writeSSEEvent(w, flusher, "connected", map[string]string{"status": "connected"})

		// Keep connection alive until client disconnects
		<-r.Context().Done()

		return
	}

	// Subscribe to all events
	subID := s.config.EventBus.SubscribeAll(func(e events.Event) {
		s.writeSSEEvent(w, flusher, string(e.Type), e.Data)
	})
	defer s.config.EventBus.Unsubscribe(subID)

	// Send initial connection event
	s.writeSSEEvent(w, flusher, "connected", map[string]string{"status": "connected"})

	// Wait for client disconnect
	<-r.Context().Done()
}

// Index handler serves the main UI page (fallback when templates fail to load).
func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	// For now, return a simple HTML page
	// Future: Replace with proper Go templates for richer UI
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Mehrhof Web UI</title>
    <script src="https://unpkg.com/htmx.org@2.0.4"></script>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto p-8">
        <h1 class="text-3xl font-bold mb-4">Mehrhof Web UI</h1>
        <p class="text-gray-600 mb-8">Mode: ` + s.modeString() + `</p>

        <div class="bg-white rounded-lg shadow p-6 mb-4">
            <h2 class="text-xl font-semibold mb-2">Status</h2>
            <div id="status" hx-get="/api/v1/status" hx-trigger="load" hx-swap="innerHTML">
                Loading...
            </div>
        </div>

        <div class="bg-white rounded-lg shadow p-6">
            <h2 class="text-xl font-semibold mb-2">API Endpoints</h2>
            <ul class="list-disc list-inside text-gray-700">
                <li><a href="/health" class="text-blue-600 hover:underline">/health</a> - Health check</li>
                <li><a href="/api/v1/status" class="text-blue-600 hover:underline">/api/v1/status</a> - Server status</li>
                <li><a href="/api/v1/context" class="text-blue-600 hover:underline">/api/v1/context</a> - Server context</li>
                <li><a href="/api/v1/tasks" class="text-blue-600 hover:underline">/api/v1/tasks</a> - List tasks</li>
            </ul>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(html)); err != nil {
		slog.Error("failed to write HTML response", "error", err)
	}
}

// writeJSON writes a JSON response.
func (s *Server) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// writeError writes an error response.
func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{
		"error": message,
	})
}

// writeSSEEvent writes a Server-Sent Event.
func (s *Server) writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal SSE data", "error", err)

		return
	}

	if _, err = w.Write([]byte("event: " + eventType + "\n")); err != nil {
		slog.Error("failed to write SSE event", "error", err)

		return
	}
	if _, err = w.Write([]byte("data: " + string(jsonData) + "\n\n")); err != nil {
		slog.Error("failed to write SSE data", "error", err)

		return
	}
	flusher.Flush()
}
