package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
)

type optimizeCommandOptions struct {
	Agent string `json:"agent,omitempty"`
}

type submitCommandOptions struct {
	Provider string   `json:"provider,omitempty"`
	Labels   []string `json:"labels,omitempty"`
	DryRun   bool     `json:"dry_run,omitempty"`
}

type submitSourceCommandOptions struct {
	Source       string   `json:"source,omitempty"`
	Provider     string   `json:"provider,omitempty"`
	Notes        []string `json:"notes,omitempty"`
	Title        string   `json:"title,omitempty"`
	Instructions string   `json:"instructions,omitempty"`
	Labels       []string `json:"labels,omitempty"`
	QueueID      string   `json:"queue_id,omitempty"`
	Optimize     bool     `json:"optimize,omitempty"`
	DryRun       bool     `json:"dry_run,omitempty"`
	Priority     int      `json:"priority,omitempty"`
}

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "delete",
			Aliases:      []string{"del", "rm"},
			Description:  "Delete a queued task",
			Category:     "project",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleDeleteQueueTask,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "export",
			Description:  "Export a queued task to markdown",
			Category:     "project",
			RequiresTask: false,
			MutatesState: false,
		},
		Handler: handleExportQueueTask,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "optimize",
			Description:  "Optimize a queued task with AI",
			Category:     "project",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleOptimizeQueueTask,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "submit",
			Description:  "Submit a queued task to provider",
			Category:     "project",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleSubmitQueueTask,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "submit-source",
			Description:  "Create a queue task from source and submit it",
			Category:     "project",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleSubmitSourceTask,
	})
}

func handleDeleteQueueTask(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	if len(inv.Args) == 0 {
		return nil, errors.New("delete requires a task reference (e.g., quick-tasks/task-1)")
	}

	queueID, taskID, err := conductor.ParseQueueTaskRef(inv.Args[0])
	if err != nil {
		return nil, err
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		return nil, fmt.Errorf("queue not found: %s", queueID)
	}
	if !queue.RemoveTask(taskID) {
		return nil, fmt.Errorf("task not found: %s/%s", queueID, taskID)
	}
	if err := queue.Save(); err != nil {
		return nil, fmt.Errorf("save queue: %w", err)
	}

	_ = ws.DeleteFile(ws.QueueNotePath(queueID, taskID))

	return NewResult(fmt.Sprintf("Deleted task %s from %s", taskID, queueID)).WithData(map[string]any{
		"success":  true,
		"queue_id": queueID,
		"task_id":  taskID,
		"message":  fmt.Sprintf("Deleted task %s from %s", taskID, queueID),
	}), nil
}

func handleExportQueueTask(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	if len(inv.Args) == 0 {
		return nil, errors.New("export requires a task reference (e.g., quick-tasks/task-1)")
	}

	queueID, taskID, err := conductor.ParseQueueTaskRef(inv.Args[0])
	if err != nil {
		return nil, err
	}

	markdown, err := cond.ExportQueueTask(queueID, taskID)
	if err != nil {
		return nil, fmt.Errorf("export task: %w", err)
	}

	return NewResult(fmt.Sprintf("Exported %s/%s", queueID, taskID)).WithData(map[string]any{
		"success":  true,
		"queue_id": queueID,
		"task_id":  taskID,
		"markdown": markdown,
	}), nil
}

func handleOptimizeQueueTask(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	if len(inv.Args) == 0 {
		return nil, errors.New("optimize requires a task reference (e.g., quick-tasks/task-1)")
	}

	queueID, taskID, err := conductor.ParseQueueTaskRef(inv.Args[0])
	if err != nil {
		return nil, err
	}

	opts, err := DecodeOptions[optimizeCommandOptions](inv)
	if err != nil {
		return nil, err
	}

	if agent := strings.TrimSpace(opts.Agent); agent != "" {
		cond.SetAgent(agent)
		defer cond.ClearAgent()
	}

	optimized, err := cond.OptimizeQueueTask(ctx, queueID, taskID)
	if err != nil {
		return nil, fmt.Errorf("optimize task: %w", err)
	}

	return NewResult("Task optimized").WithData(optimized), nil
}

