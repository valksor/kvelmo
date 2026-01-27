//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

func TestLabelCommand_Properties(t *testing.T) {
	if labelCmd.Use != "label" {
		t.Errorf("Use = %q, want %q", labelCmd.Use, "label")
	}

	if labelCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if labelCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if labelCmd.RunE != nil {
		t.Error("RunE should be nil for parent command")
	}
}

func TestLabelCommand_HasSubcommands(t *testing.T) {
	expectedSubcommands := []string{"add", "remove", "set", "list"}

	for _, sub := range expectedSubcommands {
		found := false
		for _, cmd := range labelCmd.Commands() {
			if cmd.Name() == sub {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("Missing subcommand %q", sub)
		}
	}
}

func TestLabelAddCommand_Properties(t *testing.T) {
	if labelAddCmd.Use != "add <task-id> <label>..." {
		t.Errorf("Use = %q, want %q", labelAddCmd.Use, "add <task-id> <label>...")
	}

	if labelAddCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if labelAddCmd.Args == nil {
		t.Error("Args not set")
	}

	if labelAddCmd.RunE == nil {
		t.Error("RunE not set")
	}

	// Args should require at least 2 args (task-id + label)
	if err := labelAddCmd.Args(labelAddCmd, []string{}); err == nil {
		t.Error("Args validation should require at least 2 arguments")
	}
}

func TestLabelRemoveCommand_Properties(t *testing.T) {
	if labelRemoveCmd.Use != "remove <task-id> <label>..." {
		t.Errorf("Use = %q, want %q", labelRemoveCmd.Use, "remove <task-id> <label>...")
	}

	if labelRemoveCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if labelRemoveCmd.Args == nil {
		t.Error("Args not set")
	}

	if labelRemoveCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestLabelSetCommand_Properties(t *testing.T) {
	if labelSetCmd.Use != "set <task-id> <label>..." {
		t.Errorf("Use = %q, want %q", labelSetCmd.Use, "set <task-id> <label>...")
	}

	if labelSetCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if labelSetCmd.Args == nil {
		t.Error("Args not set")
	}

	if labelSetCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestLabelListCommand_Properties(t *testing.T) {
	if labelListCmd.Use != "list <task-id>" {
		t.Errorf("Use = %q, want %q", labelListCmd.Use, "list <task-id>")
	}

	if labelListCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if labelListCmd.Args == nil {
		t.Error("Args not set")
	}

	if labelListCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestLabelAddCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "label" {
			found = true

			break
		}
	}
	if !found {
		t.Error("label command not registered in root command")
	}
}

func TestLabelAddCommand_HasValidArgsPattern(t *testing.T) {
	// Check that the Args function is cobra.MinimumNArgs(2)
	// by verifying it requires at least 2 arguments
	minArgs := 2

	if labelAddCmd.Args == nil {
		t.Fatal("Args validation not configured")
	}

	// Test with insufficient args
	args := []string{"task-id-only"}
	err := labelAddCmd.Args(labelAddCmd, args)
	if err == nil {
		t.Errorf("Expected error with %d args, got nil", len(args))
	}

	// Test with sufficient args
	args = []string{"task-id", "label1"}
	err = labelAddCmd.Args(labelAddCmd, args)
	if err != nil && minArgs <= len(args) {
		t.Errorf("Expected no error with %d args (minimum %d), got %v", len(args), minArgs, err)
	}
}

func TestLabelRemoveCommand_HasValidArgsPattern(t *testing.T) {
	minArgs := 2

	if labelRemoveCmd.Args == nil {
		t.Fatal("Args validation not configured")
	}

	args := []string{"task-id-only"}
	err := labelRemoveCmd.Args(labelRemoveCmd, args)
	if err == nil {
		t.Errorf("Expected error with %d args, got nil", len(args))
	}

	args = []string{"task-id", "label1"}
	err = labelRemoveCmd.Args(labelRemoveCmd, args)
	if err != nil && minArgs <= len(args) {
		t.Errorf("Expected no error with %d args (minimum %d), got %v", len(args), minArgs, err)
	}
}

func TestLabelSetCommand_HasValidArgsPattern(t *testing.T) {
	// set command can accept 1+ args (task-id is required, labels are optional)
	minArgs := 1

	if labelSetCmd.Args == nil {
		t.Fatal("Args validation not configured")
	}

	// Test with no args (should fail since min is 1)
	args := []string{}
	err := labelSetCmd.Args(labelSetCmd, args)
	if err == nil {
		t.Errorf("Expected error with %d args, got nil", len(args))
	}

	args = []string{"task-id"}
	err = labelSetCmd.Args(labelSetCmd, args)
	if err != nil && minArgs <= len(args) {
		t.Errorf("Expected no error with %d args (minimum %d), got %v", len(args), minArgs, err)
	}
}

func TestLabelListCommand_HasValidArgsPattern(t *testing.T) {
	// list command requires exactly 1 arg (task-id)
	if labelListCmd.Args == nil {
		t.Fatal("Args validation not configured")
	}

	// Test with no args (should fail)
	args := []string{}
	err := labelListCmd.Args(labelListCmd, args)
	if err == nil {
		t.Error("Expected error with 0 args, got nil")
	}

	// Test with 1 arg (should pass)
	args = []string{"task-id"}
	err = labelListCmd.Args(labelListCmd, args)
	if err != nil {
		t.Errorf("Expected no error with 1 arg, got %v", err)
	}

	// Test with 2 args (should fail - requires exactly 1)
	args = []string{"task-id", "extra"}
	err = labelListCmd.Args(labelListCmd, args)
	if err == nil {
		t.Error("Expected error with 2 args, got nil")
	}
}

func TestLabelCommand_ShortDescription(t *testing.T) {
	expected := "Manage task labels"
	if labelCmd.Short != expected {
		t.Errorf("Short = %q, want %q", labelCmd.Short, expected)
	}
}

func TestLabelAddCommand_ShortDescription(t *testing.T) {
	expected := "Add labels to a task"
	if labelAddCmd.Short != expected {
		t.Errorf("Short = %q, want %q", labelAddCmd.Short, expected)
	}
}

func TestLabelRemoveCommand_ShortDescription(t *testing.T) {
	expected := "Remove labels from a task"
	if labelRemoveCmd.Short != expected {
		t.Errorf("Short = %q, want %q", labelRemoveCmd.Short, expected)
	}
}

func TestLabelSetCommand_ShortDescription(t *testing.T) {
	expected := "Set task labels (replace all)"
	if labelSetCmd.Short != expected {
		t.Errorf("Short = %q, want %q", labelSetCmd.Short, expected)
	}
}

func TestLabelListCommand_ShortDescription(t *testing.T) {
	expected := "List labels for a task"
	if labelListCmd.Short != expected {
		t.Errorf("Short = %q, want %q", labelListCmd.Short, expected)
	}
}

func TestLabelCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"Add",
		"remove",
		"list",
		"manage",
	}

	for _, substr := range contains {
		if !containsString(labelCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}
