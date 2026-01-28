package conductor

import (
	"testing"

	"github.com/valksor/go-mehrhof/internal/agent"
)

func TestStandaloneDiffMode_Constants(t *testing.T) {
	// Verify the constants have expected values
	tests := []struct {
		mode     StandaloneDiffMode
		expected string
	}{
		{DiffModeUncommitted, "uncommitted"},
		{DiffModeBranch, "branch"},
		{DiffModeRange, "range"},
		{DiffModeFiles, "files"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if string(tt.mode) != tt.expected {
				t.Errorf("StandaloneDiffMode = %q, want %q", string(tt.mode), tt.expected)
			}
		})
	}
}

func TestStandaloneDiffOptions_DefaultContext(t *testing.T) {
	opts := StandaloneDiffOptions{
		Mode:    DiffModeUncommitted,
		Context: 0, // Zero should default to 3
	}

	// Context is used when calling GetStandaloneDiff
	// Default is applied there, not in the struct
	if opts.Context != 0 {
		t.Errorf("Context should be 0 before processing, got %d", opts.Context)
	}
}

func TestParseStandaloneReviewVerdict(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected string
	}{
		{
			name:     "approved uppercase",
			response: "This code is APPROVED",
			expected: "APPROVED",
		},
		{
			name:     "approved lowercase",
			response: "This code is approved and looks good",
			expected: "APPROVED",
		},
		{
			name:     "needs changes",
			response: "NEEDS_CHANGES: please fix these issues",
			expected: "NEEDS_CHANGES",
		},
		{
			name:     "changes requested",
			response: "CHANGES_REQUESTED by reviewer",
			expected: "NEEDS_CHANGES",
		},
		{
			name:     "comment only",
			response: "Here are some observations",
			expected: "COMMENT",
		},
		{
			name:     "empty response",
			response: "",
			expected: "COMMENT",
		},
		{
			name:     "mixed case approved",
			response: "approved with minor suggestions",
			expected: "APPROVED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseStandaloneReviewVerdict(tt.response)
			if result != tt.expected {
				t.Errorf("parseStandaloneReviewVerdict(%q) = %q, want %q",
					tt.response, result, tt.expected)
			}
		})
	}
}

func TestExtractStandaloneReviewSummary(t *testing.T) {
	tests := []struct {
		name     string
		response string
		contains string
	}{
		{
			name: "with summary heading ##",
			response: `## Summary

This PR adds authentication to the API.

## Issues

Some issues here.`,
			contains: "authentication",
		},
		{
			name: "with summary heading #",
			response: `# Summary

This PR improves performance.

## Details`,
			contains: "performance",
		},
		{
			name: "no summary heading - uses first lines",
			response: `This code implements caching.
It uses Redis for storage.
Additional implementation details follow.`,
			contains: "caching",
		},
		{
			name:     "empty response",
			response: "",
			contains: "",
		},
		{
			name:     "heading only lines skipped",
			response: "# Heading\n## Another Heading\nActual content here",
			contains: "Actual content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractStandaloneReviewSummary(tt.response)
			if tt.contains != "" {
				if result == "" {
					t.Errorf("extractStandaloneReviewSummary() returned empty, want to contain %q",
						tt.contains)
				} else if !containsIgnoreCase(result, tt.contains) {
					t.Errorf("extractStandaloneReviewSummary() = %q, want to contain %q",
						result, tt.contains)
				}
			}
		})
	}
}

func TestStandaloneReviewOptions_EmbedsDiffOptions(t *testing.T) {
	opts := StandaloneReviewOptions{
		StandaloneDiffOptions: StandaloneDiffOptions{
			Mode:       DiffModeBranch,
			BaseBranch: "main",
			Context:    5,
		},
		Agent: "claude",
	}

	// Verify embedded options are accessible
	if opts.Mode != DiffModeBranch {
		t.Errorf("Mode = %v, want %v", opts.Mode, DiffModeBranch)
	}
	if opts.BaseBranch != "main" {
		t.Errorf("BaseBranch = %s, want main", opts.BaseBranch)
	}
	if opts.Context != 5 {
		t.Errorf("Context = %d, want 5", opts.Context)
	}
	if opts.Agent != "claude" {
		t.Errorf("Agent = %s, want claude", opts.Agent)
	}
}

