//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

func TestQuickCommand_Properties(t *testing.T) {
	if quickCmd.Use != "quick <description>" {
		t.Errorf("Use = %q, want %q", quickCmd.Use, "quick <description>")
	}

	if quickCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if quickCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if quickCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestQuickCommand_ShortDescription(t *testing.T) {
	expected := "Create a quick task without full planning"
	if quickCmd.Short != expected {
		t.Errorf("Short = %q, want %q", quickCmd.Short, expected)
	}
}

func TestQuickCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"Quick tasks",
		"queue",
		"Iterated on with notes",
		"Optimized by AI",
		"Submitted to external providers",
		"Exported to markdown",
	}

	for _, substr := range contains {
		if !containsString(quickCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestQuickCommand_ExamplesContains(t *testing.T) {
	examples := []string{
		`mehr quick "fix typo in README.md line 42"`,
		`--label bug`,
		`--priority 1`,
		`--title "Auth Fix"`,
	}

	for _, example := range examples {
		if !containsString(quickCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestQuickCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "quick <description>" {
			found = true

			break
		}
	}
	if !found {
		t.Error("quick command not registered in root command")
	}
}

func TestQuickCommand_HasRequiredFlags(t *testing.T) {
	flags := quickCmd.Flags()

	requiredFlags := []string{
		"label",
		"priority",
		"queue",
		"title",
		"agent",
	}

	for _, flagName := range requiredFlags {
		if flags.Lookup(flagName) == nil {
			t.Errorf("Missing required flag: %s", flagName)
		}
	}
}

func TestQuickCommand_PriorityDefault(t *testing.T) {
	flag := quickCmd.Flags().Lookup("priority")
	if flag == nil {
		t.Fatal("priority flag not found")
	}

	// Check flag exists and default is documented
	if flag.DefValue != "2" {
		t.Errorf("priority default = %q, want %q", flag.DefValue, "2")
	}

	// Also check description documents the levels
	if !containsString(flag.Usage, "1=high") && !containsString(flag.Usage, "high") {
		t.Error("priority flag description should document priority levels")
	}
}

func TestQuickCommand_QueueDefault(t *testing.T) {
	flag := quickCmd.Flags().Lookup("queue")
	if flag == nil {
		t.Fatal("queue flag not found")
	}

	// The flag default is "", but the code uses "quick-tasks" as default
	// The flag description documents this
	if !containsString(flag.Usage, "default: quick-tasks") {
		t.Error("queue flag description should document default as 'quick-tasks'")
	}
}
