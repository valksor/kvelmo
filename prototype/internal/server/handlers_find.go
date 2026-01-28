package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rs/xid"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/server/views"
)

// handleFindSearch handles find search requests via Web UI.
// Supports both JSON responses and SSE streaming.
func (s *Server) handleFindSearch(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Parse query from both GET and POST
	var query string
	if r.Method == http.MethodPost {
		// Parse JSON body
		var payload struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body")

			return
		}
		query = payload.Query
	} else {
		// GET request
		query = r.URL.Query().Get("q")
	}

	if query == "" {
		s.writeError(w, http.StatusBadRequest, "query is required")

		return
	}

	// Parse options
	path := r.URL.Query().Get("path")
	pattern := r.URL.Query().Get("pattern")

	contextLines := 3
	if c := r.URL.Query().Get("context"); c != "" {
		if n, err := strconv.Atoi(c); err == nil && n > 0 {
			contextLines = n
		}
	}

	// Check if streaming is requested
	stream := r.URL.Query().Get("stream") == "true"

	if stream {
		s.streamFindResults(w, r, query, path, pattern, contextLines)
	} else {
		s.handleFindSearchJSON(w, r, query, path, pattern, contextLines)
	}
}

// streamFindResults streams search results via Server-Sent Events.
func (s *Server) streamFindResults(w http.ResponseWriter, r *http.Request, query, path, pattern string, contextLines int) {
	// Check if response writer supports flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming not supported")

		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Build find options
	findOpts := conductor.FindOptions{
		Query:   query,
		Path:    path,
		Pattern: pattern,
		Context: contextLines,
	}

	// Get result channel from conductor
	resultChan, err := s.config.Conductor.Find(r.Context(), findOpts)
	if err != nil {
		s.writeSSEEvent(w, flusher, "error", map[string]string{
			"message": err.Error(),
		})

		return
	}

	// Generate session ID
	sessionID := xid.New().String()

	// Send started event
	s.writeSSEEvent(w, flusher, "started", map[string]string{
		"session_id": sessionID,
		"query":      query,
	})

	// Stream results
	count := 0
	for result := range resultChan {
		if result.File == "__error__" {
			s.writeSSEEvent(w, flusher, "error", map[string]string{
				"message": result.Snippet,
			})

			continue
		}

		count++
		s.writeSSEEvent(w, flusher, "result", map[string]any{
			"file":    result.File,
			"line":    result.Line,
			"snippet": result.Snippet,
			"context": result.Context,
			"reason":  result.Reason,
		})
		flusher.Flush()
	}

	// Send complete event
	s.writeSSEEvent(w, flusher, "complete", map[string]any{
		"session_id": sessionID,
		"count":      count,
	})
}

// handleFindSearchJSON returns search results as JSON.
func (s *Server) handleFindSearchJSON(w http.ResponseWriter, r *http.Request, query, path, pattern string, contextLines int) {
	// Build find options
	findOpts := conductor.FindOptions{
		Query:   query,
		Path:    path,
		Pattern: pattern,
		Context: contextLines,
	}

	// Get result channel from conductor
	resultChan, err := s.config.Conductor.Find(r.Context(), findOpts)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())

		return
	}

	// Collect all results
	var results []conductor.FindResult
	for result := range resultChan {
		if result.File == "__error__" {
			s.writeError(w, http.StatusInternalServerError, result.Snippet)

			return
		}
		results = append(results, result)
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"query":   query,
		"count":   len(results),
		"matches": results,
	})
}

// handleFindUI renders the find search UI page.
func (s *Server) handleFindUI(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil {
		s.writeError(w, http.StatusInternalServerError, "renderer not loaded")

		return
	}

	pageData := views.ComputePageData(
		s.modeString(),
		s.config.Mode == ModeGlobal,
		s.config.AuthStore != nil,
		s.canSwitchProject(),
		s.getCurrentUser(r),
	)

	data := map[string]any{
		"PageData": pageData,
		"Title":    "Find - Code Search",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.Render(w, "find", data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}
