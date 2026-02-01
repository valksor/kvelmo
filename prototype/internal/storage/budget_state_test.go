package storage

import (
	"context"
	"testing"
	"time"
)

func TestMonthlyBudgetState_Rollover(t *testing.T) {
	repoRoot := t.TempDir()
	homeDir := t.TempDir()
	cfg := NewDefaultWorkspaceConfig()
	cfg.Storage.HomeDir = homeDir

	ws, err := OpenWorkspace(context.Background(), repoRoot, cfg)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	state, err := ws.LoadMonthlyBudgetState()
	if err != nil {
		t.Fatalf("load monthly budget state: %v", err)
	}
	currentMonth := time.Now().Format("2006-01")
	if state.Month != currentMonth {
		t.Fatalf("Month = %q, want %q", state.Month, currentMonth)
	}

	state.Month = "2000-01"
	state.Spent = 42
	if err := ws.SaveMonthlyBudgetState(state); err != nil {
		t.Fatalf("save monthly budget state: %v", err)
	}

	rolled, err := ws.LoadMonthlyBudgetState()
	if err != nil {
		t.Fatalf("load monthly budget state after rollover: %v", err)
	}
	if rolled.Month != currentMonth {
		t.Fatalf("Month after rollover = %q, want %q", rolled.Month, currentMonth)
	}
	if rolled.Spent != 0 {
		t.Fatalf("Spent after rollover = %v, want 0", rolled.Spent)
	}
}

func TestMonthlyBudgetState_SaveLoad(t *testing.T) {
	repoRoot := t.TempDir()
	homeDir := t.TempDir()
	cfg := NewDefaultWorkspaceConfig()
	cfg.Storage.HomeDir = homeDir

	ws, err := OpenWorkspace(context.Background(), repoRoot, cfg)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	state := &MonthlyBudgetState{
		Month:       time.Now().Format("2006-01"),
		Spent:       12.34,
		WarningSent: true,
	}
	if err := ws.SaveMonthlyBudgetState(state); err != nil {
		t.Fatalf("save monthly budget state: %v", err)
	}

	loaded, err := ws.LoadMonthlyBudgetState()
	if err != nil {
		t.Fatalf("load monthly budget state: %v", err)
	}
	if loaded.Month != state.Month {
		t.Fatalf("Month = %q, want %q", loaded.Month, state.Month)
	}
	if loaded.Spent != state.Spent {
		t.Fatalf("Spent = %v, want %v", loaded.Spent, state.Spent)
	}
	if loaded.WarningSent != state.WarningSent {
		t.Fatalf("WarningSent = %v, want %v", loaded.WarningSent, state.WarningSent)
	}
}
