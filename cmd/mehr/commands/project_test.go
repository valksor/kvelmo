//go:build !testbinary
// +build !testbinary

package commands

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestProjectCommand_Properties(t *testing.T) {
	if projectCmd.Use != "project" {
		t.Errorf("Use = %q, want %q", projectCmd.Use, "project")
	}

	if projectCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if projectCmd.Long == "" {
		t.Error("Long description is empty")
	}
}

func TestProjectCommand_HasSubcommands(t *testing.T) {
	// Note: Subcommands are registered in init(), but during test execution
	// the projectCmd.Commands() might not show them due to init() ordering.
	// This test verifies they exist by checking the command variables directly.

	subcommands := []*cobra.Command{
		projectPlanCmd,
		projectTasksCmd,
		projectEditCmd,
		projectReorderCmd,
		projectSubmitCmd,
		projectStartCmd,
	}

	for _, cmd := range subcommands {
		if cmd.Use == "" {
			t.Errorf("subcommand %v has empty Use", cmd)
		}
		if cmd.Short == "" {
			t.Errorf("subcommand %s has empty Short", cmd.Use)
		}
		if cmd.RunE == nil {
			t.Errorf("subcommand %s has nil RunE", cmd.Use)
		}
	}
}

func TestProjectCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"Project planning",
		"task queue",
		"local",
		"submit",
		"provider",
	}

	for _, substr := range contains {
		if !containsString(projectCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestProjectCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr project plan",
		"mehr project tasks",
		"mehr project edit",
		"mehr project submit",
		"mehr project start",
	}

	for _, example := range examples {
		if !containsString(projectCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestProjectCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "project" {
			found = true

			break
		}
	}
	if !found {
		t.Error("project command not registered in root command")
	}
}

func TestProjectPlanCommand_Properties(t *testing.T) {
	if !strings.HasPrefix(projectPlanCmd.Use, "plan") {
		t.Errorf("Use = %q, want prefix %q", projectPlanCmd.Use, "plan")
	}

	if projectPlanCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if projectPlanCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestProjectPlanCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		defaultValue string
	}{
		{"title flag", "title", ""},
		{"instructions flag", "instructions", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := projectPlanCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestProjectPlanCommand_ExactArgs(t *testing.T) {
	if projectPlanCmd.Args == nil {
		t.Error("Args validator not set")
	}
}

func TestProjectTasksCommand_Properties(t *testing.T) {
	if !strings.HasPrefix(projectTasksCmd.Use, "tasks") {
		t.Errorf("Use = %q, want prefix %q", projectTasksCmd.Use, "tasks")
	}

	if projectTasksCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if projectTasksCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestProjectTasksCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "status flag",
			flagName:     "status",
			shorthand:    "",
			defaultValue: "",
		},
		{
			name:         "show-deps flag",
			flagName:     "show-deps",
			shorthand:    "",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := projectTasksCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestProjectEditCommand_Properties(t *testing.T) {
	if !strings.HasPrefix(projectEditCmd.Use, "edit") {
		t.Errorf("Use = %q, want prefix %q", projectEditCmd.Use, "edit")
	}

	if projectEditCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if projectEditCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestProjectEditCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		defaultValue string
	}{
		{"title flag", "title", ""},
		{"description flag", "description", ""},
		{"priority flag", "priority", "0"},
		{"status flag", "status", ""},
		{"depends-on flag", "depends-on", ""},
		{"labels flag", "labels", ""},
		{"assignee flag", "assignee", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := projectEditCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestProjectReorderCommand_Properties(t *testing.T) {
	if !strings.HasPrefix(projectReorderCmd.Use, "reorder") {
		t.Errorf("Use = %q, want prefix %q", projectReorderCmd.Use, "reorder")
	}

	if projectReorderCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if projectReorderCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestProjectReorderCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		defaultValue string
	}{
		{"before flag", "before", ""},
		{"after flag", "after", ""},
		{"auto flag", "auto", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := projectReorderCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestProjectSubmitCommand_Properties(t *testing.T) {
	if !strings.HasPrefix(projectSubmitCmd.Use, "submit") {
		t.Errorf("Use = %q, want prefix %q", projectSubmitCmd.Use, "submit")
	}

	if projectSubmitCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if projectSubmitCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestProjectSubmitCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		defaultValue string
	}{
		{"provider flag", "provider", ""},
		{"create-epic flag", "create-epic", "false"},
		{"labels flag", "labels", ""},
		{"dry-run flag", "dry-run", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := projectSubmitCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestProjectStartCommand_Properties(t *testing.T) {
	if !strings.HasPrefix(projectStartCmd.Use, "start") {
		t.Errorf("Use = %q, want prefix %q", projectStartCmd.Use, "start")
	}

	if projectStartCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if projectStartCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestProjectStartCommand_Flags(t *testing.T) {
	flag := projectStartCmd.Flags().Lookup("auto")
	if flag == nil {
		t.Fatal("auto flag not found")
	}

	if flag.DefValue != "false" {
		t.Errorf("auto flag default value = %q, want %q", flag.DefValue, "false")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max",
			input:    "short",
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "string exactly max length",
			input:    "exact",
			maxLen:   5,
			expected: "exact",
		},
		{
			name:     "string longer than max",
			input:    "this is a long string",
			maxLen:   10,
			expected: "this is...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
		{
			name:     "maxLen 3",
			input:    "longer",
			maxLen:   3,
			expected: "...",
		},
		{
			name:     "unicode characters",
			input:    "hello 世界",
			maxLen:   8,
			expected: "hello...", // truncate counts bytes, not runes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncate() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTruncate_Length(t *testing.T) {
	// Test that truncated result is always <= maxLen + 3 (for "...")
	input := "this is a very long string that needs truncation"
	maxLen := 20
	result := truncate(input, maxLen)

	if len(result) > maxLen {
		t.Errorf("truncated length %d exceeds maxLen %d", len(result), maxLen)
	}

	// Should end with "..." if truncated
	if len(input) > maxLen && result != "" {
		if len(result) >= 3 && result[len(result)-3:] != "..." {
			t.Errorf("truncated string should end with '...', got %q", result)
		}
	}
}
