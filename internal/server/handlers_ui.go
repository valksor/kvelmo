package server

import (
	"net/http"
	"strings"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// handleDashboard renders the main dashboard page.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if s.templates == nil {
		// Fallback to old handler if templates not loaded
		s.handleIndex(w, r)

		return
	}

	data := DashboardData{
		Mode:             s.modeString(),
		AuthEnabled:      s.config.AuthStore != nil,
		IsGlobalMode:     s.config.Mode == ModeGlobal,
		CanSwitchProject: s.canSwitchProject(),
	}

	// In global mode, load registered projects
	if s.config.Mode == ModeGlobal {
		if registry, err := storage.LoadRegistry(); err == nil {
			data.Projects = registry.List()
		}
	}

	// Get active task info
	if s.config.Mode == ModeProject && s.config.Conductor != nil {
		activeTask := s.config.Conductor.GetActiveTask()
		if activeTask != nil {
			data.HasTask = true
			data.Task = &TaskData{
				ID:       activeTask.ID,
				Title:    "", // Will be populated from work metadata
				State:    activeTask.State,
				Branch:   activeTask.Branch,
				Worktree: activeTask.WorktreePath,
				Started:  activeTask.Started,
				Ref:      activeTask.Ref,
			}

			// Get title from work metadata
			taskWork := s.config.Conductor.GetTaskWork()
			if taskWork != nil {
				data.Task.Title = taskWork.Metadata.Title
			}

			// Get guide info for actions
			data.Guide = s.getGuideData()

			// Get specifications
			data.Specs = s.getSpecsData(activeTask.ID)

			// Get pending question
			data.PendingQuestion = s.getPendingQuestionData(activeTask.ID)

			// Get costs
			data.Costs = s.getCostsData(activeTask.ID)
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.RenderDashboard(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}

// handleTaskPartial renders the task card partial.
func (s *Server) handleTaskPartial(w http.ResponseWriter, r *http.Request) {
	if s.templates == nil || s.config.Conductor == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	data := &TaskData{
		ID:       activeTask.ID,
		State:    activeTask.State,
		Branch:   activeTask.Branch,
		Worktree: activeTask.WorktreePath,
		Started:  activeTask.Started,
		Ref:      activeTask.Ref,
	}

	taskWork := s.config.Conductor.GetTaskWork()
	if taskWork != nil {
		data.Title = taskWork.Metadata.Title
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.RenderPartial(w, "task_card", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleActionsPartial renders the actions partial.
func (s *Server) handleActionsPartial(w http.ResponseWriter, r *http.Request) {
	if s.templates == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	data := s.getGuideData()
	if data == nil {
		data = &GuideData{}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.RenderPartial(w, "actions", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleSpecsPartial renders the specifications partial.
func (s *Server) handleSpecsPartial(w http.ResponseWriter, r *http.Request) {
	if s.templates == nil || s.config.Conductor == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	data := s.getSpecsData(activeTask.ID)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.RenderPartial(w, "specs", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleQuestionPartial renders the pending question partial.
func (s *Server) handleQuestionPartial(w http.ResponseWriter, r *http.Request) {
	if s.templates == nil || s.config.Conductor == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	data := s.getPendingQuestionData(activeTask.ID)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.RenderPartial(w, "question", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleCostsPartial renders the costs partial.
func (s *Server) handleCostsPartial(w http.ResponseWriter, r *http.Request) {
	if s.templates == nil || s.config.Conductor == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	data := s.getCostsData(activeTask.ID)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.RenderPartial(w, "costs", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleLoginPageUI renders the login page using templates.
func (s *Server) handleLoginPageUI(w http.ResponseWriter, r *http.Request, errorMsg string) {
	if s.templates == nil {
		// Fallback to old handler
		s.handleLoginPage(w, r, errorMsg)

		return
	}

	data := LoginData{
		Mode:             s.modeString(),
		AuthEnabled:      s.config.AuthStore != nil,
		CanSwitchProject: s.canSwitchProject(),
		Error:            errorMsg,
		Redirect:         r.URL.Query().Get("redirect"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.RenderLogin(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}

// handleProjectUI renders the project planning page.
func (s *Server) handleProjectUI(w http.ResponseWriter, _ *http.Request) {
	if s.templates == nil {
		s.writeError(w, http.StatusInternalServerError, "templates not loaded")

		return
	}

	data := ProjectData{
		Mode:             s.modeString(),
		AuthEnabled:      s.config.AuthStore != nil,
		CanSwitchProject: s.canSwitchProject(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.RenderProject(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}

// Helper functions to get data for templates.

func (s *Server) getGuideData() *GuideData {
	if s.config.Conductor == nil {
		return nil
	}

	guide := &GuideData{
		NextActions: []ActionData{},
	}

	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		return guide
	}

	// Determine next actions based on state
	state := activeTask.State
	switch state {
	case "idle":
		// Check if specs exist
		ws := s.config.Conductor.GetWorkspace()
		if ws != nil {
			specs, _ := ws.ListSpecifications(activeTask.ID)
			if len(specs) > 0 {
				guide.NextActions = append(guide.NextActions, ActionData{
					Command:     "implement",
					Description: "Implement",
					Endpoint:    "/api/v1/workflow/implement",
					Method:      "POST",
				})
			} else {
				guide.NextActions = append(guide.NextActions, ActionData{
					Command:     "plan",
					Description: "Plan",
					Endpoint:    "/api/v1/workflow/plan",
					Method:      "POST",
				})
			}
		}
		guide.NextActions = append(guide.NextActions, ActionData{
			Command:     "abandon",
			Description: "Abandon",
			Endpoint:    "/api/v1/workflow/abandon",
			Method:      "POST",
			Dangerous:   true,
		})

	case "planning", "implementing", "reviewing":
		// Show undo if available
		guide.NextActions = append(guide.NextActions, ActionData{
			Command:     "undo",
			Description: "Undo",
			Endpoint:    "/api/v1/workflow/undo",
			Method:      "POST",
		})

	case "done":
		guide.NextActions = append(guide.NextActions, ActionData{
			Command:     "finish",
			Description: "Finish & Push",
			Endpoint:    "/api/v1/workflow/finish",
			Method:      "POST",
		})

	case "waiting":
		// Show that we're waiting for input
		guide.NextActions = append(guide.NextActions, ActionData{
			Command:     "undo",
			Description: "Undo",
			Endpoint:    "/api/v1/workflow/undo",
			Method:      "POST",
		})

	case "failed":
		guide.NextActions = append(guide.NextActions, ActionData{
			Command:     "undo",
			Description: "Undo & Retry",
			Endpoint:    "/api/v1/workflow/undo",
			Method:      "POST",
		})
		guide.NextActions = append(guide.NextActions, ActionData{
			Command:     "abandon",
			Description: "Abandon",
			Endpoint:    "/api/v1/workflow/abandon",
			Method:      "POST",
			Dangerous:   true,
		})
	}

	return guide
}

func (s *Server) getSpecsData(taskID string) []SpecData {
	if s.config.Conductor == nil {
		return nil
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		return nil
	}

	specList, err := ws.ListSpecificationsWithStatus(taskID)
	if err != nil {
		return nil
	}

	var specs []SpecData
	for _, spec := range specList {
		status := "draft"
		if spec.Status != "" {
			status = spec.Status
		}

		name := "spec-" + itoa(spec.Number)

		specs = append(specs, SpecData{
			Name:   name,
			Title:  spec.Title,
			Status: status,
		})
	}

	return specs
}

func (s *Server) getPendingQuestionData(taskID string) *QuestionData {
	if s.config.Conductor == nil {
		return nil
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		return nil
	}

	if !ws.HasPendingQuestion(taskID) {
		return nil
	}

	question, err := ws.LoadPendingQuestion(taskID)
	if err != nil {
		return nil
	}

	// Convert QuestionOption to string labels
	var options []string
	for _, opt := range question.Options {
		options = append(options, opt.Label)
	}

	return &QuestionData{
		Question: question.Question,
		Options:  options,
	}
}

func (s *Server) getCostsData(taskID string) *CostsData {
	if s.config.Conductor == nil {
		return nil
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		return nil
	}

	work, err := ws.LoadWork(taskID)
	if err != nil {
		return nil
	}

	costs := work.Costs
	total := costs.TotalInputTokens + costs.TotalOutputTokens
	if total == 0 {
		return nil
	}

	cachedPercent := 0.0
	if total > 0 {
		cachedPercent = float64(costs.TotalCachedTokens) / float64(total) * 100
	}

	return &CostsData{
		TotalCostUSD:  costs.TotalCostUSD,
		TotalTokens:   total,
		InputTokens:   costs.TotalInputTokens,
		OutputTokens:  costs.TotalOutputTokens,
		CachedTokens:  costs.TotalCachedTokens,
		CachedPercent: cachedPercent,
	}
}

// handleRecentTasksPartial renders the recent tasks list.
func (s *Server) handleRecentTasksPartial(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if s.config.Conductor == nil {
		_, _ = w.Write([]byte(`<p class="text-gray-500 text-center py-4">No tasks found</p>`))

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		_, _ = w.Write([]byte(`<p class="text-gray-500 text-center py-4">No workspace configured</p>`))

		return
	}

	taskIDs, err := ws.ListWorks()
	if err != nil || len(taskIDs) == 0 {
		_, _ = w.Write([]byte(`<p class="text-gray-500 text-center py-4">No tasks found</p>`))

		return
	}

	// Build HTML for task list
	html := `<ul class="divide-y divide-gray-200">`
	var htmlSb422 strings.Builder
	for _, id := range taskIDs {
		work, err := ws.LoadWork(id)
		if err != nil {
			continue
		}
		title := work.Metadata.Title
		if title == "" {
			title = id
		}
		htmlSb422.WriteString(`<li class="py-3 flex items-center justify-between hover:bg-gray-50 px-2 rounded">`)
		htmlSb422.WriteString(`<span class="font-medium text-gray-900">` + title + `</span>`)
		htmlSb422.WriteString(`<span class="ml-2 text-xs text-gray-400 font-mono">` + id[:8] + `</span>`)
		htmlSb422.WriteString(`</li>`)
	}
	html += htmlSb422.String()
	html += `</ul>`

	_, _ = w.Write([]byte(html))
}

// itoa converts int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	return string(digits)
}
