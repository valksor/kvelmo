package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/provider/httpclient"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// Project workflow request/response types

type projectPlanRequest struct {
	Source       string `json:"source"`
	Title        string `json:"title,omitempty"`
	Instructions string `json:"instructions,omitempty"`
}

type projectPlanResponse struct {
	QueueID   string                 `json:"queue_id"`
	Title     string                 `json:"title"`
	Tasks     []*projectTaskResponse `json:"tasks"`
	Questions []string               `json:"questions,omitempty"`
	Blockers  []string               `json:"blockers,omitempty"`
}

type projectTaskResponse struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"`
	Priority    int      `json:"priority"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Blocks      []string `json:"blocks,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Assignee    string   `json:"assignee,omitempty"`
	ExternalID  string   `json:"external_id,omitempty"`
	ExternalURL string   `json:"external_url,omitempty"`
}

type projectTaskEditRequest struct {
	Title       *string  `json:"title,omitempty"`
	Description *string  `json:"description,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
	Status      *string  `json:"status,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Assignee    *string  `json:"assignee,omitempty"`
}

type projectReorderRequest struct {
	Auto        bool   `json:"auto,omitempty"`
	TaskID      string `json:"task_id,omitempty"`
	Position    string `json:"position,omitempty"` // "before" or "after"
	ReferenceID string `json:"reference_id,omitempty"`
}

type projectSubmitRequest struct {
	QueueID    string   `json:"queue_id,omitempty"`
	Provider   string   `json:"provider"`
	CreateEpic bool     `json:"create_epic,omitempty"`
	Labels     []string `json:"labels,omitempty"`
	DryRun     bool     `json:"dry_run,omitempty"`
}

type projectSubmitResponse struct {
	DryRun bool                    `json:"dry_run"`
	Epic   *projectSubmittedItem   `json:"epic,omitempty"`
	Tasks  []*projectSubmittedTask `json:"tasks"`
}

type projectSubmittedItem struct {
	ExternalID  string `json:"external_id"`
	ExternalURL string `json:"external_url"`
	Title       string `json:"title"`
}

type projectSubmittedTask struct {
	LocalID     string `json:"local_id"`
	ExternalID  string `json:"external_id"`
	ExternalURL string `json:"external_url"`
	Title       string `json:"title"`
}

type projectStartRequest struct {
	QueueID string `json:"queue_id,omitempty"`
	TaskID  string `json:"task_id,omitempty"`
	Auto    bool   `json:"auto,omitempty"`
}

type projectStartResponse struct {
	TaskID string `json:"task_id"`
	Title  string `json:"title"`
}

type projectQueueListResponse struct {
	Queues []*projectQueueSummary `json:"queues"`
}

type projectQueueSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	TaskCount int    `json:"task_count"`
}

// handleProjectPlan creates a project plan from a source.
// POST /api/v1/project/plan.
func (s *Server) handleProjectPlan(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req projectPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.Source == "" {
		s.writeError(w, http.StatusBadRequest, "source is required")

		return
	}

	opts := conductor.ProjectPlanOptions{
		Title:              req.Title,
		CustomInstructions: req.Instructions,
	}

	result, err := s.config.Conductor.CreateProjectPlan(r.Context(), req.Source, opts)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "create plan failed: "+err.Error())

		return
	}

	resp := projectPlanResponse{
		QueueID:   result.Queue.ID,
		Title:     result.Queue.Title,
		Tasks:     convertTasks(result.Tasks),
		Questions: result.Questions,
		Blockers:  result.Blockers,
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// handleProjectQueues lists all project queues.
// GET /api/v1/project/queues.
func (s *Server) handleProjectQueues(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	queueIDs, err := ws.ListQueues()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "list queues failed: "+err.Error())

		return
	}

	queues := make([]*projectQueueSummary, 0, len(queueIDs))
	for _, id := range queueIDs {
		queue, err := storage.LoadTaskQueue(ws, id)
		if err != nil {
			continue // Skip broken queues
		}
		queues = append(queues, &projectQueueSummary{
			ID:        queue.ID,
			Title:     queue.Title,
			Status:    string(queue.Status),
			TaskCount: len(queue.Tasks),
		})
	}

	s.writeJSON(w, http.StatusOK, projectQueueListResponse{Queues: queues})
}

