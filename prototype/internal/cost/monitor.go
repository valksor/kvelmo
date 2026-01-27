package cost

import (
	"fmt"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// LimitAction represents the action to take when a budget threshold is hit.
type LimitAction string

const (
	ActionNone  LimitAction = "none"
	ActionWarn  LimitAction = "warn"
	ActionPause LimitAction = "pause"
	ActionStop  LimitAction = "stop"
)

// CheckResult captures the outcome of a budget check.
type CheckResult struct {
	Action  LimitAction
	Reason  string
	Scope   string // "task" or "monthly"
	Used    float64
	Limit   float64
	Percent float64
}

// CheckBudget evaluates task and monthly budgets and returns any required action.
func CheckBudget(work *storage.TaskWork, cfg *storage.WorkspaceConfig, monthly *storage.MonthlyBudgetState) CheckResult {
	if work == nil || cfg == nil {
		return CheckResult{Action: ActionNone}
	}

	taskBudget := effectiveTaskBudget(work, cfg)
	limitAction := resolveOnLimit(taskBudget.OnLimit, cfg.Budget.PerTask.OnLimit)
	warnAt := resolveWarningAt(taskBudget.WarningAt, cfg.Budget.PerTask.WarningAt)

	currentTokens := work.Costs.TotalInputTokens + work.Costs.TotalOutputTokens
	currentCost := work.Costs.TotalCostUSD

	// Monthly budget check.
	if monthly != nil && cfg.Budget.Monthly.MaxCost > 0 && monthly.Spent >= cfg.Budget.Monthly.MaxCost {
		return CheckResult{
			Action:  ActionStop,
			Scope:   "monthly",
			Reason:  "monthly budget exceeded",
			Used:    monthly.Spent,
			Limit:   cfg.Budget.Monthly.MaxCost,
			Percent: percent(monthly.Spent, cfg.Budget.Monthly.MaxCost),
		}
	}
	if monthly != nil && cfg.Budget.Monthly.MaxCost > 0 && cfg.Budget.Monthly.WarningAt > 0 &&
		monthly.Spent >= cfg.Budget.Monthly.MaxCost*cfg.Budget.Monthly.WarningAt {
		return CheckResult{
			Action:  ActionWarn,
			Scope:   "monthly",
			Reason:  fmt.Sprintf("monthly budget warning (%.0f%%)", cfg.Budget.Monthly.WarningAt*100),
			Used:    monthly.Spent,
			Limit:   cfg.Budget.Monthly.MaxCost,
			Percent: percent(monthly.Spent, cfg.Budget.Monthly.MaxCost),
		}
	}

	// Task limit check (cost).
	if taskBudget.MaxCost > 0 && currentCost >= taskBudget.MaxCost {
		return CheckResult{
			Action:  toAction(limitAction),
			Scope:   "task",
			Reason:  "task cost limit reached",
			Used:    currentCost,
			Limit:   taskBudget.MaxCost,
			Percent: percent(currentCost, taskBudget.MaxCost),
		}
	}

	// Task limit check (tokens).
	if taskBudget.MaxTokens > 0 && currentTokens >= taskBudget.MaxTokens {
		return CheckResult{
			Action:  toAction(limitAction),
			Scope:   "task",
			Reason:  "task token limit reached",
			Used:    float64(currentTokens),
			Limit:   float64(taskBudget.MaxTokens),
			Percent: percent(float64(currentTokens), float64(taskBudget.MaxTokens)),
		}
	}

	// Warning threshold (cost).
	if taskBudget.MaxCost > 0 && warnAt > 0 && currentCost >= taskBudget.MaxCost*warnAt {
		return CheckResult{
			Action:  ActionWarn,
			Scope:   "task",
			Reason:  fmt.Sprintf("task budget warning (%.0f%%)", warnAt*100),
			Used:    currentCost,
			Limit:   taskBudget.MaxCost,
			Percent: percent(currentCost, taskBudget.MaxCost),
		}
	}

	// Warning threshold (tokens).
	if taskBudget.MaxTokens > 0 && warnAt > 0 && float64(currentTokens) >= float64(taskBudget.MaxTokens)*warnAt {
		return CheckResult{
			Action:  ActionWarn,
			Scope:   "task",
			Reason:  fmt.Sprintf("task token warning (%.0f%%)", warnAt*100),
			Used:    float64(currentTokens),
			Limit:   float64(taskBudget.MaxTokens),
			Percent: percent(float64(currentTokens), float64(taskBudget.MaxTokens)),
		}
	}

	return CheckResult{Action: ActionNone}
}

func effectiveTaskBudget(work *storage.TaskWork, cfg *storage.WorkspaceConfig) storage.BudgetConfig {
	if work.Budget != nil {
		return *work.Budget
	}

	return cfg.Budget.PerTask
}

func resolveOnLimit(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	if fallback != "" {
		return fallback
	}

	return "warn"
}

func resolveWarningAt(primary, fallback float64) float64 {
	if primary > 0 {
		return primary
	}
	if fallback > 0 {
		return fallback
	}

	return 0.8
}

func toAction(value string) LimitAction {
	switch value {
	case "pause":
		return ActionPause
	case "stop":
		return ActionStop
	case "warn":
		return ActionWarn
	default:
		return ActionWarn
	}
}

func percent(used, limit float64) float64 {
	if limit <= 0 {
		return 0
	}

	return (used / limit) * 100
}
