package server

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/valksor/go-toolkit/eventbus"
)

// handleAgentLogs streams agent output logs via SSE.
func (s *Server) handleAgentLogs(w http.ResponseWriter, r *http.Request) {
	// Disable write timeout for SSE (allows indefinite streaming during long agent operations)
	// This prevents the connection from being killed by the server's WriteTimeout (default 30s)
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		slog.Debug("could not disable write deadline for SSE", "error", err)
	}

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

	// Use channel-based event delivery to prevent panic on client disconnect.
	// The callback only writes to a channel (never panics), while the main loop
	// checks ctx.Done() before writing to ResponseWriter.
	eventCh := make(chan eventbus.Event, 100)
	subID := s.config.EventBus.SubscribeAll(func(e eventbus.Event) {
		// Filter events for this task
		eventTaskID, _ := e.Data["task_id"].(string)
		if eventTaskID != taskID {
			return
		}
		select {
		case eventCh <- e:
		default:
			// Channel full, drop event (non-blocking to prevent callback from hanging)
		}
	})
	defer s.config.EventBus.Unsubscribe(subID)
	defer close(eventCh)

	// Run combined event + heartbeat loop (blocks until client disconnects)
	s.runSSEEventLoop(w, flusher, r.Context(), eventCh)
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
	transcripts, err := ws.ListTranscripts(taskID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "failed to load transcripts",
		})

		return
	}

	sort.Strings(transcripts)

	// Return transcript lines as agent log history so UI can hydrate terminal output.
	var logs []map[string]any
	lineIndex := 0
	for _, transcriptFile := range transcripts {
		content, loadErr := ws.LoadTranscript(taskID, transcriptFile)
		if loadErr != nil {
			continue
		}

		kind, startedAt := parseTranscriptMetadata(transcriptFile)
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			logs = append(logs, map[string]any{
				"index":      lineIndex,
				"kind":       kind,
				"started_at": startedAt,
				"file":       transcriptFile,
				"type":       "output",
				"message":    line,
			})
			lineIndex++
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"logs":    logs,
		"task_id": taskID,
		"count":   len(logs),
	})
}

func parseTranscriptMetadata(filename string) (string, string) {
	kind := "unknown"
	startedAt := ""

	trimmed := strings.TrimSuffix(filename, ".log")
	lastDash := strings.LastIndex(trimmed, "-")
	if lastDash <= 0 || lastDash >= len(trimmed)-1 {
		return kind, startedAt
	}

	timestampPart := trimmed[:lastDash]
	kind = trimmed[lastDash+1:]
	if parsed, err := time.Parse("2006-01-02T15-04-05", timestampPart); err == nil {
		startedAt = parsed.Format(time.RFC3339)
	}

	return kind, startedAt
}
