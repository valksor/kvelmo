package server

import (
	"fmt"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/server/views"
)

// handleTaskContentPartial returns the full task content as rendered HTML in a modal.
// Called via HTMX to show task content in a popup modal.
func (s *Server) handleTaskContentPartial(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.getWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Get task ID from query param or use active task
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		activeTask := s.config.Conductor.GetActiveTask()
		if activeTask == nil {
			s.writeError(w, http.StatusNotFound, "no active task")

			return
		}
		taskID = activeTask.ID
	}

	// Load source content
	content, err := ws.GetSourceContent(taskID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to load task content: "+err.Error())

		return
	}

	// Get title from work metadata
	title := taskID
	if work, err := ws.LoadWork(taskID); err == nil && work.Metadata.Title != "" {
		title = work.Metadata.Title
	}

	// Render markdown to HTML
	contentHTML, err := views.RenderMarkdown(content)
	if err != nil {
		// Fallback to plain text
		contentHTML = "<pre>" + content + "</pre>"
	}

	// Return modal HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(buildTaskContentModal(title, contentHTML)))
}

// buildTaskContentModal builds the HTML for the task content modal.
func buildTaskContentModal(title, contentHTML string) string {
	return fmt.Sprintf(`<div id="task-content-modal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4" onclick="if(event.target===this)this.remove()">
    <div class="bg-base-100 rounded-2xl shadow-2xl max-w-4xl w-full max-h-[85vh] flex flex-col" onclick="event.stopPropagation()">
        <div class="px-6 py-4 border-b border-base-300 flex items-center justify-between flex-shrink-0">
            <h3 class="text-lg font-bold text-base-content truncate">%s</h3>
            <button type="button" onclick="document.getElementById('task-content-modal').remove()"
                    class="p-2 rounded-lg text-base-content/40 hover:text-base-content/60 hover:bg-base-200 transition-colors">
                <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd"></path>
                </svg>
            </button>
        </div>
        <div class="p-6 overflow-y-auto prose prose-sm dark:prose-invert max-w-none">
            %s
        </div>
    </div>
</div>`, title, contentHTML)
}
