package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// MonthlyBudgetState tracks monthly spending for a workspace.
type MonthlyBudgetState struct {
	Month       string  `yaml:"month"`
	Spent       float64 `yaml:"spent"`
	WarningSent bool    `yaml:"warning_sent,omitempty"`
}

// BudgetStatePath returns the path to the monthly budget state file.
func (w *Workspace) BudgetStatePath() string {
	return filepath.Join(w.workspaceRoot, "budget.yaml")
}

// LoadMonthlyBudgetState loads the monthly budget state, initializing if missing.
func (w *Workspace) LoadMonthlyBudgetState() (*MonthlyBudgetState, error) {
	path := w.BudgetStatePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &MonthlyBudgetState{Month: currentBudgetMonth()}, nil
		}

		return nil, fmt.Errorf("read budget state: %w", err)
	}

	var state MonthlyBudgetState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse budget state: %w", err)
	}

	// Roll over on month change.
	month := currentBudgetMonth()
	if state.Month != month {
		return &MonthlyBudgetState{Month: month}, nil
	}

	return &state, nil
}

// SaveMonthlyBudgetState persists the monthly budget state.
func (w *Workspace) SaveMonthlyBudgetState(state *MonthlyBudgetState) error {
	if err := os.MkdirAll(w.workspaceRoot, 0o755); err != nil {
		return fmt.Errorf("ensure workspace root: %w", err)
	}
	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal budget state: %w", err)
	}

	return os.WriteFile(w.BudgetStatePath(), data, 0o644)
}

// AddMonthlyBudgetSpend increments monthly spend, handling month rollover.
func (w *Workspace) AddMonthlyBudgetSpend(delta float64) error {
	if delta == 0 {
		return nil
	}

	state, err := w.LoadMonthlyBudgetState()
	if err != nil {
		return err
	}
	state.Spent += delta

	return w.SaveMonthlyBudgetState(state)
}

// ResetMonthlyBudget resets the monthly budget tracking to zero.
func (w *Workspace) ResetMonthlyBudget() error {
	state := &MonthlyBudgetState{
		Month:       currentBudgetMonth(),
		Spent:       0,
		WarningSent: false,
	}

	return w.SaveMonthlyBudgetState(state)
}

func currentBudgetMonth() string {
	return time.Now().Format("2006-01")
}
