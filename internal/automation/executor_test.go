package automation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/conductor"
)

func TestExecutor_ParseCommand(t *testing.T) {
	e := &Executor{}

	tests := []struct {
		name       string
		body       string
		presetCmd  string
		expectName string
		expectArgs []string
	}{
		{
			name:       "preset_command",
			body:       "anything here",
			presetCmd:  "fix",
			expectName: "fix",
			expectArgs: nil,
		},
		{
			name:       "mehr_fix",
			body:       "@mehrhof fix",
			presetCmd:  "",
			expectName: "fix",
			expectArgs: []string{},
		},
		{
			name:       "mehr_review_with_args",
			body:       "@mehrhof review --detailed",
			presetCmd:  "",
			expectName: "review",
			expectArgs: []string{"--detailed"},
		},
		{
			name:       "mehr_status",
			body:       "Please check @mehrhof status for this issue",
			presetCmd:  "",
			expectName: "status",
			expectArgs: []string{"for", "this", "issue"},
		},
		{
			name:       "mehr_help",
			body:       "@mehrhof help",
			presetCmd:  "",
			expectName: "help",
			expectArgs: []string{},
		},
		{
			name:       "mehr_only",
			body:       "@mehrhof",
			presetCmd:  "",
			expectName: "help",
			expectArgs: nil,
		},
		{
			name:       "no_command",
			body:       "This is just a comment without any command",
			presetCmd:  "",
			expectName: "unknown",
			expectArgs: nil,
		},
		{
			name:       "case_insensitive",
			body:       "@MEHRHOF FIX",
			presetCmd:  "",
			expectName: "FIX",
			expectArgs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := e.parseCommand(tt.body, tt.presetCmd)

			if cmd.Name != tt.expectName {
				t.Errorf("Expected command name %q, got %q", tt.expectName, cmd.Name)
			}

			if tt.expectArgs != nil {
				if len(cmd.Args) != len(tt.expectArgs) {
					t.Errorf("Expected %d args, got %d: %v", len(tt.expectArgs), len(cmd.Args), cmd.Args)
				}
			}
		})
	}
}

func TestBuildReviewComment(t *testing.T) {
	tests := []struct {
		name     string
		result   *conductor.StandaloneReviewResult
		contains []string
	}{
		{
			name: "approved",
			result: &conductor.StandaloneReviewResult{
				Verdict: "APPROVED",
				Summary: "Code looks good!",
				Issues:  nil,
			},
			contains: []string{"✅", "Approved", "Code looks good!"},
		},
		{
			name: "needs_changes",
			result: &conductor.StandaloneReviewResult{
				Verdict: "NEEDS_CHANGES",
				Summary: "Found some issues",
				Issues: []conductor.ReviewIssue{
					{File: "main.go", Line: 10, Message: "Missing error check"},
				},
			},
			contains: []string{"⚠️", "Changes Requested", "Issues Found", "main.go:10"},
		},
		{
			name: "comment_only",
			result: &conductor.StandaloneReviewResult{
				Verdict: "COMMENT",
				Summary: "Just some observations",
				Issues:  nil,
			},
			contains: []string{"📝", "Review", "Just some observations"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment := buildReviewComment(tt.result)

			for _, expected := range tt.contains {
				if !strings.Contains(comment, expected) {
					t.Errorf("Expected comment to contain %q, got:\n%s", expected, comment)
				}
			}
		})
	}
}

func TestFormatReviewIssue(t *testing.T) {
	tests := []struct {
		name     string
		issue    conductor.ReviewIssue
		contains []string
	}{
		{
			name: "with_category_and_severity",
			issue: conductor.ReviewIssue{
				File:     "main.go",
				Line:     42,
				Category: "security",
				Severity: "high",
				Message:  "SQL injection vulnerability",
			},
			contains: []string{"security", "high", "SQL injection vulnerability"},
		},
		{
			name: "message_only",
			issue: conductor.ReviewIssue{
				File:    "main.go",
				Line:    10,
				Message: "Consider using constants",
			},
			contains: []string{"Consider using constants"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := formatReviewIssue(tt.issue)

			for _, expected := range tt.contains {
				if !strings.Contains(formatted, expected) {
					t.Errorf("Expected formatted issue to contain %q, got: %s", expected, formatted)
				}
			}
		})
	}
}

func TestExecutor_ExecuteCommand_NoComment(t *testing.T) {
	e := &Executor{}
	job := &WebhookJob{
		Event: &WebhookEvent{
			Comment: nil,
		},
	}

	err := e.executeCommand(context.Background(), job)
	if err == nil {
		t.Error("Expected error when no comment in event")
	}

	if !strings.Contains(err.Error(), "no comment") {
		t.Errorf("Expected 'no comment' error, got: %v", err)
	}
}

