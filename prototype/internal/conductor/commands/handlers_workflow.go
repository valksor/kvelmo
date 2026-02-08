package commands

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

type startOptions struct {
	Ref      string `json:"ref"`
	Template string `json:"template,omitempty"`
	NoBranch bool   `json:"no_branch,omitempty"`
}

type implementOptions struct {
	Component string `json:"component,omitempty"`
	Parallel  string `json:"parallel,omitempty"`
}

type continueOptions struct {
	Auto bool `json:"auto,omitempty"`
}

func init() {
	// Register workflow commands
	Register(Command{
		Info: CommandInfo{
			Name:        "start",
			Description: "Start a new task from a reference",
			Category:    "workflow",
			Args: []CommandArg{
				{Name: "reference", Required: true, Description: "Task reference (e.g., github:123, file:task.md)"},
			},
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleStart,
	})

	Register(Command{
		Info: CommandInfo{
			Name:        "plan",
			Aliases:     []string{"pl"},
			Description: "Enter planning phase",
			Category:    "workflow",
			Args: []CommandArg{
				{Name: "prompt", Required: false, Description: "Optional planning prompt"},
			},
			RequiresTask: true,
			MutatesState: true,
		},
		Handler: handlePlan,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "implement",
			Aliases:      []string{"impl", "do"},
			Description:  "Execute specifications",
			Category:     "workflow",
			RequiresTask: true,
			MutatesState: true,
			Subcommands:  []string{"review"},
		},
		Handler: handleImplement,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "review",
			Aliases:      []string{"rv"},
			Description:  "Run code review",
			Category:     "workflow",
			RequiresTask: true,
			MutatesState: true,
			Subcommands:  []string{"view"},
		},
		Handler: handleReview,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "continue",
			Aliases:      []string{"cont"},
			Description:  "Resume from waiting/paused state",
			Category:     "workflow",
			RequiresTask: true,
			MutatesState: true,
		},
		Handler: handleContinue,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "finish",
			Description:  "Complete the current task",
			Category:     "workflow",
			RequiresTask: true,
			MutatesState: true,
		},
		Handler: handleFinish,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "abandon",
			Description:  "Discard the current task",
			Category:     "workflow",
			RequiresTask: true,
			MutatesState: true,
		},
		Handler: handleAbandon,
	})
}

func handleStart(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	opts, err := DecodeOptions[startOptions](inv)
	if err != nil {
		return nil, err
	}

	ref := strings.TrimSpace(opts.Ref)
	if ref == "" && len(inv.Args) > 0 {
		ref = strings.Join(inv.Args, " ")
	}
	if ref == "" {
		return nil, errors.New("start requires a reference (e.g., start github:123)")
	}

	if conflict := cond.CheckActiveTaskConflict(ctx); conflict != nil {
		return &Result{
			Type:    ResultConflict,
			Message: "Another task is already active. Use worktree mode for parallel tasks, or finish/abandon current task first.",
			Data: map[string]any{
				"conflict_type": "active_task",
				"active_task": map[string]any{
					"id":             conflict.ActiveTaskID,
					"title":          conflict.ActiveTaskTitle,
					"branch":         conflict.ActiveBranch,
					"using_worktree": conflict.UsingWorktree,
				},
			},
		}, nil
	}

	if err := cond.Start(ctx, ref); err != nil {
		return nil, fmt.Errorf("failed to start task: %w", err)
	}

	task := cond.GetActiveTask()
	if task == nil {
		return NewResult("Task started"), nil
	}

	return NewResult("Started task: " + task.ID).
		WithState(task.State).
		WithTaskID(task.ID), nil
}

func handlePlan(ctx context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	// Note: prompt argument currently ignored in conductor.Plan()
	if err := cond.Plan(ctx); err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	result := NewResult("Planning phase started").WithState(string(workflow.StatePlanning))
	result.Executor = cond.RunPlanning

	return result, nil
}

