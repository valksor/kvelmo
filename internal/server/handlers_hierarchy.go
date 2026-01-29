package server

import (
	"net/http"

	"github.com/valksor/go-mehrhof/internal/server/views"
)

// handleGetHierarchy returns hierarchical context for the active task.
func (s *Server) handleGetHierarchy(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"active":    false,
			"hierarchy": views.HierarchyData{},
		})

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	data := views.ComputeHierarchyContext(s.config.Conductor, ws, activeTask.ID)
	if data == nil {
		data = &views.HierarchyData{}
	}

	s.writeJSON(w, http.StatusOK, data)
}
