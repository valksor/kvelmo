package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rs/xid"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/conductor/commands"
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

	result, err := commands.ExecuteWithRun(r.Context(), s.config.Conductor, "find", commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"query":   query,
			"path":    path,
			"pattern": pattern,
			"context": contextLines,
		},
	})
	if err != nil {
		s.writeSSEEvent(w, flusher, "error", map[string]string{
			"message": err.Error(),
		})

		return
	}
	if result == nil {
		s.writeSSEEvent(w, flusher, "error", map[string]string{"message": "empty response"})

		return
	}
	if result.Type == commands.ResultError {
		s.writeSSEEvent(w, flusher, "error", map[string]string{"message": result.Message})

		return
	}
	data, ok := result.Data.(map[string]any)
	if !ok {
		s.writeSSEEvent(w, flusher, "error", map[string]string{"message": "invalid response payload"})

		return
	}

	matches, _ := data["matches"].([]conductor.FindResult)

	// Generate session ID
	sessionID := xid.New().String()

	// Send started event
	s.writeSSEEvent(w, flusher, "started", map[string]string{
		"session_id": sessionID,
		"query":      query,
	})

	// Stream results
	count := 0
	for _, match := range matches {
		count++
		s.writeSSEEvent(w, flusher, "result", map[string]any{
			"file":    match.File,
			"line":    match.Line,
			"snippet": match.Snippet,
			"context": match.Context,
			"reason":  match.Reason,
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
	result, err := commands.ExecuteWithRun(r.Context(), s.config.Conductor, "find", commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"query":   query,
			"path":    path,
			"pattern": pattern,
			"context": contextLines,
		},
	})
	if err != nil {
		s.mapErrorToHTTP(w, err)

		return
	}
	if result == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"query":   query,
			"count":   0,
			"matches": []conductor.FindResult{},
		})

		return
	}
	if payload, ok := result.Data.(map[string]any); ok {
		s.writeJSON(w, http.StatusOK, payload)

		return
	}

	s.writeCommandResult(w, result)
}
