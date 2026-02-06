package server

import (
	"net/http"
	"strconv"
)

// handleGetSpecificationDiff returns a unified diff for a specification's implemented file.
func (s *Server) handleGetSpecificationDiff(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	taskID := r.PathValue("id")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}

	specNumberRaw := r.PathValue("number")
	if specNumberRaw == "" {
		s.writeError(w, http.StatusBadRequest, "specification number is required")

		return
	}

	specNumber, err := strconv.Atoi(specNumberRaw)
	if err != nil || specNumber <= 0 {
		s.writeError(w, http.StatusBadRequest, "specification number must be a positive integer")

		return
	}

	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		s.writeError(w, http.StatusBadRequest, "file query parameter is required")

		return
	}

	contextLines := 3
	if contextRaw := r.URL.Query().Get("context"); contextRaw != "" {
		parsed, parseErr := strconv.Atoi(contextRaw)
		if parseErr != nil || parsed < 0 {
			s.writeError(w, http.StatusBadRequest, "context must be a non-negative integer")

			return
		}
		contextLines = parsed
	}

	diff, err := s.config.Conductor.GetSpecificationFileDiff(r.Context(), taskID, specNumber, filePath, contextLines)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"task_id":       taskID,
		"specification": specNumber,
		"file":          filePath,
		"context":       contextLines,
		"has_diff":      diff != "",
		"diff":          diff,
	})
}
