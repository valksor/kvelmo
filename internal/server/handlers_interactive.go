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
	"github.com/valksor/go-mehrhof/internal/browser"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/conductor/commands"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/library"
	"github.com/valksor/go-mehrhof/internal/memory"
	"github.com/valksor/go-mehrhof/internal/stack"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/template"
	"github.com/valksor/go-mehrhof/internal/validation"
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
		// Handle "implement review <n>" subcommand
		if len(args) > 0 && args[0] == "review" {
			if len(args) < 2 {
				err = errors.New("usage: implement review <number>")
			} else if num, parseErr := strconv.Atoi(args[1]); parseErr != nil {
				err = errors.New("review number must be an integer")
			} else if num <= 0 {
				err = fmt.Errorf("review number must be positive, got %d", num)
			} else {
				// Pre-validate review availability before changing state
				task := cond.GetActiveTask()
				if task == nil {
					err = errors.New("no active task")
				} else {
					ws := cond.GetWorkspace()
					reviews, listErr := ws.ListReviews(task.ID)
					if listErr != nil {
						err = fmt.Errorf("list reviews: %w", listErr)
					} else if len(reviews) == 0 {
						err = errors.New("no reviews found - run 'review' first to generate code review")
					} else {
						// Check if the requested review exists
						reviewExists := false
						for _, r := range reviews {
							if r == num {
								reviewExists = true

								break
							}
						}
						if !reviewExists {
							if len(reviews) == 1 {
								err = fmt.Errorf("review %d not found - only review %d exists", num, reviews[0])
							} else {
								err = fmt.Errorf("review %d not found - available reviews: %v", num, reviews)
							}
						} else if implErr := cond.ImplementReview(ctx, num); implErr != nil {
							err = implErr
						} else if runErr := cond.RunReviewImplementation(ctx, num); runErr != nil {
							err = runErr
						} else {
							result = fmt.Sprintf("Review %d fixes applied", num)
						}
					}
				}
			}
		} else {
			err = cond.Implement(ctx)
			if err == nil {
				result = "Implementation started"
			}
		}

	case "review":
		// Handle "review <n>" for viewing reviews, "review" alone runs review workflow
		if len(args) > 0 {
			// If first arg is a number, view that review
			if num, parseErr := strconv.Atoi(args[0]); parseErr == nil {
				task := cond.GetActiveTask()
				if task == nil {
					err = errors.New("no active task")
				} else {
					ws := cond.GetWorkspace()
					review, loadErr := ws.LoadReview(task.ID, num)
					if loadErr != nil {
						err = loadErr
					} else {
						preview := review
						if len(preview) > 500 {
							preview = preview[:500] + "..."
						}
						result = fmt.Sprintf("Review %d:\n%s", num, preview)
					}
				}
			} else if args[0] == "view" && len(args) > 1 {
				// Handle "review view <n>"
				if num, parseErr := strconv.Atoi(args[1]); parseErr == nil {
					task := cond.GetActiveTask()
					if task == nil {
						err = errors.New("no active task")
					} else {
						ws := cond.GetWorkspace()
						review, loadErr := ws.LoadReview(task.ID, num)
						if loadErr != nil {
							err = loadErr
						} else {
							preview := review
							if len(preview) > 500 {
								preview = preview[:500] + "..."
							}
							result = fmt.Sprintf("Review %d:\n%s", num, preview)
						}
					}
				} else {
					err = errors.New("review number must be an integer")
				}
			} else {
				err = errors.New("usage: review <number> or review view <number>")
			}
		} else {
			// No args - run review workflow
			err = cond.Review(ctx)
			if err == nil {
				result = "Review started"
			}
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

	case "reset":
		err = cond.ResetState(ctx)
		if err == nil {
			result = "Workflow reset to idle"
		}

	case "auto":
		// Auto-execute the next workflow step based on current state
		task := cond.GetActiveTask()
		if task == nil {
			err = errors.New("no active task")
		} else {
			switch task.State {
			case "idle":
				err = errors.New("no active task, use 'start' first")
			case "planning":
				err = cond.Plan(ctx)
				if err == nil {
					result = "Planning started"
				}
			case "implementing":
				err = cond.Implement(ctx)
				if err == nil {
					result = "Implementation started"
				}
			case "reviewing":
				err = cond.Review(ctx)
				if err == nil {
					result = "Review started"
				}
			case "waiting":
				err = errors.New("task is waiting for user input")
			case "done", "failed":
				err = errors.New("task is already completed")
			default:
				err = fmt.Errorf("cannot auto-execute in state: %s", task.State)
			}
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

	case "note":
		if len(args) == 0 {
			err = errors.New("note requires a message")
		} else {
			task := cond.GetActiveTask()
			if task == nil {
				err = errors.New("no active task")
			} else {
				ws := cond.GetWorkspace()
				noteMsg := strings.Join(args, " ")
				if noteErr := ws.AppendNote(task.ID, noteMsg, task.State); noteErr != nil {
					err = noteErr
				} else {
					result = "Note saved"
				}
			}
		}

	case "quick":
		if len(args) == 0 {
			err = errors.New("quick requires a description")
		} else {
			quickResult, quickErr := cond.CreateQuickTask(ctx, conductor.QuickTaskOptions{
				Description: strings.Join(args, " "),
				QueueID:     "quick-tasks",
			})
			if quickErr != nil {
				err = quickErr
			} else {
				result = "Quick task created: " + quickResult.TaskID
			}
		}

	case "specification", "spec":
		task := cond.GetActiveTask()
		if task == nil {
			err = errors.New("no active task")
		} else {
			ws := cond.GetWorkspace()
			if len(args) == 0 {
				specs, specErr := ws.ListSpecificationsWithStatus(task.ID)
				if specErr != nil {
					err = specErr
				} else {
					result = fmt.Sprintf("Found %d specifications", len(specs))
				}
			} else {
				num, parseErr := strconv.Atoi(args[0])
				if parseErr != nil {
					err = errors.New("specification number must be an integer")
				} else {
					spec, loadErr := ws.LoadSpecification(task.ID, num)
					if loadErr != nil {
						err = loadErr
					} else {
						// Spec is raw markdown content, show first 500 chars
						preview := spec
						if len(preview) > 500 {
							preview = preview[:500] + "..."
						}
						result = fmt.Sprintf("Specification %d:\n%s", num, preview)
					}
				}
			}
		}

	case "find":
		if len(args) == 0 {
			err = errors.New("find requires a query")
		} else {
			findOpts := conductor.FindOptions{
				Query:     strings.Join(args, " "),
				Path:      "",
				Pattern:   "",
				Context:   3,
				Workspace: cond.GetWorkspace(),
			}
			resultChan, findErr := cond.Find(ctx, findOpts)
			if findErr != nil {
				err = findErr
			} else {
				var results []conductor.FindResult
				for findResult := range resultChan {
					if findResult.File != "__error__" {
						results = append(results, findResult)
					}
				}
				if len(results) == 0 {
					result = "No matches found"
				} else {
					var lines []string
					for _, r := range results {
						lines = append(lines, fmt.Sprintf("• %s:%d - %s", r.File, r.Line, r.Reason))
					}
					result = fmt.Sprintf("Found %d match(es):\n%s", len(results), strings.Join(lines, "\n"))
				}
			}
		}

	case "memory":
		mem := cond.GetMemory()
		if mem == nil {
			err = errors.New("memory system is not enabled")
		} else if len(args) == 0 {
			err = errors.New("memory requires a subcommand: search <query>, index <task-id>, stats")
		} else {
			subcommand := args[0]
			subArgs := args[1:]
			switch subcommand {
			case "search":
				if len(subArgs) == 0 {
					err = errors.New("memory search requires a query")
				} else {
					query := strings.Join(subArgs, " ")
					memResults, memErr := mem.Search(ctx, query, memory.SearchOptions{
						Limit:    5,
						MinScore: 0.65,
					})
					if memErr != nil {
						err = memErr
					} else if len(memResults) == 0 {
						result = "No similar tasks found"
					} else {
						var lines []string
						for _, r := range memResults {
							taskID := ""
							if r.Document != nil {
								taskID = r.Document.TaskID
							}
							lines = append(lines, fmt.Sprintf("• %s (%.0f%% similar)", taskID, r.Score*100))
						}
						result = fmt.Sprintf("Found %d similar task(s):\n%s", len(memResults), strings.Join(lines, "\n"))
					}
				}
			case "index":
				if len(subArgs) == 0 {
					err = errors.New("memory index requires a task ID")
				} else {
					ws := cond.GetWorkspace()
					if ws == nil {
						err = errors.New("workspace not initialized")
					} else {
						taskID := subArgs[0]
						// Verify task exists
						if _, loadErr := ws.LoadWork(taskID); loadErr != nil {
							err = fmt.Errorf("task not found: %w", loadErr)
						} else {
							indexer := memory.NewIndexer(mem, ws, nil)
							if indexErr := indexer.IndexTask(ctx, taskID); indexErr != nil {
								err = fmt.Errorf("failed to index task: %w", indexErr)
							} else {
								result = fmt.Sprintf("Task %s indexed successfully", taskID)
							}
						}
					}
				}
			case "stats":
				ws := cond.GetWorkspace()
				if ws == nil {
					err = errors.New("workspace not initialized")
				} else {
					indexer := memory.NewIndexer(mem, ws, nil)
					stats, statsErr := indexer.GetStats(ctx)
					if statsErr != nil {
						err = fmt.Errorf("failed to get stats: %w", statsErr)
					} else {
						var lines []string
						lines = append(lines, fmt.Sprintf("Total documents: %d", stats.TotalDocuments))
						if len(stats.ByType) > 0 {
							lines = append(lines, "By type:")
							for docType, count := range stats.ByType {
								lines = append(lines, fmt.Sprintf("  • %s: %d", docType, count))
							}
						}
						result = strings.Join(lines, "\n")
					}
				}
			default:
				// Backwards compatibility: treat unknown subcommand as search query
				query := strings.Join(args, " ")
				memResults, memErr := mem.Search(ctx, query, memory.SearchOptions{
					Limit:    5,
					MinScore: 0.65,
				})
				if memErr != nil {
					err = memErr
				} else if len(memResults) == 0 {
					result = "No similar tasks found"
				} else {
					var lines []string
					for _, r := range memResults {
						taskID := ""
						if r.Document != nil {
							taskID = r.Document.TaskID
						}
						lines = append(lines, fmt.Sprintf("• %s (%.0f%% similar)", taskID, r.Score*100))
					}
					result = fmt.Sprintf("Found %d similar task(s):\n%s", len(memResults), strings.Join(lines, "\n"))
				}
			}
		}

	case "library":
		lib := cond.GetLibrary()
		if lib == nil {
			// Check if there was an initialization error
			if initErr := cond.GetLibraryError(); initErr != nil {
				err = initErr
			} else {
				err = errors.New("library system is not enabled. Use the Library panel or enable in .mehrhof/config.yaml under 'library:'")
			}
		} else {
			// Default to list if no subcommand
			subcommand := "list"
			if len(args) > 0 {
				subcommand = args[0]
				args = args[1:]
			}
			switch subcommand {
			case "list", "ls":
				collections, listErr := lib.List(ctx, &library.ListOptions{})
				if listErr != nil {
					err = listErr
				} else if len(collections) == 0 {
					result = "No library collections. Use the Library panel or run 'mehr library pull <source>' to add documentation."
				} else {
					var lines []string
					for _, c := range collections {
						lines = append(lines, fmt.Sprintf("• %s [%s, %s] - %d pages",
							c.Name, c.IncludeMode, c.Location, c.PageCount))
					}
					result = fmt.Sprintf("%d Collection(s):\n%s", len(collections), strings.Join(lines, "\n"))
				}
			case "show":
				if len(args) == 0 {
					err = errors.New("usage: library show <name>")
				} else {
					coll, showErr := lib.Show(ctx, args[0])
					if showErr != nil {
						err = showErr
					} else {
						result = fmt.Sprintf("Collection: %s\nSource: %s\nType: %s\nMode: %s\nPages: %d",
							coll.Name, coll.Source, coll.SourceType, coll.IncludeMode, coll.PageCount)
					}
				}
			case "search":
				if len(args) == 0 {
					err = errors.New("usage: library search <query>")
				} else {
					query := strings.Join(args, " ")
					docCtx, searchErr := lib.GetDocsForQuery(ctx, query, 10000)
					if searchErr != nil {
						err = searchErr
					} else if docCtx == nil || len(docCtx.Pages) == 0 {
						result = "No matching documentation found"
					} else {
						// Extract unique collection names from pages
						collectionSet := make(map[string]bool)
						for _, p := range docCtx.Pages {
							collectionSet[p.CollectionName] = true
						}
						var collNames []string
						for name := range collectionSet {
							collNames = append(collNames, name)
						}
						result = fmt.Sprintf("Found %d page(s) from %d collection(s): %s",
							len(docCtx.Pages), len(collNames), strings.Join(collNames, ", "))
					}
				}
			case "pull":
				if len(args) == 0 {
					err = errors.New("usage: library pull <source> [--name <name>] [--shared]")
				} else {
					source := args[0]
					opts := &library.PullOptions{}
					// Parse simple flags
					for i := 1; i < len(args); i++ {
						if args[i] == "--name" && i+1 < len(args) {
							opts.Name = args[i+1]
							i++
						} else if args[i] == "--shared" {
							opts.Shared = true
						}
					}
					pullResult, pullErr := lib.Pull(ctx, source, opts)
					if pullErr != nil {
						err = pullErr
					} else {
						result = fmt.Sprintf("Pulled collection: %s (%d pages)", pullResult.Collection.Name, pullResult.Collection.PageCount)
					}
				}
			case "remove", "rm":
				if len(args) == 0 {
					err = errors.New("usage: library remove <name>")
				} else {
					if removeErr := lib.Remove(ctx, args[0], false); removeErr != nil {
						err = removeErr
					} else {
						result = fmt.Sprintf("Collection '%s' removed", args[0])
					}
				}
			case "stats":
				collections, listErr := lib.List(ctx, &library.ListOptions{})
				if listErr != nil {
					err = listErr
				} else {
					var totalPages int
					var sharedCount, projectCount int
					for _, c := range collections {
						totalPages += c.PageCount
						if c.Location == "shared" {
							sharedCount++
						} else {
							projectCount++
						}
					}
					result = fmt.Sprintf("Library Stats:\n• Collections: %d (%d shared, %d project)\n• Total pages: %d",
						len(collections), sharedCount, projectCount, totalPages)
				}
			default:
				// Treat as collection name for show
				coll, showErr := lib.Show(ctx, subcommand)
				if showErr != nil {
					err = showErr
				} else {
					result = fmt.Sprintf("Collection: %s\nSource: %s\nType: %s\nMode: %s\nPages: %d",
						coll.Name, coll.Source, coll.SourceType, coll.IncludeMode, coll.PageCount)
				}
			}
		}

	case "links":
		ws := cond.GetWorkspace()
		if ws == nil {
			err = errors.New("workspace not initialized")
		} else {
			linkMgr := storage.GetLinkManager(ctx, ws)
			if linkMgr == nil {
				err = errors.New("links system is not available")
			} else {
				subcommand := "list"
				if len(args) > 0 {
					subcommand = args[0]
					args = args[1:]
				}
				switch subcommand {
				case "list", "ls":
					linkIndex := linkMgr.GetIndex()
					var totalLinks int
					for _, forwardLinks := range linkIndex.Forward {
						totalLinks += len(forwardLinks)
					}
					result = fmt.Sprintf("Total links: %d (from %d sources)", totalLinks, len(linkIndex.Forward))
				case "backlinks":
					if len(args) == 0 {
						err = errors.New("usage: links backlinks <entity-id>")
					} else {
						incoming := linkMgr.GetIncoming(args[0])
						if len(incoming) == 0 {
							result = "No backlinks to " + args[0]
						} else {
							var lines []string
							for _, link := range incoming {
								lines = append(lines, "• "+link.Source)
							}
							result = fmt.Sprintf("Backlinks to %s:\n%s", args[0], strings.Join(lines, "\n"))
						}
					}
				case "search":
					if len(args) == 0 {
						err = errors.New("usage: links search <query>")
					} else {
						query := strings.Join(args, " ")
						queryLower := strings.ToLower(query)
						names := linkMgr.GetNames()
						var matches []string
						// Search in specs
						for name := range names.Specs {
							if strings.Contains(strings.ToLower(name), queryLower) {
								matches = append(matches, "spec: "+name)
							}
						}
						// Search in decisions
						for name := range names.Decisions {
							if strings.Contains(strings.ToLower(name), queryLower) {
								matches = append(matches, "decision: "+name)
							}
						}
						if len(matches) == 0 {
							result = "No matching entities found"
						} else {
							result = fmt.Sprintf("Found %d match(es):\n• %s", len(matches), strings.Join(matches, "\n• "))
						}
					}
				case "stats":
					stats := linkMgr.GetStats()
					if stats == nil {
						err = errors.New("failed to get link stats")
					} else {
						result = fmt.Sprintf("Link Stats:\n• Total links: %d\n• Sources: %d\n• Targets: %d\n• Orphans: %d",
							stats.TotalLinks, stats.TotalSources, stats.TotalTargets, stats.OrphanEntities)
					}
				case "rebuild":
					if rebuildErr := linkMgr.Rebuild(); rebuildErr != nil {
						err = fmt.Errorf("rebuild failed: %w", rebuildErr)
					} else {
						stats := linkMgr.GetStats()
						result = fmt.Sprintf("Index rebuilt: %d links from %d sources", stats.TotalLinks, stats.TotalSources)
					}
				default:
					// Treat as entity ID for getting links
					outgoing := linkMgr.GetOutgoing(subcommand)
					incoming := linkMgr.GetIncoming(subcommand)
					if len(outgoing) == 0 && len(incoming) == 0 {
						result = "No links found for " + subcommand
					} else {
						var lines []string
						if len(outgoing) > 0 {
							lines = append(lines, fmt.Sprintf("Outgoing (%d):", len(outgoing)))
							for _, link := range outgoing {
								lines = append(lines, "  → "+link.Target)
							}
						}
						if len(incoming) > 0 {
							lines = append(lines, fmt.Sprintf("Incoming (%d):", len(incoming)))
							for _, link := range incoming {
								lines = append(lines, "  ← "+link.Source)
							}
						}
						result = strings.Join(lines, "\n")
					}
				}
			}
		}

	case "browser":
		ctrl := cond.GetBrowser(ctx)
		if ctrl == nil {
			err = errors.New("browser not configured. Start the browser with 'mehr browser --keep-alive status'")
		} else {
			subcommand := "status"
			subArgs := []string{}
			if len(args) > 0 {
				subcommand = strings.ToLower(args[0])
				subArgs = args[1:]
			}

			switch subcommand {
			case "status":
				tabs, tabErr := ctrl.ListTabs(ctx)
				if tabErr != nil {
					err = tabErr
				} else {
					var lines []string
					lines = append(lines, fmt.Sprintf("Connected to Chrome (port %d)", ctrl.GetPort()))
					lines = append(lines, fmt.Sprintf("Tabs: %d", len(tabs)))
					for i, tab := range tabs {
						title := tab.Title
						if len(title) > 50 {
							title = title[:47] + "..."
						}
						lines = append(lines, fmt.Sprintf("  %d. [%s] %s", i+1, tab.ID[:8], title))
					}
					result = strings.Join(lines, "\n")
				}

			case "tabs":
				tabs, tabErr := ctrl.ListTabs(ctx)
				if tabErr != nil {
					err = tabErr
				} else if len(tabs) == 0 {
					result = "No tabs open"
				} else {
					var lines []string
					for i, tab := range tabs {
						lines = append(lines, fmt.Sprintf("%d. [%s] %s\n   %s", i+1, tab.ID, tab.Title, tab.URL))
					}
					result = strings.Join(lines, "\n")
				}

			case "goto":
				if len(subArgs) == 0 {
					err = errors.New("usage: browser goto <url>")
				} else {
					tab, openErr := ctrl.OpenTab(ctx, subArgs[0])
					if openErr != nil {
						err = openErr
					} else {
						result = fmt.Sprintf("Opened: %s\nTab ID: %s", tab.Title, tab.ID)
					}
				}

			case "navigate":
				if len(subArgs) == 0 {
					err = errors.New("usage: browser navigate <url>")
				} else {
					tabs, tabErr := ctrl.ListTabs(ctx)
					if tabErr != nil || len(tabs) == 0 {
						err = errors.New("no tabs open")
					} else {
						if navErr := ctrl.Navigate(ctx, tabs[0].ID, subArgs[0]); navErr != nil {
							err = navErr
						} else {
							result = "Navigated to: " + subArgs[0]
						}
					}
				}

			case "close":
				if len(subArgs) == 0 {
					err = errors.New("usage: browser close <tab-id>")
				} else {
					if closeErr := ctrl.CloseTab(ctx, subArgs[0]); closeErr != nil {
						err = closeErr
					} else {
						result = "Closed tab: " + subArgs[0]
					}
				}

			case "reload":
				tabs, tabErr := ctrl.ListTabs(ctx)
				if tabErr != nil || len(tabs) == 0 {
					err = errors.New("no tabs open")
				} else {
					hard := len(subArgs) > 0 && subArgs[0] == "--hard"
					if reloadErr := ctrl.Reload(ctx, tabs[0].ID, hard); reloadErr != nil {
						err = reloadErr
					} else {
						result = "Page reloaded"
					}
				}

			case "screenshot":
				tabs, tabErr := ctrl.ListTabs(ctx)
				if tabErr != nil || len(tabs) == 0 {
					err = errors.New("no tabs open")
				} else {
					opts := browser.ScreenshotOptions{Format: "png", Quality: 80}
					data, ssErr := ctrl.Screenshot(ctx, tabs[0].ID, opts)
					if ssErr != nil {
						err = ssErr
					} else {
						result = fmt.Sprintf("Screenshot captured (%d bytes, use Web UI to view)", len(data))
					}
				}

			case "click":
				if len(subArgs) == 0 {
					err = errors.New("usage: browser click <selector>")
				} else {
					tabs, tabErr := ctrl.ListTabs(ctx)
					if tabErr != nil || len(tabs) == 0 {
						err = errors.New("no tabs open")
					} else {
						selector := strings.Join(subArgs, " ")
						if clickErr := ctrl.Click(ctx, tabs[0].ID, selector); clickErr != nil {
							err = clickErr
						} else {
							result = "Clicked: " + selector
						}
					}
				}

			case "type":
				if len(subArgs) < 2 {
					err = errors.New("usage: browser type <selector> <text>")
				} else {
					tabs, tabErr := ctrl.ListTabs(ctx)
					if tabErr != nil || len(tabs) == 0 {
						err = errors.New("no tabs open")
					} else {
						selector := subArgs[0]
						text := strings.Join(subArgs[1:], " ")
						if typeErr := ctrl.Type(ctx, tabs[0].ID, selector, text, false); typeErr != nil {
							err = typeErr
						} else {
							result = "Typed into: " + selector
						}
					}
				}

			case "dom":
				if len(subArgs) == 0 {
					err = errors.New("usage: browser dom <selector>")
				} else {
					tabs, tabErr := ctrl.ListTabs(ctx)
					if tabErr != nil || len(tabs) == 0 {
						err = errors.New("no tabs open")
					} else {
						selector := strings.Join(subArgs, " ")
						elem, domErr := ctrl.QuerySelector(ctx, tabs[0].ID, selector)
						if domErr != nil {
							err = domErr
						} else if elem == nil {
							result = "No element found for: " + selector
						} else {
							text := elem.TextContent
							if len(text) > 100 {
								text = text[:97] + "..."
							}
							result = fmt.Sprintf("<%s>\nText: %s\nVisible: %v", elem.TagName, text, elem.Visible)
						}
					}
				}

			case "eval":
				if len(subArgs) == 0 {
					err = errors.New("usage: browser eval <expression>")
				} else {
					tabs, tabErr := ctrl.ListTabs(ctx)
					if tabErr != nil || len(tabs) == 0 {
						err = errors.New("no tabs open")
					} else {
						expression := strings.Join(subArgs, " ")
						evalResult, evalErr := ctrl.Eval(ctx, tabs[0].ID, expression)
						if evalErr != nil {
							err = evalErr
						} else {
							result = fmt.Sprintf("Result: %v", evalResult)
						}
					}
				}

			case "console":
				tabs, tabErr := ctrl.ListTabs(ctx)
				if tabErr != nil || len(tabs) == 0 {
					err = errors.New("no tabs open")
				} else {
					duration := 3 * time.Second
					messages, consoleErr := ctrl.GetConsoleLogs(ctx, tabs[0].ID, duration)
					if consoleErr != nil {
						err = consoleErr
					} else if len(messages) == 0 {
						result = "No console messages captured"
					} else {
						var lines []string
						for _, msg := range messages {
							lines = append(lines, fmt.Sprintf("[%s] %s", msg.Level, msg.Text))
						}
						result = strings.Join(lines, "\n")
					}
				}

			case "network":
				tabs, tabErr := ctrl.ListTabs(ctx)
				if tabErr != nil || len(tabs) == 0 {
					err = errors.New("no tabs open")
				} else {
					duration := 3 * time.Second
					requests, netErr := ctrl.GetNetworkRequests(ctx, tabs[0].ID, duration)
					if netErr != nil {
						err = netErr
					} else if len(requests) == 0 {
						result = "No network requests captured"
					} else {
						var lines []string
						for _, req := range requests {
							status := ""
							if req.Status > 0 {
								status = fmt.Sprintf(" → %d", req.Status)
							}
							lines = append(lines, fmt.Sprintf("[%s] %s %s%s",
								req.Timestamp.Format("15:04:05"), req.Method, truncateURL(req.URL, 60), status))
						}
						result = strings.Join(lines, "\n")
					}
				}

			case "source":
				tabs, tabErr := ctrl.ListTabs(ctx)
				if tabErr != nil || len(tabs) == 0 {
					err = errors.New("no tabs open")
				} else {
					source, sourceErr := ctrl.GetPageSource(ctx, tabs[0].ID)
					if sourceErr != nil {
						err = sourceErr
					} else {
						if len(source) > 2000 {
							source = source[:2000] + "\n... (truncated)"
						}
						result = fmt.Sprintf("Page source (%d bytes):\n%s", len(source), source)
					}
				}

			default:
				result = fmt.Sprintf("Unknown browser subcommand: %s\nUse: status, tabs, goto, navigate, close, reload, screenshot, click, type, dom, eval, console, network, source", subcommand)
			}
		}

	case "simplify":
		task := cond.GetActiveTask()
		if task == nil {
			err = errors.New("no active task")
		} else {
			err = cond.Simplify(ctx, "", true)
			if err == nil {
				result = "Simplification complete"
			}
		}

	case "label":
		task := cond.GetActiveTask()
		if task == nil {
			err = errors.New("no active task")
		} else {
			ws := cond.GetWorkspace()
			if len(args) == 0 {
				labels, _ := ws.GetLabels(task.ID)
				if len(labels) == 0 {
					result = "No labels"
				} else {
					result = "Labels: " + strings.Join(labels, ", ")
				}
			} else {
				subCmd := args[0]
				subArgs := args[1:]
				switch subCmd {
				case "add":
					for _, label := range subArgs {
						_ = ws.AddLabel(task.ID, label)
					}
					result = fmt.Sprintf("Added %d label(s)", len(subArgs))
				case "remove", "rm":
					for _, label := range subArgs {
						_ = ws.RemoveLabel(task.ID, label)
					}
					result = fmt.Sprintf("Removed %d label(s)", len(subArgs))
				case "clear":
					_ = ws.SetLabels(task.ID, []string{})
					result = "Labels cleared"
				case "list", "ls":
					labels, _ := ws.GetLabels(task.ID)
					if len(labels) == 0 {
						result = "No labels"
					} else {
						result = "Labels: " + strings.Join(labels, ", ")
					}
				default:
					// Treat as adding labels directly
					for _, label := range args {
						_ = ws.AddLabel(task.ID, label)
					}
					result = fmt.Sprintf("Added %d label(s)", len(args))
				}
			}
		}

	case "budget":
		task := cond.GetActiveTask()
		if task == nil {
			err = errors.New("no active task")
		} else {
			ws := cond.GetWorkspace()
			work, loadErr := ws.LoadWork(task.ID)
			if loadErr != nil {
				err = loadErr
			} else {
				cfg, cfgErr := ws.LoadConfig()
				if cfgErr != nil {
					err = cfgErr
				} else {
					taskBudget := cfg.Budget.PerTask
					if work.Budget != nil {
						taskBudget = *work.Budget
					}
					costs := work.Costs
					totalTokens := costs.TotalInputTokens + costs.TotalOutputTokens
					result = fmt.Sprintf("Tokens: %d\nCost: $%.4f / $%.2f budget",
						totalTokens, costs.TotalCostUSD, taskBudget.MaxCost)
				}
			}
		}

	case "delete":
		// Delete a queue task: delete <queue>/<task-id>
		if len(args) == 0 {
			err = errors.New("delete requires a task reference (e.g., quick-tasks/task-1)")
		} else {
			queueID, taskID, parseErr := conductor.ParseQueueTaskRef(args[0])
			if parseErr != nil {
				err = parseErr
			} else {
				ws := cond.GetWorkspace()
				queue, loadErr := storage.LoadTaskQueue(ws, queueID)
				if loadErr != nil {
					err = fmt.Errorf("queue not found: %s", queueID)
				} else if !queue.RemoveTask(taskID) {
					err = fmt.Errorf("task not found: %s/%s", queueID, taskID)
				} else if saveErr := queue.Save(); saveErr != nil {
					err = fmt.Errorf("save queue: %w", saveErr)
				} else {
					// Delete notes file
					notesPath := ws.QueueNotePath(queueID, taskID)
					_ = ws.DeleteFile(notesPath)
					result = fmt.Sprintf("Deleted task %s from %s", taskID, queueID)
				}
			}
		}

	case "export":
		// Export a queue task to markdown: export <queue>/<task-id>
		if len(args) == 0 {
			err = errors.New("export requires a task reference (e.g., quick-tasks/task-1)")
		} else {
			queueID, taskID, parseErr := conductor.ParseQueueTaskRef(args[0])
			if parseErr != nil {
				err = parseErr
			} else {
				markdown, exportErr := cond.ExportQueueTask(queueID, taskID)
				if exportErr != nil {
					err = exportErr
				} else {
					// Return the markdown content (truncated for display)
					preview := markdown
					if len(preview) > 1000 {
						preview = preview[:1000] + "\n... (truncated)"
					}
					result = fmt.Sprintf("Exported %s/%s:\n%s", queueID, taskID, preview)
				}
			}
		}

	case "optimize":
		// AI optimize a queue task: optimize <queue>/<task-id>
		if len(args) == 0 {
			err = errors.New("optimize requires a task reference (e.g., quick-tasks/task-1)")
		} else {
			queueID, taskID, parseErr := conductor.ParseQueueTaskRef(args[0])
			if parseErr != nil {
				err = parseErr
			} else {
				optimized, optimizeErr := cond.OptimizeQueueTask(ctx, queueID, taskID)
				if optimizeErr != nil {
					err = optimizeErr
				} else {
					var changes []string
					if optimized.OriginalTitle != optimized.OptimizedTitle {
						changes = append(changes, fmt.Sprintf("Title: %s → %s", optimized.OriginalTitle, optimized.OptimizedTitle))
					}
					if len(optimized.AddedLabels) > 0 {
						changes = append(changes, "Added labels: "+strings.Join(optimized.AddedLabels, ", "))
					}
					if len(optimized.ImprovementNotes) > 0 {
						changes = append(changes, "Improvements: "+strings.Join(optimized.ImprovementNotes, "; "))
					}
					if len(changes) == 0 {
						result = "Task optimized (no major changes)"
					} else {
						result = "Task optimized:\n• " + strings.Join(changes, "\n• ")
					}
				}
			}
		}

	case "submit":
		// Submit a queue task to provider: submit <queue>/<task-id> <provider>
		if len(args) < 2 {
			err = errors.New("submit requires: submit <queue>/<task-id> <provider>")
		} else {
			queueID, taskID, parseErr := conductor.ParseQueueTaskRef(args[0])
			if parseErr != nil {
				err = parseErr
			} else {
				providerName := args[1]
				submitResult, submitErr := cond.SubmitQueueTask(ctx, queueID, taskID, conductor.SubmitOptions{
					Provider: providerName,
					TaskIDs:  []string{taskID},
				})
				if submitErr != nil {
					err = submitErr
				} else if len(submitResult.Tasks) == 0 {
					result = "No tasks submitted"
				} else {
					r := submitResult.Tasks[0]
					result = fmt.Sprintf("Submitted to %s: %s\nURL: %s", providerName, r.ExternalID, r.ExternalURL)
				}
			}
		}

	case "sync":
		// Sync task from provider: sync <task-id>
		task := cond.GetActiveTask()
		if task == nil {
			err = errors.New("no active task to sync")
		} else {
			// For now, indicate that sync is available - full implementation would require
			// provider fetch and delta spec generation
			result = fmt.Sprintf("Sync requested for task %s. Use 'mehr sync %s' from CLI for full provider sync.", task.ID, task.ID)
		}

	case "answer":
		if len(args) == 0 {
			err = errors.New("answer requires a response")
		} else {
			task := cond.GetActiveTask()
			if task == nil {
				err = errors.New("no active task")
			} else {
				ws := cond.GetWorkspace()
				// Clear pending question
				if clearErr := ws.ClearPendingQuestion(task.ID); clearErr != nil {
					slog.Warn("clear pending question", "error", clearErr)
				}
				// Save answer as note
				response := strings.Join(args, " ")
				if noteErr := ws.AppendNote(task.ID, response, task.State); noteErr != nil {
					err = noteErr
				} else {
					result = "Answer saved, resuming..."
					// Resume workflow based on state with timeout context
					state := workflow.State(task.State)
					const resumeTimeout = 5 * time.Minute
					switch state {
					case workflow.StatePlanning:
						go func(ctx context.Context) {
							resumeCtx, cancel := context.WithTimeout(ctx, resumeTimeout)
							defer cancel()
							if resumeErr := cond.Plan(resumeCtx); resumeErr != nil {
								slog.Error("workflow resume failed", "step", "plan", "error", resumeErr)
							}
						}(ctx)
					case workflow.StateImplementing:
						go func(ctx context.Context) {
							resumeCtx, cancel := context.WithTimeout(ctx, resumeTimeout)
							defer cancel()
							if resumeErr := cond.Implement(resumeCtx); resumeErr != nil {
								slog.Error("workflow resume failed", "step", "implement", "error", resumeErr)
							}
						}(ctx)
					case workflow.StateReviewing:
						go func(ctx context.Context) {
							resumeCtx, cancel := context.WithTimeout(ctx, resumeTimeout)
							defer cancel()
							if resumeErr := cond.Review(resumeCtx); resumeErr != nil {
								slog.Error("workflow resume failed", "step", "review", "error", resumeErr)
							}
						}(ctx)
					case workflow.StateIdle, workflow.StateDone, workflow.StateFailed,
						workflow.StateWaiting, workflow.StatePaused,
						workflow.StateCheckpointing, workflow.StateReverting, workflow.StateRestoring:
						// These states are not resumable - do nothing
					}
				}
			}
		}

	case "question":
		if len(args) == 0 {
			err = errors.New("question requires a message")
		} else {
			question := strings.Join(args, " ")
			questionErr := cond.AskQuestion(ctx, question)
			if questionErr != nil {
				err = questionErr
			} else {
				result = "Question sent to agent"
			}
		}

	case "project":
		ws := cond.GetWorkspace()
		if ws == nil {
			err = errors.New("workspace not initialized")
		} else {
			subcommand := "help"
			subArgs := []string{}
			if len(args) > 0 {
				subcommand = strings.ToLower(args[0])
				subArgs = args[1:]
			}

			switch subcommand {
			case "plan":
				if len(subArgs) == 0 {
					err = errors.New("usage: project plan <source>")
				} else {
					source := subArgs[0]
					opts := conductor.ProjectPlanOptions{}
					// Parse optional --title flag
					for i := 1; i < len(subArgs); i++ {
						if subArgs[i] == "--title" && i+1 < len(subArgs) {
							opts.Title = subArgs[i+1]
							i++
						}
					}
					planResult, planErr := cond.CreateProjectPlan(ctx, source, opts)
					if planErr != nil {
						err = planErr
					} else {
						result = fmt.Sprintf("Created queue: %s\n  %d tasks identified", planResult.Queue.ID, len(planResult.Tasks))
						if len(planResult.Questions) > 0 {
							result += fmt.Sprintf("\n  %d questions to resolve", len(planResult.Questions))
						}
					}
				}

			case "tasks":
				var queueID string
				if len(subArgs) > 0 {
					queueID = subArgs[0]
				} else {
					queues, listErr := ws.ListQueues()
					if listErr != nil {
						err = listErr
					} else if len(queues) == 0 {
						err = errors.New("no queues found")
					} else {
						queueID = queues[len(queues)-1]
					}
				}
				if err == nil && queueID != "" {
					queue, loadErr := storage.LoadTaskQueue(ws, queueID)
					if loadErr != nil {
						err = loadErr
					} else {
						var lines []string
						lines = append(lines, fmt.Sprintf("Queue: %s (%d tasks)", queue.ID, len(queue.Tasks)))
						for _, task := range queue.Tasks {
							lines = append(lines, fmt.Sprintf("  • %s: %s [%s]", task.ID, truncateStr(task.Title, 40), task.Status))
						}
						result = strings.Join(lines, "\n")
					}
				}

			case "edit":
				if len(subArgs) == 0 {
					err = errors.New("usage: project edit <task-id> [--title <title>] [--status <status>]")
				} else {
					taskID := subArgs[0]
					queues, listErr := ws.ListQueues()
					if listErr != nil {
						err = listErr
					} else if len(queues) == 0 {
						err = errors.New("no queues found")
					} else {
						queueID := queues[len(queues)-1]
						queue, loadErr := storage.LoadTaskQueue(ws, queueID)
						if loadErr != nil {
							err = loadErr
						} else {
							updateErr := queue.UpdateTask(taskID, func(task *storage.QueuedTask) {
								for i := 1; i < len(subArgs); i++ {
									switch subArgs[i] {
									case "--title":
										if i+1 < len(subArgs) {
											task.Title = subArgs[i+1]
											i++
										}
									case "--status":
										if i+1 < len(subArgs) {
											task.Status = storage.TaskStatus(subArgs[i+1])
											i++
										}
									case "--priority":
										if i+1 < len(subArgs) {
											if p, pErr := strconv.Atoi(subArgs[i+1]); pErr == nil {
												task.Priority = p
											}
											i++
										}
									}
								}
							})
							if updateErr != nil {
								err = updateErr
							} else if saveErr := queue.Save(); saveErr != nil {
								err = saveErr
							} else {
								result = "Updated task: " + taskID
							}
						}
					}
				}

			case "reorder":
				if len(subArgs) == 0 {
					err = errors.New("usage: project reorder <task-id> --before|--after <target-id>")
				} else if len(subArgs) >= 3 {
					taskID := subArgs[0]
					queues, listErr := ws.ListQueues()
					if listErr != nil {
						err = listErr
					} else if len(queues) == 0 {
						err = errors.New("no queues found")
					} else {
						queueID := queues[len(queues)-1]
						queue, loadErr := storage.LoadTaskQueue(ws, queueID)
						if loadErr != nil {
							err = loadErr
						} else {
							var targetIndex int
							for i := 1; i < len(subArgs); i++ {
								if subArgs[i] == "--before" && i+1 < len(subArgs) {
									for j, t := range queue.Tasks {
										if t.ID == subArgs[i+1] {
											targetIndex = j

											break
										}
									}
								} else if subArgs[i] == "--after" && i+1 < len(subArgs) {
									for j, t := range queue.Tasks {
										if t.ID == subArgs[i+1] {
											targetIndex = j + 1

											break
										}
									}
								}
							}
							if reorderErr := queue.ReorderTask(taskID, targetIndex); reorderErr != nil {
								err = reorderErr
							} else if saveErr := queue.Save(); saveErr != nil {
								err = saveErr
							} else {
								result = fmt.Sprintf("Moved task %s to position %d", taskID, targetIndex+1)
							}
						}
					}
				} else {
					err = errors.New("usage: project reorder <task-id> --before|--after <target-id>")
				}

			case "submit":
				if len(subArgs) < 1 {
					err = errors.New("usage: project submit --provider <provider>")
				} else {
					var providerName string
					for i := 0; i < len(subArgs); i++ {
						if subArgs[i] == "--provider" && i+1 < len(subArgs) {
							providerName = subArgs[i+1]
							i++
						}
					}
					if providerName == "" {
						err = errors.New("--provider is required")
					} else {
						queues, listErr := ws.ListQueues()
						if listErr != nil {
							err = listErr
						} else if len(queues) == 0 {
							err = errors.New("no queues found")
						} else {
							queueID := queues[len(queues)-1]
							opts := conductor.SubmitOptions{Provider: providerName}
							submitResult, submitErr := cond.SubmitProjectTasks(ctx, queueID, opts)
							if submitErr != nil {
								err = submitErr
							} else {
								result = fmt.Sprintf("Submitted %d tasks to %s", len(submitResult.Tasks), providerName)
							}
						}
					}
				}

			case "start":
				queues, listErr := ws.ListQueues()
				if listErr != nil {
					err = listErr
				} else if len(queues) == 0 {
					err = errors.New("no queues found")
				} else {
					queueID := queues[len(queues)-1]
					task, startErr := cond.StartNextTask(ctx, queueID)
					if startErr != nil {
						err = startErr
					} else {
						result = fmt.Sprintf("Started task: %s - %s", task.ID, task.Title)
					}
				}

			case "sync":
				if len(subArgs) == 0 {
					err = errors.New("usage: project sync <provider:reference>")
				} else {
					reference := subArgs[0]
					opts := conductor.SyncProjectOptions{}
					syncResult, syncErr := cond.SyncProject(ctx, reference, opts)
					if syncErr != nil {
						err = syncErr
					} else {
						result = fmt.Sprintf("Synced project: %s\n  Queue: %s\n  Tasks: %d synced", syncResult.Queue.Title, syncResult.Queue.ID, syncResult.TasksSync)
					}
				}

			default:
				result = `Project commands:
• project plan <source> - Create task breakdown from source
• project tasks [queue-id] - View tasks in queue
• project edit <task-id> --title|--status|--priority - Edit task
• project reorder <task-id> --before|--after <target> - Reorder tasks
• project submit --provider <name> - Submit to provider
• project start - Start next task from queue
• project sync <provider:ref> - Sync from provider`
			}
		}

	case "stack":
		ws := cond.GetWorkspace()
		if ws == nil {
			err = errors.New("workspace not initialized")
		} else {
			subcommand := "list"
			subArgs := []string{}
			if len(args) > 0 {
				subcommand = strings.ToLower(args[0])
				subArgs = args[1:]
			}

			stackStorage := stack.NewStorage(ws.DataRoot())
			if loadErr := stackStorage.Load(); loadErr != nil {
				err = fmt.Errorf("load stacks: %w", loadErr)
			} else {
				switch subcommand {
				case "list", "ls":
					stacks := stackStorage.ListStacks()
					if len(stacks) == 0 {
						result = "No stacked features found.\nUse 'mehr start <task> --depends-on <parent>' to create a stacked feature."
					} else {
						var lines []string
						for _, s := range stacks {
							lines = append(lines, fmt.Sprintf("Stack: %s (%d tasks)", s.ID, s.TaskCount()))
							for _, t := range s.Tasks {
								lines = append(lines, fmt.Sprintf("  • %s [%s] %s", t.ID, t.State, t.Branch))
							}
						}
						result = strings.Join(lines, "\n")
					}

				case "rebase":
					git := cond.GetGit()
					if git == nil {
						err = errors.New("not in a git repository")
					} else {
						rebaser := stack.NewRebaser(stackStorage, git)
						if len(subArgs) > 0 {
							// Single task rebase
							taskID := subArgs[0]
							preview, previewErr := rebaser.PreviewTask(ctx, taskID)
							if previewErr != nil {
								err = previewErr
							} else if preview.WouldConflict {
								err = fmt.Errorf("cannot rebase %s: conflicts detected", taskID)
							} else {
								rebaseResult, rebaseErr := rebaser.RebaseTask(ctx, taskID)
								if rebaseErr != nil {
									err = rebaseErr
								} else if rebaseResult.FailedTask != nil {
									err = fmt.Errorf("rebase failed for %s", rebaseResult.FailedTask.TaskID)
								} else {
									result = "Rebased task " + taskID
								}
							}
						} else {
							// Rebase all needing it
							var stacksWithRebase []*stack.Stack
							for _, s := range stackStorage.ListStacks() {
								if len(s.GetTasksNeedingRebase()) > 0 {
									stacksWithRebase = append(stacksWithRebase, s)
								}
							}
							if len(stacksWithRebase) == 0 {
								result = "No tasks need rebasing"
							} else {
								var rebased int
								for _, s := range stacksWithRebase {
									rebaseResult, rebaseErr := rebaser.RebaseAll(ctx, s.ID)
									if rebaseErr != nil {
										err = rebaseErr

										break
									}
									rebased += len(rebaseResult.RebasedTasks)
								}
								if err == nil {
									result = fmt.Sprintf("Rebased %d task(s)", rebased)
								}
							}
						}
					}

				case "sync":
					result = "Stack sync requires provider configuration. Use CLI for full sync."

				default:
					result = `Stack commands:
• stack - List stacked features
• stack rebase [task-id] - Rebase stacked tasks
• stack sync - Sync PR status`
				}
			}
		}

	case "config":
		// Config validate/explain commands
		if len(args) > 0 {
			subCmd := args[0]
			switch subCmd {
			case "validate":
				ws := s.config.Conductor.GetWorkspace()
				if ws == nil {
					result = "Workspace not initialized"
				} else {
					validator := validation.New(ws.Root(), validation.Options{})
					validationResult, validateErr := validator.Validate(ctx)
					if validateErr != nil {
						result = fmt.Sprintf("Validation error: %s", validateErr)
					} else if validationResult.Valid {
						result = "✓ Configuration is valid"
					} else {
						result = validationResult.Format("text")
					}
				}
			case "explain":
				// Config explain requires step argument and complex resolution
				result = "Config explain requires step argument. Use CLI: mehr config explain --agent <planning|implementing|reviewing>"
			default:
				result = `Config commands:
• config validate - Check configuration for issues
• config explain --agent <step> - Explain agent resolution (CLI only)`
			}
		} else {
			result = `Config commands:
• config validate - Check configuration for issues
• config explain --agent <step> - Explain agent resolution (CLI only)`
		}

	case "agents":
		// Agents list/explain commands
		if len(args) > 0 {
			subCmd := args[0]
			switch subCmd {
			case "list":
				registry := s.config.Conductor.GetAgentRegistry()
				if registry == nil {
					result = "Agent registry not initialized"
				} else {
					agentNames := registry.List()
					if len(agentNames) == 0 {
						result = "No agents configured"
					} else {
						var sb strings.Builder
						sb.WriteString("Available agents:\n")
						for _, name := range agentNames {
							sb.WriteString("• " + name)
							// Check availability
							if ag, agErr := registry.Get(name); agErr == nil {
								if ag.Available() != nil {
									sb.WriteString(" (unavailable)")
								}
							}
							sb.WriteString("\n")
						}
						result = sb.String()
					}
				}
			case "explain":
				if len(args) > 1 {
					agentName := args[1]
					registry := s.config.Conductor.GetAgentRegistry()
					if registry == nil {
						result = "Agent registry not initialized"
					} else {
						ag, agErr := registry.Get(agentName)
						if agErr != nil {
							result = "Agent not found: " + agentName
						} else {
							var sb strings.Builder
							sb.WriteString(fmt.Sprintf("Agent: %s\n", ag.Name()))
							if ag.Available() == nil {
								sb.WriteString("Status: Available\n")
							} else {
								sb.WriteString(fmt.Sprintf("Status: Unavailable (%s)\n", ag.Available()))
							}
							result = sb.String()
						}
					}
				} else {
					result = "Usage: agents explain <name>"
				}
			default:
				result = `Agents commands:
• agents list - List available agents
• agents explain <name> - Show agent details`
			}
		} else {
			result = `Agents commands:
• agents list - List available agents
• agents explain <name> - Show agent details`
		}

	case "providers":
		// Providers list/info/status commands
		if len(args) > 0 {
			subCmd := args[0]
			switch subCmd {
			case "list":
				result = `Available providers:
• file (f) - Single markdown file
• dir (d) - Directory with README.md
• github (gh) - GitHub issues and pull requests
• gitlab - GitLab issues and merge requests
• jira - Atlassian Jira tickets
• linear - Linear issues
• notion - Notion pages and databases
• wrike - Wrike tasks
• youtrack (yt) - JetBrains YouTrack issues

Usage: mehr start <scheme>:<reference>`
			case "info":
				if len(args) > 1 {
					providerName := strings.ToLower(args[1])
					info := getProviderInfoText(providerName)
					if info != "" {
						result = info
					} else {
						result = fmt.Sprintf("Unknown provider: %s\nRun 'providers list' to see available providers.", providerName)
					}
				} else {
					result = "Usage: providers info <name>"
				}
			case "status":
				// Provider status requires conductor initialization and health checks
				result = "Provider status check requires CLI. Use: mehr providers status"
			default:
				result = `Providers commands:
• providers list - List available providers
• providers info <name> - Show provider details
• providers status - Check connection status (CLI only)`
			}
		} else {
			result = `Providers commands:
• providers list - List available providers
• providers info <name> - Show provider details
• providers status - Check connection status (CLI only)`
		}

	case "templates":
		// Templates list/show/apply commands
		if len(args) > 0 {
			subCmd := args[0]
			switch subCmd {
			case "list":
				templateNames := template.BuiltInTemplates()
				if len(templateNames) == 0 {
					result = "No templates available"
				} else {
					var sb strings.Builder
					sb.WriteString("Available templates:\n")
					for _, name := range templateNames {
						if tpl, tplErr := template.LoadBuiltIn(name); tplErr == nil {
							sb.WriteString("• " + name)
							if tpl.Description != "" {
								sb.WriteString(" - " + tpl.Description)
							}
							sb.WriteString("\n")
						} else {
							sb.WriteString(fmt.Sprintf("• %s\n", name))
						}
					}
					result = sb.String()
				}
			case "show":
				if len(args) > 1 {
					templateName := args[1]
					tpl, tplErr := template.LoadBuiltIn(templateName)
					if tplErr != nil {
						result = "Template not found: " + templateName
					} else {
						result = tpl.GetDescription()
					}
				} else {
					result = "Usage: templates show <name>"
				}
			case "apply":
				result = "Template apply requires file selection. Use CLI: mehr templates apply <name> <file>"
			default:
				result = `Templates commands:
• templates list - List available templates
• templates show <name> - Show template content
• templates apply <name> <file> - Apply template (CLI only)`
			}
		} else {
			result = `Templates commands:
• templates list - List available templates
• templates show <name> - Show template content
• templates apply <name> <file> - Apply template (CLI only)`
		}

	case "scan":
		// Security scanning
		result = "Security scanning requires CLI for full output. Use: mehr scan [--gosec] [--gitleaks] [--govulncheck]"

	case "commit":
		// AI commit assistance
		result = "Commit assistance requires CLI for git integration. Use: mehr commit [--analyze] [--preview] [--execute]"

	case "help":
		result = `Commands:
• start <ref> - Start a task
• plan - Run planning
• implement - Run implementation
• implement review <n> - Fix issues from review
• review - Run code review
• review <n> - View review content
• continue - Resume paused
• finish - Complete task
• abandon - Discard task
• auto - Auto-execute next step
• reset - Reset workflow to idle
• undo/redo - Checkpoints
• status - Show status
• cost - Show token usage
• budget - Show budget status
• list - List tasks
• note <msg> - Add a note
• question <msg> - Ask agent a question
• answer <resp> - Answer agent question
• find <query> - Search code
• memory search <query> - Search similar tasks
• memory index <task-id> - Index task to memory
• memory stats - Show memory statistics
• library list/show/search/pull/remove/stats - Documentation library
• links list/backlinks/search/stats/rebuild - Entity links
• browser status/tabs/goto/navigate/close/reload - Browser automation
• browser screenshot/click/type/dom/eval - Page interaction
• browser console/network/source - DevTools data
• project plan/tasks/edit/reorder/submit/start/sync - Project planning
• stack/stack rebase/stack sync - Stacked features
• specification [n] - View specifications
• quick <desc> - Create quick task
• delete <queue>/<id> - Delete queue task
• export <queue>/<id> - Export task to markdown
• optimize <queue>/<id> - AI optimize task
• submit <queue>/<id> <provider> - Submit to provider
• sync - Sync task from provider
• simplify - Simplify code
• label [add|rm] <labels> - Manage labels
• config validate/explain - Configuration validation
• agents list/explain - Agent management
• providers list/info/status - Provider management
• templates list/show - Template management
• scan - Security scanning (CLI)
• commit - AI commit assistance (CLI)
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

	// First, check if the command is handled by the unified router
	if commands.IsKnownCommand(req.Command) {
		result, err := commands.Execute(opCtx, cond, req.Command, req.Args)
		if err != nil {
			// Map specific errors to HTTP status codes
			if errors.Is(err, commands.ErrNoActiveTask) {
				s.writeError(w, http.StatusBadRequest, "no active task")

				return
			}
			if errors.Is(err, commands.ErrUnknownCommand) {
				s.writeError(w, http.StatusBadRequest, err.Error())

				return
			}
			s.writeError(w, http.StatusInternalServerError, err.Error())

			return
		}

		// Handle special result types
		if result != nil {
			// Handle exit signal (shouldn't happen in Web, but handle for completeness)
			if result.Type == commands.ResultExit {
				s.writeJSON(w, http.StatusOK, commandResponse{
					Success: true,
					Message: "exit",
					State:   result.State,
				})

				return
			}

			// Convert router result to JSON response
			response := s.routerResultToJSON(result)
			s.writeJSON(w, http.StatusOK, response)

			// Publish state update if applicable
			if result.TaskID != "" && result.State != "" {
				s.config.EventBus.PublishRaw(eventbus.Event{
					Type: events.TypeStateChanged,
					Data: map[string]any{
						"task_id": result.TaskID,
						"state":   result.State,
					},
				})
			}

			return
		}

		// Fallback response
		s.writeJSON(w, http.StatusOK, commandResponse{Success: true, Message: "OK"})

		return
	}

	// Handle Web-specific commands that aren't in the router
	var message string

	switch req.Command {
	case "reset":
		err = cond.ResetState(opCtx)
		message = "Workflow reset to idle"

	case "auto":
		// Auto-execute the next workflow step based on current state
		task := cond.GetActiveTask()
		if task == nil {
			s.writeError(w, http.StatusBadRequest, "no active task")

			return
		}
		switch task.State {
		case "idle":
			s.writeError(w, http.StatusBadRequest, "no active task, use 'start' first")

			return
		case "planning":
			err = cond.Plan(opCtx)
			message = "Planning started"
		case "implementing":
			err = cond.Implement(opCtx)
			message = "Implementation started"
		case "reviewing":
			err = cond.Review(opCtx)
			message = "Review started"
		case "waiting":
			s.writeError(w, http.StatusBadRequest, "task is waiting for user input")

			return
		case "done", "failed":
			s.writeError(w, http.StatusBadRequest, "task is already completed")

			return
		default:
			s.writeError(w, http.StatusBadRequest, "cannot auto-execute in state: "+task.State)

			return
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
			s.writeError(w, http.StatusBadRequest, "memory requires a subcommand: search <query>, index <task-id>, stats")

			return
		}
		subcommand := req.Args[0]
		subArgs := req.Args[1:]
		switch subcommand {
		case "search":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "memory search requires a query")

				return
			}
			query := strings.Join(subArgs, " ")
			results, searchErr := mem.Search(r.Context(), query, memory.SearchOptions{
				Limit:    5,
				MinScore: 0.65,
			})
			if searchErr != nil {
				s.writeError(w, http.StatusInternalServerError, "memory search: "+searchErr.Error())

				return
			}
			message = fmt.Sprintf("Found %d similar task(s)", len(results))
		case "index":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "memory index requires a task ID")

				return
			}
			ws := cond.GetWorkspace()
			if ws == nil {
				s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

				return
			}
			taskID := subArgs[0]
			if _, loadErr := ws.LoadWork(taskID); loadErr != nil {
				s.writeError(w, http.StatusNotFound, "task not found: "+loadErr.Error())

				return
			}
			indexer := memory.NewIndexer(mem, ws, nil)
			if indexErr := indexer.IndexTask(r.Context(), taskID); indexErr != nil {
				s.writeError(w, http.StatusInternalServerError, "failed to index task: "+indexErr.Error())

				return
			}
			message = fmt.Sprintf("Task %s indexed successfully", taskID)
		case "stats":
			ws := cond.GetWorkspace()
			if ws == nil {
				s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

				return
			}
			indexer := memory.NewIndexer(mem, ws, nil)
			stats, statsErr := indexer.GetStats(r.Context())
			if statsErr != nil {
				s.writeError(w, http.StatusInternalServerError, "failed to get stats: "+statsErr.Error())

				return
			}
			message = fmt.Sprintf("Total documents: %d", stats.TotalDocuments)
		default:
			// Backwards compatibility: treat unknown subcommand as search query
			query := strings.Join(req.Args, " ")
			results, searchErr := mem.Search(r.Context(), query, memory.SearchOptions{
				Limit:    5,
				MinScore: 0.65,
			})
			if searchErr != nil {
				s.writeError(w, http.StatusInternalServerError, "memory search: "+searchErr.Error())

				return
			}
			message = fmt.Sprintf("Found %d similar task(s)", len(results))
		}

	case "library":
		lib := cond.GetLibrary()
		if lib == nil {
			// Check if there was an initialization error
			errMsg := "library system is not enabled. Use the Library panel or enable in .mehrhof/config.yaml under 'library:'"
			if initErr := cond.GetLibraryError(); initErr != nil {
				errMsg = initErr.Error()
			}
			s.writeError(w, http.StatusServiceUnavailable, errMsg)

			return
		}
		// Default to list if no subcommand
		subcommand := "list"
		subArgs := req.Args
		if len(subArgs) > 0 {
			subcommand = subArgs[0]
			subArgs = subArgs[1:]
		}
		switch subcommand {
		case "list", "ls":
			collections, listErr := lib.List(r.Context(), &library.ListOptions{})
			if listErr != nil {
				s.writeError(w, http.StatusInternalServerError, "list collections: "+listErr.Error())

				return
			}
			message = fmt.Sprintf("Found %d collection(s)", len(collections))
		case "show":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "usage: library show <name>")

				return
			}
			coll, showErr := lib.Show(r.Context(), subArgs[0])
			if showErr != nil {
				s.writeError(w, http.StatusInternalServerError, "show collection: "+showErr.Error())

				return
			}
			message = fmt.Sprintf("Collection: %s (%d pages)", coll.Name, coll.PageCount)
		case "search":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "usage: library search <query>")

				return
			}
			searchQuery := strings.Join(subArgs, " ")
			docCtx, searchErr := lib.GetDocsForQuery(r.Context(), searchQuery, 10000)
			if searchErr != nil {
				s.writeError(w, http.StatusInternalServerError, "search library: "+searchErr.Error())

				return
			}
			if docCtx == nil || len(docCtx.Pages) == 0 {
				message = "No matching documentation found"
			} else {
				// Count unique collections
				collectionSet := make(map[string]bool)
				for _, p := range docCtx.Pages {
					collectionSet[p.CollectionName] = true
				}
				message = fmt.Sprintf("Found %d page(s) from %d collection(s)", len(docCtx.Pages), len(collectionSet))
			}
		case "pull":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "usage: library pull <source>")

				return
			}
			source := subArgs[0]
			opts := &library.PullOptions{}
			// Parse simple flags
			for i := 1; i < len(subArgs); i++ {
				if subArgs[i] == "--name" && i+1 < len(subArgs) {
					opts.Name = subArgs[i+1]
					i++
				} else if subArgs[i] == "--shared" {
					opts.Shared = true
				}
			}
			pullResult, pullErr := lib.Pull(r.Context(), source, opts)
			if pullErr != nil {
				s.writeError(w, http.StatusInternalServerError, "pull library: "+pullErr.Error())

				return
			}
			message = fmt.Sprintf("Pulled collection: %s (%d pages)", pullResult.Collection.Name, pullResult.Collection.PageCount)
		case "remove", "rm":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "usage: library remove <name>")

				return
			}
			if removeErr := lib.Remove(r.Context(), subArgs[0], false); removeErr != nil {
				s.writeError(w, http.StatusInternalServerError, "remove collection: "+removeErr.Error())

				return
			}
			message = fmt.Sprintf("Collection '%s' removed", subArgs[0])
		case "stats":
			collections, listErr := lib.List(r.Context(), &library.ListOptions{})
			if listErr != nil {
				s.writeError(w, http.StatusInternalServerError, "list collections: "+listErr.Error())

				return
			}
			var totalPages int
			for _, c := range collections {
				totalPages += c.PageCount
			}
			message = fmt.Sprintf("%d collections, %d total pages", len(collections), totalPages)
		default:
			// Treat as collection name for show
			coll, showErr := lib.Show(r.Context(), subcommand)
			if showErr != nil {
				s.writeError(w, http.StatusInternalServerError, "show collection: "+showErr.Error())

				return
			}
			message = fmt.Sprintf("Collection: %s (%d pages)", coll.Name, coll.PageCount)
		}

	case "links":
		ws := cond.GetWorkspace()
		if ws == nil {
			s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

			return
		}
		linkMgr := storage.GetLinkManager(r.Context(), ws)
		if linkMgr == nil {
			s.writeError(w, http.StatusServiceUnavailable, "links system is not available")

			return
		}
		subcommand := "list"
		subArgs := req.Args
		if len(subArgs) > 0 {
			subcommand = subArgs[0]
			subArgs = subArgs[1:]
		}
		switch subcommand {
		case "list", "ls":
			linkIndex := linkMgr.GetIndex()
			var totalLinks int
			for _, forwardLinks := range linkIndex.Forward {
				totalLinks += len(forwardLinks)
			}
			message = fmt.Sprintf("%d links from %d sources", totalLinks, len(linkIndex.Forward))
		case "backlinks":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "usage: links backlinks <entity-id>")

				return
			}
			incoming := linkMgr.GetIncoming(subArgs[0])
			message = fmt.Sprintf("%d backlinks to %s", len(incoming), subArgs[0])
		case "search":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "usage: links search <query>")

				return
			}
			query := strings.Join(subArgs, " ")
			queryLower := strings.ToLower(query)
			names := linkMgr.GetNames()
			var matchCount int
			for name := range names.Specs {
				if strings.Contains(strings.ToLower(name), queryLower) {
					matchCount++
				}
			}
			for name := range names.Decisions {
				if strings.Contains(strings.ToLower(name), queryLower) {
					matchCount++
				}
			}
			message = fmt.Sprintf("Found %d matching entities", matchCount)
		case "stats":
			stats := linkMgr.GetStats()
			if stats == nil {
				s.writeError(w, http.StatusInternalServerError, "failed to get link stats")

				return
			}
			message = fmt.Sprintf("%d links, %d sources, %d targets", stats.TotalLinks, stats.TotalSources, stats.TotalTargets)
		case "rebuild":
			if rebuildErr := linkMgr.Rebuild(); rebuildErr != nil {
				s.writeError(w, http.StatusInternalServerError, "rebuild failed: "+rebuildErr.Error())

				return
			}
			stats := linkMgr.GetStats()
			message = fmt.Sprintf("Index rebuilt: %d links", stats.TotalLinks)
		default:
			// Treat as entity ID
			outgoing := linkMgr.GetOutgoing(subcommand)
			incoming := linkMgr.GetIncoming(subcommand)
			message = fmt.Sprintf("%d outgoing, %d incoming links for %s", len(outgoing), len(incoming), subcommand)
		}

	case "question":
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "question requires a message")

			return
		}
		question := strings.Join(req.Args, " ")
		err = cond.AskQuestion(opCtx, question)
		message = "Question sent to agent"

	case "answer":
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "answer requires a response")

			return
		}
		task := cond.GetActiveTask()
		if task == nil {
			s.writeError(w, http.StatusBadRequest, "no active task")

			return
		}
		ws := cond.GetWorkspace()
		// Clear pending question
		if clearErr := ws.ClearPendingQuestion(task.ID); clearErr != nil {
			slog.Warn("clear pending question", "error", clearErr)
		}
		// Save answer as note
		response := strings.Join(req.Args, " ")
		if noteErr := ws.AppendNote(task.ID, response, task.State); noteErr != nil {
			s.writeError(w, http.StatusInternalServerError, "save answer: "+noteErr.Error())

			return
		}
		message = "Answer saved"
		// Resume workflow based on state
		state := workflow.State(task.State)
		switch state {
		case workflow.StatePlanning:
			go func(ctx context.Context) { _ = cond.Plan(ctx) }(opCtx)
		case workflow.StateImplementing:
			go func(ctx context.Context) { _ = cond.Implement(ctx) }(opCtx)
		case workflow.StateReviewing:
			go func(ctx context.Context) { _ = cond.Review(ctx) }(opCtx)
		case workflow.StateIdle, workflow.StateDone, workflow.StateFailed,
			workflow.StateWaiting, workflow.StatePaused,
			workflow.StateCheckpointing, workflow.StateReverting, workflow.StateRestoring:
			// These states are not resumable - do nothing
		}

	case "delete":
		// Delete a queue task: delete <queue>/<task-id>
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "delete requires a task reference (e.g., quick-tasks/task-1)")

			return
		}
		queueID, taskID, parseErr := conductor.ParseQueueTaskRef(req.Args[0])
		if parseErr != nil {
			s.writeError(w, http.StatusBadRequest, parseErr.Error())

			return
		}
		ws := cond.GetWorkspace()
		queue, loadErr := storage.LoadTaskQueue(ws, queueID)
		if loadErr != nil {
			s.writeError(w, http.StatusNotFound, "queue not found: "+queueID)

			return
		}
		if !queue.RemoveTask(taskID) {
			s.writeError(w, http.StatusNotFound, fmt.Sprintf("task not found: %s/%s", queueID, taskID))

			return
		}
		if saveErr := queue.Save(); saveErr != nil {
			s.writeError(w, http.StatusInternalServerError, "save queue: "+saveErr.Error())

			return
		}
		// Delete notes file
		notesPath := ws.QueueNotePath(queueID, taskID)
		_ = ws.DeleteFile(notesPath)
		message = fmt.Sprintf("Deleted task %s from %s", taskID, queueID)

	case "export":
		// Export a queue task to markdown: export <queue>/<task-id>
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "export requires a task reference (e.g., quick-tasks/task-1)")

			return
		}
		queueID, taskID, parseErr := conductor.ParseQueueTaskRef(req.Args[0])
		if parseErr != nil {
			s.writeError(w, http.StatusBadRequest, parseErr.Error())

			return
		}
		markdown, exportErr := cond.ExportQueueTask(queueID, taskID)
		if exportErr != nil {
			s.writeError(w, http.StatusInternalServerError, "export task: "+exportErr.Error())

			return
		}
		// Return the full markdown content
		s.writeJSON(w, http.StatusOK, map[string]any{
			"success":  true,
			"message":  fmt.Sprintf("Exported %s/%s", queueID, taskID),
			"markdown": markdown,
		})

		return

	case "optimize":
		// AI optimize a queue task: optimize <queue>/<task-id>
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "optimize requires a task reference (e.g., quick-tasks/task-1)")

			return
		}
		queueID, taskID, parseErr := conductor.ParseQueueTaskRef(req.Args[0])
		if parseErr != nil {
			s.writeError(w, http.StatusBadRequest, parseErr.Error())

			return
		}
		optimized, optimizeErr := cond.OptimizeQueueTask(opCtx, queueID, taskID)
		if optimizeErr != nil {
			s.writeError(w, http.StatusInternalServerError, "optimize task: "+optimizeErr.Error())

			return
		}
		s.writeJSON(w, http.StatusOK, map[string]any{
			"success":           true,
			"message":           "Task optimized",
			"original_title":    optimized.OriginalTitle,
			"optimized_title":   optimized.OptimizedTitle,
			"added_labels":      optimized.AddedLabels,
			"improvement_notes": optimized.ImprovementNotes,
		})

		return

	case "submit":
		// Submit a queue task to provider: submit <queue>/<task-id> <provider>
		if len(req.Args) < 2 {
			s.writeError(w, http.StatusBadRequest, "submit requires: submit <queue>/<task-id> <provider>")

			return
		}
		queueID, taskID, parseErr := conductor.ParseQueueTaskRef(req.Args[0])
		if parseErr != nil {
			s.writeError(w, http.StatusBadRequest, parseErr.Error())

			return
		}
		providerName := req.Args[1]
		submitResult, submitErr := cond.SubmitQueueTask(opCtx, queueID, taskID, conductor.SubmitOptions{
			Provider: providerName,
			TaskIDs:  []string{taskID},
		})
		if submitErr != nil {
			s.writeError(w, http.StatusInternalServerError, "submit task: "+submitErr.Error())

			return
		}
		if len(submitResult.Tasks) == 0 {
			s.writeError(w, http.StatusInternalServerError, "no tasks submitted")

			return
		}
		submittedTask := submitResult.Tasks[0]
		s.writeJSON(w, http.StatusOK, map[string]any{
			"success":     true,
			"message":     "Submitted to " + providerName,
			"external_id": submittedTask.ExternalID,
			"url":         submittedTask.ExternalURL,
		})

		return

	case "sync":
		// Sync task from provider: sync <task-id>
		task := cond.GetActiveTask()
		if task == nil {
			s.writeError(w, http.StatusBadRequest, "no active task to sync")

			return
		}
		// For now, indicate that sync is available - full implementation would require
		// provider fetch and delta spec generation
		message = fmt.Sprintf("Sync requested for task %s. Use CLI for full provider sync.", task.ID)

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
