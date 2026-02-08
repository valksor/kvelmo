package commands

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "task",
			Description:  "Get active task details",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleTask,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "tasks",
			Description:  "List all tasks in workspace",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleTasks,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "work",
			Description:  "Get work data by task ID (active or completed)",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleWork,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "costs",
			Description:  "Get task costs or aggregate costs",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleCosts,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "budget-monthly",
			Description:  "Get monthly budget status",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleBudgetMonthly,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "specifications",
			Aliases:      []string{"specs"},
			Description:  "List specifications for a task",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleSpecifications,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "sessions",
			Description:  "List sessions for a task",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleSessions,
	})
}

func handleTask(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	if !ws.HasActiveTask() {
		return NewResult("No active task").WithData(map[string]any{
			"active": false,
		}), nil
	}

	activeTask, err := ws.LoadActiveTask()
	if err != nil {
		return nil, fmt.Errorf("failed to load active task: %w", err)
	}

	response := map[string]any{
		"active": true,
		"task": map[string]any{
			"id":             activeTask.ID,
			"state":          activeTask.State,
			"progress_phase": string(ComputeProgressPhase(ws, activeTask.ID)),
			"ref":            activeTask.Ref,
			"branch":         activeTask.Branch,
			"worktree_path":  activeTask.WorktreePath,
			"started":        activeTask.Started,
		},
	}

	if work, err := ws.LoadWork(activeTask.ID); err == nil && work != nil {
		workResponse := map[string]any{
			"title":        work.Metadata.Title,
			"external_key": work.Metadata.ExternalKey,
			"created_at":   work.Metadata.CreatedAt,
			"updated_at":   work.Metadata.UpdatedAt,
			"costs":        work.Costs,
		}
		if content, err := ws.GetSourceContent(activeTask.ID); err == nil && content != "" {
			workResponse["description"] = content
		}
		response["work"] = workResponse
	}

	if q, err := ws.LoadPendingQuestion(activeTask.ID); err == nil && q != nil {
		options := make([]map[string]string, 0, len(q.Options))
		for _, opt := range q.Options {
			options = append(options, map[string]string{
				"label":       opt.Label,
				"value":       opt.Value,
				"description": opt.Description,
			})
		}
		response["pending_question"] = map[string]any{
			"question": q.Question,
			"options":  options,
			"phase":    q.Phase,
		}
	}

	return NewResult("Task loaded").WithData(response), nil
}

func handleTasks(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	taskIDs, err := ws.ListWorks()
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	var activeTask *storage.ActiveTask
	if ws.HasActiveTask() {
		activeTask, _ = ws.LoadActiveTask()
	}

	tasks := make([]map[string]any, 0, len(taskIDs))
	for _, id := range taskIDs {
		work, err := ws.LoadWork(id)
		if err != nil || work == nil {
			continue
		}

		state := work.Metadata.State
		if activeTask != nil && activeTask.ID == id {
			state = activeTask.State
		}

		task := map[string]any{
			"id":             id,
			"title":          work.Metadata.Title,
			"state":          state,
			"progress_phase": string(ComputeProgressPhase(ws, id)),
			"created_at":     work.Metadata.CreatedAt,
		}
		if work.Git.WorktreePath != "" {
			task["worktree_path"] = work.Git.WorktreePath
		}

		tasks = append(tasks, task)
	}

	return NewResult(fmt.Sprintf("%d task(s)", len(tasks))).WithData(map[string]any{
		"tasks": tasks,
		"count": len(tasks),
	}), nil
}