func TestStandaloneReviewOptions_FixModeFields(t *testing.T) {
	tests := []struct {
		name             string
		applyFixes       bool
		createCheckpoint bool
	}{
		{
			name:             "fix mode with checkpoint",
			applyFixes:       true,
			createCheckpoint: true,
		},
		{
			name:             "fix mode without checkpoint",
			applyFixes:       true,
			createCheckpoint: false,
		},
		{
			name:             "review only mode",
			applyFixes:       false,
			createCheckpoint: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := StandaloneReviewOptions{
				StandaloneDiffOptions: StandaloneDiffOptions{
					Mode: DiffModeUncommitted,
				},
				ApplyFixes:       tt.applyFixes,
				CreateCheckpoint: tt.createCheckpoint,
			}

			if opts.ApplyFixes != tt.applyFixes {
				t.Errorf("ApplyFixes = %v, want %v", opts.ApplyFixes, tt.applyFixes)
			}
			if opts.CreateCheckpoint != tt.createCheckpoint {
				t.Errorf("CreateCheckpoint = %v, want %v", opts.CreateCheckpoint, tt.createCheckpoint)
			}
		})
	}
}

func TestStandaloneSimplifyOptions_EmbedsDiffOptions(t *testing.T) {
	opts := StandaloneSimplifyOptions{
		StandaloneDiffOptions: StandaloneDiffOptions{
			Mode:  DiffModeFiles,
			Files: []string{"main.go", "util.go"},
		},
		Agent:            "opus",
		CreateCheckpoint: true,
	}

	// Verify embedded options are accessible
	if opts.Mode != DiffModeFiles {
		t.Errorf("Mode = %v, want %v", opts.Mode, DiffModeFiles)
	}
	if len(opts.Files) != 2 {
		t.Errorf("Files count = %d, want 2", len(opts.Files))
	}
	if opts.Agent != "opus" {
		t.Errorf("Agent = %s, want opus", opts.Agent)
	}
	if !opts.CreateCheckpoint {
		t.Error("CreateCheckpoint = false, want true")
	}
}

func TestStandaloneReviewResult_Fields(t *testing.T) {
	result := StandaloneReviewResult{
		Diff:    "diff content",
		Summary: "test summary",
		Verdict: "APPROVED",
		Issues: []ReviewIssue{
			{File: "main.go", Line: 10, Message: "test issue"},
		},
	}

	if result.Diff != "diff content" {
		t.Errorf("Diff = %s, want 'diff content'", result.Diff)
	}
	if result.Summary != "test summary" {
		t.Errorf("Summary = %s, want 'test summary'", result.Summary)
	}
	if result.Verdict != "APPROVED" {
		t.Errorf("Verdict = %s, want APPROVED", result.Verdict)
	}
	if len(result.Issues) != 1 {
		t.Errorf("Issues count = %d, want 1", len(result.Issues))
	}
}

func TestStandaloneReviewResult_WithChanges(t *testing.T) {
	result := StandaloneReviewResult{
		Diff:    "diff content",
		Summary: "fixed issues",
		Verdict: "APPROVED",
		Issues: []ReviewIssue{
			{File: "main.go", Line: 10, Message: "fixed this issue"},
		},
		Changes: []agent.FileChange{
			{Path: "main.go", Operation: agent.FileOpUpdate},
			{Path: "util.go", Operation: agent.FileOpUpdate},
		},
	}

	if len(result.Changes) != 2 {
		t.Errorf("Changes count = %d, want 2", len(result.Changes))
	}
	if result.Changes[0].Path != "main.go" {
		t.Errorf("Changes[0].Path = %s, want main.go", result.Changes[0].Path)
	}
	if result.Changes[0].Operation != agent.FileOpUpdate {
		t.Errorf("Changes[0].Operation = %v, want update", result.Changes[0].Operation)
	}
}

func TestStandaloneSimplifyResult_Fields(t *testing.T) {
	result := StandaloneSimplifyResult{
		Diff:    "diff content",
		Summary: "simplified code",
	}

	if result.Diff != "diff content" {
		t.Errorf("Diff = %s, want 'diff content'", result.Diff)
	}
	if result.Summary != "simplified code" {
		t.Errorf("Summary = %s, want 'simplified code'", result.Summary)
	}
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(substr) == 0 ||
			(len(s) > 0 && standaloneContains(toLowerCase(s), toLowerCase(substr))))
}

func toLowerCase(s string) string {
	result := make([]byte, len(s))
	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}

	return string(result)
}

func standaloneContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
