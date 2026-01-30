package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/memory"
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

	// Parse request
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
			Data: map[string]any{"event": event},
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

// handleInteractiveCommand executes a workflow command.
// POST /api/v1/interactive/command.
func (s *Server) handleInteractiveCommand(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")

		return
	}

	// Parse request
	var req commandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	// Create cancellable context for ALL commands
	opCtx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Register operation for cancellation
	sessionID := s.getSessionID(r)
	s.registerOperation(sessionID, cancel, req.Command)
	defer s.unregisterOperation(sessionID)

	cond := s.config.Conductor
	var err error
	var message string

	// Execute command
	switch req.Command {
	case "start":
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "start requires a reference")

			return
		}
		err = cond.Start(opCtx, req.Args[0])
		message = "Task started"

	case "plan":
		err = cond.Plan(opCtx)
		message = "Planning started"

	case "implement":
		err = cond.Implement(opCtx)
		message = "Implementation started"

	case "review":
		err = cond.Review(opCtx)
		message = "Review started"

	case "continue":
		err = cond.ResumePaused(opCtx)
		message = "Resumed"

	case "undo":
		err = cond.Undo(opCtx)
		message = "Undo complete"

	case "redo":
		err = cond.Redo(opCtx)
		message = "Redo complete"

	case "finish":
		if cond.GetActiveTask() == nil {
			s.writeError(w, http.StatusBadRequest, "no active task")

			return
		}
		opts := conductor.FinishOptions{}
		err = cond.Finish(opCtx, opts)
		message = "Task completed"

	case "abandon":
		if cond.GetActiveTask() == nil {
			s.writeError(w, http.StatusBadRequest, "no active task")

			return
		}
		opts := conductor.DeleteOptions{
			Force:      true,
			KeepBranch: false,
			DeleteWork: conductor.BoolPtr(true),
		}
		err = cond.Delete(opCtx, opts)
		message = "Task abandoned"

	case "note":
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "note requires a message")

			return
		}
		if cond.GetActiveTask() == nil {
			s.writeError(w, http.StatusBadRequest, "no active task")

			return
		}
		task := cond.GetActiveTask()
		ws := cond.GetWorkspace()
		noteMsg := strings.Join(req.Args, " ")
		if err := ws.AppendNote(task.ID, noteMsg, task.State); err != nil {
			s.writeError(w, http.StatusInternalServerError, "save note: "+err.Error())

			return
		}
		message = "Note saved"

	case "quick":
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "quick requires a description")

			return
		}
		result, err := cond.CreateQuickTask(opCtx, conductor.QuickTaskOptions{
			Description: strings.Join(req.Args, " "),
			QueueID:     "quick-tasks",
		})
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "create quick task: "+err.Error())

			return
		}
		message = "Quick task created: " + result.TaskID

	case "cost":
		if cond.GetActiveTask() == nil {
			s.writeError(w, http.StatusBadRequest, "no active task")

			return
		}
		work := cond.GetTaskWork()
		if work == nil {
			s.writeError(w, http.StatusInternalServerError, "unable to load task work")

			return
		}
		costs := work.Costs
		message = fmt.Sprintf("Input: %d, Output: %d, Total: $%.4f",
			costs.TotalInputTokens, costs.TotalOutputTokens, costs.TotalCostUSD)

	case "list":
		ws := cond.GetWorkspace()
		taskIDs, err := ws.ListWorks()
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "list tasks: "+err.Error())

			return
		}
		message = fmt.Sprintf("Found %d tasks", len(taskIDs))

	case "specification":
		if cond.GetActiveTask() == nil {
			s.writeError(w, http.StatusBadRequest, "no active task")

			return
		}
		ws := cond.GetWorkspace()
		task := cond.GetActiveTask()
		if len(req.Args) == 0 {
			specs, err := ws.ListSpecificationsWithStatus(task.ID)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, "list specifications: "+err.Error())

				return
			}
			message = fmt.Sprintf("Found %d specifications", len(specs))
		} else {
			num, err := strconv.Atoi(req.Args[0])
			if err != nil {
				s.writeError(w, http.StatusBadRequest, "specification number must be an integer")

				return
			}
			_, err = ws.LoadSpecification(task.ID, num)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, "load specification: "+err.Error())

				return
			}
			message = fmt.Sprintf("Specification %d loaded", num)
		}

	case "find":
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "find requires a query")

			return
		}
		findOpts := conductor.FindOptions{
			Query:     strings.Join(req.Args, " "),
			Path:      "",
			Pattern:   "",
			Context:   3,
			Workspace: cond.GetWorkspace(),
		}
		resultChan, err := cond.Find(r.Context(), findOpts)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "find: "+err.Error())

			return
		}
		var results []conductor.FindResult
		for result := range resultChan {
			if result.File != "__error__" {
				results = append(results, result)
			}
		}
		message = fmt.Sprintf("Found %d match(es)", len(results))

	case "simplify":
		if cond.GetActiveTask() == nil {
			s.writeError(w, http.StatusBadRequest, "no active task")

			return
		}
		err = cond.Simplify(opCtx, "", true)
		message = "Simplification complete"

	case "label":
		task := cond.GetActiveTask()
		if task == nil {
			s.writeError(w, http.StatusBadRequest, "no active task")

			return
		}
		ws := cond.GetWorkspace()
		if len(req.Args) == 0 {
			labels, _ := ws.GetLabels(task.ID)
			message = fmt.Sprintf("Labels: %v", labels)
		} else {
			subCmd := req.Args[0]
			subArgs := req.Args[1:]
			switch subCmd {
			case "add":
				for _, label := range subArgs {
					_ = ws.AddLabel(task.ID, label)
				}
				message = fmt.Sprintf("Added %d label(s)", len(subArgs))
			case "remove", "rm":
				for _, label := range subArgs {
					_ = ws.RemoveLabel(task.ID, label)
				}
				message = fmt.Sprintf("Removed %d label(s)", len(subArgs))
			case "clear":
				_ = ws.SetLabels(task.ID, []string{})
				message = "Labels cleared"
			case "list", "ls":
				labels, _ := ws.GetLabels(task.ID)
				message = fmt.Sprintf("Labels: %v", labels)
			default:
				for _, label := range req.Args {
					_ = ws.AddLabel(task.ID, label)
				}
				message = fmt.Sprintf("Added %d label(s)", len(req.Args))
			}
		}

	case "memory":
		mem := cond.GetMemory()
		if mem == nil {
			s.writeError(w, http.StatusServiceUnavailable, "memory system is not enabled")

			return
		}
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "memory requires a query")

			return
		}
		query := strings.Join(req.Args, " ")
		results, err := mem.Search(r.Context(), query, memory.SearchOptions{
			Limit:    5,
			MinScore: 0.65,
		})
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "memory search: "+err.Error())

			return
		}
		message = fmt.Sprintf("Found %d similar task(s)", len(results))

	case "budget":
		task := cond.GetActiveTask()
		if task == nil {
			s.writeError(w, http.StatusBadRequest, "no active task")

			return
		}
		ws := cond.GetWorkspace()
		work, err := ws.LoadWork(task.ID)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "load task: "+err.Error())

			return
		}
		cfg, err := ws.LoadConfig()
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "load config: "+err.Error())

			return
		}
		taskBudget := cfg.Budget.PerTask
		if work.Budget != nil {
			taskBudget = *work.Budget
		}
		costs := work.Costs
		totalTokens := costs.TotalInputTokens + costs.TotalOutputTokens
		message = fmt.Sprintf("Tokens: %d, Cost: $%.4f / $%.2f",
			totalTokens, costs.TotalCostUSD, taskBudget.MaxCost)

	default:
		s.writeError(w, http.StatusBadRequest, "unknown command: "+req.Command)

		return
	}

	// Handle cancellation gracefully
	if errors.Is(err, context.Canceled) {
		state := ""
		if task := cond.GetActiveTask(); task != nil {
			state = task.State
		}
		s.writeJSON(w, http.StatusOK, commandResponse{
			Success: true,
			Message: req.Command + " cancelled",
			State:   state,
		})

		return
	}

	if err != nil {
		slog.Error("command error", "command", req.Command, "error", err)
		s.writeJSON(w, http.StatusInternalServerError, commandResponse{
			Success: false,
			Message: err.Error(),
		})

		return
	}

	// Get current state
	state := ""
	taskID := ""
	title := ""
	if task := cond.GetActiveTask(); task != nil {
		state = task.State
		taskID = task.ID
		if work := cond.GetTaskWork(); work != nil {
			title = work.Metadata.Title
		}
	}

	s.writeJSON(w, http.StatusOK, commandResponse{
		Success: true,
		Message: message,
		State:   state,
	})

	// Also publish state update
	if taskID != "" {
		s.config.EventBus.PublishRaw(eventbus.Event{
			Type: events.TypeStateChanged,
			Data: map[string]any{
				"task_id": taskID,
				"state":   state,
				"title":   title,
			},
		})
	}
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

	// Add answer as a note
	if err := cond.GetWorkspace().AppendNote(task.ID, string(workflow.State(task.State)), req.Response); err != nil {
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

// handleInteractivePage renders the interactive page.
// GET /interactive.
func (s *Server) handleInteractivePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")

		return
	}

	// Get current state
	var state, taskID, title string
	if s.config.Conductor != nil {
		if task := s.config.Conductor.GetActiveTask(); task != nil {
			state = task.State
			taskID = task.ID
			if work := s.config.Conductor.GetTaskWork(); work != nil {
				title = work.Metadata.Title
			}
		}
	}

	data := map[string]any{
		"State":  state,
		"TaskID": taskID,
		"Title":  title,
	}

	if err := s.renderer.Render(w, "interactive", data); err != nil {
		slog.Error("render interactive page", "error", err)
	}
}
