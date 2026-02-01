package automation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestNewResultHandler(t *testing.T) {
	cfg := &storage.AutomationLabelConfig{
		MehrhofGenerated: "custom-label",
		InProgress:       "custom-processing",
		Failed:           "custom-failed",
	}

	handler := NewResultHandler(nil, cfg)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}

	if handler.config.MehrhofGenerated != "custom-label" {
		t.Errorf("Expected MehrhofGenerated 'custom-label', got %q", handler.config.MehrhofGenerated)
	}
}

func TestNewResultHandler_NilConfig(t *testing.T) {
	handler := NewResultHandler(nil, nil)

	if handler == nil {
		t.Fatal("Expected handler to be created with nil config")
	}

	// Should have defaults.
	if handler.config.MehrhofGenerated != "mehrhof-generated" {
		t.Errorf("Expected default MehrhofGenerated 'mehrhof-generated', got %q", handler.config.MehrhofGenerated)
	}

	if handler.config.InProgress != "mehrhof-processing" {
		t.Errorf("Expected default InProgress 'mehrhof-processing', got %q", handler.config.InProgress)
	}

	if handler.config.Failed != "mehrhof-failed" {
		t.Errorf("Expected default Failed 'mehrhof-failed', got %q", handler.config.Failed)
	}
}

func TestResultHandler_BuildSuccessComment(t *testing.T) {
	handler := NewResultHandler(nil, nil)

	tests := []struct {
		name     string
		job      *WebhookJob
		contains []string
	}{
		{
			name: "issue_fix_success",
			job: &WebhookJob{
				WorkflowType: WorkflowTypeIssueFix,
				Result: &JobResult{
					Success: true,
					PRURL:   "https://github.com/owner/repo/pull/42",
				},
			},
			contains: []string{"Success", "pull request", "https://github.com/owner/repo/pull/42"},
		},
		{
			name: "pr_review_with_comments",
			job: &WebhookJob{
				WorkflowType: WorkflowTypePRReview,
				Result: &JobResult{
					Success:        true,
					CommentsPosted: 5,
				},
			},
			contains: []string{"Success", "review", "5 review comments"},
		},
		{
			name: "pr_review_no_issues",
			job: &WebhookJob{
				WorkflowType: WorkflowTypePRReview,
				Result: &JobResult{
					Success:        true,
					CommentsPosted: 0,
				},
			},
			contains: []string{"Success", "review", "No issues found"},
		},
		{
			name: "command_success",
			job: &WebhookJob{
				WorkflowType: WorkflowTypeCommand,
				Result:       &JobResult{Success: true},
			},
			contains: []string{"Success", "executed successfully"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment := handler.buildSuccessComment(tt.job)

			for _, expected := range tt.contains {
				if !strings.Contains(comment, expected) {
					t.Errorf("Expected comment to contain %q, got:\n%s", expected, comment)
				}
			}
		})
	}
}

func TestResultHandler_BuildFailureComment(t *testing.T) {
	handler := NewResultHandler(nil, nil)

	tests := []struct {
		name     string
		job      *WebhookJob
		err      error
		contains []string
	}{
		{
			name: "issue_fix_failure",
			job: &WebhookJob{
				ID:           "job-123",
				WorkflowType: WorkflowTypeIssueFix,
				Attempts:     2,
				MaxAttempts:  3,
			},
			err:      errors.New("failed to create PR"),
			contains: []string{"Failed", "unable to automatically fix", "failed to create PR", "job-123", "2/3"},
		},
		{
			name: "pr_review_failure",
			job: &WebhookJob{
				ID:           "job-456",
				WorkflowType: WorkflowTypePRReview,
				Attempts:     1,
				MaxAttempts:  3,
			},
			err:      errors.New("review timeout"),
			contains: []string{"Failed", "error while reviewing", "review timeout"},
		},
		{
			name: "command_failure",
			job: &WebhookJob{
				ID:           "job-789",
				WorkflowType: WorkflowTypeCommand,
				Attempts:     1,
				MaxAttempts:  1,
			},
			err:      errors.New("unknown command"),
			contains: []string{"Failed", "command failed", "unknown command"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment := handler.buildFailureComment(tt.job, tt.err)

			for _, expected := range tt.contains {
				if !strings.Contains(comment, expected) {
					t.Errorf("Expected comment to contain %q, got:\n%s", expected, comment)
				}
			}
		})
	}
}

// Mock implementations for testing.
type mockLabelManager struct {
	addedLabels   []string
	removedLabels []string
	err           error
}

func (m *mockLabelManager) AddLabels(ctx context.Context, workUnitID string, labels []string) error {
	if m.err != nil {
		return m.err
	}
	m.addedLabels = append(m.addedLabels, labels...)

	return nil
}

func (m *mockLabelManager) RemoveLabels(ctx context.Context, workUnitID string, labels []string) error {
	if m.err != nil {
		return m.err
	}
	m.removedLabels = append(m.removedLabels, labels...)

	return nil
}

type mockResultCommenter struct {
	comments []string
	err      error
}

func (m *mockResultCommenter) AddComment(_ context.Context, _ string, body string) (any, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.comments = append(m.comments, body)

	return struct{}{}, nil // Return empty struct to indicate success
}

// Combined mock for testing both interfaces.
type mockProvider struct {
	mockResultCommenter
	mockLabelManager
}

