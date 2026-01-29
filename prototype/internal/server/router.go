package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/server/static"
)

// setupRouter creates and configures the HTTP router.
func (s *Server) setupRouter() http.Handler {
	mux := http.NewServeMux()

	// Serve static assets (self-hosted JS/CSS)
	staticFS := http.FileServer(http.FS(static.Public()))
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticFS))

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

	// License routes (available in both project and global modes)
	mux.HandleFunc("GET /api/v1/license", s.handleLicense)
	mux.HandleFunc("GET /api/v1/license/info", s.handleLicenseInfo)
	mux.HandleFunc("GET /license", s.handleLicensePage)

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
		mux.HandleFunc("POST /api/v1/workflow/resume", s.handleWorkflowResume)
		mux.HandleFunc("POST /api/v1/workflow/abandon", s.handleWorkflowAbandon)
		mux.HandleFunc("POST /api/v1/workflow/continue", s.handleWorkflowContinue)
		mux.HandleFunc("POST /api/v1/workflow/auto", s.handleWorkflowAuto)
		mux.HandleFunc("POST /api/v1/workflow/question", s.handleWorkflowQuestion)
		mux.HandleFunc("GET /api/v1/workflow/diagram", s.handleWorkflowDiagram)

		// Notes endpoints
		mux.HandleFunc("POST /api/v1/tasks/{id}/notes", s.handleAddNote)
		mux.HandleFunc("GET /api/v1/tasks/{id}/notes", s.handleGetNotes)

		// Labels endpoints
		mux.HandleFunc("GET /api/v1/task/labels", s.handleTaskLabels)
		mux.HandleFunc("POST /api/v1/task/labels", s.handleTaskLabels)
		mux.HandleFunc("GET /api/v1/labels", s.handleListLabels)

		// Hierarchy endpoint
		mux.HandleFunc("GET /api/v1/task/hierarchy", s.handleGetHierarchy)

		// Cost tracking endpoints
		mux.HandleFunc("GET /api/v1/tasks/{id}/costs", s.handleGetTaskCosts)
		mux.HandleFunc("GET /api/v1/costs", s.handleGetAllCosts)

		// Guide endpoint
		mux.HandleFunc("GET /api/v1/guide", s.handleGuide)

		// Info endpoints
		mux.HandleFunc("GET /api/v1/agents", s.handleListAgents)
		mux.HandleFunc("GET /api/v1/providers", s.handleListProviders)

		// Agent Alias endpoints
		mux.HandleFunc("GET /api/v1/agents/aliases", s.handleListAgentAliases)
		mux.HandleFunc("POST /api/v1/agents/aliases", s.handleCreateAgentAlias)
		mux.HandleFunc("DELETE /api/v1/agents/aliases/", s.handleDeleteAgentAlias)

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

		// Links endpoints
		mux.HandleFunc("GET /api/v1/links", s.handleListLinks)
		mux.HandleFunc("GET /api/v1/links/", s.handleGetEntityLinks)
		mux.HandleFunc("GET /api/v1/links/search", s.handleSearchLinks)
		mux.HandleFunc("GET /api/v1/links/stats", s.handleLinksStats)
		mux.HandleFunc("POST /api/v1/links/rebuild", s.handleRebuildLinks)

		// Find search endpoints (available in both project and global mode)
		mux.HandleFunc("GET /api/v1/find", s.handleFindSearch)
		mux.HandleFunc("POST /api/v1/find", s.handleFindSearch)
		mux.HandleFunc("GET /find", s.handleFindUI)

		// Budget endpoints
		mux.HandleFunc("GET /api/v1/budget/monthly/status", s.handleBudgetMonthlyStatus)
		mux.HandleFunc("POST /api/v1/budget/monthly/reset", s.handleBudgetMonthlyReset)

		// Sync and simplify endpoints
		mux.HandleFunc("POST /api/v1/workflow/sync", s.handleWorkflowSync)
		mux.HandleFunc("POST /api/v1/workflow/simplify", s.handleWorkflowSimplify)

		// Standalone review/simplify endpoints (no active task required)
		mux.HandleFunc("POST /api/v1/workflow/review/standalone", s.handleStandaloneReview)
		mux.HandleFunc("POST /api/v1/workflow/simplify/standalone", s.handleStandaloneSimplify)

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

		// Sandbox endpoints
		mux.HandleFunc("GET /api/v1/sandbox/status", s.handleSandboxStatus)
		mux.HandleFunc("POST /api/v1/sandbox/enable", s.handleSandboxEnable)
		mux.HandleFunc("POST /api/v1/sandbox/disable", s.handleSandboxDisable)

		// Project planning UI
		mux.HandleFunc("GET /project", s.handleProjectUI)

		// Browser control panel UI
		mux.HandleFunc("GET /browser", s.handleBrowserUI)

		// Interactive chat UI
		mux.HandleFunc("GET /interactive", s.handleInteractivePage)
		mux.HandleFunc("POST /api/v1/interactive/chat", s.handleInteractiveChat)
		mux.HandleFunc("POST /api/v1/interactive/command", s.handleInteractiveCommand)
		mux.HandleFunc("POST /api/v1/interactive/answer", s.handleInteractiveAnswer)
		mux.HandleFunc("GET /api/v1/interactive/state", s.handleInteractiveState)
		mux.HandleFunc("POST /api/v1/interactive/stop", s.handleInteractiveStop)

		// Task history UI
		mux.HandleFunc("GET /history", s.handleHistoryUI)

		// Memory UI
		mux.HandleFunc("GET /memory", s.handleMemoryUI)

		// Links UI
		mux.HandleFunc("GET /links", s.handleLinksUI)

		// Stack management UI and API
		mux.HandleFunc("GET /stack", s.handleStacksUI)
		mux.HandleFunc("GET /api/v1/stack", s.handleStackList)
		mux.HandleFunc("POST /api/v1/stack/sync", s.handleStackSync)
		mux.HandleFunc("POST /api/v1/stack/rebase", s.handleStackRebase)

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
		mux.HandleFunc("POST /api/v1/project/sync", s.handleProjectSync)

		// Quick tasks endpoints
		mux.HandleFunc("GET /quick", s.handleQuickTasksUI)
		mux.HandleFunc("GET /api/v1/quick", s.handleQuickTaskList)
		mux.HandleFunc("POST /api/v1/quick", s.handleQuickTaskCreate)
		// Quick task item endpoints using Go 1.22+ wildcard patterns
		mux.HandleFunc("GET /api/v1/quick/{taskId}", s.handleQuickTaskGet)
		mux.HandleFunc("POST /api/v1/quick/{taskId}/note", s.handleQuickTaskNote)
		mux.HandleFunc("POST /api/v1/quick/{taskId}/optimize", s.handleQuickTaskOptimize)
		mux.HandleFunc("POST /api/v1/quick/{taskId}/export", s.handleQuickTaskExport)
		mux.HandleFunc("POST /api/v1/quick/{taskId}/submit", s.handleQuickTaskSubmit)
		mux.HandleFunc("POST /api/v1/quick/{taskId}/start", s.handleQuickTaskStart)
		mux.HandleFunc("DELETE /api/v1/quick/{taskId}", s.handleQuickTaskDelete)
		mux.HandleFunc("GET /api/v1/quick/{taskId}/card", s.handleQuickTaskCard)
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

		// Budget status endpoint (returns placeholder when no workspace)
		mux.HandleFunc("GET /api/v1/budget/monthly/status", s.handleBudgetMonthlyStatus)

		// Sandbox endpoints (also available in global mode)
		mux.HandleFunc("GET /api/v1/sandbox/status", s.handleSandboxStatus)
		mux.HandleFunc("POST /api/v1/sandbox/enable", s.handleSandboxEnable)
		mux.HandleFunc("POST /api/v1/sandbox/disable", s.handleSandboxDisable)
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
	mux.HandleFunc("GET /ui/partials/specification", s.handleSpecificationPartial)
	mux.HandleFunc("GET /ui/partials/question", s.handleQuestionPartial)
	mux.HandleFunc("GET /ui/partials/costs", s.handleCostsPartial)
	mux.HandleFunc("GET /ui/partials/hierarchy", s.handleHierarchyPartial)
	mux.HandleFunc("GET /ui/partials/workspace-stats", s.handleWorkspaceStatsPartial)
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

	statusCode     int
	headersWritten bool
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	if !rw.headersWritten {
		rw.ResponseWriter.WriteHeader(code)
		rw.headersWritten = true
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	// Mark headers as written on first Write call
	if !rw.headersWritten {
		rw.headersWritten = true
	}

	return rw.ResponseWriter.Write(b)
}

// Flush implements http.Flusher for SSE (Server-Sent Events) support.
// It delegates to the underlying ResponseWriter's Flush method if available.
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
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
			"state":      work.Metadata.State,
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
	// Set CORS header first (before any error response)
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Check if response writer supports flushing BEFORE setting SSE headers
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming not supported")

		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

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
