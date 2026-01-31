//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

func TestResetCommand_Properties(t *testing.T) {
	if resetCmd.Use != "reset" {
		t.Errorf("Use = %q, want %q", resetCmd.Use, "reset")
	}

	if resetCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if resetCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if resetCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestResetCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "yes flag",
			flagName:     "yes",
			shorthand:    "y",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := resetCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			// Check default value
			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}

			// Check shorthand if specified
			if tt.shorthand != "" {
				shorthand := resetCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestResetCommand_ShortDescription(t *testing.T) {
	expected := "Reset workflow state to idle without losing work"
	if resetCmd.Short != expected {
		t.Errorf("Short = %q, want %q", resetCmd.Short, expected)
	}
}

func TestResetCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"workflow state",
		"idle",
		"preserves",
		"specifications",
		"notes",
		"code changes",
	}

	for _, substr := range contains {
		if !containsString(resetCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestResetCommand_DocumentsUseCases(t *testing.T) {
	useCases := []string{
		"Agent hangs",
		"stuck in planning",
		"retry a step",
	}

	for _, useCase := range useCases {
		if !containsString(resetCmd.Long, useCase) {
			t.Errorf("Long description does not document use case %q", useCase)
		}
	}
}

func TestResetCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr reset",
		"--yes",
	}

	for _, example := range examples {
		if !containsString(resetCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestResetCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "reset" {
			found = true

			break
		}
	}
	if !found {
		t.Error("reset command not registered in root command")
	}
}

func TestResetCommand_NoAliases(t *testing.T) {
	if len(resetCmd.Aliases) > 0 {
		t.Errorf("reset command should have no aliases, got %v", resetCmd.Aliases)
	}
}