func handleImplement(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	args := inv.Args

	// Handle "implement review <n>" subcommand
	if len(args) > 0 && args[0] == "review" {
		if len(args) < 2 {
			return nil, errors.New("usage: implement review <number>")
		}
		reviewNum, err := strconv.Atoi(args[1])
		if err != nil {
			return nil, errors.New("review number must be an integer")
		}
		if reviewNum <= 0 {
			return nil, fmt.Errorf("review number must be positive, got %d", reviewNum)
		}

		return handleImplementReview(ctx, cond, reviewNum)
	}

	opts, err := DecodeOptions[implementOptions](inv)
	if err != nil {
		return nil, err
	}

	hasImplementationOptions := opts.Component != "" || opts.Parallel != ""
	if hasImplementationOptions {
		cond.SetImplementationOptions(opts.Component, opts.Parallel)
	}

	if err := cond.Implement(ctx); err != nil {
		if hasImplementationOptions {
			cond.ClearImplementationOptions()
		}

		return nil, fmt.Errorf("implementation failed: %w", err)
	}

	result := NewResult("Implementation phase started").WithState(string(workflow.StateImplementing))
	result.Executor = func(execCtx context.Context) error {
		if hasImplementationOptions {
			defer cond.ClearImplementationOptions()
		}

		return cond.RunImplementation(execCtx)
	}

	return result, nil
}

func handleImplementReview(ctx context.Context, cond *conductor.Conductor, reviewNum int) (*Result, error) {
	if err := cond.ImplementReview(ctx, reviewNum); err != nil {
		return nil, fmt.Errorf("implement review failed: %w", err)
	}

	result := NewResult(fmt.Sprintf("Implementing fixes for review #%d", reviewNum)).
		WithState(string(workflow.StateImplementing))
	result.Executor = func(execCtx context.Context) error {
		return cond.RunReviewImplementation(execCtx, reviewNum)
	}

	return result, nil
}

func handleReview(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	args := inv.Args

	// Handle "review <n>" or "review view <n>" for viewing reviews
	if len(args) > 0 {
		// Check if first arg is a number (view that review)
		if num, err := strconv.Atoi(args[0]); err == nil {
			return handleReviewView(ctx, cond, num)
		}
		// "review view <n>" subcommand
		if args[0] == "view" && len(args) > 1 {
			if num, err := strconv.Atoi(args[1]); err == nil {
				return handleReviewView(ctx, cond, num)
			}

			return nil, errors.New("review view requires a number")
		}
	}

	// Run the review workflow
	if err := cond.Review(ctx); err != nil {
		return nil, fmt.Errorf("review failed: %w", err)
	}

	result := NewResult("Code review started").WithState(string(workflow.StateReviewing))
	result.Executor = cond.RunReview

	return result, nil
}

func handleReviewView(_ context.Context, cond *conductor.Conductor, num int) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("no workspace available")
	}

	task := cond.GetActiveTask()
	if task == nil {
		return nil, ErrNoActiveTask
	}

	review, err := ws.LoadReview(task.ID, num)
	if err != nil {
		return nil, fmt.Errorf("failed to load review #%d: %w", num, err)
	}

	// Return the review content
	return NewResult(fmt.Sprintf("Review #%d", num)).WithData(review), nil
}

func handleContinue(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	opts, err := DecodeOptions[continueOptions](inv)
	if err != nil {
		return nil, err
	}

	task := cond.GetActiveTask()
	if task == nil {
		return nil, ErrNoActiveTask
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}
	question, err := ws.LoadPendingQuestion(task.ID)
	if err == nil && question != nil {
		return nil, errors.New("agent has a pending question - use 'answer <response>'")
	}

	status, err := cond.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("continue failed: %w", err)
	}

	state := workflow.State(status.State)
	nextActions := continueNextActionsForState(state, status.Specifications)

	if opts.Auto {
		action, actionErr := continueExecuteNextStep(ctx, cond, status)
		if actionErr != nil {
			return nil, fmt.Errorf("continue auto failed: %w", actionErr)
		}

		updatedStatus, statusErr := cond.Status(ctx)
		if statusErr == nil && updatedStatus != nil {
			state = workflow.State(updatedStatus.State)
			nextActions = continueNextActionsForState(state, updatedStatus.Specifications)
		}

		return NewResult("auto-executed: " + action).
			WithState(string(state)).
			WithData(map[string]any{
				"action":       action,
				"next_actions": nextActions,
			}), nil
	}

	if err := cond.ResumePaused(ctx); err != nil {
		return nil, fmt.Errorf("continue failed: %w", err)
	}

	return NewResult("task resumed").
		WithState(status.State).
		WithData(map[string]any{
			"next_actions": nextActions,
		}), nil
}