// handleProjectQueue gets a specific queue by ID.
// GET /api/v1/project/queue/{id}.
func (s *Server) handleProjectQueue(w http.ResponseWriter, _ *http.Request, queueID string) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "queue not found: "+err.Error())

		return
	}

	resp := projectPlanResponse{
		QueueID:   queue.ID,
		Title:     queue.Title,
		Tasks:     convertQueuedTasks(queue.Tasks),
		Questions: queue.Questions,
		Blockers:  queue.Blockers,
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// handleProjectQueueDelete deletes a queue.
// DELETE /api/v1/project/queue/{id}.
func (s *Server) handleProjectQueueDelete(w http.ResponseWriter, _ *http.Request, queueID string) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	if err := ws.DeleteQueue(queueID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete queue failed: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleProjectTasks lists tasks with optional filtering.
// GET /api/v1/project/tasks?queue_id=xxx&status=ready&include_deps=true.
func (s *Server) handleProjectTasks(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	queueID := r.URL.Query().Get("queue_id")
	statusFilter := r.URL.Query().Get("status")

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// If no queue specified, use most recent
	if queueID == "" {
		queueIDs, err := ws.ListQueues()
		if err != nil || len(queueIDs) == 0 {
			s.writeError(w, http.StatusNotFound, "no queues found")

			return
		}
		queueID = queueIDs[len(queueIDs)-1]
	}

	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "queue not found: "+err.Error())

		return
	}

	// Filter tasks
	var tasks []*storage.QueuedTask
	for _, task := range queue.Tasks {
		if statusFilter != "" && string(task.Status) != statusFilter {
			continue
		}
		tasks = append(tasks, task)
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"queue_id":  queue.ID,
		"tasks":     convertQueuedTasks(tasks),
		"questions": queue.Questions,
		"blockers":  queue.Blockers,
	})
}

// handleProjectTaskEdit updates a task.
// PUT /api/v1/project/tasks/{id}.
func (s *Server) handleProjectTaskEdit(w http.ResponseWriter, r *http.Request, taskID string) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req projectTaskEditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	queueID := r.URL.Query().Get("queue_id")
	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// If no queue specified, use most recent
	if queueID == "" {
		queueIDs, err := ws.ListQueues()
		if err != nil || len(queueIDs) == 0 {
			s.writeError(w, http.StatusNotFound, "no queues found")

			return
		}
		queueID = queueIDs[len(queueIDs)-1]
	}

	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "queue not found: "+err.Error())

		return
	}

	// Update task
	err = queue.UpdateTask(taskID, func(task *storage.QueuedTask) {
		if req.Title != nil {
			task.Title = *req.Title
		}
		if req.Description != nil {
			task.Description = *req.Description
		}
		if req.Priority != nil {
			task.Priority = *req.Priority
		}
		if req.Status != nil {
			task.Status = storage.TaskStatus(*req.Status)
		}
		if req.DependsOn != nil {
			task.DependsOn = req.DependsOn
		}
		if req.Labels != nil {
			task.Labels = req.Labels
		}
		if req.Assignee != nil {
			task.Assignee = *req.Assignee
		}
	})
	if err != nil {
		s.writeError(w, http.StatusNotFound, "task not found: "+err.Error())

		return
	}

	// Recompute relationships
	queue.ComputeBlocksRelations()
	queue.ComputeTaskStatuses()

	// Save queue
	if err := queue.Save(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "save failed: "+err.Error())

		return
	}

	// Return updated task
	task := queue.GetTask(taskID)
	s.writeJSON(w, http.StatusOK, convertQueuedTask(task))
}

