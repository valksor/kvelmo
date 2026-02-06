package server

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/valksor/go-mehrhof/internal/conductor"
)

// executeInteractiveWorkflowCommand handles workflow-related interactive commands.
// Commands: status, st, start, plan, implement, review, continue, finish, abandon, undo, redo, reset, auto.
func (s *Server) executeInteractiveWorkflowCommand(ctx context.Context, command string, args []string) (string, error) {
	cond := s.config.Conductor

	switch command {
	case "status", "st":
		status, statusErr := cond.Status(ctx)
		if statusErr != nil {
			return "", statusErr
		}
		result := "State: " + status.State
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

		return result, nil

	case "start":
		if len(args) == 0 {
			return "", errors.New("start requires a reference (e.g., start github:123)")
		}
		if err := cond.Start(ctx, args[0]); err != nil {
			return "", err
		}

		return "Task started", nil

	case "plan":
		if err := cond.Plan(ctx); err != nil {
			return "", err
		}

		return "Planning started", nil

	case "implement":
		// Handle "implement review <n>" subcommand
		if len(args) > 0 && args[0] == "review" {
			if len(args) < 2 {
				return "", errors.New("usage: implement review <number>")
			}
			num, parseErr := strconv.Atoi(args[1])
			if parseErr != nil {
				return "", errors.New("review number must be an integer")
			}
			if num <= 0 {
				return "", fmt.Errorf("review number must be positive, got %d", num)
			}
			// Pre-validate review availability before changing state
			task := cond.GetActiveTask()
			if task == nil {
				return "", errors.New("no active task")
			}
			ws := cond.GetWorkspace()
			reviews, listErr := ws.ListReviews(task.ID)
			if listErr != nil {
				return "", fmt.Errorf("list reviews: %w", listErr)
			}
			if len(reviews) == 0 {
				return "", errors.New("no reviews found - run 'review' first to generate code review")
			}
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
					return "", fmt.Errorf("review %d not found - only review %d exists", num, reviews[0])
				}

				return "", fmt.Errorf("review %d not found - available reviews: %v", num, reviews)
			}
			if implErr := cond.ImplementReview(ctx, num); implErr != nil {
				return "", implErr
			}
			if runErr := cond.RunReviewImplementation(ctx, num); runErr != nil {
				return "", runErr
			}

			return fmt.Sprintf("Review %d fixes applied", num), nil
		}
		if err := cond.Implement(ctx); err != nil {
			return "", err
		}

		return "Implementation started", nil

	case "review":
		// Handle "review <n>" for viewing reviews, "review" alone runs review workflow
		if len(args) > 0 {
			// If first arg is a number, view that review
			if num, parseErr := strconv.Atoi(args[0]); parseErr == nil {
				task := cond.GetActiveTask()
				if task == nil {
					return "", errors.New("no active task")
				}
				ws := cond.GetWorkspace()
				review, loadErr := ws.LoadReview(task.ID, num)
				if loadErr != nil {
					return "", loadErr
				}
				preview := review
				if len(preview) > 500 {
					preview = preview[:500] + "..."
				}

				return fmt.Sprintf("Review %d:\n%s", num, preview), nil
			} else if args[0] == "view" && len(args) > 1 {
				// Handle "review view <n>"
				if num, parseErr := strconv.Atoi(args[1]); parseErr == nil {
					task := cond.GetActiveTask()
					if task == nil {
						return "", errors.New("no active task")
					}
					ws := cond.GetWorkspace()
					review, loadErr := ws.LoadReview(task.ID, num)
					if loadErr != nil {
						return "", loadErr
					}
					preview := review
					if len(preview) > 500 {
						preview = preview[:500] + "..."
					}

					return fmt.Sprintf("Review %d:\n%s", num, preview), nil
				}

				return "", errors.New("review number must be an integer")
			}

			return "", errors.New("usage: review <number> or review view <number>")
		}
		// No args - run review workflow
		if err := cond.Review(ctx); err != nil {
			return "", err
		}

		return "Review started", nil

	case "continue":
		if err := cond.ResumePaused(ctx); err != nil {
			return "", err
		}

		return "Resumed", nil

	case "finish":
		if cond.GetActiveTask() == nil {
			return "", errors.New("no active task")
		}
		if err := cond.Finish(ctx, conductor.FinishOptions{}); err != nil {
			return "", err
		}

		return "Task completed", nil

	case "abandon":
		if cond.GetActiveTask() == nil {
			return "", errors.New("no active task")
		}
		if err := cond.Delete(ctx, conductor.DeleteOptions{
			Force:      true,
			KeepBranch: false,
			DeleteWork: conductor.BoolPtr(true),
		}); err != nil {
			return "", err
		}

		return "Task abandoned", nil

	case "undo":
		if err := cond.Undo(ctx); err != nil {
			return "", err
		}

		return "Undo complete", nil

	case "redo":
		if err := cond.Redo(ctx); err != nil {
			return "", err
		}

		return "Redo complete", nil

	case "reset":
		if err := cond.ResetState(ctx); err != nil {
			return "", err
		}

		return "Workflow reset to idle", nil

	case "auto":
		// Auto-execute the next workflow step based on current state
		task := cond.GetActiveTask()
		if task == nil {
			return "", errors.New("no active task")
		}
		switch task.State {
		case "idle":
			return "", errors.New("no active task, use 'start' first")
		case "planning":
			if err := cond.Plan(ctx); err != nil {
				return "", err
			}

			return "Planning started", nil
		case "implementing":
			if err := cond.Implement(ctx); err != nil {
				return "", err
			}

			return "Implementation started", nil
		case "reviewing":
			if err := cond.Review(ctx); err != nil {
				return "", err
			}

			return "Review started", nil
		case "waiting":
			return "", errors.New("task is waiting for user input")
		case "done", "failed":
			return "", errors.New("task is already completed")
		default:
			return "", fmt.Errorf("cannot auto-execute in state: %s", task.State)
		}

	default:
		return "", fmt.Errorf("unknown workflow command: %s", command)
	}
}