func TestResultHandler_HandleJobStart(t *testing.T) {
	mock := &mockProvider{}

	handler := NewResultHandler(func(name string) (any, error) {
		return mock, nil
	}, nil)

	job := &WebhookJob{
		Event: &WebhookEvent{
			Provider: "github",
			Issue:    &IssueInfo{Number: 42},
		},
	}

	err := handler.HandleJobStart(context.Background(), job)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have added in-progress label.
	found := false
	for _, label := range mock.addedLabels {
		if label == "mehrhof-processing" {
			found = true

			break
		}
	}
	if !found {
		t.Error("Expected in-progress label to be added")
	}
}

func TestResultHandler_HandleJobStart_NoLabel(t *testing.T) {
	handler := NewResultHandler(nil, &storage.AutomationLabelConfig{
		InProgress: "", // Empty = disabled
	})

	job := &WebhookJob{
		Event: &WebhookEvent{
			Provider: "github",
			Issue:    &IssueInfo{Number: 42},
		},
	}

	// Should not error even with no provider.
	err := handler.HandleJobStart(context.Background(), job)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestResultHandler_HandleJobSuccess(t *testing.T) {
	mock := &mockProvider{}

	handler := NewResultHandler(func(name string) (any, error) {
		return mock, nil
	}, nil)

	job := &WebhookJob{
		WorkflowType: WorkflowTypeIssueFix,
		Event: &WebhookEvent{
			Provider: "github",
			Issue:    &IssueInfo{Number: 42},
		},
		Result: &JobResult{
			Success: true,
			PRURL:   "https://github.com/owner/repo/pull/1",
		},
	}

	// First add in-progress label.
	mock.addedLabels = []string{"mehrhof-processing"}

	err := handler.HandleJobSuccess(context.Background(), job)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have removed in-progress label.
	found := false
	for _, label := range mock.removedLabels {
		if label == "mehrhof-processing" {
			found = true

			break
		}
	}
	if !found {
		t.Error("Expected in-progress label to be removed")
	}

	// Should have posted success comment.
	if len(mock.comments) == 0 {
		t.Error("Expected success comment to be posted")
	}
}

func TestResultHandler_HandleJobFailure(t *testing.T) {
	mock := &mockProvider{}

	handler := NewResultHandler(func(name string) (any, error) {
		return mock, nil
	}, nil)

	job := &WebhookJob{
		ID:           "job-123",
		WorkflowType: WorkflowTypeIssueFix,
		Attempts:     1,
		MaxAttempts:  3,
		Event: &WebhookEvent{
			Provider: "github",
			Issue:    &IssueInfo{Number: 42},
		},
	}

	testErr := errors.New("test failure")
	err := handler.HandleJobFailure(context.Background(), job, testErr)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have added failed label.
	foundFailed := false
	for _, label := range mock.addedLabels {
		if label == "mehrhof-failed" {
			foundFailed = true

			break
		}
	}
	if !foundFailed {
		t.Error("Expected failed label to be added")
	}

	// Should have removed in-progress label.
	foundRemoved := false
	for _, label := range mock.removedLabels {
		if label == "mehrhof-processing" {
			foundRemoved = true

			break
		}
	}
	if !foundRemoved {
		t.Error("Expected in-progress label to be removed")
	}

	// Should have posted failure comment.
	if len(mock.comments) == 0 {
		t.Error("Expected failure comment to be posted")
	}

	if !strings.Contains(mock.comments[0], "test failure") {
		t.Error("Expected failure comment to contain error message")
	}
}

func TestResultHandler_AddMehrhofLabel(t *testing.T) {
	mock := &mockProvider{}

	handler := NewResultHandler(func(name string) (any, error) {
		return mock, nil
	}, nil)

	job := &WebhookJob{
		Event: &WebhookEvent{
			Provider:    "github",
			PullRequest: &PullRequestInfo{Number: 42},
		},
	}

	err := handler.AddMehrhofLabel(context.Background(), job)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	found := false
	for _, label := range mock.addedLabels {
		if label == "mehrhof-generated" {
			found = true

			break
		}
	}
	if !found {
		t.Error("Expected mehrhof-generated label to be added")
	}
}

func TestResultHandler_AddMehrhofLabel_Disabled(t *testing.T) {
	handler := NewResultHandler(nil, &storage.AutomationLabelConfig{
		MehrhofGenerated: "", // Empty = disabled
	})

	job := &WebhookJob{
		Event: &WebhookEvent{
			Provider:    "github",
			PullRequest: &PullRequestInfo{Number: 42},
		},
	}

	// Should not error.
	err := handler.AddMehrhofLabel(context.Background(), job)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestResultHandler_PostComment_NoIssueOrPR(t *testing.T) {
	mock := &mockResultCommenter{}

	handler := NewResultHandler(func(name string) (any, error) {
		return mock, nil
	}, nil)

	job := &WebhookJob{
		Event: &WebhookEvent{
			Provider:    "github",
			Issue:       nil,
			PullRequest: nil,
		},
	}

	// Should not error, just skip.
	err := handler.postComment(context.Background(), job, "test")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should not have posted any comment.
	if len(mock.comments) != 0 {
		t.Error("Expected no comments to be posted")
	}
}

func TestResultHandler_ProviderError(t *testing.T) {
	handler := NewResultHandler(func(name string) (any, error) {
		return nil, errors.New("provider not found")
	}, nil)

	job := &WebhookJob{
		Event: &WebhookEvent{
			Provider: "github",
			Issue:    &IssueInfo{Number: 42},
		},
	}

	// postComment returns error when provider fails.
	err := handler.postComment(context.Background(), job, "test")
	if err == nil {
		t.Error("Expected error when provider not found")
	}
}
