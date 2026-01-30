package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

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
