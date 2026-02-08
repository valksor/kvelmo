package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "chat",
			Aliases:      []string{"c", "ask"},
			Description:  "Chat with the AI agent",
			Category:     "interactive",
			RequiresTask: false,
		},
		Handler: handleChat,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "interactive-state",
			Description:  "Get the current interactive session state",
			Category:     "interactive",
			RequiresTask: false,
		},
		Handler: handleInteractiveState,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "interactive-commands",
			Description:  "List available commands for discovery",
			Category:     "interactive",
			RequiresTask: false,
		},
		Handler: handleInteractiveCommands,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "interactive-answer",
			Description:  "Respond to a pending question in interactive mode",
			Category:     "interactive",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleInteractiveAnswer,
	})
}

// handleInteractiveState returns the current task state for the interactive session.
func handleInteractiveState(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	response := map[string]any{
		"success": true,
	}

	if cond == nil {
		return NewResult("State loaded").WithData(response), nil
	}

	task := cond.GetActiveTask()
	if task != nil {
		response["state"] = task.State
		response["task_id"] = task.ID
		if work := cond.GetTaskWork(); work != nil {
			response["title"] = work.Metadata.Title
		}
	}

	return NewResult("State loaded").WithData(response), nil
}

// handleInteractiveCommands returns all registered commands for IDE/UI discovery.
func handleInteractiveCommands(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
	cmds := Metadata()

	return NewResult("Commands loaded").WithData(map[string]any{
		"commands": cmds,
	}), nil
}

// handleInteractiveAnswer clears the pending question and resumes the workflow.
func handleInteractiveAnswer(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	response := GetString(inv.Options, "response")
	if response == "" {
		return nil, fmt.Errorf("%w: response is required", ErrBadRequest)
	}

	task := cond.GetActiveTask()
	if task == nil {
		return nil, errors.New("no active task")
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	if err := ws.ClearPendingQuestion(task.ID); err != nil {
		return nil, fmt.Errorf("clear pending question: %w", err)
	}

	if err := ws.AppendNote(task.ID, task.State, response); err != nil {
		return nil, fmt.Errorf("save answer: %w", err)
	}

	state := workflow.State(task.State)

	switch state {
	case workflow.StatePlanning:
		if err := cond.Plan(ctx); err != nil {
			return nil, fmt.Errorf("resume planning: %w", err)
		}
	case workflow.StateImplementing:
		if err := cond.Implement(ctx); err != nil {
			return nil, fmt.Errorf("resume implementing: %w", err)
		}
	case workflow.StateReviewing:
		if err := cond.Review(ctx); err != nil {
			return nil, fmt.Errorf("resume reviewing: %w", err)
		}
	case workflow.StateIdle:
		return nil, fmt.Errorf("%w: task not started, use 'plan' to begin", ErrBadRequest)
	case workflow.StateDone:
		return nil, fmt.Errorf("%w: task already completed", ErrBadRequest)
	case workflow.StateFailed:
		return nil, fmt.Errorf("%w: task failed, use 'reset' to recover", ErrBadRequest)
	case workflow.StateWaiting:
		return nil, fmt.Errorf("%w: task already waiting for answer", ErrBadRequest)
	case workflow.StatePaused:
		return nil, fmt.Errorf("%w: task paused due to budget limits", ErrBadRequest)
	case workflow.StateCheckpointing, workflow.StateReverting, workflow.StateRestoring:
		return nil, fmt.Errorf("%w: operation in progress, please wait", ErrBadRequest)
	}

	return NewResult("Answer sent, resuming..."), nil
}

// handleChat sends a chat message to the agent.
// Supports optional streaming via inv.StreamCB.
func handleChat(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	message := strings.Join(inv.Args, " ")
	if message == "" {
		return nil, fmt.Errorf("%w: message cannot be empty", ErrBadRequest)
	}

	activeAgent := cond.GetActiveAgent()
	if activeAgent == nil {
		return nil, fmt.Errorf("%w: no agent available", ErrBadRequest)
	}

	// Build prompt with context
	prompt := buildChatPrompt(cond, message)

	var response *agent.Response
	var err error

	if inv.StreamCB != nil {
		// Use streaming callback if provided
		response, err = activeAgent.RunWithCallback(ctx, prompt, inv.StreamCB)
	} else {
		// Fall back to non-streaming for API/Web
		response, err = activeAgent.Run(ctx, prompt)
	}

	if err != nil {
		return nil, fmt.Errorf("agent error: %w", err)
	}

	// Build response data
	data := map[string]any{
		"success": true,
	}

	if response != nil {
		if response.Summary != "" {
			data["response"] = response.Summary
		}
		// If the agent asked a question, save it and return a question result
		if response.Question != nil {
			// Save pending question to storage
			if task := cond.GetActiveTask(); task != nil {
				if ws := cond.GetWorkspace(); ws != nil {
					pq := &storage.PendingQuestion{
						Question: response.Question.Text,
						Phase:    "chat",
						AskedAt:  time.Now(),
					}
					for _, opt := range response.Question.Options {
						pq.Options = append(pq.Options, storage.QuestionOption{
							Label:       opt.Label,
							Description: opt.Description,
						})
					}
					if err := ws.SavePendingQuestion(task.ID, pq); err != nil {
						slog.Debug("failed to save pending question", "task_id", task.ID, "error", err)
					}
				}
			}

			return &Result{
				Type:    ResultQuestion,
				Message: response.Question.Text,
				Data: WaitingData{
					Question: response.Question.Text,
					Options:  convertQuestionOptions(response.Question.Options),
					Phase:    "chat",
				},
			}, nil
		}
	}

	return NewChatResult("Chat response", data), nil
}

// convertQuestionOptions converts agent options to router options.
func convertQuestionOptions(opts []agent.QuestionOption) []QuestionOption {
	result := make([]QuestionOption, len(opts))
	for i, opt := range opts {
		result[i] = QuestionOption{
			Label:       opt.Label,
			Value:       opt.Value,
			Description: opt.Description,
		}
	}

	return result
}

// buildChatPrompt creates a prompt with task context.
func buildChatPrompt(cond *conductor.Conductor, message string) string {
	var builder strings.Builder

	builder.WriteString("You are an AI assistant helping with a software development task.\n\n")

	// Add current task context if available
	task := cond.GetActiveTask()
	if task != nil {
		if work := cond.GetTaskWork(); work != nil {
			builder.WriteString(fmt.Sprintf("Task: %s\n", work.Metadata.Title))
			builder.WriteString(fmt.Sprintf("Current State: %s\n\n", task.State))
		}
	}

	builder.WriteString("User message: ")
	builder.WriteString(message)

	return builder.String()
}
