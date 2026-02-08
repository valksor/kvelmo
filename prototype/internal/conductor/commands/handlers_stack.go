package commands

import (
	"context"
	"errors"
	"log/slog"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/stack"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "stack",
			Description:  "Stack management commands",
			Category:     "tools",
			Subcommands:  []string{"list", "sync", "rebase", "rebase-preview"},
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleStack,
	})
}

func handleStack(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	sub := GetString(inv.Options, "subcommand")

	switch sub {
	case "list":
		return handleStackList(ctx, cond, inv)
	case "sync":
		return handleStackSync(ctx, cond, inv)
	case "rebase":
		return handleStackRebase(ctx, cond, inv)
	case "rebase-preview":
		return handleStackRebasePreview(ctx, cond, inv)
	default:
		return nil, errors.New("unknown stack subcommand: " + sub)
	}
}

// handleStackList loads and returns all stacks with their task states.
func handleStackList(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	stor := stack.NewStorage(ws.DataRoot())
	if err := stor.Load(); err != nil {
		return NewListResult("No stacks found", map[string]any{ //nolint:nilerr // No stacks file = empty list
			"stacks": []map[string]any{},
			"count":  0,
		}), nil
	}

	stacks := stor.ListStacks()
	summaries := make([]map[string]any, 0, len(stacks))

	for _, st := range stacks {
		tasks := make([]map[string]any, 0, len(st.Tasks))
		hasRebase := false
		hasConflict := false

		for _, task := range st.Tasks {
			tasks = append(tasks, map[string]any{
				"id":         task.ID,
				"branch":     task.Branch,
				"state":      string(task.State),
				"pr_number":  task.PRNumber,
				"pr_url":     task.PRURL,
				"depends_on": task.DependsOn,
				"state_icon": getStackStateIcon(task.State),
			})

			if task.State == stack.StateNeedsRebase {
				hasRebase = true
			}
			if task.State == stack.StateConflict {
				hasConflict = true
			}
		}

		summaries = append(summaries, map[string]any{
			"id":           st.ID,
			"root_task":    st.RootTask,
			"task_count":   st.TaskCount(),
			"tasks":        tasks,
			"created_at":   st.CreatedAt.Format("2006-01-02T15:04:05"),
			"updated_at":   st.UpdatedAt.Format("2006-01-02T15:04:05"),
			"has_rebase":   hasRebase,
			"has_conflict": hasConflict,
		})
	}

	return NewListResult("Stacks loaded", map[string]any{
		"stacks": summaries,
		"count":  len(stacks),
	}), nil
}

// handleStackSync syncs PR status for all stacks.
// Currently a placeholder returning 0 updated - full sync requires provider integration.
func handleStackSync(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	stor := stack.NewStorage(ws.DataRoot())
	if err := stor.Load(); err != nil {
		return NewResult("Stack sync complete").WithData(map[string]any{ //nolint:nilerr // No stacks = nothing to sync
			"success": true,
			"updated": 0,
		}), nil
	}

	slog.Info("stack sync requested via command handler")

	return NewResult("Stack sync complete").WithData(map[string]any{
		"success": true,
		"updated": 0,
	}), nil
}

// handleStackRebase rebases stacked tasks.
// Supports task_id (single task), stack_id (single stack), or rebase-all (all stacks).
func handleStackRebase(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	git := cond.GetGit()
	if git == nil {
		return nil, errors.New("git not initialized")
	}

	stackID := GetString(inv.Options, "stack_id")
	taskID := GetString(inv.Options, "task_id")
	rebaseAll := GetBool(inv.Options, "rebase_all")

	stor := stack.NewStorage(ws.DataRoot())
	rebaser := stack.NewRebaser(stor, git)

	// Rebase specific task.
	if taskID != "" {
		return doRebaseTask(ctx, rebaser, taskID)
	}

	// Rebase specific stack.
	if stackID != "" {
		return doRebaseStack(ctx, rebaser, stackID)
	}

	// Rebase all stacks with tasks needing rebase.
	if rebaseAll {
		return doRebaseAllStacks(ctx, stor, rebaser)
	}

	return nil, errors.New("rebase requires stack_id, task_id, or rebase_all option")
}

