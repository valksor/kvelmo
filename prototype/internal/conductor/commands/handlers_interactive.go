package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

func init() {
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

	switch state { //nolint:exhaustive // Only planning/implementing/reviewing can be resumed
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
	default:
		return nil, fmt.Errorf("%w: cannot resume from state %s", ErrBadRequest, state)
	}

	return NewResult("Answer sent, resuming..."), nil
}
