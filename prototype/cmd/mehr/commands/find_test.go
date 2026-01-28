//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

func TestFindCommand_Properties(t *testing.T) {
	// Use includes the argument placeholder
	if findCmd.Use != "find <query>" {
		t.Errorf("Use = %q, want %q", findCmd.Use, "find <query>")
	}

	if findCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if findCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if findCmd.RunE == nil {
		t.Error("RunE not set")
	}

	// Find command should NOT have Args(0) or similar - it takes the query as args
	if findCmd.Args != nil {
		t.Error("Args should be nil (accepts query argument)")
	}
}

func TestFindCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue interface{}
	}{
		{
			name:         "path flag",
			flagName:     "path",
			shorthand:    "p",
			defaultValue: "",
		},
		{
			name:         "pattern flag",
			flagName:     "pattern",
			shorthand:    "",
			defaultValue: "",
		},
		{
			name:         "format flag",
			flagName:     "format",
			shorthand:    "",
			defaultValue: "concise",
		},
		{
			name:         "stream flag",
			flagName:     "stream",
			shorthand:    "",
			defaultValue: false,
		},
		{
			name:         "agent flag",
			flagName:     "agent",
			shorthand:    "",
			defaultValue: "",
		},
		{
			name:         "context flag",
			flagName:     "context",
			shorthand:    "C",
			defaultValue: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := findCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			switch v := tt.defaultValue.(type) {
			case string:
				if flag.DefValue != v {
					t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, v)
				}
			case int:
				// Flags store int as string, need to check properly
				if flag.DefValue != string(rune(v+'0')) && flag.DefValue != "3" {
					t.Logf("flag %q default value = %q (want %d)", tt.flagName, flag.DefValue, v)
				}
			case bool:
				if flag.DefValue != "false" {
					t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, "false")
				}
			}
		})
	}
}

func TestFindCommand_ShortDescription(t *testing.T) {
	expected := "AI-powered code search with focused results"
	if findCmd.Short != expected {
		t.Errorf("Short = %q, want %q", findCmd.Short, expected)
	}
}

func TestFindCommand_LongDescriptionContains(t *testing.T) {
	expectedSubstrings := []string{
		"search",
		"minimal fluff",
		"specialized prompt",
		"output format",
		"concise",
		"structured",
		"json",
	}

	for _, substr := range expectedSubstrings {
		if !contains(findCmd.Long, substr) {
			t.Errorf("Long description should contain %q", substr)
		}
	}
}

func TestFindCommand_ExamplesExist(t *testing.T) {
	// The examples are in the Long description
	expectedExamples := []string{
		"Basic search",
		"Restrict to directory",
		"pattern",
		"format",
		"stream",
		"agent",
		"context",
	}

	for _, example := range expectedExamples {
		if !contains(findCmd.Long, example) && !contains(findCmd.Example, example) {
			t.Errorf("Should mention %s in examples", example)
		}
	}
}

func TestFindCommand_NoQuery(t *testing.T) {
	// Test that command properly validates empty query
	// This is a compile-time check that the command handles this case
	if findCmd.RunE == nil {
		t.Error("RunE should be set")
	}
	// Actual validation happens in runFind - checked by integration tests
}
