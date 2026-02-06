package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/conductor/commands"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/eventbus"
)

// Interactive chat request/response types.

type chatMessage struct {
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type chatRequest struct {
	Message string `json:"message"`
}

type chatResponse struct {
	Success  bool          `json:"success"`
	Message  string        `json:"message,omitempty"`
	Messages []chatMessage `json:"messages,omitempty"`
}

type commandRequest struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

type commandResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type stateResponse struct {
	Success bool   `json:"success"`
	State   string `json:"state,omitempty"`
	TaskID  string `json:"task_id,omitempty"`
	Title   string `json:"title,omitempty"`
}

// parseChatRequest parses chat request from either JSON or form data.
// Supports both JSON and form-encoded requests.
func parseChatRequest(r *http.Request) (chatRequest, error) {
	contentType := r.Header.Get("Content-Type")
	var req chatRequest

	if strings.HasPrefix(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return req, fmt.Errorf("invalid JSON: %w", err)
		}
	} else {
		// Form-encoded
		if err := r.ParseForm(); err != nil {
			return req, fmt.Errorf("invalid form data: %w", err)
		}
		req.Message = r.FormValue("message")
	}

	return req, nil
}

// parseCommandRequest parses command request from either JSON or form data.
// Supports both JSON and form-encoded requests.
func parseCommandRequest(r *http.Request) (commandRequest, error) {
	contentType := r.Header.Get("Content-Type")
	var req commandRequest

	if strings.HasPrefix(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return req, fmt.Errorf("invalid JSON: %w", err)
		}
	} else {
		// Form-encoded
		if err := r.ParseForm(); err != nil {
			return req, fmt.Errorf("invalid form data: %w", err)
		}
		req.Command = r.FormValue("command")
		// Args can be passed as comma-separated or multiple values
		if args := r.Form["args"]; len(args) > 0 {
			req.Args = args
		} else if argsStr := r.FormValue("args"); argsStr != "" {
			req.Args = strings.Split(argsStr, ",")
		}
	}

	return req, nil
}

// routerResultToJSON converts a router Result to a JSON-serializable map.
func (s *Server) routerResultToJSON(result *commands.Result) map[string]any {
	response := map[string]any{
		"success": true,
		"message": result.Message,
	}

	if result.State != "" {
		response["state"] = result.State
	}
	if result.TaskID != "" {
		response["task_id"] = result.TaskID
	}

	// Add type-specific data
	switch result.Type {
	case commands.ResultStatus:
		if data, ok := result.Data.(commands.StatusData); ok {
			response["status"] = map[string]any{
				"taskId":         data.TaskID,
				"title":          data.Title,
				"state":          data.State,
				"ref":            data.Ref,
				"branch":         data.Branch,
				"specifications": data.SpecCount,
			}
		}

	case commands.ResultCost:
		if data, ok := result.Data.(commands.CostData); ok {
			response["cost"] = map[string]any{
				"totalTokens":   data.TotalTokens,
				"inputTokens":   data.InputTokens,
				"outputTokens":  data.OutputTokens,
				"cachedTokens":  data.CachedTokens,
				"cachedPercent": data.CachedPercent,
				"totalCostUSD":  data.TotalCostUSD,
			}
		}

	case commands.ResultBudget:
		if data, ok := result.Data.(commands.BudgetData); ok {
			response["budget"] = map[string]any{
				"type":       data.Type,
				"used":       data.Used,
				"max":        data.Max,
				"percentage": data.Percentage,
				"warned":     data.Warned,
			}
		}

	case commands.ResultList:
		response["data"] = result.Data

	case commands.ResultHelp:
		response["commands"] = result.Data

	case commands.ResultSpecifications:
		response["specifications"] = result.Data

	case commands.ResultMessage:
		// Message type - data already in message field, include extra data if present
		if result.Data != nil {
			response["data"] = result.Data
		}

	case commands.ResultQuestion:
		// Question pending for user
		if result.Data != nil {
			response["question"] = result.Data
		}

	case commands.ResultChat:
		// Chat response from agent
		if result.Data != nil {
			response["chat"] = result.Data
		}

	case commands.ResultError:
		// Error response - message contains error details
		response["success"] = false

	case commands.ResultExit:
		// Exit signal - handled separately before this switch
		// Include for exhaustive switch compliance
	}

	return response
}

