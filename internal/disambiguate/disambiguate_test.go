package disambiguate

import (
	"testing"

	"github.com/spf13/cobra"
)

func newTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "mehr"}

	// Dummy run function to make commands "available"
	noop := func(cmd *cobra.Command, args []string) {}

	// Top-level commands
	root.AddCommand(
		&cobra.Command{Use: "plan", Short: "Create implementation specifications", Run: noop},
		&cobra.Command{Use: "plugins", Short: "Manage plugins", Run: noop},
		&cobra.Command{Use: "providers", Short: "Manage providers", Run: noop},
		&cobra.Command{Use: "start", Short: "Start a new task", Run: noop},
		&cobra.Command{Use: "status", Short: "Show task status", Run: noop},
		&cobra.Command{Use: "sync", Short: "Sync with provider", Run: noop},
		&cobra.Command{Use: "guide", Short: "Show next action", Run: noop},
		&cobra.Command{Use: "implement", Short: "Implement specifications", Run: noop},
		&cobra.Command{Use: "finish", Short: "Complete the task", Run: noop},
		&cobra.Command{Use: "continue", Short: "Continue work", Run: noop},
	)

	// Config with subcommands
	configCmd := &cobra.Command{Use: "config", Short: "Configuration management"}
	configCmd.AddCommand(
		&cobra.Command{Use: "validate", Short: "Validate configuration", Run: noop},
		&cobra.Command{Use: "init", Short: "Initialize configuration", Run: noop},
	)
	root.AddCommand(configCmd)

	// Agents with subcommands
	agentsCmd := &cobra.Command{Use: "agents", Short: "Agent management"}
	agentsCmd.AddCommand(
		&cobra.Command{Use: "list", Short: "List available agents", Run: noop},
		&cobra.Command{Use: "explain", Short: "Explain agent capabilities", Run: noop},
	)
	root.AddCommand(agentsCmd)

	return root
}

func TestFindPrefixMatches(t *testing.T) {
	root := newTestRoot()

	tests := []struct {
		name     string
		prefix   string
		expected int
		cmdNames []string
	}{
		{
			name:     "exact match - plan",
			prefix:   "plan",
			expected: 1,
			cmdNames: []string{"plan"},
		},
		{
			name:     "unambiguous prefix - pl",
			prefix:   "pl",
			expected: 2,
			cmdNames: []string{"plan", "plugins"},
		},
		{
			name:     "ambiguous s prefix",
			prefix:   "s",
			expected: 3,
			cmdNames: []string{"start", "status", "sync"},
		},
		{
			name:     "ambiguous st prefix",
			prefix:   "st",
			expected: 2,
			cmdNames: []string{"start", "status"},
		},
		{
			name:     "unambiguous gu prefix",
			prefix:   "gu",
			expected: 1,
			cmdNames: []string{"guide"},
		},
		{
			name:     "unambiguous imp prefix",
			prefix:   "imp",
			expected: 1,
			cmdNames: []string{"implement"},
		},
		{
			name:     "case insensitive",
			prefix:   "PL",
			expected: 2,
			cmdNames: []string{"plan", "plugins"},
		},
		{
			name:     "no match",
			prefix:   "xyz",
			expected: 0,
			cmdNames: nil,
		},
		{
			name:     "p matches plan, plugins, providers",
			prefix:   "p",
			expected: 3,
			cmdNames: []string{"plan", "plugins", "providers"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := FindPrefixMatches(root, tt.prefix)
			if len(matches) != tt.expected {
				t.Errorf("expected %d matches, got %d", tt.expected, len(matches))
				for _, m := range matches {
					t.Logf("  matched: %s", m.Command.Name())
				}
			}

			if tt.cmdNames != nil {
				for _, name := range tt.cmdNames {
					found := false
					for _, m := range matches {
						if m.Command.Name() == name {
							found = true

							break
						}
					}
					if !found {
						t.Errorf("expected to find command %q in matches", name)
					}
				}
			}
		})
	}
}

func TestResolveColonPath(t *testing.T) {
	root := newTestRoot()

	tests := []struct {
		name          string
		path          string
		expectedPath  []string
		expectMatches int // 0 = resolved, >0 = ambiguous
		expectError   bool
	}{
		{
			name:          "config:validate with full prefix",
			path:          "config:v",
			expectedPath:  []string{"config", "validate"},
			expectMatches: 0,
		},
		{
			name:          "config:init with full prefix",
			path:          "config:i",
			expectedPath:  []string{"config", "init"},
			expectMatches: 0,
		},
		{
			name:          "agents:list",
			path:          "a:l",
			expectedPath:  []string{"agents", "list"},
			expectMatches: 0,
		},
		{
			name:          "agents:explain",
			path:          "a:e",
			expectedPath:  []string{"agents", "explain"},
			expectMatches: 0,
		},
		{
			name:          "full names with colon",
			path:          "config:validate",
			expectedPath:  []string{"config", "validate"},
			expectMatches: 0,
		},
		{
			name:          "ambiguous first segment",
			path:          "c:v",
			expectedPath:  []string{}, // Empty because first segment is ambiguous
			expectMatches: 2,          // config, continue both match "c"
		},
		{
			name:          "trailing colon lists subcommands",
			path:          "config:",
			expectedPath:  []string{"config"},
			expectMatches: 2, // validate, init
		},
		{
			name:          "not a colon path",
			path:          "plan",
			expectError:   true,
			expectMatches: 0,
		},
		{
			name:          "no match in subcommand",
			path:          "config:xyz",
			expectedPath:  []string{"config"},
			expectError:   true,
			expectMatches: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, matches, err := ResolveColonPath(root, tt.path)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}

				return
			}

			if err != nil && tt.expectMatches == 0 {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if tt.expectMatches > 0 {
				if len(matches) != tt.expectMatches {
					t.Errorf("expected %d matches, got %d", tt.expectMatches, len(matches))
				}
			} else {
				// Check resolved path
				if len(resolved) != len(tt.expectedPath) {
					t.Errorf("expected path %v, got %v", tt.expectedPath, resolved)
				} else {
					for i, seg := range tt.expectedPath {
						if resolved[i] != seg {
							t.Errorf("path segment %d: expected %q, got %q", i, seg, resolved[i])
						}
					}
				}
			}
		})
	}
}

func TestFormatAmbiguousError(t *testing.T) {
	matches := []CommandMatch{
		{Command: &cobra.Command{Use: "start", Short: "Start a task"}},
		{Command: &cobra.Command{Use: "status", Short: "Show status"}},
	}

	err := FormatAmbiguousError("st", matches)

	if err == "" {
		t.Error("expected non-empty error message")
	}

	// Check it contains the command names
	if !contains(err, "start") || !contains(err, "status") {
		t.Errorf("error message should contain command names: %s", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