func doRebaseTask(ctx context.Context, rebaser *stack.Rebaser, taskID string) (*Result, error) {
	result, err := rebaser.RebaseTask(ctx, taskID)

	return buildRebaseResponse(result, err)
}

func doRebaseStack(ctx context.Context, rebaser *stack.Rebaser, stackID string) (*Result, error) {
	result, err := rebaser.RebaseAll(ctx, stackID)

	return buildRebaseResponse(result, err)
}

func doRebaseAllStacks(ctx context.Context, stor *stack.Storage, rebaser *stack.Rebaser) (*Result, error) {
	if err := stor.Load(); err != nil {
		return NewResult("Stack rebase complete").WithData(map[string]any{ //nolint:nilerr // No stacks = nothing to rebase
			"success": true,
			"rebased": 0,
			"results": []map[string]any{},
		}), nil
	}

	allResults := make([]map[string]any, 0)

	for _, st := range stor.ListStacks() {
		if len(st.GetTasksNeedingRebase()) == 0 {
			continue
		}

		result, err := rebaser.RebaseAll(ctx, st.ID)
		if err != nil {
			// On failure, return what we have so far plus the failure info.
			return buildRebaseResponse(result, err)
		}

		for _, tr := range result.RebasedTasks {
			allResults = append(allResults, map[string]any{
				"task_id":  tr.TaskID,
				"branch":   tr.Branch,
				"old_base": tr.OldBase,
				"new_base": tr.NewBase,
			})
		}
	}

	slog.Info("stack rebase completed", "rebased", len(allResults))

	return NewResult("Stack rebase complete").WithData(map[string]any{
		"success": true,
		"rebased": len(allResults),
		"results": allResults,
	}), nil
}

// buildRebaseResponse converts rebase results into a command Result.
func buildRebaseResponse(result *stack.RebaseResult, err error) (*Result, error) {
	if err != nil {
		data := map[string]any{
			"success": false,
			"error":   err.Error(),
		}

		if result != nil && result.FailedTask != nil {
			data["failed"] = map[string]any{
				"task_id":       result.FailedTask.TaskID,
				"branch":        result.FailedTask.Branch,
				"onto_base":     result.FailedTask.OntoBase,
				"is_conflict":   result.FailedTask.IsConflict,
				"conflict_hint": result.FailedTask.ConflictHint,
			}
		}

		return NewResult("Stack rebase failed").WithData(data), nil
	}

	results := make([]map[string]any, 0, len(result.RebasedTasks))
	for _, tr := range result.RebasedTasks {
		results = append(results, map[string]any{
			"task_id":  tr.TaskID,
			"branch":   tr.Branch,
			"old_base": tr.OldBase,
			"new_base": tr.NewBase,
		})
	}

	slog.Info("stack rebase completed", "rebased", len(result.RebasedTasks))

	return NewResult("Stack rebase complete").WithData(map[string]any{
		"success": true,
		"rebased": len(result.RebasedTasks),
		"results": results,
	}), nil
}

// handleStackRebasePreview returns a preview of what would happen during rebase.
// Supports task_id (single task), stack_id (single stack), or preview-all (all stacks).
func handleStackRebasePreview(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	git := cond.GetGit()
	if git == nil {
		return nil, errors.New("git not initialized")
	}

	stackID := GetString(inv.Options, "stack_id")
	taskID := GetString(inv.Options, "task_id")
	previewAll := GetBool(inv.Options, "preview_all")

	stor := stack.NewStorage(ws.DataRoot())
	if err := stor.Load(); err != nil { //nolint:nilerr // No stacks yet, return empty preview
		return NewResult("Rebase preview").WithData(map[string]any{
			"tasks":          []map[string]any{},
			"has_conflicts":  false,
			"safe_count":     0,
			"conflict_count": 0,
		}), nil
	}

	rebaser := stack.NewRebaser(stor, git)

	// Preview single task.
	if taskID != "" {
		return doPreviewTask(ctx, rebaser, taskID)
	}

	// Preview specific stack.
	if stackID != "" {
		return doPreviewStack(ctx, rebaser, stackID)
	}

	// Preview all stacks.
	if previewAll {
		return doPreviewAllStacks(ctx, stor, rebaser)
	}

	return nil, errors.New("rebase-preview requires stack_id, task_id, or preview_all option")
}