// handleInteractiveChat processes a chat message from the Web UI.
// POST /api/v1/interactive/chat.
func (s *Server) handleInteractiveChat(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")

		return
	}

	// Parse request (supports both JSON and form-encoded)
	req, err := parseChatRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.Message == "" {
		s.writeError(w, http.StatusBadRequest, "message is required")

		return
	}

	// Get active agent
	activeAgent := s.config.Conductor.GetActiveAgent()
	if activeAgent == nil {
		s.writeError(w, http.StatusServiceUnavailable, "no agent available")

		return
	}

	// Create cancellable context for this operation
	opCtx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Register operation for cancellation
	sessionID := s.getSessionID(r)
	s.registerOperation(sessionID, cancel, "chat")
	defer s.unregisterOperation(sessionID)

	// Build prompt with context
	prompt := s.buildChatPrompt(req.Message)

	// Run agent with streaming callback
	// We'll collect messages for the response
	var messages []chatMessage
	messages = append(messages, chatMessage{
		Role:      "user",
		Content:   req.Message,
		Timestamp: time.Now(),
	})

	response, err := activeAgent.RunWithCallback(opCtx, prompt, func(event agent.Event) error {
		// Stream event via SSE to connected clients
		s.config.EventBus.PublishRaw(eventbus.Event{
			Type: events.TypeAgentMessage,
			Data: map[string]any{
				"content": event.Text,
				"type":    string(event.Type),
			},
		})

		return nil
	})
	if err != nil {
		// Handle cancellation gracefully
		if errors.Is(err, context.Canceled) {
			s.writeJSON(w, http.StatusOK, chatResponse{
				Success: true,
				Message: "Chat cancelled",
			})

			return
		}
		slog.Error("agent chat error", "error", err)
		s.writeJSON(w, http.StatusInternalServerError, chatResponse{
			Success: false,
			Message: err.Error(),
		})

		return
	}

	// Add agent response
	if response != nil && response.Summary != "" {
		messages = append(messages, chatMessage{
			Role:      "assistant",
			Content:   response.Summary,
			Timestamp: time.Now(),
		})
	}

	// Check if agent asked a question
	if response != nil && response.Question != nil {
		// Save pending question
		task := s.config.Conductor.GetActiveTask()
		if task != nil {
			pendingQuestion := &storage.PendingQuestion{
				Question: response.Question.Text,
			}
			for _, opt := range response.Question.Options {
				pendingQuestion.Options = append(pendingQuestion.Options, storage.QuestionOption{
					Label:       opt.Label,
					Description: opt.Description,
				})
			}
			if err := s.config.Conductor.GetWorkspace().SavePendingQuestion(task.ID, pendingQuestion); err != nil {
				slog.Error("save pending question", "error", err)
			}
		}
	}

	s.writeJSON(w, http.StatusOK, chatResponse{
		Success:  true,
		Messages: messages,
	})
}

// handleInteractiveAnswer responds to an agent question.
// POST /api/v1/interactive/answer.
func (s *Server) handleInteractiveAnswer(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")

		return
	}

	// Parse request
	var req struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.Response == "" {
		s.writeError(w, http.StatusBadRequest, "response is required")

		return
	}

	ctx := r.Context()
	cond := s.config.Conductor
	task := cond.GetActiveTask()

	if task == nil {
		s.writeError(w, http.StatusServiceUnavailable, "no active task")

		return
	}

	// Clear the pending question
	if err := cond.GetWorkspace().ClearPendingQuestion(task.ID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "clear pending question: "+err.Error())

		return
	}

	// Add an answer as a note
	if err := cond.GetWorkspace().AppendNote(task.ID, task.State, req.Response); err != nil {
		s.writeError(w, http.StatusInternalServerError, "save answer: "+err.Error())

		return
	}

	// Resume workflow based on state
	var err error
	state := workflow.State(task.State)

	switch state {
	case workflow.StatePlanning:
		err = cond.Plan(ctx)
	case workflow.StateImplementing:
		err = cond.Implement(ctx)
	case workflow.StateReviewing:
		err = cond.Review(ctx)
	case workflow.StateIdle, workflow.StateDone, workflow.StateFailed,
		workflow.StateWaiting, workflow.StatePaused, workflow.StateCheckpointing,
		workflow.StateReverting, workflow.StateRestoring:
		s.writeError(w, http.StatusBadRequest, "cannot resume from state: "+string(state))

		return
	}

	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "resume workflow: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, commandResponse{
		Success: true,
		Message: "Answer sent, resuming...",
	})
}

