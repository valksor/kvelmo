package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// handleProjectTaskEditCmd updates a task in a queue.
func handleProjectTaskEditCmd(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	taskID := strings.TrimSpace(GetString(inv.Options, "task_id"))
	if taskID == "" && len(inv.Args) > 0 {
		taskID = strings.TrimSpace(inv.Args[0])
	}
	if taskID == "" {
		return nil, errors.New("task ID is required")
	}

	queueID := strings.TrimSpace(GetString(inv.Options, "queue_id"))
	if queueID == "" {
		queueID = mostRecentQueue(ws)
		if queueID == "" {
			return nil, errors.New("no queues found")
		}
	}

	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		return nil, fmt.Errorf("queue not found: %w", err)
	}

	// Apply updates from options
	err = queue.UpdateTask(taskID, func(task *storage.QueuedTask) {
		if title := GetString(inv.Options, "title"); title != "" {
			task.Title = title
		}
		if desc := GetString(inv.Options, "description"); desc != "" {
			task.Description = desc
		}
		if priority := GetInt(inv.Options, "priority"); priority != 0 {
			task.Priority = priority
		}
		if status := GetString(inv.Options, "status"); status != "" {
			task.Status = storage.TaskStatus(status)
		}
		if parentID, ok := inv.Options["parent_id"]; ok {
			task.ParentID = fmt.Sprintf("%v", parentID)
		}
		if dependsOn, ok := inv.Options["depends_on"]; ok {
			task.DependsOn = toStringSlice(dependsOn)
		}
		if labels, ok := inv.Options["labels"]; ok {
			task.Labels = toStringSlice(labels)
		}
		if assignee, ok := inv.Options["assignee"]; ok {
			task.Assignee = fmt.Sprintf("%v", assignee)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Recompute relationships
	queue.ComputeBlocksRelations()
	queue.ComputeSubtaskRelations()
	queue.ComputeTaskStatuses()

	if err := queue.Save(); err != nil {
		return nil, fmt.Errorf("save failed: %w", err)
	}

	updated := queue.GetTask(taskID)

	return NewResult("Task updated").WithData(convertQueuedTaskMap(updated)), nil
}

// handleProjectReorderCmd reorders tasks in a queue.
func handleProjectReorderCmd(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	queueID := strings.TrimSpace(GetString(inv.Options, "queue_id"))
	if queueID == "" {
		queueID = mostRecentQueue(ws)
		if queueID == "" {
			return nil, errors.New("no queues found")
		}
	}

	// Auto reorder mode
	if GetBool(inv.Options, "auto") {
		result, err := cond.AutoReorderTasks(ctx, queueID)
		if err != nil {
			return nil, fmt.Errorf("auto reorder failed: %w", err)
		}

		return NewResult("Tasks reordered by AI").WithData(map[string]any{
			"old_order": result.OldOrder,
			"new_order": result.NewOrder,
			"reasoning": result.Reasoning,
		}), nil
	}

	// Manual reorder mode
	taskID := strings.TrimSpace(GetString(inv.Options, "task_id"))
	referenceID := strings.TrimSpace(GetString(inv.Options, "reference_id"))
	position := strings.TrimSpace(GetString(inv.Options, "position"))

	if taskID == "" || referenceID == "" || position == "" {
		return nil, errors.New("manual reorder requires task_id, reference_id, and position (before/after)")
	}

	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		return nil, fmt.Errorf("queue not found: %w", err)
	}

	// Find target index based on reference task and position
	var targetIndex int
	for i, task := range queue.Tasks {
		if task.ID == referenceID {
			if position == "before" {
				targetIndex = i
			} else {
				targetIndex = i + 1
			}

			break
		}
	}

	if err := queue.ReorderTask(taskID, targetIndex); err != nil {
		return nil, fmt.Errorf("reorder failed: %w", err)
	}

	if err := queue.Save(); err != nil {
		return nil, fmt.Errorf("save failed: %w", err)
	}

	return NewResult("Task reordered").WithData(map[string]any{
		"task_id":  taskID,
		"position": targetIndex,
	}), nil
}

// handleProjectSubmitCmd submits tasks to a provider.
func handleProjectSubmitCmd(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	provider := strings.TrimSpace(GetString(inv.Options, "provider"))
	if provider == "" && len(inv.Args) > 0 {
		provider = strings.TrimSpace(inv.Args[0])
	}
	if provider == "" {
		return nil, errors.New("provider is required")
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	queueID := strings.TrimSpace(GetString(inv.Options, "queue_id"))
	if queueID == "" {
		queueID = mostRecentQueue(ws)
		if queueID == "" {
			return nil, errors.New("no queues found")
		}
	}

	opts := conductor.SubmitOptions{
		Provider:   provider,
		CreateEpic: GetBool(inv.Options, "create_epic"),
		DryRun:     GetBool(inv.Options, "dry_run"),
		Mention:    strings.TrimSpace(GetString(inv.Options, "mention")),
	}
	if labels, ok := inv.Options["labels"]; ok {
		opts.Labels = toStringSlice(labels)
	}
	if taskIDs, ok := inv.Options["task_ids"]; ok {
		opts.TaskIDs = toStringSlice(taskIDs)
	}

	result, err := cond.SubmitProjectTasks(ctx, queueID, opts)
	if err != nil {
		return nil, fmt.Errorf("submit failed: %w", err)
	}

	resp := map[string]any{
		"dry_run": result.DryRun,
	}

	if result.Epic != nil {
		resp["epic"] = map[string]any{
			"external_id":  result.Epic.ExternalID,
			"external_url": result.Epic.ExternalURL,
			"title":        result.Epic.Title,
		}
	}

	tasks := make([]map[string]any, 0, len(result.Tasks))
	for _, task := range result.Tasks {
		tasks = append(tasks, map[string]any{
			"local_id":     task.LocalID,
			"external_id":  task.ExternalID,
			"external_url": task.ExternalURL,
			"title":        task.Title,
		})
	}
	resp["tasks"] = tasks

	msg := fmt.Sprintf("Submitted %d task(s) to %s", len(result.Tasks), provider)
	if result.DryRun {
		msg = fmt.Sprintf("Dry run: %d task(s) would be submitted to %s", len(result.Tasks), provider)
	}

	return NewResult(msg).WithData(resp), nil
}

// handleProjectStartCmd starts implementing tasks from a queue.
func handleProjectStartCmd(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	// Auto mode: full project automation
	if GetBool(inv.Options, "auto") {
		ref := strings.TrimSpace(GetString(inv.Options, "ref"))
		if ref == "" && len(inv.Args) > 0 {
			ref = strings.TrimSpace(inv.Args[0])
		}

		opts := conductor.ProjectAutoOptions{}
		result, err := cond.RunProjectAuto(ctx, ref, opts)
		if err != nil {
			return nil, fmt.Errorf("auto failed: %w", err)
		}

		return NewResult("Project auto completed").WithData(map[string]any{
			"tasks_planned":   result.TasksPlanned,
			"tasks_submitted": result.TasksSubmitted,
			"tasks_completed": result.TasksCompleted,
		}), nil
	}

	// Next task mode
	queueID := strings.TrimSpace(GetString(inv.Options, "queue_id"))
	if queueID == "" {
		queueID = mostRecentQueue(ws)
		if queueID == "" {
			return nil, errors.New("no queues found")
		}
	}

	task, err := cond.StartNextTask(ctx, queueID)
	if err != nil {
		return nil, fmt.Errorf("start failed: %w", err)
	}

	return NewResult("Started task: " + task.Title).WithData(map[string]any{
		"task_id": task.ID,
		"title":   task.Title,
	}), nil
}

// handleProjectSyncCmd syncs a project structure from a provider.
func handleProjectSyncCmd(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	reference := strings.TrimSpace(GetString(inv.Options, "reference"))
	if reference == "" && len(inv.Args) > 0 {
		reference = strings.TrimSpace(inv.Args[0])
	}
	if reference == "" {
		return nil, errors.New("reference is required (e.g., wrike:https://..., jira:PROJ-123)")
	}

	opts := conductor.SyncProjectOptions{
		PreserveExternal: GetBool(inv.Options, "preserve_external"),
		MaxDepth:         GetInt(inv.Options, "max_depth"),
	}
	if includeStatus, ok := inv.Options["include_status"]; ok {
		opts.IncludeStatus = toStringSlice(includeStatus)
	}

	result, err := cond.SyncProject(ctx, reference, opts)
	if err != nil {
		return nil, fmt.Errorf("sync failed: %w", err)
	}

	return NewResult(fmt.Sprintf("Synced %d task(s)", result.TasksSync)).WithData(map[string]any{
		"queue_id":   result.Queue.ID,
		"title":      result.Queue.Title,
		"tasks_sync": result.TasksSync,
		"source":     result.Source,
		"url":        result.URL,
	}), nil
}

// handleProjectUploadCmd accepts a pre-resolved source path from file upload.
// Actual file upload processing stays in the server's ParseFn.
func handleProjectUploadCmd(_ context.Context, _ *conductor.Conductor, inv Invocation) (*Result, error) {
	source := strings.TrimSpace(GetString(inv.Options, "source"))
	if source == "" && len(inv.Args) > 0 {
		source = strings.TrimSpace(inv.Args[0])
	}
	if source == "" {
		return nil, errors.New("source path is required")
	}

	return NewResult("Upload processed").WithData(map[string]any{
		"source": source,
	}), nil
}

// handleProjectSourceCmd accepts a pre-resolved source string.
// Actual URL fetching and text saving stays in the server's ParseFn.
func handleProjectSourceCmd(_ context.Context, _ *conductor.Conductor, inv Invocation) (*Result, error) {
	source := strings.TrimSpace(GetString(inv.Options, "source"))
	if source == "" && len(inv.Args) > 0 {
		source = strings.TrimSpace(inv.Args[0])
	}
	if source == "" {
		return nil, errors.New("source is required")
	}

	return NewResult("Source resolved").WithData(map[string]any{
		"source": source,
	}), nil
}

// toStringSlice converts an any value to a string slice.
// Handles []any (from JSON), []string, and single string values.
func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []string:
		return val
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			} else {
				result = append(result, fmt.Sprintf("%v", item))
			}
		}

		return result
	case string:
		if val == "" {
			return nil
		}

		return []string{val}
	default:
		return nil
	}
}
