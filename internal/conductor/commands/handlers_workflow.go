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
		},
		Handler: handleContinue,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "finish",
			Description:  "Complete the current task",
			Category:     "workflow",
			RequiresTask: true,
		},
		Handler: handleFinish,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "abandon",
			Description:  "Discard the current task",
			Category:     "workflow",
			RequiresTask: true,
		},
		Handler: handleAbandon,
	})
}

func handleStart(ctx context.Context, cond *conductor.Conductor, args []string) (*Result, error) {
	if len(args) == 0 {
		return nil, errors.New("start requires a reference (e.g., start github:123)")
	}

	ref := strings.Join(args, " ")
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

func handlePlan(ctx context.Context, cond *conductor.Conductor, _ []string) (*Result, error) {
	// Note: prompt argument currently ignored in conductor.Plan()
	if err := cond.Plan(ctx); err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	return NewResult("Planning phase started").WithState(string(workflow.StatePlanning)), nil
}

func handleImplement(ctx context.Context, cond *conductor.Conductor, args []string) (*Result, error) {
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

	if err := cond.Implement(ctx); err != nil {
		return nil, fmt.Errorf("implementation failed: %w", err)
	}

	return NewResult("Implementation phase started").WithState(string(workflow.StateImplementing)), nil
}

func handleImplementReview(ctx context.Context, cond *conductor.Conductor, reviewNum int) (*Result, error) {
	if err := cond.ImplementReview(ctx, reviewNum); err != nil {
		return nil, fmt.Errorf("implement review failed: %w", err)
	}

	return NewResult(fmt.Sprintf("Implementing fixes for review #%d", reviewNum)).
		WithState(string(workflow.StateImplementing)), nil
}

func handleReview(ctx context.Context, cond *conductor.Conductor, args []string) (*Result, error) {
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

	return NewResult("Code review started").WithState(string(workflow.StateReviewing)), nil
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

func handleContinue(ctx context.Context, cond *conductor.Conductor, _ []string) (*Result, error) {
	// Check if there's a pending question
	task := cond.GetActiveTask()
	if task == nil {
		return nil, ErrNoActiveTask
	}

	ws := cond.GetWorkspace()
	question, err := ws.LoadPendingQuestion(task.ID)
	if err == nil && question != nil {
		return nil, errors.New("agent has a pending question - use 'answer <response>'")
	}

	// Resume workflow
	if err := cond.ResumePaused(ctx); err != nil {
		return nil, fmt.Errorf("continue failed: %w", err)
	}

	return NewResult("Resumed workflow"), nil
}

func handleFinish(ctx context.Context, cond *conductor.Conductor, _ []string) (*Result, error) {
	opts := conductor.FinishOptions{}
	if err := cond.Finish(ctx, opts); err != nil {
		return nil, fmt.Errorf("finish failed: %w", err)
	}

	return NewResult("Task completed").WithState(string(workflow.StateDone)), nil
}

func handleAbandon(ctx context.Context, cond *conductor.Conductor, _ []string) (*Result, error) {
	opts := conductor.DeleteOptions{
		Force:      true,
		KeepBranch: false,
		DeleteWork: conductor.BoolPtr(true),
	}

	if err := cond.Delete(ctx, opts); err != nil {
		return nil, fmt.Errorf("abandon failed: %w", err)
	}

	return NewResult("Task abandoned").WithState(string(workflow.StateIdle)), nil
}
