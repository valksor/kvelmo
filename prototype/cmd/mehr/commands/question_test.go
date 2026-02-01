//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

func TestQuestionCommand_Properties(t *testing.T) {
	if questionCmd.Use != "question [message]" {
		t.Errorf("Use = %q, want %q", questionCmd.Use, "question [message]")
	}

	if questionCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if questionCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if questionCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestQuestionCommand_HasAliases(t *testing.T) {
	expectedAliases := []string{"ask", "q"}

	if len(questionCmd.Aliases) != len(expectedAliases) {
		t.Fatalf("Aliases count = %d, want %d", len(questionCmd.Aliases), len(expectedAliases))
	}

	for i, expected := range expectedAliases {
		if questionCmd.Aliases[i] != expected {
			t.Errorf("Alias[%d] = %q, want %q", i, questionCmd.Aliases[i], expected)
		}
	}
}

func TestQuestionCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "question [message]" {
			found = true

			break
		}
	}
	if !found {
		t.Error("question command not registered in root command")
	}
}

func TestQuestionCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"question",
		"WHEN TO USE",
		"RELATED COMMANDS",
	}

	for _, substr := range contains {
		if !containsString(questionCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestJoinArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "empty args",
			args:     []string{},
			expected: "",
		},
		{
			name:     "single arg",
			args:     []string{"hello"},
			expected: "hello",
		},
		{
			name:     "multiple args",
			args:     []string{"hello", "world"},
			expected: "hello world",
		},
		{
			name:     "three args",
			args:     []string{"Why", "did", "you?"},
			expected: "Why did you?",
		},
		{
			name:     "args with spaces",
			args:     []string{"arg with spaces", "another"},
			expected: "arg with spaces another",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinArgs(tt.args)
			if got != tt.expected {
				t.Errorf("joinArgs(%v) = %q, want %q", tt.args, got, tt.expected)
			}
		})
	}
}
