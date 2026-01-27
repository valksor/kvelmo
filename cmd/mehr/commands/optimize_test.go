//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

func TestOptimizeCommand_Properties(t *testing.T) {
	if optimizeCmd.Use != "optimize --task <queue>/<task-id>" {
		t.Errorf("Use = %q, want %q", optimizeCmd.Use, "optimize --task <queue>/<task-id>")
	}

	if optimizeCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if optimizeCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if optimizeCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestOptimizeCommand_ShortDescription(t *testing.T) {
	expected := "AI optimizes a task based on its notes"
	if optimizeCmd.Short != expected {
		t.Errorf("Short = %q, want %q", optimizeCmd.Short, expected)
	}
}

func TestOptimizeCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"Review the current task",
		"Enhance the title",
		"Expand the description",
		"Suggest relevant labels",
		"Explain improvements",
	}

	for _, substr := range contains {
		if !containsString(optimizeCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestOptimizeCommand_ExamplesContains(t *testing.T) {
	examples := []string{
		`mehr optimize --task=quick-tasks/task-1`,
		`--agent claude-opus`,
		`mehr note --task=quick-tasks/task-1`,
	}

	for _, example := range examples {
		if !containsString(optimizeCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestOptimizeCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "optimize --task <queue>/<task-id>" {
			found = true

			break
		}
	}
	if !found {
		t.Error("optimize command not registered in root command")
	}
}

func TestOptimizeCommand_HasRequiredFlags(t *testing.T) {
	flags := optimizeCmd.Flags()

	requiredFlags := []string{
		"task",
		"agent",
	}

	for _, flagName := range requiredFlags {
		if flags.Lookup(flagName) == nil {
			t.Errorf("Missing required flag: %s", flagName)
		}
	}
}

func TestOptimizeCommand_TaskFlagRequired(t *testing.T) {
	flags := optimizeCmd.Flags()
	taskFlag := flags.Lookup("task")
	if taskFlag == nil {
		t.Fatal("task flag not found")
	}
}

func TestOptimizeCommand_SeeAlsoReferences(t *testing.T) {
	// Check that related commands are referenced
	seeAlso := []string{
		"mehr quick",
		"mehr note",
		"mehr export",
	}

	for _, ref := range seeAlso {
		if !containsString(optimizeCmd.Long, ref) {
			t.Errorf("Long description should reference %s", ref)
		}
	}
}
