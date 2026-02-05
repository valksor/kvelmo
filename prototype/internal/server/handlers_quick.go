package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// Quick task request/response types.

type createQuickTaskRequest struct {
	Description string   `json:"description"`
	Title       string   `json:"title,omitempty"`
	Priority    int      `json:"priority"`
	Labels      []string `json:"labels,omitempty"`
}

type quickTaskResponse struct {
	QueueID   string `json:"queue_id"`
	TaskID    string `json:"task_id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
}

type quickTaskListItem struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Priority  int      `json:"priority"`
	Labels    []string `json:"labels"`
	Status    string   `json:"status"`
	NoteCount int      `json:"note_count"`
}

type submitSourceRequest struct {
	Source       string   `json:"source"`
	Provider     string   `json:"provider"`
	Notes        []string `json:"notes"`
	Title        string   `json:"title,omitempty"`
	Instructions string   `json:"instructions,omitempty"`
	Labels       []string `json:"labels,omitempty"`
	QueueID      string   `json:"queue_id,omitempty"`
	Optimize     bool     `json:"optimize"`
	DryRun       bool     `json:"dry_run"`
}

// handleQuickTaskGet returns a single quick task with its notes.
// GET /api/v1/quick/{taskId}.
func (s *Server) handleQuickTaskGet(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}
	s.handleQuickTaskGetWithID(w, r, taskID)
}

// handleQuickTaskNote adds a note to a quick task.
// POST /api/v1/quick/{taskId}/note.
func (s *Server) handleQuickTaskNote(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}
	s.handleQuickTaskNoteWithID(w, r, taskID)
}

// handleQuickTaskOptimize runs AI optimization on a task.
// POST /api/v1/quick/{taskId}/optimize.
func (s *Server) handleQuickTaskOptimize(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}
	s.handleQuickTaskOptimizeWithID(w, r, taskID)
}

// handleQuickTaskExport exports a task to markdown.
// POST /api/v1/quick/{taskId}/export.
func (s *Server) handleQuickTaskExport(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}
	s.handleQuickTaskExportWithID(w, r, taskID)
}

// handleQuickTaskSubmit submits a task to a provider.
// POST /api/v1/quick/{taskId}/submit.
func (s *Server) handleQuickTaskSubmit(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}
	s.handleQuickTaskSubmitWithID(w, r, taskID)
}

// handleQuickTaskSubmitSource creates a quick task from source and submits it.
// POST /api/v1/quick/submit-source.
func (s *Server) handleQuickTaskSubmitSource(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req submitSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if strings.TrimSpace(req.Source) == "" {
		s.writeError(w, http.StatusBadRequest, "source is required")

		return
	}
	if strings.TrimSpace(req.Provider) == "" {
		s.writeError(w, http.StatusBadRequest, "provider is required")

		return
	}

	result, err := s.config.Conductor.CreateQueueTaskFromSource(r.Context(), req.Source, conductor.SourceTaskOptions{
		QueueID:      req.QueueID,
		Title:        req.Title,
		Instructions: req.Instructions,
		Notes:        req.Notes,
		Provider:     req.Provider,
		Labels:       req.Labels,
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "create task from source failed: "+err.Error())

		return
	}

	if req.Optimize {
		if _, err := s.config.Conductor.OptimizeQueueTask(r.Context(), result.QueueID, result.TaskID); err != nil {
			s.writeError(w, http.StatusInternalServerError, "optimize task failed: "+err.Error())

			return
		}
	}

	submitResult, err := s.config.Conductor.SubmitQueueTask(r.Context(), result.QueueID, result.TaskID, conductor.SubmitOptions{
		Provider: req.Provider,
		Labels:   req.Labels,
		TaskIDs:  []string{result.TaskID},
		DryRun:   req.DryRun,
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "submission failed: "+err.Error())

		return
	}

	response := map[string]any{
		"success":  true,
		"queue_id": result.QueueID,
		"task_id":  result.TaskID,
		"provider": req.Provider,
		"dry_run":  submitResult.DryRun,
	}
	if len(submitResult.Tasks) > 0 {
		response["external_id"] = submitResult.Tasks[0].ExternalID
		response["external_url"] = submitResult.Tasks[0].ExternalURL
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleQuickTaskStart starts working on a task.
// POST /api/v1/quick/{taskId}/start.
func (s *Server) handleQuickTaskStart(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}
	s.handleQuickTaskStartWithID(w, r, taskID)
}

// handleQuickTaskDelete deletes a quick task.
// DELETE /api/v1/quick/{taskId}.
func (s *Server) handleQuickTaskDelete(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}
	s.handleQuickTaskDeleteWithID(w, r, taskID)
}

// handleQuickTaskCard renders a single task card for HTMX swapping.
// GET /api/v1/quick/{taskId}/card.
func (s *Server) handleQuickTaskCard(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}
	s.handleQuickTaskCardWithID(w, r, taskID)
}

// Implementation handlers

// handleQuickTaskCreate creates a new quick task.
// POST /api/v1/quick.
func (s *Server) handleQuickTaskCreate(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Parse request - handle both form and JSON submissions
	var req createQuickTaskRequest
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid form data: "+err.Error())

			return
		}
		req.Description = r.FormValue("description")
		req.Title = r.FormValue("title")
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

			return
		}
	}

	if req.Description == "" {
		s.writeError(w, http.StatusBadRequest, "description is required")

		return
	}

	// Create quick task
	result, err := s.config.Conductor.CreateQuickTask(r.Context(), conductor.QuickTaskOptions{
		Description: req.Description,
		Title:       req.Title,
		Priority:    req.Priority,
		Labels:      req.Labels,
		QueueID:     "quick-tasks",
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to create task: "+err.Error())

		return
	}

	slog.Info("quick task created", "task_id", result.TaskID, "title", result.Title)

	s.writeJSON(w, http.StatusOK, quickTaskResponse{
		QueueID:   result.QueueID,
		TaskID:    result.TaskID,
		Title:     result.Title,
		CreatedAt: result.CreatedAt.Format("2006-01-02T15:04:05"),
	})
}

// handleQuickTaskList returns all quick tasks.
// GET /api/v1/quick.
func (s *Server) handleQuickTaskList(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Load quick-tasks queue
	queueID := "quick-tasks"
	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		// Queue doesn't exist yet, return empty list
		s.writeJSON(w, http.StatusOK, map[string]any{
			"tasks": []quickTaskListItem{},
			"count": 0,
		})

		return
	}

	// Build task list with note counts
	var tasks []quickTaskListItem
	for _, task := range queue.Tasks {
		notes, _ := ws.LoadQueueNotes(queueID, task.ID)

		tasks = append(tasks, quickTaskListItem{
			ID:        task.ID,
			Title:     task.Title,
			Priority:  task.Priority,
			Labels:    append([]string{}, task.Labels...),
			Status:    string(task.Status),
			NoteCount: len(notes),
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"tasks": tasks,
		"count": len(tasks),
	})
}

// handleQuickTaskGetWithID returns a single quick task with its notes.
//
//nolint:unparam // r is part of the handler signature
func (s *Server) handleQuickTaskGetWithID(w http.ResponseWriter, r *http.Request, taskID string) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Load queue
	queueID := "quick-tasks"
	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "task not found")

		return
	}

	task := queue.GetTask(taskID)
	if task == nil {
		s.writeError(w, http.StatusNotFound, "task not found")

		return
	}

	// Load notes
	notes, _ := ws.LoadQueueNotes(queueID, taskID)

	// Build notes response
	var notesList []map[string]any
	for _, note := range notes {
		notesList = append(notesList, map[string]any{
			"timestamp": note.Timestamp,
			"content":   note.Content,
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"id":          task.ID,
		"title":       task.Title,
		"description": task.Description,
		"priority":    task.Priority,
		"labels":      task.Labels,
		"status":      string(task.Status),
		"notes":       notesList,
	})
}

// handleQuickTaskNoteWithID adds a note to a quick task.
func (s *Server) handleQuickTaskNoteWithID(w http.ResponseWriter, r *http.Request, taskID string) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Parse request
	var req addNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.Note == "" {
		s.writeError(w, http.StatusBadRequest, "note is required")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Add note
	if err := ws.AppendQueueNote("quick-tasks", taskID, req.Note); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to save note: "+err.Error())

		return
	}

	slog.Info("quick task note added", "task_id", taskID)

	// For HTMX requests, return the updated card
	if r.Header.Get("Hx-Request") == "true" {
		s.handleQuickTaskCardWithID(w, r, taskID)

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "note saved",
	})
}

// handleQuickTaskOptimizeWithID runs AI optimization on a task.
func (s *Server) handleQuickTaskOptimizeWithID(w http.ResponseWriter, r *http.Request, taskID string) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Parse request
	var req optimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	// Apply agent override if specified
	if req.Agent != "" {
		s.config.Conductor.SetAgent(req.Agent)
		defer s.config.Conductor.ClearAgent()
	}

	// Optimize task
	result, err := s.config.Conductor.OptimizeQueueTask(r.Context(), "quick-tasks", taskID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "optimization failed: "+err.Error())

		return
	}

	slog.Info("quick task optimized", "task_id", taskID)

	// For HTMX requests, return the updated card
	if r.Header.Get("Hx-Request") == "true" {
		s.handleQuickTaskCardWithID(w, r, taskID)

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success":        true,
		"title":          result.OptimizedTitle,
		"original_title": result.OriginalTitle,
		"added_labels":   result.AddedLabels,
		"improvements":   result.ImprovementNotes,
	})
}

// handleQuickTaskExportWithID exports a task to markdown.
func (s *Server) handleQuickTaskExportWithID(w http.ResponseWriter, r *http.Request, taskID string) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Parse request
	var req exportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	// Export task
	markdown, err := s.config.Conductor.ExportQueueTask("quick-tasks", taskID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "export failed: "+err.Error())

		return
	}

	// If output specified, save to file
	if req.Output != "" {
		ws := s.config.Conductor.GetWorkspace()
		outputPath := ws.CodeAbsolutePath(req.Output)
		if err := ws.SaveFile(outputPath, []byte(markdown)); err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to save file: "+err.Error())

			return
		}

		slog.Info("quick task exported", "task_id", taskID, "output", req.Output)

		s.writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "exported to " + req.Output,
			"path":    req.Output,
		})

		return
	}

	// Return as downloadable file
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.md", taskID))
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	//nolint:errcheck // HTTP response write errors are handled by the server
	w.Write([]byte(markdown))

	slog.Info("quick task exported", "task_id", taskID)
}

// handleQuickTaskSubmitWithID submits a task to a provider.
func (s *Server) handleQuickTaskSubmitWithID(w http.ResponseWriter, r *http.Request, taskID string) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Parse request
	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.Provider == "" {
		s.writeError(w, http.StatusBadRequest, "provider is required")

		return
	}

	// Submit task
	result, err := s.config.Conductor.SubmitQueueTask(r.Context(), "quick-tasks", taskID, conductor.SubmitOptions{
		Provider: req.Provider,
		Labels:   req.Labels,
		TaskIDs:  []string{taskID},
		DryRun:   req.DryRun,
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "submission failed: "+err.Error())

		return
	}

	if !req.DryRun {
		slog.Info("quick task submitted", "task_id", taskID, "provider", req.Provider)
	}

	// Build response
	var responseData map[string]any
	if len(result.Tasks) > 0 {
		task := result.Tasks[0]
		responseData = map[string]any{
			"success":      true,
			"provider":     req.Provider,
			"external_id":  task.ExternalID,
			"external_url": task.ExternalURL,
		}
		if result.Epic != nil {
			responseData["epic_id"] = result.Epic.ExternalID
			responseData["epic_url"] = result.Epic.ExternalURL
		}
		if req.DryRun {
			responseData["dry_run"] = true
			responseData["title"] = task.Title
		}
	} else {
		responseData = map[string]any{
			"success": true,
			"dry_run": req.DryRun,
		}
	}

	s.writeJSON(w, http.StatusOK, responseData)
}

// handleQuickTaskStartWithID starts working on a task.
func (s *Server) handleQuickTaskStartWithID(w http.ResponseWriter, r *http.Request, taskID string) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Start task
	ref := "queue:quick-tasks/" + taskID
	if err := s.config.Conductor.Start(r.Context(), ref); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to start task: "+err.Error())

		return
	}

	slog.Info("quick task started", "task_id", taskID)

	// Check if this is HTMX request
	if r.Header.Get("Hx-Request") == "true" {
		// Redirect to dashboard for HTMX
		w.Header().Set("Hx-Redirect", "/")

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "task started",
	})
}

// handleQuickTaskDeleteWithID deletes a quick task.
//
//nolint:unparam // r is part of the handler signature
func (s *Server) handleQuickTaskDeleteWithID(w http.ResponseWriter, r *http.Request, taskID string) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Load queue
	queueID := "quick-tasks"
	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "task not found")

		return
	}

	// Remove task
	if !queue.RemoveTask(taskID) {
		s.writeError(w, http.StatusNotFound, "task not found")

		return
	}

	// Save queue
	if err := queue.Save(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to save queue: "+err.Error())

		return
	}

	// Delete notes file
	notesPath := ws.QueueNotePath(queueID, taskID)
	_ = ws.DeleteFile(notesPath)

	slog.Info("quick task deleted", "task_id", taskID)

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "task deleted",
	})
}

// handleQuickTaskCardWithID renders a single task card for HTMX swapping.
//
//nolint:unparam // r is part of the handler signature
func (s *Server) handleQuickTaskCardWithID(w http.ResponseWriter, r *http.Request, taskID string) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Load queue
	queueID := "quick-tasks"
	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		//nolint:errcheck // HTTP response write errors are handled by the server
		w.Write([]byte("<div class='text-red-500'>Error loading task</div>"))

		return
	}

	task := queue.GetTask(taskID)
	if task == nil {
		//nolint:errcheck // HTTP response write errors are handled by the server
		w.Write([]byte("<div class='text-red-500'>Task not found</div>"))

		return
	}

	// Load notes
	notes, _ := ws.LoadQueueNotes(queueID, taskID)

	// Render card HTML
	html := s.renderQuickTaskCard(task, notes)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	//nolint:errcheck // HTTP response write errors are handled by the server
	w.Write([]byte(html))
}

// renderQuickTaskCard renders HTML for a single task card.
func (s *Server) renderQuickTaskCard(task *storage.QueuedTask, notes []storage.QueueNote) string {
	var labelsHTML string
	var labelsHTMLSb656 strings.Builder
	for _, label := range task.Labels {
		labelsHTMLSb656.WriteString(fmt.Sprintf(`<span class="inline-block px-2 py-1 text-xs rounded-full bg-gray-100 text-gray-700 mr-1">%s</span>`, label))
	}
	labelsHTML += labelsHTMLSb656.String()

	var notesHTML string
	var notesHTMLSb661 strings.Builder
	for i, note := range notes {
		if i >= 3 {
			notesHTMLSb661.WriteString(fmt.Sprintf(`<div class="text-sm text-gray-500">...and %d more</div>`, len(notes)-3))

			break
		}
		notesHTMLSb661.WriteString(fmt.Sprintf(`<div class="text-sm text-gray-600 mb-1"><span class="text-gray-400">[%s]</span> %s</div>`,
			note.Timestamp, note.Content))
	}
	notesHTML += notesHTMLSb661.String()
	if len(notes) == 0 {
		notesHTML = `<div class="text-sm text-gray-400 italic">No notes yet</div>`
	}

	return fmt.Sprintf(`
<div class="bg-white rounded-lg shadow p-4 mb-4" id="card-%s">
    <div class="flex justify-between items-start">
        <div class="flex-1">
            <h3 class="font-semibold text-lg">%s</h3>
            <p class="text-gray-600 text-sm mt-1">%s</p>
            <div class="mt-2">%s</div>
        </div>
        <div class="flex gap-2 ml-4">
            <button hx-post="/api/v1/quick/%s/optimize"
                    hx-target="#card-%s"
                    hx-swap="outerHTML"
                    class="px-3 py-1 text-sm bg-purple-100 hover:bg-purple-200 text-purple-700 rounded">
                ✨ Optimize
            </button>
            <button hx-post="/api/v1/quick/%s/start"
                    class="px-3 py-1 text-sm bg-green-100 hover:bg-green-200 text-green-700 rounded">
                ▶️ Start
            </button>
            <button hx-delete="/api/v1/quick/%s"
                    hx-target="#card-%s"
                    hx-swap="outerHTML"
                    class="px-3 py-1 text-sm bg-red-100 hover:bg-red-200 text-red-700 rounded">
                🗑️
            </button>
        </div>
    </div>
    <div class="mt-4 pt-4 border-t">
        <details class="group">
            <summary class="cursor-pointer text-sm font-medium text-gray-500 hover:text-gray-700">
                Notes (%d)
            </summary>
            <div class="mt-2 space-y-1">
                %s
            </div>
            <form hx-post="/api/v1/quick/%s/note" hx-target="#card-%s" hx-swap="outerHTML" class="mt-2">
                <input type="text" name="note" placeholder="Add a note..."
                       class="w-full px-3 py-1 text-sm border rounded focus:outline-none focus:ring-2 focus:ring-blue-500">
            </form>
        </details>
    </div>
</div>`,
		task.ID,
		task.Title,
		truncateText(task.Description, 100),
		labelsHTML,
		task.ID, task.ID,
		task.ID,
		task.ID, task.ID,
		len(notes),
		notesHTML,
		task.ID, task.ID,
	)
}

// truncateText truncates text to maxLen chars.
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	return text[:maxLen-3] + "..."
}