func handleFinish(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	opts, err := DecodeOptions[conductor.FinishOptions](inv)
	if err != nil {
		return nil, err
	}

	// Capture task ID before Finish() clears c.activeTask.
	taskID := cond.GetTaskID()

	if err := cond.Finish(ctx, opts); err != nil {
		return nil, fmt.Errorf("finish failed: %w", err)
	}

	// Return StateIdle (not StateDone) because the task is fully cleared.
	// Setting TaskID ensures publishStateChangeEvent fires in handleViaRouter.
	return NewResult("Task completed").
		WithState(string(workflow.StateIdle)).
		WithTaskID(taskID), nil
}

func handleAbandon(ctx context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	// Capture task ID before Delete() clears c.activeTask.
	taskID := cond.GetTaskID()

	opts := conductor.DeleteOptions{
		Force:      true,
		KeepBranch: false,
		DeleteWork: conductor.BoolPtr(true),
	}

	if err := cond.Delete(ctx, opts); err != nil {
		return nil, fmt.Errorf("abandon failed: %w", err)
	}

	// Setting TaskID ensures publishStateChangeEvent fires in handleViaRouter.
	return NewResult("Task abandoned").
		WithState(string(workflow.StateIdle)).
		WithTaskID(taskID), nil
}

func continueExecuteNextStep(ctx context.Context, cond *conductor.Conductor, status *conductor.TaskStatus) (string, error) {
	switch workflow.State(status.State) {
	case workflow.StateIdle:
		if status.Specifications == 0 {
			if err := cond.Plan(ctx); err != nil {
				return "", err
			}

			return "plan", nil
		}
		if err := cond.Implement(ctx); err != nil {
			return "", err
		}

		return "implement", nil
	case workflow.StatePlanning:
		if err := cond.Implement(ctx); err != nil {
			return "", err
		}

		return "implement", nil
	case workflow.StateImplementing, workflow.StateReviewing:
		return "none", nil
	case workflow.StateDone:
		return "none", nil
	case workflow.StateFailed, workflow.StateWaiting, workflow.StatePaused:
		return "none", nil
	case workflow.StateCheckpointing, workflow.StateReverting, workflow.StateRestoring:
		return "none", nil
	}

	return "none", nil
}

func continueNextActionsForState(state workflow.State, specifications int) []string {
	switch state {
	case workflow.StateIdle:
		if specifications == 0 {
			return []string{
				"POST /api/v1/workflow/plan",
				"POST /api/v1/tasks/{id}/notes",
			}
		}

		return []string{
			"POST /api/v1/workflow/implement",
			"POST /api/v1/workflow/plan",
			"POST /api/v1/tasks/{id}/notes",
		}
	case workflow.StatePlanning:
		return []string{
			"POST /api/v1/workflow/implement",
			"POST /api/v1/workflow/question",
			"POST /api/v1/tasks/{id}/notes",
		}
	case workflow.StateImplementing:
		return []string{
			"POST /api/v1/workflow/implement",
			"POST /api/v1/workflow/question",
			"POST /api/v1/workflow/undo",
			"POST /api/v1/workflow/finish",
			"POST /api/v1/tasks/{id}/notes",
		}
	case workflow.StateReviewing:
		return []string{
			"POST /api/v1/workflow/finish",
			"POST /api/v1/workflow/implement",
			"POST /api/v1/workflow/question",
		}
	case workflow.StateFailed:
		return []string{
			"GET /api/v1/task",
			"POST /api/v1/workflow/implement",
			"POST /api/v1/tasks/{id}/notes",
		}
	case workflow.StateWaiting:
		return []string{
			"POST /api/v1/workflow/answer",
		}
	case workflow.StatePaused:
		return []string{
			"POST /api/v1/workflow/resume",
			"GET /api/v1/costs",
		}
	case workflow.StateDone:
		return []string{
			"POST /api/v1/workflow/start",
		}
	case workflow.StateCheckpointing, workflow.StateReverting, workflow.StateRestoring:
		return []string{
			"GET /api/v1/task",
		}
	}

	return []string{
		"GET /api/v1/task",
		"POST /api/v1/tasks/{id}/notes",
	}
}