// handleProjectReorder reorders tasks in the queue.
// POST /api/v1/project/reorder.
func (s *Server) handleProjectReorder(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req projectReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	queueID := r.URL.Query().Get("queue_id")
	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// If no queue specified, use most recent
	if queueID == "" {
		queueIDs, err := ws.ListQueues()
		if err != nil || len(queueIDs) == 0 {
			s.writeError(w, http.StatusNotFound, "no queues found")

			return
		}
		queueID = queueIDs[len(queueIDs)-1]
	}

	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "queue not found: "+err.Error())

		return
	}

	if req.Auto {
		result, err := s.config.Conductor.AutoReorderTasks(r.Context(), queueID)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "auto reorder failed: "+err.Error())

			return
		}

		s.writeJSON(w, http.StatusOK, map[string]any{
			"old_order": result.OldOrder,
			"new_order": result.NewOrder,
			"reasoning": result.Reasoning,
		})

		return
	}

	if req.TaskID == "" || req.ReferenceID == "" || req.Position == "" {
		s.writeError(w, http.StatusBadRequest, "task_id, reference_id, and position are required")

		return
	}

	// Find target index
	var targetIndex int
	for i, task := range queue.Tasks {
		if task.ID == req.ReferenceID {
			if req.Position == "before" {
				targetIndex = i
			} else {
				targetIndex = i + 1
			}

			break
		}
	}

	// Reorder
	if err := queue.ReorderTask(req.TaskID, targetIndex); err != nil {
		s.writeError(w, http.StatusBadRequest, "reorder failed: "+err.Error())

		return
	}

	// Save queue
	if err := queue.Save(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "save failed: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"task_id":  req.TaskID,
		"position": targetIndex,
	})
}

// handleProjectSubmit submits tasks to a provider.
// POST /api/v1/project/submit.
func (s *Server) handleProjectSubmit(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req projectSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.Provider == "" {
		s.writeError(w, http.StatusBadRequest, "provider is required")

		return
	}

	queueID := req.QueueID
	if queueID == "" {
		ws := s.config.Conductor.GetWorkspace()
		if ws == nil {
			s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

			return
		}

		queueIDs, err := ws.ListQueues()
		if err != nil || len(queueIDs) == 0 {
			s.writeError(w, http.StatusNotFound, "no queues found")

			return
		}
		queueID = queueIDs[len(queueIDs)-1]
	}

	opts := conductor.SubmitOptions{
		Provider:   req.Provider,
		CreateEpic: req.CreateEpic,
		Labels:     req.Labels,
		DryRun:     req.DryRun,
	}

	result, err := s.config.Conductor.SubmitProjectTasks(r.Context(), queueID, opts)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "submit failed: "+err.Error())

		return
	}

	resp := projectSubmitResponse{
		DryRun: result.DryRun,
		Tasks:  make([]*projectSubmittedTask, 0, len(result.Tasks)),
	}

	if result.Epic != nil {
		resp.Epic = &projectSubmittedItem{
			ExternalID:  result.Epic.ExternalID,
			ExternalURL: result.Epic.ExternalURL,
			Title:       result.Epic.Title,
		}
	}

	for _, task := range result.Tasks {
		resp.Tasks = append(resp.Tasks, &projectSubmittedTask{
			LocalID:     task.LocalID,
			ExternalID:  task.ExternalID,
			ExternalURL: task.ExternalURL,
			Title:       task.Title,
		})
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// handleProjectStart starts implementing tasks from a queue.
// POST /api/v1/project/start.
func (s *Server) handleProjectStart(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req projectStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	queueID := req.QueueID
	if queueID == "" {
		ws := s.config.Conductor.GetWorkspace()
		if ws == nil {
			s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

			return
		}

		queueIDs, err := ws.ListQueues()
		if err != nil || len(queueIDs) == 0 {
			s.writeError(w, http.StatusNotFound, "no queues found")

			return
		}
		queueID = queueIDs[len(queueIDs)-1]
	}

	if req.Auto {
		// Run full automation
		opts := conductor.ProjectAutoOptions{}
		result, err := s.config.Conductor.RunProjectAuto(r.Context(), "", opts)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "auto failed: "+err.Error())

			return
		}
		s.writeJSON(w, http.StatusOK, map[string]any{
			"tasks_planned":   result.TasksPlanned,
			"tasks_submitted": result.TasksSubmitted,
			"tasks_completed": result.TasksCompleted,
		})

		return
	}

	// Start next task
	task, err := s.config.Conductor.StartNextTask(r.Context(), queueID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "start failed: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, projectStartResponse{
		TaskID: task.ID,
		Title:  task.Title,
	})
}

