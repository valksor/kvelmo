package automation

import (
	"testing"
	"time"
)

func TestEventType_String(t *testing.T) {
	tests := []struct {
		name     string
		et       EventType
		expected string
	}{
		{"issue_opened", EventTypeIssueOpened, "issue_opened"},
		{"issue_closed", EventTypeIssueClosed, "issue_closed"},
		{"issue_labeled", EventTypeIssueLabeled, "issue_labeled"},
		{"issue_comment", EventTypeIssueComment, "issue_comment"},
		{"pr_opened", EventTypePROpened, "pr_opened"},
		{"pr_closed", EventTypePRClosed, "pr_closed"},
		{"pr_merged", EventTypePRMerged, "pr_merged"},
		{"pr_updated", EventTypePRUpdated, "pr_updated"},
		{"pr_comment", EventTypePRComment, "pr_comment"},
		{"unknown", EventTypeUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.et); got != tt.expected {
				t.Errorf("EventType = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWorkflowType_String(t *testing.T) {
	tests := []struct {
		name     string
		wt       WorkflowType
		expected string
	}{
		{"issue_fix", WorkflowTypeIssueFix, "issue_fix"},
		{"pr_review", WorkflowTypePRReview, "pr_review"},
		{"command", WorkflowTypeCommand, "command"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.wt); got != tt.expected {
				t.Errorf("WorkflowType = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestJobStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		js       JobStatus
		expected string
	}{
		{"pending", JobStatusPending, "pending"},
		{"running", JobStatusRunning, "running"},
		{"completed", JobStatusCompleted, "completed"},
		{"failed", JobStatusFailed, "failed"},
		{"cancelled", JobStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.js); got != tt.expected {
				t.Errorf("JobStatus = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWebhookEvent_ProviderReference(t *testing.T) {
	tests := []struct {
		name     string
		event    WebhookEvent
		expected string
	}{
		{
			name: "github_issue",
			event: WebhookEvent{
				Provider: "github",
				Repository: RepositoryInfo{
					FullName: "owner/repo",
				},
				Issue: &IssueInfo{
					Number: 123,
				},
			},
			expected: "github:owner/repo#123",
		},
		{
			name: "github_pr",
			event: WebhookEvent{
				Provider: "github",
				Repository: RepositoryInfo{
					FullName: "owner/repo",
				},
				PullRequest: &PullRequestInfo{
					Number: 456,
				},
			},
			expected: "github:owner/repo#456",
		},
		{
			name: "gitlab_issue",
			event: WebhookEvent{
				Provider: "gitlab",
				Repository: RepositoryInfo{
					FullName: "group/project",
				},
				Issue: &IssueInfo{
					Number: 789,
				},
			},
			expected: "gitlab:group/project#789",
		},
		{
			name: "no_issue_or_pr",
			event: WebhookEvent{
				Provider: "github",
				Repository: RepositoryInfo{
					FullName: "owner/repo",
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.ProviderReference(); got != tt.expected {
				t.Errorf("ProviderReference() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWebhookJob_CanRetry(t *testing.T) {
	tests := []struct {
		name     string
		job      *WebhookJob
		expected bool
	}{
		{
			name: "can_retry",
			job: &WebhookJob{
				Status:      JobStatusFailed,
				Attempts:    1,
				MaxAttempts: 3,
			},
			expected: true,
		},
		{
			name: "max_attempts_reached",
			job: &WebhookJob{
				Status:      JobStatusFailed,
				Attempts:    3,
				MaxAttempts: 3,
			},
			expected: false,
		},
		{
			name: "not_failed",
			job: &WebhookJob{
				Status:      JobStatusCompleted,
				Attempts:    1,
				MaxAttempts: 3,
			},
			expected: false,
		},
		{
			name: "cancelled",
			job: &WebhookJob{
				Status:      JobStatusCancelled,
				Attempts:    1,
				MaxAttempts: 3,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.job.CanRetry(); got != tt.expected {
				t.Errorf("CanRetry() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWebhookJob_IsTerminal(t *testing.T) {
	tests := []struct {
		name     string
		status   JobStatus
		expected bool
	}{
		{"pending", JobStatusPending, false},
		{"running", JobStatusRunning, false},
		{"completed", JobStatusCompleted, true},
		{"failed", JobStatusFailed, true},
		{"cancelled", JobStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &WebhookJob{Status: tt.status}
			if got := job.IsTerminal(); got != tt.expected {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestQueueStatus_Fields(t *testing.T) {
	status := QueueStatus{
		Enabled:       true,
		Running:       true,
		Workers:       4,
		PendingJobs:   10,
		RunningJobs:   2,
		CompletedJobs: 100,
		FailedJobs:    5,
		CancelledJobs: 3,
	}

	if !status.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if !status.Running {
		t.Error("Expected Running to be true")
	}

	if status.Workers != 4 {
		t.Errorf("Expected Workers 4, got %d", status.Workers)
	}

	if status.PendingJobs != 10 {
		t.Errorf("Expected PendingJobs 10, got %d", status.PendingJobs)
	}
}

func TestJobResult_Fields(t *testing.T) {
	result := JobResult{
		Success:        true,
		PRNumber:       42,
		PRURL:          "https://github.com/owner/repo/pull/42",
		CommentsPosted: 5,
		Duration:       10 * time.Second,
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if result.PRNumber != 42 {
		t.Errorf("Expected PRNumber 42, got %d", result.PRNumber)
	}

	if result.CommentsPosted != 5 {
		t.Errorf("Expected CommentsPosted 5, got %d", result.CommentsPosted)
	}

	if result.Duration != 10*time.Second {
		t.Errorf("Expected Duration 10s, got %v", result.Duration)
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{123, "123"},
		{-1, "-1"},
		{-42, "-42"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := itoa(tt.n); got != tt.expected {
				t.Errorf("itoa(%d) = %v, want %v", tt.n, got, tt.expected)
			}
		})
	}
}
