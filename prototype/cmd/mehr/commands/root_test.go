package commands

import (
	"errors"
	"testing"
)

// TestFormatError verifies error formatting behavior.
func TestFormatError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "simple error",
			err:      errors.New("failed"),
			expected: "Error: failed\n",
		},
		{
			name:     "multi-line error",
			err:      errors.New("line1\nline2"),
			expected: "line1\nline2\n",
		},
		{
			name:     "error with suggestions",
			err:      errors.New("cmd not found\n\nDid you mean: help"),
			expected: "cmd not found\n\nDid you mean: help\n",
		},
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "empty error",
			err:      errors.New(""),
			expected: "Error: \n",
		},
		{
			name:     "multi-line with trailing newline",
			err:      errors.New("error message\n"),
			expected: "error message\n\n",
		},
		{
			name:     "complex multi-line error",
			err:      errors.New("step 1 failed\nstep 2 failed\nstep 3 failed"),
			expected: "step 1 failed\nstep 2 failed\nstep 3 failed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatError(tt.err)
			if result != tt.expected {
				t.Errorf("FormatError() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestShouldAttemptDisambiguation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "unknown command error",
			err:      errors.New("unknown command \"foo\""),
			expected: true,
		},
		{
			name:     "unknown command in middle of message",
			err:      errors.New("error: unknown command found"),
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("connection refused"),
			expected: false,
		},
		{
			name:     "empty error",
			err:      errors.New(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldAttemptDisambiguation(tt.err)
			if got != tt.expected {
				t.Errorf("shouldAttemptDisambiguation(%q) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestGetSettings(t *testing.T) {
	// GetSettings returns the loaded settings (may be nil before PersistentPreRunE)
	got := GetSettings()
	// Just verify it doesn't panic - the value may be nil or non-nil
	_ = got
}

func TestResolveCommandArgs_NoColon(t *testing.T) {
	args := []string{"status"}
	resolved, err := resolveCommandArgs(args)
	if err != nil {
		t.Fatalf("resolveCommandArgs(%v) error = %v", args, err)
	}
	if resolved != nil {
		t.Errorf("resolveCommandArgs(%v) = %v, want nil", args, resolved)
	}
}

func TestResolveCommandArgs_EmptyArgs(t *testing.T) {
	resolved, err := resolveCommandArgs([]string{})
	if err != nil {
		t.Fatalf("resolveCommandArgs(empty) error = %v", err)
	}
	if resolved != nil {
		t.Errorf("resolveCommandArgs(empty) = %v, want nil", resolved)
	}
}

func TestRootCommand_PersistentFlags(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
	}{
		{"verbose flag", "verbose"},
		{"quiet flag", "quiet"},
		{"no-color flag", "no-color"},
		{"sandbox flag", "sandbox"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := rootCmd.PersistentFlags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("persistent flag %q not found", tt.flagName)
			}
		})
	}
}

func TestRootCommand_Groups(t *testing.T) {
	groups := rootCmd.Groups()
	if len(groups) == 0 {
		t.Error("root command has no groups")
	}

	expectedGroups := []string{"workflow", "task", "info", "config", "utility"}
	for _, expected := range expectedGroups {
		found := false
		for _, g := range groups {
			if g.ID == expected {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("root command missing group %q", expected)
		}
	}
}

func TestRootCommand_HasVersionSubcommand(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "version" {
			found = true

			break
		}
	}
	if !found {
		t.Error("root command missing 'version' subcommand")
	}
}

func TestRootCommand_SilenceSettings(t *testing.T) {
	if !rootCmd.SilenceUsage {
		t.Error("SilenceUsage should be true")
	}
	if !rootCmd.SilenceErrors {
		t.Error("SilenceErrors should be true")
	}
}

func TestRootCommand_CompletionDisabled(t *testing.T) {
	if !rootCmd.CompletionOptions.DisableDefaultCmd {
		t.Error("default completion command should be disabled")
	}
}