// handleProjectUpload handles file/archive upload for project sources.
// POST /api/v1/project/upload.
func (s *Server) handleProjectUpload(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Reuse existing file upload handler
	taskRef, err := s.handleFileUpload(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"source": taskRef,
	})
}

// handleProjectSource handles alternative source inputs (reference, URL, text).
// POST /api/v1/project/source.
func (s *Server) handleProjectSource(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req struct {
		Type     string `json:"type"`     // "reference", "url", "text"
		Value    string `json:"value"`    // The content
		Filename string `json:"filename"` // Optional filename for text type
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	var source string

	switch req.Type {
	case "reference":
		// Provider reference (github:123, jira:PROJ-123, etc.)
		source = req.Value

	case "text":
		// Save text content to temp file
		taskRef, err := s.saveContentToFile(req.Value)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to save content: "+err.Error())

			return
		}
		source = taskRef

	case "url":
		client := httpclient.NewHTTPClient()
		httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, req.Value, nil)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid URL: "+err.Error())

			return
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			s.writeError(w, http.StatusBadGateway, "fetch failed: "+err.Error())

			return
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			s.writeError(w, http.StatusBadGateway, fmt.Sprintf("fetch failed: status %d", resp.StatusCode))

			return
		}

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "read failed: "+err.Error())

			return
		}

		taskRef, err := s.saveContentToFile(string(content))
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "save failed: "+err.Error())

			return
		}
		source = taskRef

	default:
		s.writeError(w, http.StatusBadRequest, "invalid type: must be reference, url, or text")

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"source": source,
	})
}

// Helper functions

func convertTasks(tasks []*storage.QueuedTask) []*projectTaskResponse {
	result := make([]*projectTaskResponse, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, convertQueuedTask(task))
	}

	return result
}

func convertQueuedTasks(tasks []*storage.QueuedTask) []*projectTaskResponse {
	return convertTasks(tasks)
}

func convertQueuedTask(task *storage.QueuedTask) *projectTaskResponse {
	if task == nil {
		return nil
	}

	return &projectTaskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      string(task.Status),
		Priority:    task.Priority,
		DependsOn:   task.DependsOn,
		Blocks:      task.Blocks,
		Labels:      task.Labels,
		Assignee:    task.Assignee,
		ExternalID:  task.ExternalID,
		ExternalURL: task.ExternalURL,
	}
}

// parseQueueID extracts queue ID from URL path.
func parseQueueID(path, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}

// Route handlers that extract path parameters

func (s *Server) handleProjectQueueRoute(w http.ResponseWriter, r *http.Request) {
	queueID := parseQueueID(r.URL.Path, "/api/v1/project/queue/")
	if queueID == "" {
		s.writeError(w, http.StatusBadRequest, "queue ID required")

		return
	}
	s.handleProjectQueue(w, r, queueID)
}

func (s *Server) handleProjectQueueDeleteRoute(w http.ResponseWriter, r *http.Request) {
	queueID := parseQueueID(r.URL.Path, "/api/v1/project/queue/")
	if queueID == "" {
		s.writeError(w, http.StatusBadRequest, "queue ID required")

		return
	}
	s.handleProjectQueueDelete(w, r, queueID)
}

func (s *Server) handleProjectTaskEditRoute(w http.ResponseWriter, r *http.Request) {
	taskID := strings.TrimPrefix(r.URL.Path, "/api/v1/project/tasks/")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID required")

		return
	}
	s.handleProjectTaskEdit(w, r, taskID)
}
