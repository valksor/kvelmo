package conductor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/valksor/go-mehrhof/internal/cost"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

var (
	// ErrBudgetPaused indicates the task was paused due to budget limits.
	ErrBudgetPaused = errors.New("task paused due to budget limit")
	// ErrBudgetStopped indicates the task was stopped due to budget limits.
	ErrBudgetStopped = errors.New("task stopped due to budget limit")
)

func (c *Conductor) checkBudgets(ctx context.Context, phase string) error {
	if c.activeTask == nil {
		return nil
	}

	if err := c.workspace.FlushUsage(); err != nil {
		return fmt.Errorf("flush usage before budget check: %w", err)
	}

	work, err := c.workspace.LoadWork(c.activeTask.ID)
	if err != nil {
		return fmt.Errorf("load work for budget check: %w", err)
	}

	cfg, err := c.workspace.LoadConfig()
	if err != nil {
		c.logError(fmt.Errorf("load workspace config for budget check: %w", err))

		return nil
	}

	monthlyState, err := c.workspace.LoadMonthlyBudgetState()
	if err != nil {
		c.logError(fmt.Errorf("load monthly budget state: %w", err))
	}

	result := cost.CheckBudget(work, cfg, monthlyState)
	switch result.Action {
	case cost.ActionNone:
		return nil
	case cost.ActionWarn:
		return c.handleBudgetWarning(work, monthlyState, result, phase)
	case cost.ActionPause:
		return c.handleBudgetPause(ctx, work, result, phase)
	case cost.ActionStop:
		return c.handleBudgetStop(ctx, work, result, phase)
	}

	return nil
}

func (c *Conductor) handleBudgetWarning(work *storage.TaskWork, monthly *storage.MonthlyBudgetState, result cost.CheckResult, phase string) error {
	if result.Scope == "task" {
		if work.BudgetStatus != nil && work.BudgetStatus.Warned {
			return nil
		}
		c.publishProgress(fmt.Sprintf("Budget warning (%s): %s (%.1f%%)", phase, result.Reason, result.Percent), 0)
		if work.BudgetStatus == nil {
			work.BudgetStatus = &storage.BudgetStatus{}
		}
		work.BudgetStatus.Warned = true
		work.BudgetStatus.WarnedAt = time.Now()
		if err := c.workspace.SaveWork(work); err != nil {
			return fmt.Errorf("save budget warning status: %w", err)
		}

		return nil
	}

	if result.Scope == "monthly" && monthly != nil {
		if monthly.WarningSent {
			return nil
		}
		c.publishProgress(fmt.Sprintf("Monthly budget warning (%s): %s (%.1f%%)", phase, result.Reason, result.Percent), 0)
		monthly.WarningSent = true
		if err := c.workspace.SaveMonthlyBudgetState(monthly); err != nil {
			return fmt.Errorf("save monthly budget warning: %w", err)
		}
	}

	return nil
}

func (c *Conductor) handleBudgetPause(ctx context.Context, work *storage.TaskWork, result cost.CheckResult, phase string) error {
	if work.BudgetStatus == nil {
		work.BudgetStatus = &storage.BudgetStatus{}
	}
	if !work.BudgetStatus.LimitHit {
		work.BudgetStatus.LimitHit = true
		work.BudgetStatus.LimitHitAt = time.Now()
		if err := c.workspace.SaveWork(work); err != nil {
			return fmt.Errorf("save budget limit status: %w", err)
		}
	}

	c.publishProgress(fmt.Sprintf("Paused (%s): %s (%.1f%%)", phase, result.Reason, result.Percent), 0)
	c.activeTask.State = string(workflow.StatePaused)
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		c.logError(fmt.Errorf("save active task after budget pause: %w", err))
	}
	_ = c.machine.Dispatch(ctx, workflow.EventPause)

	return ErrBudgetPaused
}

func (c *Conductor) handleBudgetStop(ctx context.Context, work *storage.TaskWork, result cost.CheckResult, phase string) error {
	if work.BudgetStatus == nil {
		work.BudgetStatus = &storage.BudgetStatus{}
	}
	if !work.BudgetStatus.LimitHit {
		work.BudgetStatus.LimitHit = true
		work.BudgetStatus.LimitHitAt = time.Now()
		if err := c.workspace.SaveWork(work); err != nil {
			return fmt.Errorf("save budget limit status: %w", err)
		}
	}

	c.publishProgress(fmt.Sprintf("Stopped (%s): %s (%.1f%%)", phase, result.Reason, result.Percent), 0)
	c.activeTask.State = string(workflow.StateFailed)
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		c.logError(fmt.Errorf("save active task after budget stop: %w", err))
	}
	_ = c.machine.Dispatch(ctx, workflow.EventAbort)

	return ErrBudgetStopped
}
