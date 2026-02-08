package commands

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/server/views"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "hierarchy",
			Description:  "Get hierarchical context for the active task",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleHierarchy,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "specification-diff",
			Aliases:      []string{"spec-diff"},
			Description:  "Get a unified diff for a specification's implemented file",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleSpecificationDiff,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "guide",
			Description:  "Get guidance on what to do next",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleGuide,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "budget-reset",
			Description:  "Reset monthly budget tracking",
			Category:     "control",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleBudgetReset,
	})
}

func handleHierarchy(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	activeTask := cond.GetActiveTask()
	if activeTask == nil {
		return NewResult("No active task").WithData(map[string]any{
			"active":    false,
			"hierarchy": views.HierarchyData{},
		}), nil
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	data := views.ComputeHierarchyContext(cond, ws, activeTask.ID)
	if data == nil {
		data = &views.HierarchyData{}
	}

	return NewResult("Hierarchy loaded").WithData(data), nil
}

func handleSpecificationDiff(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	taskID := GetString(inv.Options, "task_id")
	if taskID == "" {
		return nil, errors.New("task ID is required")
	}

	specNumberRaw := GetString(inv.Options, "spec_number")
	if specNumberRaw == "" {
		return nil, errors.New("specification number is required")
	}

	specNumber, err := strconv.Atoi(specNumberRaw)
	if err != nil || specNumber <= 0 {
		return nil, errors.New("specification number must be a positive integer")
	}

	filePath := GetString(inv.Options, "file")
	if filePath == "" {
		return nil, errors.New("file is required")
	}

	contextLines := GetInt(inv.Options, "context")
	if contextLines == 0 {
		contextLines = 3
	}

	diff, err := cond.GetSpecificationFileDiff(ctx, taskID, specNumber, filePath, contextLines)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not listed in specification") ||
			strings.Contains(errMsg, "load specification") {
			return nil, fmt.Errorf("%w: %s", ErrBadRequest, errMsg)
		}

		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	return NewResult("Diff loaded").WithData(map[string]any{
		"task_id":       taskID,
		"specification": specNumber,
		"file":          filePath,
		"context":       contextLines,
		"has_diff":      diff != "",
		"diff":          diff,
	}), nil
}

func handleGuide(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	activeTask := cond.GetActiveTask()
	if activeTask == nil {
		return NewResult("No active task").WithData(map[string]any{
			"has_task": false,
			"next_actions": []guideActionData{
				{
					Command:     "mehr start <reference>",
					Description: "Start a new task",
					Endpoint:    "POST /api/v1/workflow/start",
				},
			},
		}), nil
	}

	work, _ := ws.LoadWork(activeTask.ID)
	specs, _ := ws.ListSpecificationsWithStatus(activeTask.ID)

	response := map[string]any{
		"has_task":       true,
		"task_id":        activeTask.ID,
		"state":          activeTask.State,
		"specifications": len(specs),
	}

	if work != nil {
		response["title"] = work.Metadata.Title
	}

	// Check for pending question
	if ws.HasPendingQuestion(activeTask.ID) {
		q, _ := ws.LoadPendingQuestion(activeTask.ID)
		if q != nil {
			var options []string
			for _, opt := range q.Options {
				options = append(options, opt.Label)
			}
			response["pending_question"] = map[string]any{
				"question": q.Question,
				"options":  options,
			}
			response["next_actions"] = []guideActionData{
				{
					Command:     `mehr answer "your answer"`,
					Description: "Respond to the question",
					Endpoint:    "POST /api/v1/workflow/answer",
				},
			}

			return NewResult("Guide loaded").WithData(response), nil
		}
	}

	response["next_actions"] = computeGuideActions(workflow.State(activeTask.State), len(specs))

	return NewResult("Guide loaded").WithData(response), nil
}

// guideActionData represents a suggested next action.
type guideActionData struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Endpoint    string `json:"endpoint,omitempty"`
}

// computeGuideActions returns suggested actions based on workflow state.
func computeGuideActions(state workflow.State, specifications int) []guideActionData {
	switch state {
	case workflow.StateIdle:
		if specifications == 0 {
			return []guideActionData{
				{Command: "mehr plan", Description: "Create specifications", Endpoint: "POST /api/v1/workflow/plan"},
				{Command: "mehr note", Description: "Add requirements", Endpoint: "POST /api/v1/tasks/{id}/notes"},
			}
		}

		return []guideActionData{
			{Command: "mehr implement", Description: "Implement the specifications", Endpoint: "POST /api/v1/workflow/implement"},
			{Command: "mehr plan", Description: "Create more specifications", Endpoint: "POST /api/v1/workflow/plan"},
		}

	case workflow.StatePlanning:
		return []guideActionData{
			{Command: "mehr status", Description: "View planning progress", Endpoint: "GET /api/v1/task"},
			{Command: "mehr question", Description: "Ask the agent a question", Endpoint: "POST /api/v1/workflow/question"},
		}

	case workflow.StateImplementing:
		return []guideActionData{
			{Command: "mehr status", Description: "View implementation progress", Endpoint: "GET /api/v1/task"},
			{Command: "mehr question", Description: "Ask the agent a question", Endpoint: "POST /api/v1/workflow/question"},
			{Command: "mehr undo", Description: "Revert last change", Endpoint: "POST /api/v1/workflow/undo"},
			{Command: "mehr finish", Description: "Complete and merge", Endpoint: "POST /api/v1/workflow/finish"},
		}

	case workflow.StatePaused:
		return []guideActionData{
			{Command: "mehr budget status", Description: "Review budget limits", Endpoint: "GET /api/v1/costs"},
			{Command: "mehr budget resume --confirm", Description: "Resume after budget pause", Endpoint: "POST /api/v1/workflow/resume"},
		}

	case workflow.StateReviewing:
		return []guideActionData{
			{Command: "mehr finish", Description: "Complete and merge", Endpoint: "POST /api/v1/workflow/finish"},
			{Command: "mehr question", Description: "Ask the agent a question", Endpoint: "POST /api/v1/workflow/question"},
			{Command: "mehr implement", Description: "Make more changes", Endpoint: "POST /api/v1/workflow/implement"},
		}

	case workflow.StateDone:
		return []guideActionData{
			{Command: "mehr start <reference>", Description: "Start a new task", Endpoint: "POST /api/v1/workflow/start"},
		}

	case workflow.StateWaiting:
		return []guideActionData{
			{Command: `mehr answer "response"`, Description: "Respond to agent question", Endpoint: "POST /api/v1/workflow/answer"},
		}

	case workflow.StateFailed:
		return []guideActionData{
			{Command: "mehr status", Description: "View error details", Endpoint: "GET /api/v1/task"},
			{Command: "mehr implement", Description: "Retry implementation", Endpoint: "POST /api/v1/workflow/implement"},
		}

	case workflow.StateCheckpointing, workflow.StateReverting, workflow.StateRestoring:
		return []guideActionData{
			{Command: "mehr status", Description: "View operation progress", Endpoint: "GET /api/v1/task"},
		}
	}

	return []guideActionData{
		{Command: "mehr status", Description: "View detailed status", Endpoint: "GET /api/v1/task"},
	}
}

func handleBudgetReset(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	if err := ws.ResetMonthlyBudget(); err != nil {
		return nil, fmt.Errorf("failed to reset budget: %w", err)
	}

	return NewResult("Monthly budget reset"), nil
}
