//go:build !testbinary
// +build !testbinary

package commands

import "testing"

func TestBudgetCommand_Structure(t *testing.T) {
	if budgetCmd.Use != "budget" {
		t.Errorf("Use = %q, want %q", budgetCmd.Use, "budget")
	}
	if budgetCmd.Short == "" {
		t.Error("Short description is empty")
	}
	if budgetCmd.Long == "" {
		t.Error("Long description is empty")
	}
}

func TestBudgetCommand_RegisteredInRoot(t *testing.T) {
	if !hasCommand(rootCmd, "budget") {
		t.Error("budget command not registered with rootCmd")
	}
}

func TestBudgetCommand_Subcommands(t *testing.T) {
	expected := []string{"status", "set", "task", "resume", "reset"}
	for _, name := range expected {
		if !hasCommand(budgetCmd, name) {
			t.Errorf("budget command missing subcommand %q", name)
		}
	}
}
