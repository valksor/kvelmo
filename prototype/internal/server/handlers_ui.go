package server

import (
	"net/http"

	"github.com/valksor/go-mehrhof/internal/server/views"
)

// handleDashboard renders the main dashboard page.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil {
		s.writeError(w, http.StatusInternalServerError, "renderer not initialized")

		return
	}

	ws := s.getWorkspace()
	pageData := views.ComputePageData(
		s.modeString(),
		s.config.Mode == ModeGlobal,
		s.config.AuthStore != nil,
		s.canSwitchProject(),
		s.isViewer(r),
		s.getCurrentUser(r),
	)

	data := views.ComputeDashboard(s.config.Conductor, ws, pageData)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderDashboard(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}

// handleTaskPartial renders the task card partial.
func (s *Server) handleTaskPartial(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil || s.config.Conductor == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	ws := s.getWorkspace()
	data := views.ComputeActiveWork(s.config.Conductor, ws)
	if data == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderPartial(w, "active_work", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleActionsPartial renders the actions partial.
func (s *Server) handleActionsPartial(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	ws := s.getWorkspace()
	activeWork := views.ComputeActiveWork(s.config.Conductor, ws)
	actions := views.ComputeActions(activeWork, ws)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderPartial(w, "actions", actions); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleSpecificationPartial renders the specifications partial.
func (s *Server) handleSpecificationPartial(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil || s.config.Conductor == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	ws := s.getWorkspace()
	data := views.ComputeSpecifications(ws, activeTask.ID)
	if data == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderPartial(w, "specifications", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleQuestionPartial renders the pending question partial.
func (s *Server) handleQuestionPartial(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil || s.config.Conductor == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	ws := s.getWorkspace()
	data := views.ComputeQuestion(ws, activeTask.ID)
	if data == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderPartial(w, "question", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleCostsPartial renders the costs partial.
func (s *Server) handleCostsPartial(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil || s.config.Conductor == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	ws := s.getWorkspace()
	data := views.ComputeCosts(ws, activeTask.ID)
	if data == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderPartial(w, "costs", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleHierarchyPartial renders the hierarchy partial.
func (s *Server) handleHierarchyPartial(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil || s.config.Conductor == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	ws := s.getWorkspace()
	data := views.ComputeHierarchyContext(s.config.Conductor, ws, activeTask.ID)
	if data == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderPartial(w, "hierarchy", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleLoginPageUI renders the login page.
func (s *Server) handleLoginPageUI(w http.ResponseWriter, r *http.Request, errorMsg string) {
	if s.renderer == nil {
		s.writeError(w, http.StatusInternalServerError, "renderer not initialized")

		return
	}

	pageData := views.ComputePageData(
		s.modeString(),
		s.config.Mode == ModeGlobal,
		s.config.AuthStore != nil,
		s.canSwitchProject(),
		false, // Login page - no user
		"",
	)

	data := views.LoginData{
		PageData: pageData,
		Error:    errorMsg,
		Redirect: r.URL.Query().Get("redirect"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderLogin(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}

// handleProjectUI renders the project planning page.
func (s *Server) handleProjectUI(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil {
		s.writeError(w, http.StatusInternalServerError, "renderer not initialized")

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

	data := views.ProjectPlanningData{
		PageData: pageData,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderProject(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}

// handleHistoryUI renders the task history page.
func (s *Server) handleHistoryUI(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil {
		s.writeError(w, http.StatusInternalServerError, "renderer not initialized")

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

	data := views.HistoryData{
		PageData: pageData,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderHistory(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}

// handleRecentTasksPartial renders the recent tasks list.
func (s *Server) handleRecentTasksPartial(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if s.renderer == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	ws := s.getWorkspace()
	data := views.ComputeRecentTasks(ws, 10)
	if len(data) == 0 {
		// Render empty state instead of 204 so HTMX can swap content
		if err := s.renderer.RenderEmptyState(w, "no_recent_tasks", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	if err := s.renderer.RenderPartial(w, "recent_tasks", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleWorkspaceStatsPartial renders the workspace statistics card.
func (s *Server) handleWorkspaceStatsPartial(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if s.renderer == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	ws := s.getWorkspace()
	data := views.ComputeStats(ws)
	if data == nil || data.TotalTasks == 0 {
		// Render empty state instead of 204 so HTMX can swap content
		if err := s.renderer.RenderEmptyState(w, "no_stats", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	if err := s.renderer.RenderPartial(w, "stats", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleLicensePage renders the license information page.
func (s *Server) handleLicensePage(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil {
		s.writeError(w, http.StatusInternalServerError, "renderer not initialized")

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

	data := views.LicenseData{
		PageData: pageData,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderLicense(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}