// handleInteractiveState returns the current state.
// GET /api/v1/interactive/state.
func (s *Server) handleInteractiveState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")

		return
	}

	resp := stateResponse{
		Success: true,
	}

	// Return empty state if no conductor
	if s.config.Conductor == nil {
		s.writeJSON(w, http.StatusOK, resp)

		return
	}

	cond := s.config.Conductor
	task := cond.GetActiveTask()

	if task != nil {
		resp.State = task.State
		resp.TaskID = task.ID
		if work := cond.GetTaskWork(); work != nil {
			resp.Title = work.Metadata.Title
		}
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// handleInteractiveStop pauses the current operation.
// POST /api/v1/interactive/stop.
func (s *Server) handleInteractiveStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")

		return
	}

	sessionID := s.getSessionID(r)
	if sessionID == "" {
		s.writeJSON(w, http.StatusOK, commandResponse{
			Success: true,
			Message: "No session to cancel",
		})

		return
	}

	opName, cancelled := s.cancelOperation(sessionID)
	if !cancelled {
		s.writeJSON(w, http.StatusOK, commandResponse{
			Success: true,
			Message: "No active operation to cancel",
		})

		return
	}

	s.writeJSON(w, http.StatusOK, commandResponse{
		Success: true,
		Message: fmt.Sprintf("Cancelled %s operation", opName),
	})
}

// handleInteractiveCommands returns available commands for discovery.
// GET /api/v1/interactive/commands.
// This endpoint allows IDE plugins to discover available commands dynamically
// rather than hardcoding command lists.
func (s *Server) handleInteractiveCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")

		return
	}

	// Get all registered commands from the unified router
	cmds := commands.Metadata()

	// Include Web-specific commands that aren't in the router
	webCommands := []commands.CommandInfo{
		{
			Name:         "reset",
			Description:  "Reset workflow state to idle",
			Category:     "control",
			RequiresTask: false,
		},
		{
			Name:         "auto",
			Description:  "Auto-execute the next workflow step",
			Category:     "workflow",
			RequiresTask: true,
		},
		{
			Name:         "find",
			Aliases:      []string{"search"},
			Description:  "Search codebase for patterns",
			Category:     "exploration",
			Args:         []commands.CommandArg{{Name: "query", Required: true, Description: "Search query"}},
			RequiresTask: false,
		},
		{
			Name:         "memory",
			Aliases:      []string{"mem"},
			Description:  "Search and manage semantic memory",
			Category:     "exploration",
			Args:         []commands.CommandArg{{Name: "query", Required: false, Description: "Memory search query"}},
			RequiresTask: false,
		},
		{
			Name:         "library",
			Aliases:      []string{"lib"},
			Description:  "Search project library",
			Category:     "exploration",
			Args:         []commands.CommandArg{{Name: "query", Required: false, Description: "Library search query"}},
			RequiresTask: false,
		},
		{
			Name:         "question",
			Description:  "Ask agent a question and wait for response",
			Category:     "interaction",
			Args:         []commands.CommandArg{{Name: "question", Required: true, Description: "Question to ask"}},
			RequiresTask: true,
		},
		{
			Name:         "delete",
			Aliases:      []string{"del", "rm"},
			Description:  "Delete the current task",
			Category:     "task",
			RequiresTask: true,
		},
		{
			Name:         "export",
			Description:  "Export task to file",
			Category:     "task",
			RequiresTask: true,
		},
		{
			Name:         "submit",
			Description:  "Submit task for project",
			Category:     "workflow",
			RequiresTask: true,
		},
		{
			Name:         "sync",
			Description:  "Sync task with external source",
			Category:     "workflow",
			RequiresTask: true,
		},
	}

	// Combine router commands with Web-specific commands
	allCommands := append(cmds, webCommands...)

	s.writeJSON(w, http.StatusOK, map[string]any{
		"commands": allCommands,
	})
}

// buildChatPrompt builds a prompt for chat with context.
func (s *Server) buildChatPrompt(message string) string {
	var builder strings.Builder

	builder.WriteString("You are an AI assistant helping with a software development task.\n\n")

	// Add current task context
	task := s.config.Conductor.GetActiveTask()
	if task != nil {
		if work := s.config.Conductor.GetTaskWork(); work != nil {
			builder.WriteString(fmt.Sprintf("Task: %s\n", work.Metadata.Title))
			builder.WriteString(fmt.Sprintf("Current State: %s\n\n", task.State))
		}
	}

	builder.WriteString("User message: ")
	builder.WriteString(message)

	return builder.String()
}
