package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "undo",
			Description:  "Undo to previous checkpoint",
			Category:     "control",
			RequiresTask: true,
		},
		Handler: handleUndo,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "redo",
			Description:  "Redo to next checkpoint",
			Category:     "control",
			RequiresTask: true,
		},
		Handler: handleRedo,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "exit",
			Aliases:      []string{"quit", "q"},
			Description:  "Exit interactive mode",
			Category:     "session",
			RequiresTask: false,
		},
		Handler: handleExit,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "help",
			Aliases:      []string{"?"},
			Description:  "Show available commands",
			Category:     "session",
			RequiresTask: false,
		},
		Handler: handleHelp,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "clear",
			Description:  "Clear screen",
			Category:     "session",
			RequiresTask: false,
		},
		Handler: handleClear,
	})

	Register(Command{
		Info: CommandInfo{
			Name:        "answer",
			Aliases:     []string{"a"},
			Description: "Answer agent's question",
			Category:    "chat",
			Args: []CommandArg{
				{Name: "response", Required: true, Description: "Your answer"},
			},
			RequiresTask: true,
		},
		Handler: handleAnswer,
	})

	Register(Command{
		Info: CommandInfo{
			Name:        "note",
			Description: "Add a note to the current task",
			Category:    "task",
			Args: []CommandArg{
				{Name: "message", Required: true, Description: "Note content"},
			},
			RequiresTask: true,
		},
		Handler: handleNote,
	})

	Register(Command{
		Info: CommandInfo{
			Name:        "quick",
			Description: "Create a quick task",
			Category:    "task",
			Args: []CommandArg{
				{Name: "description", Required: true, Description: "Task description"},
			},
			RequiresTask: false,
		},
		Handler: handleQuick,
	})
}

func handleUndo(ctx context.Context, cond *conductor.Conductor, _ []string) (*Result, error) {
	if err := cond.Undo(ctx); err != nil {
		return nil, fmt.Errorf("undo failed: %w", err)
	}

	return NewResult("Undone to previous checkpoint"), nil
}

func handleRedo(ctx context.Context, cond *conductor.Conductor, _ []string) (*Result, error) {
	if err := cond.Redo(ctx); err != nil {
		return nil, fmt.Errorf("redo failed: %w", err)
	}

	return NewResult("Redone to next checkpoint"), nil
}

func handleExit(_ context.Context, _ *conductor.Conductor, _ []string) (*Result, error) {
	return ExitResult, nil
}

func handleHelp(_ context.Context, _ *conductor.Conductor, _ []string) (*Result, error) {
	return NewHelpResult(Metadata()), nil
}

func handleClear(_ context.Context, _ *conductor.Conductor, _ []string) (*Result, error) {
	// Clear is handled by the client (CLI prints escape codes, web clears console)
	return &Result{Type: ResultMessage, Message: "clear"}, nil
}

func handleAnswer(ctx context.Context, cond *conductor.Conductor, args []string) (*Result, error) {
	if len(args) == 0 {
		return nil, errors.New("answer requires a response")
	}

	response := strings.Join(args, " ")
	if err := cond.AnswerQuestion(ctx, response); err != nil {
		return nil, fmt.Errorf("failed to submit answer: %w", err)
	}

	return NewResult("Answer submitted"), nil
}

func handleNote(ctx context.Context, cond *conductor.Conductor, args []string) (*Result, error) {
	if len(args) == 0 {
		return nil, errors.New("note requires a message")
	}

	message := strings.Join(args, " ")
	if err := cond.AddNote(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to add note: %w", err)
	}

	return NewResult("Note added"), nil
}

func handleQuick(ctx context.Context, cond *conductor.Conductor, args []string) (*Result, error) {
	if len(args) == 0 {
		return nil, errors.New("quick requires a description")
	}

	description := strings.Join(args, " ")

	result, err := cond.CreateQuickTask(ctx, conductor.QuickTaskOptions{
		Description: description,
		QueueID:     "quick-tasks",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create quick task: %w", err)
	}

	return NewResult(fmt.Sprintf("Quick task created: %s (queue: %s)", result.TaskID, result.QueueID)), nil
}
