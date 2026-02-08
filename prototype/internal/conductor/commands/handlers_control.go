package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

type autoOptions struct {
	Ref           string `json:"ref,omitempty"`
	MaxRetries    int    `json:"max_retries,omitempty"`
	NoPush        bool   `json:"no_push,omitempty"`
	NoDelete      bool   `json:"no_delete,omitempty"`
	NoSquash      bool   `json:"no_squash,omitempty"`
	TargetBranch  string `json:"target_branch,omitempty"`
	QualityTarget string `json:"quality_target,omitempty"`
	NoQuality     bool   `json:"no_quality,omitempty"`
}

type simplifyOptions struct {
	NoCheckpoint bool   `json:"no_checkpoint,omitempty"`
	Agent        string `json:"agent,omitempty"`
}

type quickOptions struct {
	QueueID  string   `json:"queue_id,omitempty"`
	Title    string   `json:"title,omitempty"`
	Priority int      `json:"priority,omitempty"`
	Labels   []string `json:"labels,omitempty"`
}

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "undo",
			Description:  "Undo to previous checkpoint",
			Category:     "control",
			RequiresTask: true,
			MutatesState: true,
		},
		Handler: handleUndo,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "redo",
			Description:  "Redo to next checkpoint",
			Category:     "control",
			RequiresTask: true,
			MutatesState: true,
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
			MutatesState: true,
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
			RequiresTask: false,
			MutatesState: true,
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
			MutatesState: true,
		},
		Handler: handleQuick,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "reset",
			Description:  "Reset workflow state to idle",
			Category:     "control",
			RequiresTask: true,
			MutatesState: true,
		},
		Handler: handleReset,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "auto",
			Description:  "Run auto mode or auto-execute next workflow step",
			Category:     "workflow",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleAuto,
	})

	Register(Command{
		Info: CommandInfo{
			Name:        "question",
			Description: "Ask the agent a question",
			Category:    "chat",
			Args: []CommandArg{
				{Name: "question", Required: true, Description: "Question for the agent"},
			},
			RequiresTask: true,
			MutatesState: true,
		},
		Handler: handleQuestion,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "simplify",
			Description:  "Simplify task input/specifications based on current context",
			Category:     "workflow",
			RequiresTask: true,
			MutatesState: true,
		},
		Handler: handleSimplify,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "sync",
			Description:  "Sync active task with provider source",
			Category:     "workflow",
			RequiresTask: true,
			MutatesState: true,
		},
		Handler: handleSync,
	})
}

func handleUndo(ctx context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	if err := cond.Undo(ctx); err != nil {
		return nil, fmt.Errorf("undo failed: %w", err)
	}

	return NewResult("Undone to previous checkpoint"), nil
}

func handleRedo(ctx context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	if err := cond.Redo(ctx); err != nil {
		return nil, fmt.Errorf("redo failed: %w", err)
	}

	return NewResult("Redone to next checkpoint"), nil
}

func handleExit(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
	return ExitResult, nil
}

func handleHelp(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
	return NewHelpResult(Metadata()), nil
}

func handleClear(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
	// Clear is handled by the client (CLI prints escape codes, web clears console)
	return &Result{Type: ResultMessage, Message: "clear"}, nil
}

func handleAnswer(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	args := inv.Args
	if len(args) == 0 {
		return nil, errors.New("answer requires a response")
	}

	response := strings.Join(args, " ")

	// Load pending question BEFORE answering to check if it's a finish-phase question.
	// AnswerQuestion clears the question, so we must capture the phase first.
	var questionPhase string
	task := cond.GetActiveTask()
	ws := cond.GetWorkspace()
	if task != nil && ws != nil {
		if q, err := ws.LoadPendingQuestion(task.ID); err == nil && q != nil {
			questionPhase = q.Phase
		}
	}

	if err := cond.AnswerQuestion(ctx, response); err != nil {
		return nil, fmt.Errorf("failed to submit answer: %w", err)
	}

	// Auto-continue: if the question was from the finish flow, complete the finish.
	if questionPhase == "finishing" {
		opts := conductor.FinishOptions{FinishAction: response}
		if err := cond.Finish(ctx, opts); err != nil {
			return nil, fmt.Errorf("finish after answer: %w", err)
		}

		return NewResult("Task completed").WithState(string(workflow.StateDone)), nil
	}

	return NewResult("Answer submitted"), nil
}

func handleNote(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	args := inv.Args
	if len(args) == 0 {
		return nil, errors.New("note requires a message")
	}

	message := strings.Join(args, " ")
	task := cond.GetActiveTask()
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}
	taskID := GetString(inv.Options, "task_id")
	if taskID == "" && task != nil {
		taskID = task.ID
	}
	if taskID == "" {
		return nil, ErrNoActiveTask
	}

	// If this is the active task and a pending question exists, treat the note as an answer.
	if task != nil && task.ID == taskID && ws.HasPendingQuestion(taskID) {
		if err := cond.AnswerQuestion(ctx, message); err != nil {
			return nil, fmt.Errorf("failed to submit answer: %w", err)
		}

		return NewResult("Answer submitted").WithData(map[string]any{"was_answer": true}), nil
	}

	// Non-active task notes are still allowed for API and queue workflows.
	if task == nil || task.ID != taskID {
		if err := ws.AppendNote(taskID, message, "note"); err != nil {
			return nil, fmt.Errorf("failed to add note: %w", err)
		}

		return NewResult("Note added").WithData(map[string]any{"was_answer": false}), nil
	}

	if err := cond.AddNote(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to add note: %w", err)
	}

	return NewResult("Note added").WithData(map[string]any{"was_answer": false}), nil
}

