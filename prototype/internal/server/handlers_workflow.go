package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor/commands"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/eventbus"
)

// handleWorkflowQuestion asks the agent a question during planning/implementing/reviewing.
// Streams the agent's response via SSE.
func (s *Server) handleWorkflowQuestion(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Parse request - handle both form and JSON submissions
	var question string
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid form data: "+err.Error())

			return
		}
		question = r.FormValue("question")
	} else {
		var req questionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

			return
		}
		question = req.Question
	}

	if question == "" {
		s.writeError(w, http.StatusBadRequest, "question is required")

		return
	}

	// Validate state allows questions BEFORE starting SSE stream.
	// Once SSE headers are set, we're committed to HTTP 200.
	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		s.writeError(w, http.StatusBadRequest, "no active task")

		return
	}

	allowedStates := map[string]bool{
		string(workflow.StatePlanning):     true,
		string(workflow.StateImplementing): true,
		string(workflow.StateReviewing):    true,
	}
	if !allowedStates[activeTask.State] {
		s.writeError(w, http.StatusConflict, fmt.Sprintf(
			"cannot ask questions in state '%s'; use during planning, implementing, or reviewing",
			activeTask.State,
		))

		return
	}

	// Set SSE headers AFTER validation passes
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Subscribe to agent message events for streaming
	eventBus := s.config.Conductor.GetEventBus()
	if eventBus == nil {
		s.writeErrorSSE(w, "event bus not available")

		return
	}

	// Set up event subscription for this request
	eventCh := make(chan eventbus.Event, 100)
	unsubscribeID := eventBus.SubscribeAll(func(e eventbus.Event) {
		if e.Type == events.TypeAgentMessage {
			select {
			case eventCh <- e:
			default:
				// Channel full, drop event
			}
		}
	})
	defer eventBus.Unsubscribe(unsubscribeID)
	defer close(eventCh)

	// Start streaming in background
	ctx := r.Context()
	go func() {
		result, err := commands.ExecuteWithRun(ctx, s.config.Conductor, "question", commands.Invocation{
			Source: commands.SourceAPI,
			Args:   []string{question},
		})
		if err != nil {
			sendSSE(w, "", "{\"event\":\"error\",\"error\":\""+escapeJSON(err.Error())+"\"}")
			sendSSE(w, "", "{\"event\":\"done\"}")

			return
		}
		if result != nil && result.Type == commands.ResultWaiting {
			sendSSE(w, "", "{\"event\":\"back_question\",\"question\":\"Agent has a follow-up question\"}")
		}
		sendSSE(w, "", "{\"event\":\"done\"}")
	}()

	// Stream events to client
	for {
		select {
		case <-r.Context().Done():
			return
		case e, ok := <-eventCh:
			if !ok {
				return
			}
			if eventType, ok := e.Data["type"].(string); ok && eventType == "text" {
				if text, ok := e.Data["content"].(string); ok && text != "" {
					sendSSE(w, "", "{\"event\":\"content\",\"text\":\""+escapeJSON(text)+"\"}")
				}
			}
		}
	}
}

// sendSSE sends a Server-Sent Event to the client.
func sendSSE(w http.ResponseWriter, _ string, data string) {
	_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
	flusher, _ := w.(http.Flusher)
	if flusher != nil {
		flusher.Flush()
	}
}

// escapeJSON escapes a string for safe inclusion in JSON.
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")

	return s
}

// writeErrorSSE writes an error via SSE and closes the connection.
func (s *Server) writeErrorSSE(w http.ResponseWriter, message string) {
	sendSSE(w, "", "{\"event\":\"error\",\"error\":\""+escapeJSON(message)+"\"}")
}