func TestExecutor_ExecuteCommand_UnknownCommand(t *testing.T) {
	e := &Executor{}
	job := &WebhookJob{
		Event: &WebhookEvent{
			Comment: &CommentInfo{
				Body: "@mehrhof unknowncmd",
			},
		},
	}

	err := e.executeCommand(context.Background(), job)
	if err == nil {
		t.Error("Expected error for unknown command")
	}

	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("Expected 'unknown command' error, got: %v", err)
	}
}

func TestExecutor_ExecuteCommand_FixRequiresIssue(t *testing.T) {
	e := &Executor{}
	job := &WebhookJob{
		Event: &WebhookEvent{
			Comment: &CommentInfo{
				Body: "@mehrhof fix",
			},
			Issue: nil, // No issue context
		},
	}

	err := e.executeCommand(context.Background(), job)
	if err == nil {
		t.Error("Expected error when fix command has no issue context")
	}

	if !strings.Contains(err.Error(), "requires issue context") {
		t.Errorf("Expected 'requires issue context' error, got: %v", err)
	}
}

func TestExecutor_ExecuteCommand_ReviewRequiresPR(t *testing.T) {
	e := &Executor{}
	job := &WebhookJob{
		Event: &WebhookEvent{
			Comment: &CommentInfo{
				Body: "@mehrhof review",
			},
			PullRequest: nil, // No PR context
		},
	}

	err := e.executeCommand(context.Background(), job)
	if err == nil {
		t.Error("Expected error when review command has no PR context")
	}

	if !strings.Contains(err.Error(), "requires PR context") {
		t.Errorf("Expected 'requires PR context' error, got: %v", err)
	}
}

func TestExecutor_ExecuteIssueFix_NoIssue(t *testing.T) {
	e := &Executor{}
	job := &WebhookJob{
		Event: &WebhookEvent{
			Issue: nil,
		},
	}

	err := e.executeIssueFix(context.Background(), job)
	if err == nil {
		t.Error("Expected error when no issue in event")
	}
}

func TestExecutor_ExecutePRReview_NoPR(t *testing.T) {
	e := &Executor{}
	job := &WebhookJob{
		Event: &WebhookEvent{
			PullRequest: nil,
		},
	}

	err := e.executePRReview(context.Background(), job)
	if err == nil {
		t.Error("Expected error when no PR in event")
	}
}

// Mock types for testing.
type mockCommenter struct {
	comments []string
	err      error
}

func (m *mockCommenter) AddComment(_ context.Context, _ string, body string) (any, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.comments = append(m.comments, body)

	return struct{}{}, nil // Return empty struct to indicate success
}

func TestExecutor_PostIssueComment(t *testing.T) {
	mock := &mockCommenter{}

	e := &Executor{
		providerGetter: func(name string) (any, error) {
			return mock, nil
		},
	}

	job := &WebhookJob{
		Event: &WebhookEvent{
			Provider: "github",
			Issue: &IssueInfo{
				Number: 123,
			},
		},
	}

	err := e.postIssueComment(context.Background(), job, "Test comment")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(mock.comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(mock.comments))
	}

	if mock.comments[0] != "Test comment" {
		t.Errorf("Expected 'Test comment', got %q", mock.comments[0])
	}
}

func TestExecutor_PostIssueComment_NoProvider(t *testing.T) {
	e := &Executor{
		providerGetter: nil,
	}

	job := &WebhookJob{
		Event: &WebhookEvent{
			Provider: "github",
			Issue:    &IssueInfo{Number: 123},
		},
	}

	err := e.postIssueComment(context.Background(), job, "Test")
	if err == nil {
		t.Error("Expected error when no provider getter")
	}
}

func TestExecutor_PostIssueComment_ProviderError(t *testing.T) {
	e := &Executor{
		providerGetter: func(name string) (any, error) {
			return nil, errors.New("provider not found")
		},
	}

	job := &WebhookJob{
		Event: &WebhookEvent{
			Provider: "github",
			Issue:    &IssueInfo{Number: 123},
		},
	}

	err := e.postIssueComment(context.Background(), job, "Test")
	if err == nil {
		t.Error("Expected error when provider returns error")
	}
}

func TestCommand_Fields(t *testing.T) {
	cmd := Command{
		Name: "fix",
		Args: []string{"--force", "--verbose"},
	}

	if cmd.Name != "fix" {
		t.Errorf("Expected Name 'fix', got %q", cmd.Name)
	}

	if len(cmd.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(cmd.Args))
	}
}
