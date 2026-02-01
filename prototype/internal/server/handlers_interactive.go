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

// parseChatRequest parses chat request from either JSON or form data.
// HTMX sends form-encoded by default, but JSON is also supported.
func parseChatRequest(r *http.Request) (chatRequest, error) {
	contentType := r.Header.Get("Content-Type")
	var req chatRequest

	if strings.HasPrefix(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return req, fmt.Errorf("invalid JSON: %w", err)
		}
	} else {
		// Form-encoded (HTMX default)
		if err := r.ParseForm(); err != nil {
			return req, fmt.Errorf("invalid form data: %w", err)
		}
		req.Message = r.FormValue("message")
	}

	return req, nil
}

// parseCommandRequest parses command request from either JSON or form data.
// HTMX sends form-encoded by default, but JSON is also supported.
func parseCommandRequest(r *http.Request) (commandRequest, error) {
	contentType := r.Header.Get("Content-Type")
	var req commandRequest

	if strings.HasPrefix(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return req, fmt.Errorf("invalid JSON: %w", err)
		}
	} else {
		// Form-encoded (HTMX default)
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

	// Parse request (supports both JSON and form-encoded from HTMX)
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

// handleInteractiveSend processes input from the interactive UI and returns HTML.
// POST /ui/interactive/send.
// This handles both chat messages and workflow commands, returning HTML for HTMX.
func (s *Server) handleInteractiveSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeHTML(w, http.StatusMethodNotAllowed, `<div class="text-red-500 p-2">Method not allowed</div>`)

		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		s.writeHTML(w, http.StatusBadRequest, `<div class="text-red-500 p-2">Invalid request</div>`)

		return
	}

	message := strings.TrimSpace(r.FormValue("message"))
	if message == "" {
		return // Empty message, do nothing
	}

	// Check if it's a command (starts with known command word)
	parts := strings.Fields(message)
	command := strings.ToLower(parts[0])
	args := parts[1:]

	// List of known commands
	knownCommands := map[string]bool{
		"start": true, "plan": true, "implement": true, "review": true,
		"continue": true, "finish": true, "abandon": true,
		"undo": true, "redo": true, "status": true, "st": true,
		"cost": true, "budget": true, "list": true, "note": true,
		"quick": true, "find": true, "memory": true, "simplify": true,
		"label": true, "specification": true, "spec": true,
		"chat": true, "answer": true, "help": true,
	}

	var html string

	if knownCommands[command] {
		// It's a command - execute it
		html = s.executeInteractiveCommand(r.Context(), command, args, message)
	} else {
		// Treat as chat message
		html = s.executeInteractiveChat(r.Context(), message)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

// executeInteractiveCommand runs a workflow command and returns HTML response.
func (s *Server) executeInteractiveCommand(ctx context.Context, command string, args []string, original string) string {
	// Show user's command
	html := fmt.Sprintf(`<div class="flex gap-3 p-3 bg-brand-50 dark:bg-brand-900/20 rounded-lg">
		<span class="text-brand-600 dark:text-brand-400 font-medium">You:</span>
		<span class="text-surface-700 dark:text-surface-300">%s</span>
	</div>`, escapeHTML(original))

	if s.config.Conductor == nil {
		return html + `<div class="p-3 bg-red-50 dark:bg-red-900/20 rounded-lg text-red-600 dark:text-red-400">
			Error: Conductor not initialized
		</div>`
	}

	cond := s.config.Conductor
	var result string
	var err error

	switch command {
	case "status", "st":
		status, statusErr := cond.Status(ctx)
		if statusErr != nil {
			err = statusErr
		} else {
			result = "State: " + status.State
			if status.TaskID != "" {
				result += "\nTask: " + status.TaskID
			}
			if status.Title != "" {
				result += "\nTitle: " + status.Title
			}
			if status.Branch != "" {
				result += "\nBranch: " + status.Branch
			}
			if status.Specifications > 0 {
				result += fmt.Sprintf("\nSpecifications: %d", status.Specifications)
			}
			if status.Checkpoints > 0 {
				result += fmt.Sprintf("\nCheckpoints: %d", status.Checkpoints)
			}
		}

	case "start":
		if len(args) == 0 {
			err = errors.New("start requires a reference (e.g., start github:123)")
		} else {
			err = cond.Start(ctx, args[0])
			if err == nil {
				result = "Task started"
			}
		}

	case "plan":
		err = cond.Plan(ctx)
		if err == nil {
			result = "Planning started"
		}

	case "implement":
		err = cond.Implement(ctx)
		if err == nil {
			result = "Implementation started"
		}

	case "review":
		err = cond.Review(ctx)
		if err == nil {
			result = "Review started"
		}

	case "continue":
		err = cond.ResumePaused(ctx)
		if err == nil {
			result = "Resumed"
		}

	case "finish":
		if cond.GetActiveTask() == nil {
			err = errors.New("no active task")
		} else {
			err = cond.Finish(ctx, conductor.FinishOptions{})
			if err == nil {
				result = "Task completed"
			}
		}

	case "abandon":
		if cond.GetActiveTask() == nil {
			err = errors.New("no active task")
		} else {
			err = cond.Delete(ctx, conductor.DeleteOptions{
				Force:      true,
				KeepBranch: false,
				DeleteWork: conductor.BoolPtr(true),
			})
			if err == nil {
				result = "Task abandoned"
			}
		}

	case "undo":
		err = cond.Undo(ctx)
		if err == nil {
			result = "Undo complete"
		}

	case "redo":
		err = cond.Redo(ctx)
		if err == nil {
			result = "Redo complete"
		}

	case "cost":
		task := cond.GetActiveTask()
		if task == nil {
			err = errors.New("no active task")
		} else {
			ws := cond.GetWorkspace()
			work, loadErr := ws.LoadWork(task.ID)
			if loadErr != nil {
				err = loadErr
			} else {
				costs := work.Costs
				result = fmt.Sprintf("Input: %d tokens\nOutput: %d tokens\nTotal: $%.4f",
					costs.TotalInputTokens, costs.TotalOutputTokens, costs.TotalCostUSD)
			}
		}

	case "list":
		ws := cond.GetWorkspace()
		taskIDs, listErr := ws.ListWorks()
		if listErr != nil {
			err = listErr
		} else if len(taskIDs) == 0 {
			result = "No tasks found"
		} else {
			var lines []string
			for _, id := range taskIDs {
				work, loadErr := ws.LoadWork(id)
				if loadErr != nil {
					continue
				}
				shortID := id
				if len(id) > 8 {
					shortID = id[:8]
				}
				title := work.Metadata.Title
				if title == "" {
					title = work.Source.Ref
				}
				lines = append(lines, fmt.Sprintf("• %s: %s (%s)", shortID, title, work.Metadata.State))
			}
			result = strings.Join(lines, "\n")
		}

	case "help":
		result = `Commands:
• start <ref> - Start a task
• plan - Run planning
• implement - Run implementation
• review - Run code review
• continue - Resume paused
• finish - Complete task
• abandon - Discard task
• undo/redo - Checkpoints
• status - Show status
• cost - Show token usage
• list - List tasks
• help - Show this help`

	default:
		result = "Unknown command: " + command
	}

	if err != nil {
		html += fmt.Sprintf(`<div class="p-3 bg-red-50 dark:bg-red-900/20 rounded-lg text-red-600 dark:text-red-400">
			Error: %s
		</div>`, escapeHTML(err.Error()))
	} else if result != "" {
		html += fmt.Sprintf(`<div class="flex gap-3 p-3 bg-surface-50 dark:bg-surface-800 rounded-lg">
			<span class="text-green-600 dark:text-green-400 font-medium">System:</span>
			<pre class="text-surface-700 dark:text-surface-300 whitespace-pre-wrap">%s</pre>
		</div>`, escapeHTML(result))
	}

	return html
}

// executeInteractiveChat runs a chat message and returns HTML response.
func (s *Server) executeInteractiveChat(ctx context.Context, message string) string {
	// Show user's message
	html := fmt.Sprintf(`<div class="flex gap-3 p-3 bg-brand-50 dark:bg-brand-900/20 rounded-lg">
		<span class="text-brand-600 dark:text-brand-400 font-medium">You:</span>
		<span class="text-surface-700 dark:text-surface-300">%s</span>
	</div>`, escapeHTML(message))

	if s.config.Conductor == nil {
		return html + `<div class="p-3 bg-red-50 dark:bg-red-900/20 rounded-lg text-red-600 dark:text-red-400">
			Error: Conductor not initialized
		</div>`
	}

	activeAgent := s.config.Conductor.GetActiveAgent()
	if activeAgent == nil {
		return html + `<div class="p-3 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg text-yellow-600 dark:text-yellow-400">
			No active agent. Start a task first or check agent configuration.
		</div>`
	}

	// Build prompt and run agent
	prompt := s.buildChatPrompt(message)
	response, err := activeAgent.Run(ctx, prompt)

	if err != nil {
		if errors.Is(err, context.Canceled) {
			html += `<div class="p-3 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg text-yellow-600 dark:text-yellow-400">
				Chat cancelled
			</div>`
		} else {
			html += fmt.Sprintf(`<div class="p-3 bg-red-50 dark:bg-red-900/20 rounded-lg text-red-600 dark:text-red-400">
				Error: %s
			</div>`, escapeHTML(err.Error()))
		}
	} else if response != nil && response.Summary != "" {
		html += fmt.Sprintf(`<div class="flex gap-3 p-3 bg-surface-50 dark:bg-surface-800 rounded-lg">
			<span class="text-purple-600 dark:text-purple-400 font-medium">Agent:</span>
			<div class="text-surface-700 dark:text-surface-300 prose dark:prose-invert prose-sm max-w-none">%s</div>
		</div>`, escapeHTML(response.Summary))
	}

	return html
}

// escapeHTML escapes special HTML characters.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")

	return s
}

