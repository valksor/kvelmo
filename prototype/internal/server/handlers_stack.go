package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/server/api"
	"github.com/valksor/go-mehrhof/internal/server/views"
	"github.com/valksor/go-mehrhof/internal/stack"
)

// Stack API request/response types.

type stackListResponse struct {
	Stacks []stackSummary `json:"stacks"`
	Count  int            `json:"count"`
}

type stackSummary struct {
	ID          string      `json:"id"`
	RootTask    string      `json:"root_task"`
	TaskCount   int         `json:"task_count"`
	Tasks       []taskState `json:"tasks"`
	CreatedAt   string      `json:"created_at"`
	UpdatedAt   string      `json:"updated_at"`
	HasRebase   bool        `json:"has_rebase"`   // True if any task needs rebase
	HasConflict bool        `json:"has_conflict"` // True if any task has conflict
}

type taskState struct {
	ID        string `json:"id"`
	Branch    string `json:"branch"`
	State     string `json:"state"`
	PRNumber  int    `json:"pr_number,omitempty"`
	PRURL     string `json:"pr_url,omitempty"`
	DependsOn string `json:"depends_on,omitempty"`
	StateIcon string `json:"state_icon"`
}

type stackSyncResponse struct {
	Success      bool         `json:"success"`
	Updated      int          `json:"updated"`
	UpdatedTasks []taskUpdate `json:"updated_tasks,omitempty"`
	Errors       []string     `json:"errors,omitempty"`
}

type taskUpdate struct {
	TaskID   string `json:"task_id"`
	OldState string `json:"old_state"`
	NewState string `json:"new_state"`
	Children int    `json:"children_marked,omitempty"`
}

type rebaseRequest struct {
	StackID string `json:"stack_id,omitempty"` // If empty, rebase all stacks
	TaskID  string `json:"task_id,omitempty"`  // If set, only rebase this task
}

type rebaseResponse struct {
	Success bool              `json:"success"`
	Rebased int               `json:"rebased"`
	Results []rebaseResult    `json:"results,omitempty"`
	Failed  *failedRebaseInfo `json:"failed,omitempty"`
	Error   string            `json:"error,omitempty"`
}

type rebaseResult struct {
	TaskID  string `json:"task_id"`
	Branch  string `json:"branch"`
	OldBase string `json:"old_base"`
	NewBase string `json:"new_base"`
}

type failedRebaseInfo struct {
	TaskID       string `json:"task_id"`
	Branch       string `json:"branch"`
	OntoBase     string `json:"onto_base"`
	IsConflict   bool   `json:"is_conflict"`
	ConflictHint string `json:"conflict_hint,omitempty"`
}

// Preview response types.

type rebasePreviewResponse struct {
	Tasks             []taskPreview `json:"tasks"`
	HasConflicts      bool          `json:"has_conflicts"`
	SafeCount         int           `json:"safe_count"`
	ConflictCount     int           `json:"conflict_count"`
	Unavailable       bool          `json:"unavailable"`
	UnavailableReason string        `json:"unavailable_reason,omitempty"`
}

type taskPreview struct {
	TaskID           string   `json:"task_id"`
	Branch           string   `json:"branch"`
	OntoBase         string   `json:"onto_base"`
	Safe             bool     `json:"safe"`
	WouldConflict    bool     `json:"would_conflict,omitempty"`
	ConflictingFiles []string `json:"conflicting_files,omitempty"`
	Unavailable      bool     `json:"unavailable,omitempty"`
}

