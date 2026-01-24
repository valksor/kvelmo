package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/valksor/go-mehrhof/internal/memory"
)

// handleMemorySearch searches the memory system.
func (s *Server) handleMemorySearch(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	mem := s.config.Conductor.GetMemory()
	if mem == nil {
		s.writeJSON(w, http.StatusOK, memorySearchResponse{
			Results: []memoryResult{},
			Count:   0,
		})

		return
	}

	// Parse query parameters
	query := r.URL.Query().Get("q")
	if query == "" {
		s.writeError(w, http.StatusBadRequest, "q parameter is required")

		return
	}

	limit := 5
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Parse document types
	var docTypes []memory.DocumentType
	if typesStr := r.URL.Query().Get("types"); typesStr != "" {
		for _, t := range strings.Split(typesStr, ",") {
			switch strings.TrimSpace(strings.ToLower(t)) {
			case "code_change", "code":
				docTypes = append(docTypes, memory.TypeCodeChange)
			case "specification", "spec":
				docTypes = append(docTypes, memory.TypeSpecification)
			case "session":
				docTypes = append(docTypes, memory.TypeSession)
			case "solution":
				docTypes = append(docTypes, memory.TypeSolution)
			case "decision":
				docTypes = append(docTypes, memory.TypeDecision)
			case "error":
				docTypes = append(docTypes, memory.TypeError)
			}
		}
	}

	// Search memory
	results, err := mem.Search(r.Context(), query, memory.SearchOptions{
		Limit:         limit,
		MinScore:      0.65,
		DocumentTypes: docTypes,
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "search failed: "+err.Error())

		return
	}

	var memResults []memoryResult
	for _, result := range results {
		memResults = append(memResults, memoryResult{
			TaskID:   result.Document.TaskID,
			Type:     string(result.Document.Type),
			Score:    float64(result.Score),
			Content:  result.Document.Content,
			Metadata: result.Document.Metadata,
		})
	}

	s.writeJSON(w, http.StatusOK, memorySearchResponse{
		Results: memResults,
		Count:   len(memResults),
	})
}

// handleMemoryIndex indexes a task to memory.
func (s *Server) handleMemoryIndex(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	mem := s.config.Conductor.GetMemory()
	if mem == nil {
		s.writeError(w, http.StatusServiceUnavailable, "memory system not available")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	var req memoryIndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.TaskID == "" {
		s.writeError(w, http.StatusBadRequest, "task_id is required")

		return
	}

	// Verify task exists
	_, err := ws.LoadWork(req.TaskID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "task not found: "+err.Error())

		return
	}

	// Create indexer and index the task
	indexer := memory.NewIndexer(mem, ws, nil)
	if err := indexer.IndexTask(r.Context(), req.TaskID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to index task: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "task indexed successfully",
		"task_id": req.TaskID,
	})
}

// handleMemoryStats returns memory system statistics.
func (s *Server) handleMemoryStats(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	mem := s.config.Conductor.GetMemory()
	if mem == nil {
		s.writeJSON(w, http.StatusOK, memoryStatsResponse{
			TotalDocuments: 0,
			ByType:         map[string]int{},
			Enabled:        false,
		})

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Create indexer to get stats
	indexer := memory.NewIndexer(mem, ws, nil)
	stats, err := indexer.GetStats(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to get stats: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, memoryStatsResponse{
		TotalDocuments: stats.TotalDocuments,
		ByType:         stats.ByType,
		Enabled:        true,
	})
}