func handleQuick(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	args := inv.Args
	if len(args) == 0 {
		return nil, errors.New("quick requires a description")
	}

	opts, err := DecodeOptions[quickOptions](inv)
	if err != nil {
		return nil, err
	}

	description := strings.Join(args, " ")

	result, err := cond.CreateQuickTask(ctx, conductor.QuickTaskOptions{
		Description: description,
		Title:       strings.TrimSpace(opts.Title),
		Priority:    opts.Priority,
		Labels:      append([]string{}, opts.Labels...),
		QueueID:     strings.TrimSpace(opts.QueueID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create quick task: %w", err)
	}

	return NewResult(fmt.Sprintf("Quick task created: %s (queue: %s)", result.TaskID, result.QueueID)).WithData(map[string]any{
		"queue_id":   result.QueueID,
		"task_id":    result.TaskID,
		"title":      result.Title,
		"created_at": result.CreatedAt,
	}), nil
}

func handleReset(ctx context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	if err := cond.ResetState(ctx); err != nil {
		return nil, fmt.Errorf("reset state: %w", err)
	}

	return NewResult("Workflow reset to idle").WithState(string(workflow.StateIdle)), nil
}

func handleAuto(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	opts, err := DecodeOptions[autoOptions](inv)
	if err != nil {
		return nil, err
	}

	ref := strings.TrimSpace(opts.Ref)
	if ref == "" && len(inv.Args) > 0 {
		ref = strings.TrimSpace(inv.Args[0])
	}

	// Full auto cycle mode: `auto <ref>`.
	if ref != "" {
		if cond.GetActiveTask() != nil {
			return &Result{
				Type:    ResultConflict,
				Message: "task already active; use abandon first or status for details",
			}, nil
		}

		maxRetries := opts.MaxRetries
		if maxRetries == 0 {
			maxRetries = 3
		}

		qualityTarget := strings.TrimSpace(opts.QualityTarget)
		if qualityTarget == "" {
			qualityTarget = "quality"
		}

		if opts.NoQuality {
			maxRetries = 0
		}

		autoResult, runErr := cond.RunAuto(ctx, ref, conductor.AutoOptions{
			QualityTarget: qualityTarget,
			MaxRetries:    maxRetries,
			SquashMerge:   !opts.NoSquash,
			DeleteBranch:  !opts.NoDelete,
			TargetBranch:  strings.TrimSpace(opts.TargetBranch),
			Push:          !opts.NoPush,
		})
		if runErr != nil {
			return nil, fmt.Errorf("auto run failed: %w", runErr)
		}

		return NewResult("Auto run completed").WithData(map[string]any{
			"planning_done":    autoResult.PlanningDone,
			"implement_done":   autoResult.ImplementDone,
			"quality_attempts": autoResult.QualityAttempts,
			"quality_passed":   autoResult.QualityPassed,
			"finish_done":      autoResult.FinishDone,
		}), nil
	}

	// Interactive mode: execute the next logical step for the active task.
	if cond.GetActiveTask() == nil {
		return nil, ErrNoActiveTask
	}

	return handleContinue(ctx, cond, Invocation{
		Source: inv.Source,
		Options: map[string]any{
			"auto": true,
		},
	})
}

func handleQuestion(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	if len(inv.Args) == 0 {
		return nil, errors.New("question requires a message")
	}

	question := strings.Join(inv.Args, " ")
	if err := cond.AskQuestion(ctx, question); err != nil {
		return nil, fmt.Errorf("failed to ask question: %w", err)
	}

	return NewResult("Question sent to agent"), nil
}

func handleSimplify(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	opts, err := DecodeOptions[simplifyOptions](inv)
	if err != nil {
		return nil, err
	}

	if err := cond.Simplify(ctx, opts.Agent, !opts.NoCheckpoint); err != nil {
		return nil, fmt.Errorf("simplification failed: %w", err)
	}

	simplified := "task_input"
	task := cond.GetActiveTask()
	if ws := cond.GetWorkspace(); ws != nil && task != nil {
		specs, listErr := ws.ListSpecifications(task.ID)
		if listErr == nil && len(specs) > 0 {
			simplified = "specifications"
		}
	}

	return NewResult("simplification complete").WithData(map[string]any{
		"success":    true,
		"simplified": simplified,
		"message":    "simplification complete",
	}), nil
}

func handleSync(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	taskID := ""
	if taskIDOpt := strings.TrimSpace(GetString(inv.Options, "task_id")); taskIDOpt != "" {
		taskID = taskIDOpt
	}
	if len(inv.Args) > 0 {
		taskID = strings.TrimSpace(inv.Args[0])
	}
	if taskID == "" {
		task := cond.GetActiveTask()
		if task == nil {
			return nil, ErrNoActiveTask
		}
		taskID = task.ID
	}

	result, err := cond.SyncTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("sync failed: %w", err)
	}
	if !result.HasChanges {
		return NewResult("no changes detected").WithData(map[string]any{
			"success":     true,
			"has_changes": false,
			"message":     "no changes detected",
		}), nil
	}

	return NewResult("changes detected and delta specification generated").WithData(map[string]any{
		"success":                true,
		"has_changes":            true,
		"changes_summary":        result.ChangesSummary,
		"spec_generated":         result.SpecGenerated,
		"source_updated":         result.SourceUpdated,
		"previous_snapshot_path": result.PreviousSnapshotPath,
		"diff_path":              result.DiffPath,
		"warnings":               result.Warnings,
		"message":                "changes detected and delta specification generated",
	}), nil
}
