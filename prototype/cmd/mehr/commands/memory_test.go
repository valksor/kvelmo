//go:build !testbinary
// +build !testbinary

package commands

import (
	"strings"
	"testing"
)

func TestMemoryCommand_Properties(t *testing.T) {
	if memoryCmd.Use != "memory" {
		t.Errorf("Use = %q, want %q", memoryCmd.Use, "memory")
	}

	if memoryCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if memoryCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if memoryCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestMemoryCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "search flag",
			flagName:     "search",
			shorthand:    "s",
			defaultValue: "",
		},
		{
			name:         "limit flag",
			flagName:     "limit",
			shorthand:    "l",
			defaultValue: "5",
		},
		{
			name:         "type flag",
			flagName:     "type",
			shorthand:    "t",
			defaultValue: "[]",
		},
		{
			name:         "task flag",
			flagName:     "task",
			shorthand:    "",
			defaultValue: "",
		},
		{
			name:         "index flag",
			flagName:     "index",
			shorthand:    "",
			defaultValue: "",
		},
		{
			name:         "stats flag",
			flagName:     "stats",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "clear flag",
			flagName:     "clear",
			shorthand:    "",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := memoryCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}

			if tt.shorthand != "" {
				shorthand := memoryCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestMemoryCommand_GroupID(t *testing.T) {
	if memoryCmd.GroupID != "utility" {
		t.Errorf("GroupID = %q, want %q", memoryCmd.GroupID, "utility")
	}
}

func TestMemoryCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"semantic memory",
		"search",
		"index",
		"stats",
		"clear",
		"embeddings",
	}

	for _, substr := range contains {
		if !containsString(memoryCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestMemoryCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr memory search",
		"mehr memory index",
		"mehr memory stats",
		"mehr memory clear",
		"--limit",
		"--task",
	}

	for _, example := range examples {
		if !containsString(memoryCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestMemoryCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "memory" {
			found = true

			break
		}
	}
	if !found {
		t.Error("memory command not registered in root command")
	}
}

func TestMemoryCommand_NoAliases(t *testing.T) {
	if len(memoryCmd.Aliases) > 0 {
		t.Errorf("memory command should have no aliases, got %v", memoryCmd.Aliases)
	}
}

func TestIndentText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		indent   string
		expected string
	}{
		{
			name:     "single line",
			input:    "single line",
			indent:   "  ",
			expected: "  single line",
		},
		{
			name:     "multi line",
			input:    "line1\nline2\nline3",
			indent:   ">>",
			expected: ">>line1\n>>line2\n>>line3",
		},
		{
			name:     "empty string",
			input:    "",
			indent:   "  ",
			expected: "  ", // Split returns [""], then indent is added
		},
		{
			name:     "no indent",
			input:    "text",
			indent:   "",
			expected: "text",
		},
		{
			name:     "trailing newline preserved",
			input:    "line1\nline2\n",
			indent:   "  ",
			expected: "  line1\n  line2\n  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indentText(tt.input, tt.indent)
			if got != tt.expected {
				t.Errorf("indentText() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIndentText_EachLine(t *testing.T) {
	// Test that each line gets indented
	input := "first\nsecond\nthird"
	result := indentText(input, "  ")

	lines := strings.Split(result, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "  ") && line != "" {
			t.Errorf("line %d not indented: %q", i, line)
		}
	}
}