// writeHTML writes an HTML response.
func (s *Server) writeHTML(w http.ResponseWriter, status int, html string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(html))
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

	// Parse request (supports both JSON and form-encoded from HTMX)
	req, err := parseCommandRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	slog.Debug("interactive command received", "command", req.Command, "args", req.Args)

	// Create cancellable context for ALL commands
	opCtx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Register operation for cancellation
	sessionID := s.getSessionID(r)
	s.registerOperation(sessionID, cancel, req.Command)
	defer s.unregisterOperation(sessionID)

	cond := s.config.Conductor
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

	case "status", "st":
		status, err := cond.Status(opCtx)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to get status: "+err.Error())

			return
		}

		// Build human-readable status message
		var statusMsg strings.Builder
		statusMsg.WriteString("State: " + status.State)
		if status.TaskID != "" {
			statusMsg.WriteString("\nTask: " + status.TaskID[:min(7, len(status.TaskID))])
		}
		if status.Title != "" {
			statusMsg.WriteString("\nTitle: " + status.Title)
		}
		if status.Branch != "" {
			statusMsg.WriteString("\nBranch: " + status.Branch)
		}
		if status.Ref != "" {
			statusMsg.WriteString("\nRef: " + status.Ref)
		}
		if status.Specifications > 0 {
			statusMsg.WriteString(fmt.Sprintf("\nSpecifications: %d", status.Specifications))
		}
		if status.Checkpoints > 0 {
			statusMsg.WriteString(fmt.Sprintf("\nCheckpoints: %d", status.Checkpoints))
		}

		s.writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"status": map[string]any{
				"taskId":         status.TaskID,
				"title":          status.Title,
				"externalKey":    status.ExternalKey,
				"state":          status.State,
				"ref":            status.Ref,
				"branch":         status.Branch,
				"worktreePath":   status.WorktreePath,
				"specifications": status.Specifications,
				"checkpoints":    status.Checkpoints,
				"started":        status.Started,
				"agent":          status.Agent,
				"agentSource":    status.AgentSource,
			},
			"message": statusMsg.String(),
			"state":   status.State,
		})

		return

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
