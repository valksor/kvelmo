package vcs

import (
	"context"
	"errors"
	"testing"
)

func TestGroupByDirectory_SingleFile(t *testing.T) {
	t.Parallel()

	analyzer := NewChangeAnalyzer(nil)

	files := []FileStatus{
		{Path: "main.go"},
	}

	groups, err := analyzer.groupByDirectory(files)
	if err != nil {
		t.Fatalf("groupByDirectory() error = %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("groupByDirectory() returned %d groups, want 1", len(groups))
	}

	if len(groups[0].Files) != 1 {
		t.Errorf("groupByDirectory() groups[0].Files length = %d, want 1", len(groups[0].Files))
	}
}

func TestGroupByDirectory_MultipleDirectories(t *testing.T) {
	t.Parallel()

	analyzer := NewChangeAnalyzer(nil)

	files := []FileStatus{
		{Path: "cmd/main.go"},
		{Path: "cmd/util.go"},
		{Path: "internal/handler.go"},
	}

	groups, err := analyzer.groupByDirectory(files)
	if err != nil {
		t.Fatalf("groupByDirectory() error = %v", err)
	}

	// Should have at least 2 groups (cmd and internal)
	if len(groups) < 2 {
		t.Fatalf("groupByDirectory() returned %d groups, want >= 2", len(groups))
	}
}

func TestGroupByDirectory_DeepNesting(t *testing.T) {
	t.Parallel()

	analyzer := NewChangeAnalyzer(nil)

	files := []FileStatus{
		{Path: "internal/auth/provider/github.go"},
		{Path: "internal/auth/provider/gitlab.go"},
	}

	groups, err := analyzer.groupByDirectory(files)
	if err != nil {
		t.Fatalf("groupByDirectory() error = %v", err)
	}

	// Both should be in same group since they share the same top-level dirs
	if len(groups) != 1 {
		t.Fatalf("groupByDirectory() returned %d groups, want 1", len(groups))
	}

	if len(groups[0].Files) != 2 {
		t.Errorf("groupByDirectory() groups[0].Files length = %d, want 2", len(groups[0].Files))
	}
}

func TestFilepathDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
	}{
		{"main.go", "(root)"},
		{"cmd/main.go", "cmd"},
		{"internal/auth/provider.go", "internal/auth"},
		{"docs/cli/commit.md", "docs/cli"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := filepathDir(tt.path)
			if got != tt.want {
				t.Errorf("filepathDir(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestFormatFileStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		index    byte
		workDir  byte
		expected string
	}{
		{"untracked", '?', ' ', "?"},
		{"modified", 'M', ' ', "M"},
		{"staged modified", 'M', 'M', "M"},
		{"staged added", 'A', ' ', "A"},
		{"deleted", 'D', ' ', "D"},
		{"renamed", 'R', ' ', "R"},
		{"copied", 'C', ' ', "C"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatFileStatus(tt.index, tt.workDir)
			if got != tt.expected {
				t.Errorf("formatFileStatus(%c, %c) = %q, want %q", tt.index, tt.workDir, got, tt.expected)
			}
		})
	}
}

// mockAgent is a test double for Agent.
type mockAgent struct {
	response *AgentResponse
	err      error
}

func (m *mockAgent) Run(_ context.Context, _ string) (*AgentResponse, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.response, nil
}

func TestParseGroupingResponse_ValidJSON(t *testing.T) {
	t.Parallel()

	analyzer := NewChangeAnalyzer(nil)

	response := &AgentResponse{
		Messages: []string{`[
			{"files": ["file1.go", "file2.go"], "reason": "feature"},
			{"files": ["README.md"], "reason": "docs"}
		]`},
	}

	groups, err := analyzer.parseGroupingResponse(response)
	if err != nil {
		t.Fatalf("parseGroupingResponse() error = %v", err)
	}

	if len(groups) != 2 {
		t.Fatalf("parseGroupingResponse() returned %d groups, want 2", len(groups))
	}

	if len(groups[0].Files) != 2 {
		t.Errorf("parseGroupingResponse() groups[0].Files length = %d, want 2", len(groups[0].Files))
	}

	if groups[0].Reason != "feature" {
		t.Errorf("parseGroupingResponse() groups[0].Reason = %q, want %q", groups[0].Reason, "feature")
	}
}

func TestParseGroupingResponse_MarkdownWrapped(t *testing.T) {
	t.Parallel()

	analyzer := NewChangeAnalyzer(nil)

	response := &AgentResponse{
		Messages: []string{"```\n" + `[
			{"files": ["file1.go"], "reason": "test"}
		]` + "\n```"},
	}

	groups, err := analyzer.parseGroupingResponse(response)
	if err != nil {
		t.Fatalf("parseGroupingResponse() error = %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("parseGroupingResponse() returned %d groups, want 1", len(groups))
	}
}

func TestParseGroupingResponse_InvalidJSON(t *testing.T) {
	t.Parallel()

	analyzer := NewChangeAnalyzer(nil)

	response := &AgentResponse{
		Messages: []string{"not json"},
	}

	_, err := analyzer.parseGroupingResponse(response)
	if err == nil {
		t.Error("parseGroupingResponse() expected error for invalid JSON, got nil")
	}
}

func TestGroupWithAI_ValidResponse(t *testing.T) {
	t.Parallel()

	analyzer := NewChangeAnalyzer(nil)
	analyzer.SetAgent(&mockAgent{
		response: &AgentResponse{
			Messages: []string{`[{"files": ["file1.go"], "reason": "test"}]`},
		},
	})

	repoInfo := RepoInfo{Language: "go"}
	groups, err := analyzer.groupWithAI(context.Background(), nil, repoInfo)
	if err != nil {
		t.Fatalf("groupWithAI() error = %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("groupWithAI() returned %d groups, want 1", len(groups))
	}
}

func TestGroupWithAI_AgentError_FallsBackToDirectory(t *testing.T) {
	t.Parallel()

	analyzer := NewChangeAnalyzer(nil)
	analyzer.SetAgent(&mockAgent{
		err: errors.New("agent error"),
	})

	files := []FileStatus{
		{Path: "cmd/main.go"},
		{Path: "internal/auth.go"},
	}

	repoInfo := RepoInfo{Language: "go"}
	groups, err := analyzer.groupWithAI(context.Background(), files, repoInfo)
	if err != nil {
		t.Fatalf("groupWithAI() should not return error on agent failure, got %v", err)
	}

	// Should fallback to directory grouping
	if len(groups) == 0 {
		t.Error("groupWithAI() returned no groups on fallback")
	}
}