func doPreviewTask(ctx context.Context, rebaser *stack.Rebaser, taskID string) (*Result, error) {
	preview, err := rebaser.PreviewTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	safe := !preview.WouldConflict && !preview.Unavailable

	return NewResult("Rebase preview").WithData(map[string]any{
		"tasks": []map[string]any{
			{
				"task_id":           preview.TaskID,
				"branch":            preview.Branch,
				"onto_base":         preview.OntoBase,
				"safe":              safe,
				"would_conflict":    preview.WouldConflict,
				"conflicting_files": preview.ConflictingFiles,
				"unavailable":       preview.Unavailable,
			},
		},
		"has_conflicts":  preview.WouldConflict,
		"safe_count":     boolToInt(safe),
		"conflict_count": boolToInt(preview.WouldConflict),
		"unavailable":    preview.Unavailable,
	}), nil
}

func doPreviewStack(ctx context.Context, rebaser *stack.Rebaser, stackID string) (*Result, error) {
	preview, err := rebaser.PreviewRebase(ctx, stackID)
	if err != nil {
		return nil, err
	}

	return NewResult("Rebase preview").WithData(convertPreviewToData(preview)), nil
}

func doPreviewAllStacks(ctx context.Context, stor *stack.Storage, rebaser *stack.Rebaser) (*Result, error) {
	allTasks := make([]map[string]any, 0)

	var totalSafe, totalConflict int
	var unavailable bool
	var unavailableReason string

	for _, st := range stor.ListStacks() {
		if len(st.GetTasksNeedingRebase()) == 0 {
			continue
		}

		preview, err := rebaser.PreviewRebase(ctx, st.ID)
		if err != nil {
			slog.Warn("failed to preview stack", "stack_id", st.ID, "error", err)

			continue
		}

		for _, task := range preview.Tasks {
			allTasks = append(allTasks, map[string]any{
				"task_id":           task.TaskID,
				"branch":            task.Branch,
				"onto_base":         task.OntoBase,
				"safe":              !task.WouldConflict && !task.Unavailable,
				"would_conflict":    task.WouldConflict,
				"conflicting_files": task.ConflictingFiles,
				"unavailable":       task.Unavailable,
			})
		}

		totalSafe += preview.SafeCount
		totalConflict += preview.ConflictCount

		if preview.Unavailable {
			unavailable = true
			if unavailableReason == "" {
				unavailableReason = preview.UnavailableReason
			}
		}
	}

	return NewResult("Rebase preview").WithData(map[string]any{
		"tasks":              allTasks,
		"has_conflicts":      totalConflict > 0,
		"safe_count":         totalSafe,
		"conflict_count":     totalConflict,
		"unavailable":        unavailable,
		"unavailable_reason": unavailableReason,
	}), nil
}

// convertPreviewToData converts a stack.RebasePreview to a response map.
func convertPreviewToData(preview *stack.RebasePreview) map[string]any {
	tasks := make([]map[string]any, 0, len(preview.Tasks))
	for _, task := range preview.Tasks {
		tasks = append(tasks, map[string]any{
			"task_id":           task.TaskID,
			"branch":            task.Branch,
			"onto_base":         task.OntoBase,
			"safe":              !task.WouldConflict && !task.Unavailable,
			"would_conflict":    task.WouldConflict,
			"conflicting_files": task.ConflictingFiles,
			"unavailable":       task.Unavailable,
		})
	}

	return map[string]any{
		"tasks":              tasks,
		"has_conflicts":      preview.HasConflicts,
		"safe_count":         preview.SafeCount,
		"conflict_count":     preview.ConflictCount,
		"unavailable":        preview.Unavailable,
		"unavailable_reason": preview.UnavailableReason,
	}
}

// boolToInt converts a boolean to 0 or 1.
func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

// getStackStateIcon returns the icon name for a stack state.
func getStackStateIcon(state stack.StackState) string {
	switch state {
	case stack.StateMerged:
		return "check"
	case stack.StateNeedsRebase:
		return "refresh"
	case stack.StateConflict:
		return "x-circle"
	case stack.StatePendingReview:
		return "clock"
	case stack.StateApproved:
		return "check-circle"
	case stack.StateAbandoned:
		return "slash"
	case stack.StateActive:
		return "play"
	default:
		return "circle"
	}
}
