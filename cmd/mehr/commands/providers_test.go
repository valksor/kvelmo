//go:build !testbinary
// +build !testbinary

package commands

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestProvidersCommand_Structure(t *testing.T) {
	tests := []struct {
		cmd       *cobra.Command
		wantArgs  cobra.PositionalArgs
		name      string
		wantUse   string
		wantShort string
	}{
		{
			name:      "providers command",
			cmd:       providersCmd,
			wantUse:   "providers",
			wantShort: "List and manage task providers",
		},
		{
			name:      "providers list subcommand",
			cmd:       providersListCmd,
			wantUse:   "list",
			wantShort: "List all available providers",
		},
		{
			name:      "providers info subcommand",
			cmd:       providersInfoCmd,
			wantUse:   "info <provider>",
			wantShort: "Show provider information",
			wantArgs:  cobra.ExactArgs(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.Use != tt.wantUse {
				t.Errorf("Use = %q, want %q", tt.cmd.Use, tt.wantUse)
			}
			if tt.cmd.Short != tt.wantShort {
				t.Errorf("Short = %q, want %q", tt.cmd.Short, tt.wantShort)
			}
			if tt.wantArgs != nil && tt.cmd.Args != nil {
				// Verify that args validation is set (ExactArgs(1))
				// by checking it rejects empty args
				err := tt.cmd.Args(tt.cmd, []string{})
				if err == nil {
					t.Errorf("Expected args validation to reject empty args, but it didn't")
				}
			}
		})
	}
}

func TestProvidersCommand_SubcommandsRegistered(t *testing.T) {
	// Verify that providersCmd has the expected subcommands
	subcommands := providersCmd.Commands()
	if len(subcommands) != 3 {
		t.Fatalf("expected 3 subcommands, got %d", len(subcommands))
	}

	subcommandNames := make(map[string]bool)
	for _, cmd := range subcommands {
		subcommandNames[cmd.Name()] = true
	}

	expectedSubcommands := []string{"list", "info", "status"}
	for _, expected := range expectedSubcommands {
		if !subcommandNames[expected] {
			t.Errorf("missing subcommand %q", expected)
		}
	}
}

func TestProvidersCommand_HasParent(t *testing.T) {
	// Verify that providersCmd is added to rootCmd
	if !hasCommand(rootCmd, "providers") {
		t.Error("providers command not registered with rootCmd")
	}
}

// Helper function to check if a command exists in the command tree.
func hasCommand(cmd *cobra.Command, name string) bool {
	for _, subCmd := range cmd.Commands() {
		if subCmd.Name() == name {
			return true
		}
	}

	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests for providers.go helper functions and run functions
// ─────────────────────────────────────────────────────────────────────────────

// TestGetProviderInfo tests the getProviderInfo helper function.
func TestGetProviderInfo(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		wantNil       bool
		expectedName  string
		expectedDesc  string
		expectedUsage string
	}{
		{
			name:          "file provider",
			provider:      "file",
			wantNil:       false,
			expectedName:  "File Provider",
			expectedDesc:  "Load tasks from individual markdown files",
			expectedUsage: "mehr start file:path/to/task.md",
		},
		{
			name:         "file provider - shorthand",
			provider:     "f",
			wantNil:      false,
			expectedName: "File Provider",
		},
		{
			name:         "dir provider",
			provider:     "dir",
			wantNil:      false,
			expectedName: "Directory Provider",
		},
		{
			name:         "directory - full name",
			provider:     "directory",
			wantNil:      false,
			expectedName: "Directory Provider",
		},
		{
			name:         "github provider",
			provider:     "github",
			wantNil:      false,
			expectedName: "GitHub Provider",
			expectedDesc: "Load tasks from GitHub issues and PRs",
		},
		{
			name:         "github - shorthand gh",
			provider:     "gh",
			wantNil:      false,
			expectedName: "GitHub Provider",
		},
		{
			name:         "github - shorthand git",
			provider:     "git",
			wantNil:      false,
			expectedName: "GitHub Provider",
		},
		{
			name:         "jira provider",
			provider:     "jira",
			wantNil:      false,
			expectedName: "Jira Provider",
			expectedDesc: "Load tasks from Atlassian Jira",
		},
		{
			name:         "linear provider",
			provider:     "linear",
			wantNil:      false,
			expectedName: "Linear Provider",
		},
		{
			name:         "notion provider",
			provider:     "notion",
			wantNil:      false,
			expectedName: "Notion Provider",
		},
		{
			name:         "youtrack provider",
			provider:     "youtrack",
			wantNil:      false,
			expectedName: "YouTrack Provider",
		},
		{
			name:         "youtrack - shorthand yt",
			provider:     "yt",
			wantNil:      false,
			expectedName: "YouTrack Provider",
		},
		{
			name:         "wrike provider",
			provider:     "wrike",
			wantNil:      false,
			expectedName: "Wrike Provider",
		},
		{
			name:     "unknown provider",
			provider: "unknown",
			wantNil:  true,
		},
		{
			name:     "empty string",
			provider: "",
			wantNil:  true,
		},
		{
			name:     "case insensitive - GitHub",
			provider: "GitHub",
			wantNil:  true, // getProviderInfo uses a lowercase switch, so "GitHub" won't match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getProviderInfo(tt.provider)

			if tt.wantNil {
				if got != nil {
					t.Errorf("getProviderInfo(%q) = %v, want nil", tt.provider, got)
				}

				return
			}

			if got == nil {
				t.Fatalf("getProviderInfo(%q) returned nil, want non-nil", tt.provider)

				return
			}

			if got.Name != tt.expectedName {
				t.Errorf("getProviderInfo(%q).Name = %q, want %q", tt.provider, got.Name, tt.expectedName)
			}

			if tt.expectedDesc != "" && got.Description != tt.expectedDesc {
				t.Errorf("getProviderInfo(%q).Description = %q, want %q", tt.provider, got.Description, tt.expectedDesc)
			}

			if tt.expectedUsage != "" && got.Usage != tt.expectedUsage {
				t.Errorf("getProviderInfo(%q).Usage = %q, want %q", tt.provider, got.Usage, tt.expectedUsage)
			}
		})
	}
}

