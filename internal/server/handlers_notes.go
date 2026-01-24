package server

import (
	"encoding/json"
	"net/http"
)

// handleAddNote adds a note to a task.
func (s *Server) handleAddNote(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Get task ID from path
	taskID := r.PathValue("id")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}

	var req addNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.Content == "" {
		s.writeError(w, http.StatusBadRequest, "content is required")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Check if there's a pending question
	wasAnswer := false
	if ws.HasPendingQuestion(taskID) {
		// Load the pending question to format the answer
		q, err := ws.LoadPendingQuestion(taskID)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to load question: "+err.Error())

			return
		}

		// Save as a Q&A pair in notes
		note := "**Q:** " + q.Question + "\n\n**A:** " + req.Content
		if err := ws.AppendNote(taskID, note, "answer"); err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to save answer: "+err.Error())

			return
		}

		// Clear the pending question
		if err := ws.ClearPendingQuestion(taskID); err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to clear question: "+err.Error())

			return
		}

		wasAnswer = true
	} else {
		// Get current task state for the note tag
		state := "note"
		activeTask := s.config.Conductor.GetActiveTask()
		if activeTask != nil && activeTask.ID == taskID {
			state = activeTask.State
		}

		// Append as regular note
		if err := ws.AppendNote(taskID, req.Content, state); err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to append note: "+err.Error())

			return
		}
	}

	message := "note added"
	if wasAnswer {
		message = "answer submitted"
	}

	s.writeJSON(w, http.StatusOK, noteResponse{
		Success:   true,
		WasAnswer: wasAnswer,
		Message:   message,
	})
}

// handleGetNotes returns the notes for a task.
func (s *Server) handleGetNotes(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Get task ID from path
	taskID := r.PathValue("id")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	content, err := ws.ReadNotes(taskID)
	if err != nil {
		// Return empty content if notes file doesn't exist
		content = ""
	}

	s.writeJSON(w, http.StatusOK, notesListResponse{
		TaskID:  taskID,
		Content: content,
	})
}
