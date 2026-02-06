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
		"browser": true, "project": true, "stack": true,
		"label": true, "specification": true, "spec": true,
		"chat": true, "answer": true, "help": true, "library": true,
		"delete": true, "export": true, "optimize": true, "submit": true, "sync": true,
		"links": true,
		// Configuration commands
		"config": true, "agents": true, "providers": true, "templates": true,
		"scan": true, "commit": true,
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

	var result string
	var err error

	switch command {
	// Workflow commands
	case "status", "st", "start", "plan", "implement", "review",
		"continue", "finish", "abandon", "undo", "redo", "reset", "auto":
		result, err = s.executeInteractiveWorkflowCommand(ctx, command, args)

	// Task management commands
	case "cost", "budget", "list", "note", "quick", "specification", "spec",
		"simplify", "label", "delete", "export", "optimize", "submit", "sync",
		"answer", "question":
		result, err = s.executeInteractiveTaskCommand(ctx, command, args)

	// Exploration commands
	case "find", "memory", "library", "links":
		result, err = s.executeInteractiveExploreCommand(ctx, command, args)

	// Tools and meta commands
	case "browser", "project", "stack", "config", "agents", "providers",
		"templates", "scan", "commit", "help":
		result, err = s.executeInteractiveToolsCommand(ctx, command, args)

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

// getProviderInfoText returns provider information text for the interactive handler.
func getProviderInfoText(name string) string {
	providers := map[string]string{
		"file": `Provider: File
Scheme: file (shorthand: f)
Description: Single markdown file containing task description.
Usage: mehr start file:path/to/task.md`,

		"dir": `Provider: Directory
Scheme: dir (shorthand: d)
Description: Directory with README.md as task description.
Usage: mehr start dir:./tasks/feature-x/`,

		"github": `Provider: GitHub
Scheme: github (shorthand: gh)
Description: GitHub issues and pull requests.
Setup:
  1. Set GITHUB_TOKEN environment variable
  2. Or use 'gh auth login' for GitHub CLI auth
Usage: mehr start github:owner/repo#123`,

		"gitlab": `Provider: GitLab
Scheme: gitlab
Description: GitLab issues and merge requests.
Setup:
  1. Set GITLAB_TOKEN environment variable
  2. Configure gitlab.url in .mehrhof/config.yaml for self-hosted
Usage: mehr start gitlab:group/project#123`,

		"jira": `Provider: Jira
Scheme: jira
Description: Atlassian Jira tickets.
Setup:
  1. Set JIRA_URL, JIRA_EMAIL, JIRA_TOKEN environment variables
  2. Or configure in .mehrhof/config.yaml
Usage: mehr start jira:PROJECT-123`,

		"linear": `Provider: Linear
Scheme: linear
Description: Linear issues.
Setup:
  1. Set LINEAR_API_KEY environment variable
Usage: mehr start linear:ISSUE-123`,

		"notion": `Provider: Notion
Scheme: notion
Description: Notion pages and databases.
Setup:
  1. Set NOTION_TOKEN environment variable
  2. Share pages with your integration
Usage: mehr start notion:page-id`,

		"wrike": `Provider: Wrike
Scheme: wrike
Description: Wrike tasks.
Setup:
  1. Set WRIKE_TOKEN environment variable
Usage: mehr start wrike:task-id`,

		"youtrack": `Provider: YouTrack
Scheme: youtrack (shorthand: yt)
Description: JetBrains YouTrack issues.
Setup:
  1. Set YOUTRACK_URL and YOUTRACK_TOKEN environment variables
Usage: mehr start youtrack:PROJECT-123`,
	}

	// Check aliases
	aliases := map[string]string{
		"f":  "file",
		"d":  "dir",
		"gh": "github",
		"yt": "youtrack",
	}

	if alias, ok := aliases[name]; ok {
		name = alias
	}

	return providers[name]
}

// writeHTML writes an HTML response.
func (s *Server) writeHTML(w http.ResponseWriter, status int, html string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(html))
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

// truncateURL truncates a URL for display, keeping the beginning visible.
func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}

	return url[:maxLen-3] + "..."
}

// truncateStr truncates a string for display.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
