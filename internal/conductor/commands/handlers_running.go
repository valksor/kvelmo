package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/taskrunner"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "running-list",
			Description:  "List running parallel tasks",
			Category:     "tools",
			RequiresTask: false,
		},
		Handler: handleRunningList,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "running-cancel",
			Description:  "Cancel a running parallel task",
			Category:     "tools",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleRunningCancel,
	})
}

// handleRunningList returns all running parallel tasks.
// The task registry is injected via InjectFn as "_registry" in options.
func handleRunningList(_ context.Context, _ *conductor.Conductor, inv Invocation) (*Result, error) {
	registry := extractRegistry(inv.Options)
	if registry == nil {
		return NewListResult("No parallel tasks", map[string]any{
			"tasks":   []map[string]any{},
			"count":   0,
			"running": 0,
		}), nil
	}

	tasks := registry.List()
	jsonTasks := make([]map[string]any, 0, len(tasks))

	for _, task := range tasks {
		entry := map[string]any{
			"id":         task.ID,
			"reference":  task.Reference,
			"status":     string(task.Status),
			"started_at": task.StartedAt,
			"duration":   task.Duration().String(),
		}
		if task.TaskID != "" {
			entry["task_id"] = task.TaskID
		}
		if !task.FinishedAt.IsZero() {
			entry["finished_at"] = task.FinishedAt
		}
		if task.WorktreePath != "" {
			entry["worktree_path"] = task.WorktreePath
		}
		if task.Error != nil {
			entry["error"] = task.Error.Error()
		}
		jsonTasks = append(jsonTasks, entry)
	}

	return NewListResult("Running tasks", map[string]any{
		"tasks":   jsonTasks,
		"count":   len(jsonTasks),
		"running": registry.CountRunning(),
	}), nil
}

// handleRunningCancel cancels a specific running parallel task.
// The task registry is injected via InjectFn as "_registry" in options.
func handleRunningCancel(_ context.Context, _ *conductor.Conductor, inv Invocation) (*Result, error) {
	taskID := GetString(inv.Options, "task_id")
	if taskID == "" {
		return nil, fmt.Errorf("%w: task_id is required", ErrBadRequest)
	}

	registry := extractRegistry(inv.Options)
	if registry == nil {
		return nil, errors.New("no parallel tasks running")
	}

	task := registry.Get(taskID)
	if task == nil {
		return nil, fmt.Errorf("%w: task not found: %s", ErrBadRequest, taskID)
	}

	_ = registry.Cancel(taskID)

	return NewResult("Task cancellation requested").WithData(map[string]any{
		"success":   true,
		"task_id":   taskID,
		"reference": task.Reference,
	}), nil
}

// extractRegistry pulls the task registry from injected options.
func extractRegistry(opts map[string]any) *taskrunner.Registry {
	if opts == nil {
		return nil
	}
	registry, _ := opts["_registry"].(*taskrunner.Registry)

	return registry
}
