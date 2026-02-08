package commands

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "status",
			Aliases:      []string{"st"},
			Description:  "Show current task status",
			Category:     "info",
			RequiresTask: false,
			MutatesState: false,
		},
		Handler: handleStatus,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "cost",
			Description:  "Show token usage and costs",
			Category:     "info",
			RequiresTask: true,
			MutatesState: false,
		},
		Handler: handleCost,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "budget",
			Description:  "Show token budget status",
			Category:     "info",
			RequiresTask: true,
			MutatesState: false,
		},
		Handler: handleBudget,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "list",
			Aliases:      []string{"ls"},
			Description:  "List all tasks",
			Category:     "info",
			RequiresTask: false,
			MutatesState: false,
		},
		Handler: handleList,
	})

	Register(Command{
		Info: CommandInfo{
			Name:        "specification",
			Aliases:     []string{"spec"},
			Description: "View specification details",
			Category:    "info",
			Args: []CommandArg{
				{Name: "number", Required: false, Description: "Specification number to view"},
			},
			RequiresTask: true,
			MutatesState: false,
		},
		Handler: handleSpecification,
	})
}

func handleStatus(ctx context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	status, err := cond.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	data := StatusData{
		State:     status.State,
		TaskID:    status.TaskID,
		Title:     status.Title,
		Ref:       status.Ref,
		Branch:    status.Branch,
		SpecCount: status.Specifications,
	}

	message := "State: " + status.State
	if data.TaskID != "" {
		message = fmt.Sprintf("Task: %s | State: %s", data.TaskID, status.State)
	}

	return NewStatusResult(message, data.State, data.TaskID, data), nil
}

func handleCost(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	task := cond.GetActiveTask()
	if task == nil {
		return nil, ErrNoActiveTask
	}

	work := cond.GetTaskWork()
	if work == nil {
		return nil, errors.New("unable to load task work")
	}

	costs := work.Costs
	totalTokens := costs.TotalInputTokens + costs.TotalOutputTokens
	cachedPercent := 0.0
	if totalTokens > 0 && costs.TotalCachedTokens > 0 {
		cachedPercent = float64(costs.TotalCachedTokens) / float64(totalTokens) * 100
	}

	data := CostData{
		TotalTokens:   totalTokens,
		InputTokens:   costs.TotalInputTokens,
		OutputTokens:  costs.TotalOutputTokens,
		CachedTokens:  costs.TotalCachedTokens,
		CachedPercent: cachedPercent,
		TotalCostUSD:  costs.TotalCostUSD,
	}

	message := fmt.Sprintf("Total: %d tokens ($%.4f)", totalTokens, costs.TotalCostUSD)

	return NewCostResult(message, data), nil
}

func handleBudget(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("no workspace available")
	}

	task := cond.GetActiveTask()
	if task == nil {
		return nil, ErrNoActiveTask
	}

	work := cond.GetTaskWork()
	if work == nil {
		return nil, errors.New("unable to load task work")
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	taskBudget := cfg.Budget.PerTask
	// Allow work-level budget override
	if work.Budget != nil {
		taskBudget = *work.Budget
	}

	// Check if any budget is configured
	if taskBudget.MaxCost == 0 && taskBudget.MaxTokens == 0 {
		return NewResult("No budget configured"), nil
	}

	costs := work.Costs
	var data BudgetData
	var message string

	if taskBudget.MaxCost > 0 {
		pct := (costs.TotalCostUSD / taskBudget.MaxCost) * 100
		data = BudgetData{
			Type:       "cost",
			Used:       fmt.Sprintf("$%.4f", costs.TotalCostUSD),
			Max:        fmt.Sprintf("$%.2f", taskBudget.MaxCost),
			Percentage: pct,
			Warned:     taskBudget.WarningAt > 0 && pct >= taskBudget.WarningAt*100,
		}
		message = fmt.Sprintf("Budget: $%.4f / $%.2f (%.1f%%)", costs.TotalCostUSD, taskBudget.MaxCost, pct)
	} else if taskBudget.MaxTokens > 0 {
		totalTokens := costs.TotalInputTokens + costs.TotalOutputTokens
		pct := (float64(totalTokens) / float64(taskBudget.MaxTokens)) * 100
		data = BudgetData{
			Type:       "token",
			Used:       strconv.Itoa(totalTokens),
			Max:        strconv.Itoa(taskBudget.MaxTokens),
			Percentage: pct,
			Warned:     taskBudget.WarningAt > 0 && pct >= taskBudget.WarningAt*100,
		}
		message = fmt.Sprintf("Budget: %d / %d tokens (%.1f%%)", totalTokens, taskBudget.MaxTokens, pct)
	} else {
		return NewResult("No budget limits configured"), nil
	}

	return NewBudgetResult(message, data), nil
}

func handleList(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("no workspace available")
	}

	taskIDs, err := ws.ListWorks()
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Get active task for state lookup
	activeID := ""
	var activeTask *storage.ActiveTask
	if ws.HasActiveTask() {
		activeTask, _ = ws.LoadActiveTask()
		if activeTask != nil {
			activeID = activeTask.ID
		}
	}

	items := make([]TaskListItem, 0, len(taskIDs))
	for _, id := range taskIDs {
		work, _ := ws.LoadWork(id)
		title := "(no title)"
		if work != nil && work.Metadata.Title != "" {
			title = work.Metadata.Title
		}
		state := "idle"
		if id == activeID && activeTask != nil {
			state = activeTask.State
		}
		items = append(items, TaskListItem{
			ID:    id,
			Title: title,
			State: state,
		})
	}

	message := fmt.Sprintf("%d task(s)", len(items))

	return NewListResult(message, items), nil
}

func handleSpecification(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	args := inv.Args
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("no workspace available")
	}

	task := cond.GetActiveTask()
	if task == nil {
		return nil, ErrNoActiveTask
	}

	specs, err := ws.ListSpecificationsWithStatus(task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load specifications: %w", err)
	}

	// If no number specified, list all specs
	if len(args) == 0 {
		items := make([]SpecificationItem, 0, len(specs))
		for _, spec := range specs {
			items = append(items, SpecificationItem{
				Number: spec.Number,
				Title:  spec.Title,
				Status: spec.Status,
			})
		}

		return NewListResult(fmt.Sprintf("%d specification(s)", len(items)), items), nil
	}

	// View specific specification
	num, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.New("specification number must be an integer")
	}

	for _, spec := range specs {
		if spec.Number == num {
			item := SpecificationItem{
				Number:      spec.Number,
				Title:       spec.Title,
				Description: spec.Description,
				Status:      spec.Status,
			}

			return NewResult(fmt.Sprintf("Specification #%d: %s", num, spec.Title)).WithData(item), nil
		}
	}

	return nil, fmt.Errorf("specification #%d not found", num)
}