func handleSubmitQueueTask(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	if len(inv.Args) == 0 {
		return nil, errors.New("submit requires: submit <queue>/<task-id> <provider>")
	}

	queueID, taskID, err := conductor.ParseQueueTaskRef(inv.Args[0])
	if err != nil {
		return nil, err
	}
	opts, err := DecodeOptions[submitCommandOptions](inv)
	if err != nil {
		return nil, err
	}

	providerName := ""
	if len(inv.Args) > 1 {
		providerName = strings.TrimSpace(inv.Args[1])
	}
	if providerName == "" {
		providerName = strings.TrimSpace(opts.Provider)
	}
	if providerName == "" {
		return nil, errors.New("submit requires: submit <queue>/<task-id> <provider>")
	}

	result, err := cond.SubmitQueueTask(ctx, queueID, taskID, conductor.SubmitOptions{
		Provider: providerName,
		Labels:   append([]string{}, opts.Labels...),
		TaskIDs:  []string{taskID},
		DryRun:   opts.DryRun,
	})
	if err != nil {
		return nil, fmt.Errorf("submit task: %w", err)
	}

	data := map[string]any{
		"success":  true,
		"provider": providerName,
		"queue_id": queueID,
		"task_id":  taskID,
		"dry_run":  opts.DryRun,
	}

	if len(result.Tasks) > 0 {
		submitted := result.Tasks[0]
		data["external_id"] = submitted.ExternalID
		data["external_url"] = submitted.ExternalURL
		if opts.DryRun {
			data["title"] = submitted.Title
		}
	}
	if result.Epic != nil {
		data["epic_id"] = result.Epic.ExternalID
		data["epic_url"] = result.Epic.ExternalURL
	}

	return NewResult("Submitted to " + providerName).WithData(data), nil
}

func handleSubmitSourceTask(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	opts, err := DecodeOptions[submitSourceCommandOptions](inv)
	if err != nil {
		return nil, err
	}

	source := strings.TrimSpace(opts.Source)
	if source == "" && len(inv.Args) > 0 {
		source = strings.TrimSpace(inv.Args[0])
	}
	if source == "" {
		return nil, errors.New("source is required")
	}

	provider := strings.TrimSpace(opts.Provider)
	if provider == "" && len(inv.Args) > 1 {
		provider = strings.TrimSpace(inv.Args[1])
	}
	if provider == "" {
		return nil, errors.New("provider is required")
	}

	createResult, err := cond.CreateQueueTaskFromSource(ctx, source, conductor.SourceTaskOptions{
		QueueID:      strings.TrimSpace(opts.QueueID),
		Title:        strings.TrimSpace(opts.Title),
		Instructions: strings.TrimSpace(opts.Instructions),
		Notes:        append([]string{}, opts.Notes...),
		Provider:     provider,
		Priority:     opts.Priority,
		Labels:       append([]string{}, opts.Labels...),
	})
	if err != nil {
		return nil, fmt.Errorf("create task from source: %w", err)
	}

	if opts.Optimize {
		if _, optimizeErr := cond.OptimizeQueueTask(ctx, createResult.QueueID, createResult.TaskID); optimizeErr != nil {
			return nil, fmt.Errorf("optimize task: %w", optimizeErr)
		}
	}

	submitResult, err := cond.SubmitQueueTask(ctx, createResult.QueueID, createResult.TaskID, conductor.SubmitOptions{
		Provider: provider,
		Labels:   append([]string{}, opts.Labels...),
		TaskIDs:  []string{createResult.TaskID},
		DryRun:   opts.DryRun,
	})
	if err != nil {
		return nil, fmt.Errorf("submit task: %w", err)
	}

	data := map[string]any{
		"success":  true,
		"queue_id": createResult.QueueID,
		"task_id":  createResult.TaskID,
		"provider": provider,
		"dry_run":  submitResult.DryRun,
	}
	if len(submitResult.Tasks) > 0 {
		submitted := submitResult.Tasks[0]
		data["external_id"] = submitted.ExternalID
		data["external_url"] = submitted.ExternalURL
	}

	return NewResult("Source task submitted").WithData(data), nil
}
