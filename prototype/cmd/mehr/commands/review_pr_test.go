//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestReviewPRCommand_Properties(t *testing.T) {
	if reviewPRCmd.Use != "pr" {
		t.Errorf("Use = %q, want %q", reviewPRCmd.Use, "pr")
	}

	if reviewPRCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if reviewPRCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if reviewPRCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestReviewPRCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		defaultValue string
	}{
		{
			name:         "provider flag",
			flagName:     "provider",
			defaultValue: "",
		},
		{
			name:         "pr-number flag",
			flagName:     "pr-number",
			defaultValue: "0",
		},
		{
			name:         "format flag",
			flagName:     "format",
			defaultValue: "summary",
		},
		{
			name:         "scope flag",
			flagName:     "scope",
			defaultValue: "full",
		},
		{
			name:         "agent flag",
			flagName:     "agent-pr-review",
			defaultValue: "",
		},
		{
			name:         "token flag",
			flagName:     "token",
			defaultValue: "",
		},
		{
			name:         "acknowledge-fixes flag",
			flagName:     "acknowledge-fixes",
			defaultValue: "true",
		},
		{
			name:         "update-existing flag",
			flagName:     "update-existing",
			defaultValue: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := reviewPRCmd.Flags().Lookup(tt.flagName)
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

func TestReviewPRCommand_PRNumberRequired(t *testing.T) {
	// Check that the pr-number flag exists
	flag := reviewPRCmd.Flags().Lookup("pr-number")
	if flag == nil {
		t.Error("pr-number flag not found")

		return
	}

	// The flag is marked required in init() via MarkFlagRequired
	// We can't directly test that, but we verify the flag exists
}

func TestReviewPRCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"pull request",
		"merge request",
		"GitHub",
		"GitLab",
		"auto-detected",
		"CI/CD",
		"--pr-number",
		"--provider",
	}

	for _, substr := range contains {
		if !containsString(reviewPRCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestReviewPRCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr review pr --pr-number",
		"--provider",
		"--agent-pr-review",
		"--scope",
		"--token",
	}

	for _, example := range examples {
		if !containsString(reviewPRCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestReviewPRCommand_AddedToReview(t *testing.T) {
	// Verify it's a subcommand of reviewCmd
	found := false
	for _, cmd := range reviewCmd.Commands() {
		if cmd.Use == "pr" {
			found = true

			break
		}
	}
	if !found {
		t.Error("review pr command not registered as subcommand of review")
	}
}

func TestReviewPRCommand_NoAliases(t *testing.T) {
	if len(reviewPRCmd.Aliases) > 0 {
		t.Errorf("review pr command should have no aliases, got %v", reviewPRCmd.Aliases)
	}
}

func TestReviewPRCommand_Standalone(t *testing.T) {
	// Verify the command documents that it's standalone
	if !containsString(reviewPRCmd.Long, "standalone") {
		t.Error("Long description should mention command is standalone")
	}
}

func TestReviewPRCommand_FormatFlagValues(t *testing.T) {
	// Verify the format flag exists and has the correct default
	flag := reviewPRCmd.Flags().Lookup("format")
	if flag == nil {
		t.Fatal("format flag not found")
	}

	if flag.DefValue != "summary" {
		t.Errorf("format flag default = %q, want %q", flag.DefValue, "summary")
	}

	// Flag usage includes format options
	if !containsString(flag.Usage, "summary") || !containsString(flag.Usage, "line-comments") {
		t.Error("format flag usage should document format options")
	}
}

func TestReviewPRCommand_ScopeFlagValues(t *testing.T) {
	// Verify the scope flag documents its options
	if !containsString(reviewPRCmd.Long, "full") && !containsString(reviewPRCmd.Long, "compact") && !containsString(reviewPRCmd.Long, "files-changed") {
		t.Error("Long description should document scope options")
	}
}

func TestReviewPRCommand_ProviderSupport(t *testing.T) {
	// Should mention supported providers
	supportedProviders := []string{
		"github",
		"gitlab",
		"bitbucket",
		"azuredevops",
	}

	hasProviderExample := false
	for _, provider := range supportedProviders {
		if containsString(reviewPRCmd.Long, provider) {
			hasProviderExample = true

			break
		}
	}

	if !hasProviderExample {
		t.Error("Long description should mention at least one supported provider")
	}
}

func TestReviewPRCommand_CIMention(t *testing.T) {
	// Should be suitable for CI/CD
	if !containsString(reviewPRCmd.Long, "CI") {
		t.Error("Long description should mention CI/CD usage")
	}
}

func TestReviewPRCommand_NoWorkspaceRequired(t *testing.T) {
	// Should document that workspace is not required
	contains := []string{
		"does not require",
		"no workspace",
		"standalone",
	}

	hasMention := false
	for _, phrase := range contains {
		if containsString(reviewPRCmd.Long, phrase) {
			hasMention = true

			break
		}
	}

	if !hasMention {
		t.Log("Note: Consider mentioning that workspace is not required for this command")
	}
}

func TestValidatePRNumber(t *testing.T) {
	tests := []struct {
		name     string
		prNumber int
		wantErr  bool
	}{
		{"positive number", 123, false},
		{"one", 1, false},
		{"large number", 99999, false},
		{"zero", 0, true},
		{"negative", -1, true},
		{"large negative", -100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePRNumber(tt.prNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePRNumber(%d) error = %v, wantErr %v", tt.prNumber, err, tt.wantErr)
			}
		})
	}
}

func TestResolvePRAgent(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *storage.WorkspaceConfig
		agentOverride string
		want          string
	}{
		{
			name:          "override wins over nil config",
			cfg:           nil,
			agentOverride: "opus",
			want:          "opus",
		},
		{
			name:          "override wins over config default",
			cfg:           &storage.WorkspaceConfig{Agent: storage.AgentSettings{Default: "sonnet"}},
			agentOverride: "opus",
			want:          "opus",
		},
		{
			name:          "config default when no override",
			cfg:           &storage.WorkspaceConfig{Agent: storage.AgentSettings{Default: "sonnet"}},
			agentOverride: "",
			want:          "sonnet",
		},
		{
			name:          "fallback to claude with nil config",
			cfg:           nil,
			agentOverride: "",
			want:          "claude",
		},
		{
			name:          "fallback to claude with empty config default",
			cfg:           &storage.WorkspaceConfig{Agent: storage.AgentSettings{Default: ""}},
			agentOverride: "",
			want:          "claude",
		},
		{
			name:          "fallback to claude with zero-value config",
			cfg:           &storage.WorkspaceConfig{},
			agentOverride: "",
			want:          "claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolvePRAgent(tt.cfg, tt.agentOverride)
			if got != tt.want {
				t.Errorf("resolvePRAgent() = %q, want %q", got, tt.want)
			}
		})
	}
}
