//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

func TestSimplifyCommand_Properties(t *testing.T) {
	if simplifyCmd.Use != "simplify" {
		t.Errorf("Use = %q, want %q", simplifyCmd.Use, "simplify")
	}

	if simplifyCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if simplifyCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if simplifyCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestSimplifyCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "no-checkpoint flag",
			flagName:     "no-checkpoint",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "agent flag",
			flagName:     "agent",
			shorthand:    "",
			defaultValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := simplifyCmd.Flags().Lookup(tt.flagName)
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

func TestSimplifyCommand_ShortDescription(t *testing.T) {
	expected := "Simplify content based on current workflow state"
	if simplifyCmd.Short != expected {
		t.Errorf("Short = %q, want %q", simplifyCmd.Short, expected)
	}
}

func TestSimplifyCommand_LongDescriptionContains(t *testing.T) {
	expectedSubstrings := []string{
		"workflow state",
		"specification",
		"checkpoint",
		"implementing",
	}

	for _, substr := range expectedSubstrings {
		if !contains(simplifyCmd.Long, substr) {
			t.Errorf("Long description should contain %q", substr)
		}
	}
}

func TestSimplifyCommand_ExamplesExist(t *testing.T) {
	if simplifyCmd.Example == "" {
		t.Error("Example is empty")
	}

	// Check that key examples are mentioned
	expectedExamples := []string{
		"Auto-detect",
		"verbose",
		"agent",
		"no-checkpoint",
	}

	for _, example := range expectedExamples {
		if !contains(simplifyCmd.Example, example) && !contains(simplifyCmd.Long, example) {
			t.Errorf("Should mention %s in examples", example)
		}
	}
}