// TestGetProviderInfo_HasEnvVars tests that provider info includes env vars.
func TestGetProviderInfo_HasEnvVars(t *testing.T) {
	providersWithEnvVars := map[string]string{
		"github":   "GITHUB_TOKEN",
		"jira":     "JIRA_TOKEN",
		"linear":   "LINEAR_API_KEY",
		"notion":   "NOTION_TOKEN",
		"youtrack": "YOUTRACK_TOKEN",
		"wrike":    "WRIKE_TOKEN",
	}

	for provider, expectedEnvVar := range providersWithEnvVars {
		t.Run(provider, func(t *testing.T) {
			info := getProviderInfo(provider)
			if info == nil {
				t.Fatalf("getProviderInfo(%q) returned nil", provider)

				return
			}

			if len(info.EnvVars) == 0 {
				t.Errorf("getProviderInfo(%q).EnvVars is empty, expected to contain %q", provider, expectedEnvVar)

				return
			}

			found := false
			for _, envVar := range info.EnvVars {
				if envVar == expectedEnvVar {
					found = true

					break
				}
			}
			if !found {
				t.Errorf("getProviderInfo(%q).EnvVars = %v, expected to contain %q", provider, info.EnvVars, expectedEnvVar)
			}
		})
	}
}

// TestGetProviderInfo_HasConfig tests that provider info includes config examples.
func TestGetProviderInfo_HasConfig(t *testing.T) {
	providersWithConfig := []string{"github", "jira", "linear", "notion", "youtrack", "wrike"}

	for _, provider := range providersWithConfig {
		t.Run(provider, func(t *testing.T) {
			info := getProviderInfo(provider)
			if info == nil {
				t.Fatalf("getProviderInfo(%q) returned nil", provider)

				return
			}

			if len(info.Config) == 0 {
				t.Errorf("getProviderInfo(%q).Config is empty, expected config examples", provider)
			}
		})
	}
}

// TestRunProvidersList tests the runProvidersList function.
func TestRunProvidersList(t *testing.T) {
	tc := NewTestContext(t)
	// Add providersCmd parent first
	tc.AddSubCommand(providersCmd)
	// Then add the subcommand
	tc.AddSubCommand(providersListCmd)

	err := tc.Execute("providers", "list")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := tc.StdoutString()

	// Check for expected providers in the output
	expectedProviders := []string{
		"file", "dir", "github", "gitlab", "jira", "linear", "notion", "wrike", "youtrack",
	}

	for _, provider := range expectedProviders {
		if !strings.Contains(output, provider) {
			t.Errorf("output should contain %q provider", provider)
		}
	}

	// Check for headers
	if !strings.Contains(output, "SCHEME") {
		t.Error("output should contain SCHEME header")
	}
	if !strings.Contains(output, "PROVIDER") {
		t.Error("output should contain PROVIDER header")
	}
	if !strings.Contains(output, "DESCRIPTION") {
		t.Error("output should contain DESCRIPTION header")
	}

	// Check for a usage section
	if !strings.Contains(output, "Usage:") {
		t.Error("output should contain usage section")
	}
	if !strings.Contains(output, "mehr start") {
		t.Error("output should contain example commands")
	}
}

// TestRunProvidersInfo_KnownProvider tests the runProvidersInfo function with known providers.
func TestRunProvidersInfo_KnownProvider(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		expectedInOut []string
	}{
		{
			name:     "github",
			provider: "github",
			expectedInOut: []string{
				"Provider: GitHub Provider",
				"Scheme: github",
				"Load tasks from GitHub issues and PRs",
				"GITHUB_TOKEN",
				"owner:",
				"repo:",
			},
		},
		{
			name:     "jira",
			provider: "jira",
			expectedInOut: []string{
				"Provider: Jira Provider",
				"JIRA_TOKEN",
				"url:",
			},
		},
		{
			name:     "file",
			provider: "file",
			expectedInOut: []string{
				"Provider: File Provider",
				"file:",
				"Load tasks from individual markdown files",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTestContext(t)
			// Add providersCmd parent first
			tc.AddSubCommand(providersCmd)
			// Then add the subcommand
			tc.AddSubCommand(providersInfoCmd)

			err := tc.Execute("providers", "info", tt.provider)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}

			output := tc.StdoutString()

			for _, expected := range tt.expectedInOut {
				if !strings.Contains(output, expected) {
					t.Errorf("output should contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

// TestRunProvidersInfo_UnknownProvider tests the runProvidersInfo function with an unknown provider.
func TestRunProvidersInfo_UnknownProvider(t *testing.T) {
	tc := NewTestContext(t)
	// Add providersCmd parent first
	tc.AddSubCommand(providersCmd)
	// Then add the subcommand
	tc.AddSubCommand(providersInfoCmd)

	err := tc.Execute("providers", "info", "unknown_provider_xyz")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := tc.StdoutString()

	if !strings.Contains(output, "Unknown provider: unknown_provider_xyz") {
		t.Errorf("output should contain unknown provider error, got: %s", output)
	}
	if !strings.Contains(output, "mehr providers list") {
		t.Errorf("output should suggest running 'mehr providers list', got: %s", output)
	}
}
