package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rs/xid"

	"github.com/valksor/go-mehrhof/internal/events"
)

// handleAgentLogs streams agent output logs via SSE.
func (s *Server) handleAgentLogs(w http.ResponseWriter, r *http.Request) {
	// Set CORS header first (before any error response)
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Check if response writer supports flushing BEFORE setting SSE headers
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming not supported")

		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Get task ID from query params
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" && s.config.Conductor != nil {
		activeTask := s.config.Conductor.GetActiveTask()
		if activeTask != nil {
			taskID = activeTask.ID
		}
	}

	if taskID == "" {
		s.writeSSEEvent(w, flusher, "error", map[string]string{"message": "no active task"})

		return
	}

	// Subscribe to agent output events
	if s.config.EventBus == nil {
		s.writeSSEEvent(w, flusher, "error", map[string]string{"message": "event bus not available"})

		return
	}

	// Send initial connection event
	sessionID := xid.New().String()
	s.writeSSEEvent(w, flusher, "connected", map[string]string{
		"session_id": sessionID,
		"task_id":    taskID,
	})

	// Subscribe to all workflow events
	subID := s.config.EventBus.SubscribeAll(func(e events.Event) {
		// Filter events for this task
		eventTaskID, _ := e.Data["task_id"].(string)
		if eventTaskID != taskID {
			return
		}

		// Forward all workflow-related events (agent output, state changes, etc.)
		s.writeSSEEvent(w, flusher, string(e.Type), e.Data)
	})
	defer s.config.EventBus.Unsubscribe(subID)

	// Send keepalive comments every 15 seconds
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	done := r.Context().Done()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// Send SSE comment to keep connection alive
			if _, err := w.Write([]byte(": keepalive\n\n")); err != nil {
				// Client disconnected, exit the loop
				return
			}
			flusher.Flush()
		}
	}
}

// handleAgentLogsHistory returns recent agent log history.
func (s *Server) handleAgentLogsHistory(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" && s.config.Conductor != nil {
		activeTask := s.config.Conductor.GetActiveTask()
		if activeTask != nil {
			taskID = activeTask.ID
		}
	}

	if taskID == "" {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"logs": []map[string]any{},
		})

		return
	}

	if s.config.Conductor == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"logs":  []map[string]any{},
			"error": "conductor not initialized",
		})

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"logs": []map[string]any{},
		})

		return
	}

	// Get sessions for the task
	sessions, err := ws.ListSessions(taskID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "failed to load sessions",
		})

		return
	}

	// For now, return session metadata
	// In a real implementation, you'd store and retrieve actual log lines
	var logs []map[string]any
	for i, session := range sessions {
		logs = append(logs, map[string]any{
			"index":      i,
			"kind":       session.Kind,
			"started_at": session.Metadata.StartedAt,
			"agent":      session.Metadata.Agent,
			"state":      session.Metadata.State,
			"status":     "completed",
			"message":    fmt.Sprintf("Session %d completed", i+1),
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"logs":    logs,
		"task_id": taskID,
		"count":   len(logs),
	})
}