// handleStacksUI renders the stacks management page.
// GET /stack.
func (s *Server) handleStacksUI(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil {
		http.Error(w, "renderer not loaded", http.StatusInternalServerError)

		return
	}

	pageData := views.ComputePageData(
		s.modeString(),
		s.config.Mode == ModeGlobal,
		s.config.AuthStore != nil,
		s.canSwitchProject(),
		s.isViewer(r),
		s.getCurrentUser(r),
	)

	data := views.StackData{
		PageData: pageData,
	}

	// Load stacks if conductor is available
	if s.config.Conductor != nil {
		ws := s.config.Conductor.GetWorkspace()
		if ws != nil {
			storage := stack.NewStorage(ws.DataRoot())
			if err := storage.Load(); err == nil {
				stacks := storage.ListStacks()
				for _, st := range stacks {
					stackView := views.StackViewData{
						ID:        st.ID,
						RootTask:  st.RootTask,
						TaskCount: st.TaskCount(),
						CreatedAt: st.CreatedAt.Format("2006-01-02 15:04"),
						UpdatedAt: st.UpdatedAt.Format("2006-01-02 15:04"),
					}

					// Add tasks
					for _, task := range st.Tasks {
						taskView := views.StackTaskView{
							ID:        task.ID,
							Branch:    task.Branch,
							State:     string(task.State),
							StateIcon: getStackStateIcon(task.State),
							DependsOn: task.DependsOn,
							PRNumber:  task.PRNumber,
							PRURL:     task.PRURL,
						}
						stackView.Tasks = append(stackView.Tasks, taskView)

						// Track if stack has rebase or conflict
						if task.State == stack.StateNeedsRebase {
							stackView.HasRebase = true
						}
						if task.State == stack.StateConflict {
							stackView.HasConflict = true
						}
					}
					data.Stacks = append(data.Stacks, stackView)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderStack(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}

// handleStackList returns all stacks.
// GET /api/v1/stack.
func (s *Server) handleStackList(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	storage := stack.NewStorage(ws.DataRoot())
	if err := storage.Load(); err != nil {
		// No stacks file yet, return empty list
		s.writeJSON(w, http.StatusOK, stackListResponse{
			Stacks: []stackSummary{},
			Count:  0,
		})

		return
	}

	stacks := storage.ListStacks()
	response := stackListResponse{
		Stacks: make([]stackSummary, 0, len(stacks)),
		Count:  len(stacks),
	}

	for _, st := range stacks {
		summary := stackSummary{
			ID:        st.ID,
			RootTask:  st.RootTask,
			TaskCount: st.TaskCount(),
			Tasks:     make([]taskState, 0, len(st.Tasks)),
			CreatedAt: st.CreatedAt.Format("2006-01-02T15:04:05"),
			UpdatedAt: st.UpdatedAt.Format("2006-01-02T15:04:05"),
		}

		for _, task := range st.Tasks {
			summary.Tasks = append(summary.Tasks, taskState{
				ID:        task.ID,
				Branch:    task.Branch,
				State:     string(task.State),
				PRNumber:  task.PRNumber,
				PRURL:     task.PRURL,
				DependsOn: task.DependsOn,
				StateIcon: getStackStateIcon(task.State),
			})

			if task.State == stack.StateNeedsRebase {
				summary.HasRebase = true
			}
			if task.State == stack.StateConflict {
				summary.HasConflict = true
			}
		}

		response.Stacks = append(response.Stacks, summary)
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleStackSync syncs PR status for all stacks.
// POST /api/v1/stack/sync.
func (s *Server) handleStackSync(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	storage := stack.NewStorage(ws.DataRoot())
	if err := storage.Load(); err != nil {
		s.writeJSON(w, http.StatusOK, stackSyncResponse{
			Success: true,
			Updated: 0,
		})

		return
	}

	// For now, return a placeholder - full sync requires provider integration
	// which is done in the CLI via `mehr stack sync`
	slog.Info("stack sync requested via Web UI")

	s.writeJSON(w, http.StatusOK, stackSyncResponse{
		Success: true,
		Updated: 0,
	})
}

// handleStackRebase rebases stacked tasks.
// POST /api/v1/stack/rebase.
func (s *Server) handleStackRebase(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	git := s.config.Conductor.GetGit()
	if git == nil {
		s.writeError(w, http.StatusServiceUnavailable, "git not initialized")

		return
	}

	// Parse request
	var req rebaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	storage := stack.NewStorage(ws.DataRoot())
	rebaser := stack.NewRebaser(storage, git)

	var result *stack.RebaseResult
	var err error

	if req.TaskID != "" {
		// Rebase specific task
		result, err = rebaser.RebaseTask(r.Context(), req.TaskID)
	} else if req.StackID != "" {
		// Rebase specific stack
		result, err = rebaser.RebaseAll(r.Context(), req.StackID)
	} else {
		// Rebase all stacks that have tasks needing rebase
		if loadErr := storage.Load(); loadErr != nil {
			s.writeJSON(w, http.StatusOK, rebaseResponse{
				Success: true,
				Rebased: 0,
			})

			return
		}

		allResults := make([]rebaseResult, 0)
		for _, st := range storage.ListStacks() {
			if len(st.GetTasksNeedingRebase()) > 0 {
				result, err = rebaser.RebaseAll(r.Context(), st.ID)
				if err != nil {
					break
				}
				for _, tr := range result.RebasedTasks {
					allResults = append(allResults, rebaseResult{
						TaskID:  tr.TaskID,
						Branch:  tr.Branch,
						OldBase: tr.OldBase,
						NewBase: tr.NewBase,
					})
				}
			}
		}

		if err == nil {
			s.writeJSON(w, http.StatusOK, rebaseResponse{
				Success: true,
				Rebased: len(allResults),
				Results: allResults,
			})

			return
		}
	}

	// Handle single stack/task result
	if err != nil {
		response := rebaseResponse{
			Success: false,
			Error:   err.Error(),
		}

		if result != nil && result.FailedTask != nil {
			response.Failed = &failedRebaseInfo{
				TaskID:       result.FailedTask.TaskID,
				Branch:       result.FailedTask.Branch,
				OntoBase:     result.FailedTask.OntoBase,
				IsConflict:   result.FailedTask.IsConflict,
				ConflictHint: result.FailedTask.ConflictHint,
			}
		}

		s.writeJSON(w, http.StatusOK, response)

		return
	}

	// Build successful response
	response := rebaseResponse{
		Success: true,
		Rebased: len(result.RebasedTasks),
		Results: make([]rebaseResult, 0, len(result.RebasedTasks)),
	}

	for _, tr := range result.RebasedTasks {
		response.Results = append(response.Results, rebaseResult{
			TaskID:  tr.TaskID,
			Branch:  tr.Branch,
			OldBase: tr.OldBase,
			NewBase: tr.NewBase,
		})
	}

	slog.Info("stack rebase completed", "rebased", len(result.RebasedTasks))

	s.writeJSON(w, http.StatusOK, response)
}

// handleStackRebasePreview returns a preview of what would happen during rebase.
// GET /api/v1/stack/rebase-preview
// GET /api/v1/stack/{id}/rebase-preview.
func (s *Server) handleStackRebasePreview(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	git := s.config.Conductor.GetGit()
	if git == nil {
		s.writeError(w, http.StatusServiceUnavailable, "git not initialized")

		return
	}

	// Get stack ID from query parameter or path
	stackID := r.URL.Query().Get("stack_id")
	taskID := r.URL.Query().Get("task_id")

	storage := stack.NewStorage(ws.DataRoot())
	if err := storage.Load(); err != nil {
		// No stacks yet
		s.writeJSON(w, http.StatusOK, rebasePreviewResponse{
			Tasks: []taskPreview{},
		})

		return
	}

	rebaser := stack.NewRebaser(storage, git)

	// Preview single task
	if taskID != "" {
		preview, err := rebaser.PreviewTask(r.Context(), taskID)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "preview task: "+err.Error())

			return
		}

		response := rebasePreviewResponse{
			Tasks: []taskPreview{
				{
					TaskID:           preview.TaskID,
					Branch:           preview.Branch,
					OntoBase:         preview.OntoBase,
					Safe:             !preview.WouldConflict && !preview.Unavailable,
					WouldConflict:    preview.WouldConflict,
					ConflictingFiles: preview.ConflictingFiles,
					Unavailable:      preview.Unavailable,
				},
			},
			HasConflicts:  preview.WouldConflict,
			SafeCount:     boolToInt(!preview.WouldConflict && !preview.Unavailable),
			ConflictCount: boolToInt(preview.WouldConflict),
			Unavailable:   preview.Unavailable,
		}

		s.writeJSON(w, http.StatusOK, response)

		return
	}

	// Preview specific stack
	if stackID != "" {
		preview, err := rebaser.PreviewRebase(r.Context(), stackID)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "preview rebase: "+err.Error())

			return
		}

		response := convertPreviewToResponse(preview)

		// Return HTML for HTMX requests
		if api.IsHTMXRequest(r) {
			s.renderRebasePreviewHTML(w, response)

			return
		}

		s.writeJSON(w, http.StatusOK, response)

		return
	}

	// Preview all stacks
	allTasks := make([]taskPreview, 0)
	var totalSafe, totalConflict int
	var unavailable bool
	var unavailableReason string

	for _, st := range storage.ListStacks() {
		if len(st.GetTasksNeedingRebase()) == 0 {
			continue
		}

		preview, err := rebaser.PreviewRebase(r.Context(), st.ID)
		if err != nil {
			slog.Warn("failed to preview stack", "stack_id", st.ID, "error", err)

			continue
		}

		for _, task := range preview.Tasks {
			allTasks = append(allTasks, taskPreview{
				TaskID:           task.TaskID,
				Branch:           task.Branch,
				OntoBase:         task.OntoBase,
				Safe:             !task.WouldConflict && !task.Unavailable,
				WouldConflict:    task.WouldConflict,
				ConflictingFiles: task.ConflictingFiles,
				Unavailable:      task.Unavailable,
			})
		}

		totalSafe += preview.SafeCount
		totalConflict += preview.ConflictCount
		if preview.Unavailable {
			unavailable = true
			if unavailableReason == "" {
				unavailableReason = preview.UnavailableReason
			}
		}
	}

	s.writeJSON(w, http.StatusOK, rebasePreviewResponse{
		Tasks:             allTasks,
		HasConflicts:      totalConflict > 0,
		SafeCount:         totalSafe,
		ConflictCount:     totalConflict,
		Unavailable:       unavailable,
		UnavailableReason: unavailableReason,
	})
}

// convertPreviewToResponse converts a stack.RebasePreview to API response.
func convertPreviewToResponse(preview *stack.RebasePreview) rebasePreviewResponse {
	tasks := make([]taskPreview, 0, len(preview.Tasks))
	for _, task := range preview.Tasks {
		tasks = append(tasks, taskPreview{
			TaskID:           task.TaskID,
			Branch:           task.Branch,
			OntoBase:         task.OntoBase,
			Safe:             !task.WouldConflict && !task.Unavailable,
			WouldConflict:    task.WouldConflict,
			ConflictingFiles: task.ConflictingFiles,
			Unavailable:      task.Unavailable,
		})
	}

	return rebasePreviewResponse{
		Tasks:             tasks,
		HasConflicts:      preview.HasConflicts,
		SafeCount:         preview.SafeCount,
		ConflictCount:     preview.ConflictCount,
		Unavailable:       preview.Unavailable,
		UnavailableReason: preview.UnavailableReason,
	}
}

// boolToInt converts a boolean to 0 or 1.
func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

// renderRebasePreviewHTML renders the preview as HTML for HTMX requests.
func (s *Server) renderRebasePreviewHTML(w http.ResponseWriter, preview rebasePreviewResponse) {
	if s.renderer == nil {
		s.writeError(w, http.StatusInternalServerError, "renderer not loaded")

		return
	}

	// Convert to view data
	data := views.RebasePreviewData{
		Tasks:             make([]views.RebaseTaskPreview, 0, len(preview.Tasks)),
		HasConflicts:      preview.HasConflicts,
		SafeCount:         preview.SafeCount,
		ConflictCount:     preview.ConflictCount,
		Unavailable:       preview.Unavailable,
		UnavailableReason: preview.UnavailableReason,
	}

	for _, task := range preview.Tasks {
		data.Tasks = append(data.Tasks, views.RebaseTaskPreview{
			TaskID:           task.TaskID,
			Branch:           task.Branch,
			OntoBase:         task.OntoBase,
			Safe:             task.Safe,
			WouldConflict:    task.WouldConflict,
			ConflictingFiles: task.ConflictingFiles,
			Unavailable:      task.Unavailable,
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderRebasePreview(w, data); err != nil {
		slog.Error("failed to render rebase preview", "error", err)
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
	}
}

// getStackStateIcon returns the icon for a stack state.
func getStackStateIcon(state stack.StackState) string {
	switch state {
	case stack.StateMerged:
		return "check"
	case stack.StateNeedsRebase:
		return "refresh"
	case stack.StateConflict:
		return "x-circle"
	case stack.StatePendingReview:
		return "clock"
	case stack.StateApproved:
		return "check-circle"
	case stack.StateAbandoned:
		return "slash"
	case stack.StateActive:
		return "play"
	default:
		return "circle"
	}
}