// handleWork returns work data for a specific task ID.
// Works for both active and completed tasks as long as the work directory exists.
func handleWork(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	// Get task ID from args or options
	taskID := GetString(inv.Options, "id")
	if taskID == "" && len(inv.Args) > 0 {
		taskID = inv.Args[0]
	}
	if taskID == "" {
		return nil, errors.New("task ID is required")
	}

	// Check if this is the active task
	var activeTask *storage.ActiveTask
	if ws.HasActiveTask() {
		if active, err := ws.LoadActiveTask(); err == nil && active.ID == taskID {
			activeTask = active
		}
	}

	// Load work data from work directory
	work, err := ws.LoadWork(taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// Build response
	response := map[string]any{
		"active": activeTask != nil,
		"work": map[string]any{
			"metadata": map[string]any{
				"id":           work.Metadata.ID,
				"title":        work.Metadata.Title,
				"state":        work.Metadata.State,
				"external_key": work.Metadata.ExternalKey,
				"task_type":    work.Metadata.TaskType,
				"labels":       work.Metadata.Labels,
				"created_at":   work.Metadata.CreatedAt,
				"updated_at":   work.Metadata.UpdatedAt,
				"pull_request": work.Metadata.PullRequest,
			},
			"source": map[string]any{
				"type":    work.Source.Type,
				"ref":     work.Source.Ref,
				"read_at": work.Source.ReadAt,
			},
			"git": map[string]any{
				"branch":        work.Git.Branch,
				"base_branch":   work.Git.BaseBranch,
				"worktree_path": work.Git.WorktreePath,
			},
			"costs": work.Costs,
		},
	}

	// Add active task info if this is the active task
	if activeTask != nil {
		response["task"] = map[string]any{
			"id":             activeTask.ID,
			"state":          activeTask.State,
			"progress_phase": string(ComputeProgressPhase(ws, taskID)),
			"ref":            activeTask.Ref,
			"branch":         activeTask.Branch,
			"worktree_path":  activeTask.WorktreePath,
			"started":        activeTask.Started,
		}
	}

	// Add source content if available
	if content, err := ws.GetSourceContent(taskID); err == nil && content != "" {
		if workMap, ok := response["work"].(map[string]any); ok {
			workMap["description"] = content
		}
	}

	return NewResult("Work loaded").WithData(response), nil
}

func handleCosts(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	aggregate := GetBool(inv.Options, "aggregate")
	taskID := GetString(inv.Options, "task_id")
	if taskID == "" && len(inv.Args) > 0 {
		taskID = inv.Args[0]
	}

	cfg, _ := ws.LoadConfig()
	if aggregate {
		taskIDs, err := ws.ListWorks()
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks: %w", err)
		}

		var (
			tasks                             []map[string]any
			grandTotalInput, grandTotalOutput int
			grandTotalCached                  int
			grandTotalCost                    float64
		)
		for _, id := range taskIDs {
			work, err := ws.LoadWork(id)
			if err != nil || work == nil {
				continue
			}
			taskCost := buildTaskCostData(id, work, cfg)
			tasks = append(tasks, taskCost)

			costs := work.Costs
			grandTotalInput += costs.TotalInputTokens
			grandTotalOutput += costs.TotalOutputTokens
			grandTotalCached += costs.TotalCachedTokens
			grandTotalCost += costs.TotalCostUSD
		}

		response := map[string]any{
			"tasks": tasks,
			"grand_total": map[string]any{
				"input_tokens":  grandTotalInput,
				"output_tokens": grandTotalOutput,
				"total_tokens":  grandTotalInput + grandTotalOutput,
				"cached_tokens": grandTotalCached,
				"cost_usd":      grandTotalCost,
			},
		}

		if cfg != nil && cfg.Budget.Monthly.MaxCost > 0 {
			if state, err := ws.LoadMonthlyBudgetState(); err == nil && state != nil {
				response["monthly"] = map[string]any{
					"month":        state.Month,
					"spent":        state.Spent,
					"max_cost":     cfg.Budget.Monthly.MaxCost,
					"warning_at":   cfg.Budget.Monthly.WarningAt,
					"warning_sent": state.WarningSent,
				}
			}
		}

		return NewResult("Aggregate costs loaded").WithData(response), nil
	}

	if taskID == "" {
		if task := cond.GetActiveTask(); task != nil {
			taskID = task.ID
		}
	}
	if taskID == "" {
		return nil, errors.New("task ID is required")
	}

	work, err := ws.LoadWork(taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	return NewResult("Task costs loaded").WithData(buildTaskCostData(taskID, work, cfg)), nil
}

func handleBudgetMonthly(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()

	cfg := storage.NewDefaultWorkspaceConfig()
	var state *storage.MonthlyBudgetState
	if ws != nil {
		if loaded, err := ws.LoadConfig(); err == nil && loaded != nil {
			cfg = loaded
		}
		if loaded, err := ws.LoadMonthlyBudgetState(); err == nil {
			state = loaded
		}
	}

	enabled := cfg.Budget.Enabled
	response := map[string]any{
		"enabled": enabled,
	}
	if enabled {
		spent := 0.0
		warned := false
		if state != nil {
			spent = state.Spent
			warned = state.WarningSent
		}
		warningAt := cfg.Budget.Monthly.WarningAt
		if warningAt == 0 {
			warningAt = 0.8
		}
		currency := cfg.Budget.Monthly.Currency
		if currency == "" {
			currency = "USD"
		}
		response["max_cost"] = cfg.Budget.Monthly.MaxCost
		response["spent"] = spent
		response["remaining"] = cfg.Budget.Monthly.MaxCost - spent
		response["currency"] = currency
		response["warning_at"] = warningAt
		response["warned"] = warned
		response["limit_hit"] = spent >= cfg.Budget.Monthly.MaxCost
	}

	return NewResult("Monthly budget status loaded").WithData(response), nil
}

func handleSpecifications(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	taskID := GetString(inv.Options, "task_id")
	if taskID == "" && len(inv.Args) > 0 {
		taskID = inv.Args[0]
	}
	if taskID == "" {
		if task := cond.GetActiveTask(); task != nil {
			taskID = task.ID
		}
	}
	if taskID == "" {
		return nil, errors.New("task ID is required")
	}

	specs, err := ws.ListSpecificationsWithStatus(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list specifications: %w", err)
	}

	codeDir := cond.CodeDir()

	specList := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		specList = append(specList, map[string]any{
			"number":            spec.Number,
			"name":              fmt.Sprintf("spec-%d", spec.Number),
			"title":             spec.Title,
			"description":       spec.Content,
			"component":         spec.Component,
			"status":            spec.Status,
			"created_at":        spec.CreatedAt,
			"completed_at":      spec.CompletedAt,
			"implemented_files": deduplicateFiles(spec.ImplementedFiles, codeDir),
		})
	}

	return NewResult(fmt.Sprintf("Loaded %d specification(s)", len(specList))).WithData(map[string]any{
		"specifications": specList,
		"count":          len(specList),
	}), nil
}

