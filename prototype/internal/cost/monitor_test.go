package cost

import (
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestCheckBudget_TaskLimit(t *testing.T) {
	work := &storage.TaskWork{
		Costs: storage.CostStats{
			TotalInputTokens:  100,
			TotalOutputTokens: 100,
			TotalCostUSD:      5.0,
		},
		Budget: &storage.BudgetConfig{
			MaxCost:   5.0,
			OnLimit:   "pause",
			WarningAt: 0.8,
		},
	}
	cfg := &storage.WorkspaceConfig{
		Budget: storage.BudgetSettings{
			PerTask: storage.BudgetConfig{
				OnLimit:   "warn",
				WarningAt: 0.8,
			},
		},
	}

	result := CheckBudget(work, cfg, nil)
	if result.Action != ActionPause {
		t.Fatalf("Action = %v, want %v", result.Action, ActionPause)
	}
	if result.Scope != "task" {
		t.Fatalf("Scope = %q, want %q", result.Scope, "task")
	}
}

func TestCheckBudget_MonthlyStop(t *testing.T) {
	work := &storage.TaskWork{
		Costs: storage.CostStats{
			TotalCostUSD: 1.0,
		},
	}
	cfg := &storage.WorkspaceConfig{
		Budget: storage.BudgetSettings{
			Monthly: storage.MonthlyBudgetSettings{
				MaxCost:   10.0,
				WarningAt: 0.8,
			},
		},
	}
	monthly := &storage.MonthlyBudgetState{
		Month: time.Now().Format("2006-01"),
		Spent: 10.0,
	}

	result := CheckBudget(work, cfg, monthly)
	if result.Action != ActionStop {
		t.Fatalf("Action = %v, want %v", result.Action, ActionStop)
	}
	if result.Scope != "monthly" {
		t.Fatalf("Scope = %q, want %q", result.Scope, "monthly")
	}
}

func TestCheckBudget_Warning(t *testing.T) {
	work := &storage.TaskWork{
		Costs: storage.CostStats{
			TotalCostUSD: 8.0,
		},
		Budget: &storage.BudgetConfig{
			MaxCost:   10.0,
			WarningAt: 0.8,
		},
	}
	cfg := &storage.WorkspaceConfig{
		Budget: storage.BudgetSettings{
			PerTask: storage.BudgetConfig{
				WarningAt: 0.8,
			},
		},
	}

	result := CheckBudget(work, cfg, nil)
	if result.Action != ActionWarn {
		t.Fatalf("Action = %v, want %v", result.Action, ActionWarn)
	}
	if result.Scope != "task" {
		t.Fatalf("Scope = %q, want %q", result.Scope, "task")
	}
}