// deduplicateFiles normalizes file paths (stripping the code directory prefix)
// and returns a deduplicated list preserving order.
func deduplicateFiles(files []string, codeDir string) []string {
	if len(files) == 0 {
		return files
	}
	seen := make(map[string]struct{}, len(files))
	result := make([]string, 0, len(files))
	prefix := codeDir + string(filepath.Separator)
	for _, f := range files {
		// Normalize: strip code directory prefix, leading ./
		norm := strings.TrimPrefix(f, prefix)
		norm = strings.TrimPrefix(norm, "./")
		norm = filepath.Clean(norm)
		if _, exists := seen[norm]; exists {
			continue
		}
		seen[norm] = struct{}{}
		result = append(result, norm)
	}

	return result
}

func handleSessions(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	taskID := GetString(inv.Options, "task_id")
	if taskID == "" && len(inv.Args) > 0 {
		taskID = inv.Args[0]
	}
	if taskID == "" {
		if task := cond.GetActiveTask(); task != nil {
			taskID = task.ID
		}
	}
	if taskID == "" {
		return nil, errors.New("task ID is required")
	}

	sessions, err := ws.ListSessions(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	sessionList := make([]map[string]any, 0, len(sessions))
	for _, session := range sessions {
		sess := map[string]any{
			"kind":       session.Kind,
			"started_at": session.Metadata.StartedAt,
			"ended_at":   session.Metadata.EndedAt,
			"agent":      session.Metadata.Agent,
		}
		if session.Usage != nil {
			sess["usage"] = map[string]any{
				"input_tokens":  session.Usage.InputTokens,
				"output_tokens": session.Usage.OutputTokens,
				"cached_tokens": session.Usage.CachedTokens,
				"cost_usd":      session.Usage.CostUSD,
			}
		}
		sessionList = append(sessionList, sess)
	}

	return NewResult(fmt.Sprintf("Loaded %d session(s)", len(sessionList))).WithData(map[string]any{
		"sessions": sessionList,
		"count":    len(sessionList),
	}), nil
}

func buildTaskCostData(taskID string, work *storage.TaskWork, cfg *storage.WorkspaceConfig) map[string]any {
	costs := work.Costs
	totalTokens := costs.TotalInputTokens + costs.TotalOutputTokens
	cachedPercent := 0.0
	if totalTokens > 0 && costs.TotalCachedTokens > 0 {
		cachedPercent = float64(costs.TotalCachedTokens) / float64(totalTokens) * 100
	}

	title := work.Metadata.Title
	if title == "" {
		title = "(untitled)"
	}

	response := map[string]any{
		"task_id":        taskID,
		"title":          title,
		"total_tokens":   totalTokens,
		"input_tokens":   costs.TotalInputTokens,
		"output_tokens":  costs.TotalOutputTokens,
		"cached_tokens":  costs.TotalCachedTokens,
		"cached_percent": cachedPercent,
		"total_cost_usd": costs.TotalCostUSD,
	}

	if len(costs.ByStep) > 0 {
		byStep := make(map[string]any, len(costs.ByStep))
		for step, stats := range costs.ByStep {
			byStep[step] = map[string]any{
				"input_tokens":  stats.InputTokens,
				"output_tokens": stats.OutputTokens,
				"cached_tokens": stats.CachedTokens,
				"total_tokens":  stats.InputTokens + stats.OutputTokens,
				"cost_usd":      stats.CostUSD,
				"calls":         stats.Calls,
			}
		}
		response["by_step"] = byStep
	}

	if budget := buildBudgetInfoMap(work, cfg); budget != nil {
		response["budget"] = budget
	}

	return response
}

func buildBudgetInfoMap(work *storage.TaskWork, cfg *storage.WorkspaceConfig) map[string]any {
	if work == nil || cfg == nil || !cfg.Budget.Enabled {
		return nil
	}

	budget := cfg.Budget.PerTask
	if work.Budget != nil {
		budget = *work.Budget
	}

	info := map[string]any{
		"max_tokens": budget.MaxTokens,
		"max_cost":   budget.MaxCost,
		"currency":   budget.Currency,
		"on_limit":   budget.OnLimit,
		"warning_at": budget.WarningAt,
	}
	if work.BudgetStatus != nil {
		info["warned"] = work.BudgetStatus.Warned
		info["limit_hit"] = work.BudgetStatus.LimitHit
	}

	if budget.MaxTokens == 0 && budget.MaxCost == 0 && budget.OnLimit == "" && budget.WarningAt == 0 {
		return nil
	}

	return info
}

// ComputeProgressPhase derives a high-level progress phase from task artifacts.
func ComputeProgressPhase(ws *storage.Workspace, taskID string) display.ProgressPhase {
	if ws == nil {
		return display.PhaseStarted
	}

	hasSpecs := false
	hasImplementedFiles := false
	hasReviews := false

	if specs, err := ws.ListSpecifications(taskID); err == nil && len(specs) > 0 {
		hasSpecs = true
		for _, specNum := range specs {
			if spec, err := ws.ParseSpecification(taskID, specNum); err == nil && len(spec.ImplementedFiles) > 0 {
				hasImplementedFiles = true

				break
			}
		}
	}

	if reviews, err := ws.ListReviews(taskID); err == nil && len(reviews) > 0 {
		hasReviews = true
	}

	return display.DetectProgressPhase(hasSpecs, hasImplementedFiles, hasReviews)
}
