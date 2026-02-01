package automation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestNew(t *testing.T) {
	cfg := Config{
		Settings: &storage.AutomationSettings{
			Enabled: true,
			Queue: storage.AutomationQueueConfig{
				MaxConcurrent: 2,
			},
		},
	}

	auto := New(cfg)

	if auto == nil {
		t.Fatal("Expected automation to be created")
	}

	status := auto.Status()
	if !status.Enabled {
		t.Error("Expected automation to be enabled")
	}
}

func TestNew_NilSettings(t *testing.T) {
	cfg := Config{
		Settings: nil,
	}

	auto := New(cfg)

	if auto == nil {
		t.Fatal("Expected automation to be created with nil settings")
	}
}

func TestAutomation_Status(t *testing.T) {
	cfg := Config{
		Settings: &storage.AutomationSettings{
			Enabled: true,
			Queue: storage.AutomationQueueConfig{
				MaxConcurrent: 4,
			},
		},
	}

	auto := New(cfg)
	status := auto.Status()

	if !status.Enabled {
		t.Error("Expected enabled to be true")
	}

	if status.Workers != 4 {
		t.Errorf("Expected 4 workers, got %d", status.Workers)
	}
}

func TestAutomation_RegisterParser(t *testing.T) {
	auto := New(Config{Settings: &storage.AutomationSettings{}})

	parser := &mockParser{}
	auto.RegisterParser("github", parser)

	// We can't directly check if it was registered, but we can verify
	// the method doesn't panic.
}

func TestAutomation_StartStop(t *testing.T) {
	auto := New(Config{
		Settings: &storage.AutomationSettings{Enabled: true},
		Handler: func(ctx context.Context, job *WebhookJob) error {
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())

	// Start in background.
	go auto.Start(ctx)

	// Give it time to start.
	time.Sleep(50 * time.Millisecond)

	// Stop.
	cancel()
	err := auto.Stop(1 * time.Second)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestAutomation_GetJob(t *testing.T) {
	auto := New(Config{Settings: &storage.AutomationSettings{}})

	// Enqueue a job.
	job := &WebhookJob{
		Event: &WebhookEvent{
			Provider: "github",
			Type:     EventTypeIssueOpened,
		},
		WorkflowType: WorkflowTypeIssueFix,
	}
	err := auto.queue.Enqueue(job)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Get the job.
	retrieved, ok := auto.GetJob(job.ID)
	if !ok {
		t.Error("Expected job to be found")
	}
	if retrieved.ID != job.ID {
		t.Errorf("Expected job ID %s, got %s", job.ID, retrieved.ID)
	}

	// Get non-existent job.
	_, ok = auto.GetJob("nonexistent")
	if ok {
		t.Error("Expected job to not be found")
	}
}

func TestAutomation_ListJobs(t *testing.T) {
	auto := New(Config{Settings: &storage.AutomationSettings{}})

	// Enqueue jobs.
	for range 3 {
		job := &WebhookJob{
			Event: &WebhookEvent{
				Provider: "github",
				Type:     EventTypeIssueOpened,
			},
			WorkflowType: WorkflowTypeIssueFix,
		}
		_ = auto.queue.Enqueue(job)
	}

	// List all.
	jobs := auto.ListJobs(nil)
	if len(jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(jobs))
	}

	// List by status.
	pending := JobStatusPending
	pendingJobs := auto.ListJobs(&pending)
	if len(pendingJobs) != 3 {
		t.Errorf("Expected 3 pending jobs, got %d", len(pendingJobs))
	}
}

func TestAutomation_CancelJob(t *testing.T) {
	auto := New(Config{Settings: &storage.AutomationSettings{}})

	// Enqueue a job.
	job := &WebhookJob{
		Event: &WebhookEvent{
			Provider: "github",
			Type:     EventTypeIssueOpened,
		},
		WorkflowType: WorkflowTypeIssueFix,
	}
	err := auto.queue.Enqueue(job)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Cancel the job.
	err = auto.CancelJob(job.ID)
	if err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	// Verify status.
	retrieved, _ := auto.GetJob(job.ID)
	if retrieved.Status != JobStatusCancelled {
		t.Errorf("Expected status cancelled, got %v", retrieved.Status)
	}
}

func TestValidateGitHubSignature(t *testing.T) {
	tests := []struct {
		name      string
		signature string
		body      []byte
		secret    string
		expectErr bool
	}{
		{
			name:      "missing_signature",
			signature: "",
			body:      []byte("test"),
			secret:    "secret",
			expectErr: true,
		},
		{
			name:      "invalid_signature",
			signature: "sha256=invalid",
			body:      []byte("test"),
			secret:    "secret",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
			if tt.signature != "" {
				req.Header.Set("X-Hub-Signature-256", tt.signature)
			}

			err := ValidateGitHubSignature(req, tt.body, tt.secret)
			if tt.expectErr && err == nil {
				t.Error("Expected error")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestValidateGitLabSignature(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		secret    string
		expectErr bool
	}{
		{
			name:      "missing_token",
			token:     "",
			secret:    "secret",
			expectErr: true,
		},
		{
			name:      "valid_token",
			token:     "secret",
			secret:    "secret",
			expectErr: false,
		},
		{
			name:      "invalid_token",
			token:     "wrong",
			secret:    "secret",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
			if tt.token != "" {
				req.Header.Set("X-Gitlab-Token", tt.token)
			}

			err := ValidateGitLabSignature(req, tt.secret)
			if tt.expectErr && err == nil {
				t.Error("Expected error")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestAutomation_ShouldProcess(t *testing.T) {
	auto := New(Config{
		Settings: &storage.AutomationSettings{
			Labels: storage.AutomationLabelConfig{
				MehrhofGenerated: "mehrhof-generated",
			},
		},
	})

	tests := []struct {
		name           string
		event          *WebhookEvent
		cfg            *storage.ProviderAutoConfig
		expectProcess  bool
		expectWorkflow WorkflowType
	}{
		{
			name: "issue_opened_enabled",
			event: &WebhookEvent{
				Type:  EventTypeIssueOpened,
				Issue: &IssueInfo{Number: 1},
			},
			cfg: &storage.ProviderAutoConfig{
				TriggerOn: storage.AutomationTriggerConfig{
					IssueOpened: true,
				},
			},
			expectProcess:  true,
			expectWorkflow: WorkflowTypeIssueFix,
		},
		{
			name: "issue_opened_disabled",
			event: &WebhookEvent{
				Type:  EventTypeIssueOpened,
				Issue: &IssueInfo{Number: 1},
			},
			cfg: &storage.ProviderAutoConfig{
				TriggerOn: storage.AutomationTriggerConfig{
					IssueOpened: false,
				},
			},
			expectProcess: false,
		},
		{
			name: "pr_opened_enabled",
			event: &WebhookEvent{
				Type:        EventTypePROpened,
				PullRequest: &PullRequestInfo{Number: 1},
			},
			cfg: &storage.ProviderAutoConfig{
				TriggerOn: storage.AutomationTriggerConfig{
					PROpened: true,
				},
			},
			expectProcess:  true,
			expectWorkflow: WorkflowTypePRReview,
		},
		{
			name: "pr_opened_mehr_generated",
			event: &WebhookEvent{
				Type: EventTypePROpened,
				PullRequest: &PullRequestInfo{
					Number: 1,
					Labels: []string{"mehrhof-generated"},
				},
			},
			cfg: &storage.ProviderAutoConfig{
				TriggerOn: storage.AutomationTriggerConfig{
					PROpened: true,
				},
			},
			expectProcess: false, // Should skip mehrhof-generated PRs
		},
		{
			name: "comment_with_command",
			event: &WebhookEvent{
				Type: EventTypeIssueComment,
				Comment: &CommentInfo{
					Body: "@mehrhof fix this issue",
				},
			},
			cfg: &storage.ProviderAutoConfig{
				CommandPrefix: "@mehrhof",
				TriggerOn: storage.AutomationTriggerConfig{
					CommentCommands: true,
				},
			},
			expectProcess:  true,
			expectWorkflow: WorkflowTypeCommand,
		},
		{
			name: "comment_without_command",
			event: &WebhookEvent{
				Type: EventTypeIssueComment,
				Comment: &CommentInfo{
					Body: "Just a regular comment",
				},
			},
			cfg: &storage.ProviderAutoConfig{
				CommandPrefix: "@mehrhof",
				TriggerOn: storage.AutomationTriggerConfig{
					CommentCommands: true,
				},
			},
			expectProcess: false,
		},
		{
			name: "unknown_event",
			event: &WebhookEvent{
				Type: EventTypeUnknown,
			},
			cfg: &storage.ProviderAutoConfig{
				TriggerOn: storage.AutomationTriggerConfig{
					IssueOpened: true,
					PROpened:    true,
				},
			},
			expectProcess: false,
		},
		{
			name: "issue_labeled_matching",
			event: &WebhookEvent{
				Type: EventTypeIssueLabeled,
				Issue: &IssueInfo{
					Number: 1,
					Labels: []string{"mehr-fix", "bug"},
				},
			},
			cfg: &storage.ProviderAutoConfig{
				TriggerOn: storage.AutomationTriggerConfig{
					IssueLabeled: []string{"mehr-fix"},
				},
			},
			expectProcess:  true,
			expectWorkflow: WorkflowTypeIssueFix,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wt, shouldProcess := auto.shouldProcess(tt.event, tt.cfg)

			if shouldProcess != tt.expectProcess {
				t.Errorf("shouldProcess() = %v, want %v", shouldProcess, tt.expectProcess)
			}

			if tt.expectProcess && wt != tt.expectWorkflow {
				t.Errorf("workflow = %v, want %v", wt, tt.expectWorkflow)
			}
		})
	}
}

func TestAutomation_IsMehrhofGenerated(t *testing.T) {
	auto := New(Config{
		Settings: &storage.AutomationSettings{
			Labels: storage.AutomationLabelConfig{
				MehrhofGenerated: "mehrhof-generated",
			},
		},
	})

	tests := []struct {
		name     string
		event    *WebhookEvent
		expected bool
	}{
		{
			name: "has_mehr_generated_label",
			event: &WebhookEvent{
				PullRequest: &PullRequestInfo{
					Labels: []string{"bug", "mehrhof-generated", "enhancement"},
				},
			},
			expected: true,
		},
		{
			name: "no_mehr_generated_label",
			event: &WebhookEvent{
				PullRequest: &PullRequestInfo{
					Labels: []string{"bug", "enhancement"},
				},
			},
			expected: false,
		},
		{
			name: "no_pr",
			event: &WebhookEvent{
				PullRequest: nil,
			},
			expected: false,
		},
		{
			name: "case_insensitive",
			event: &WebhookEvent{
				PullRequest: &PullRequestInfo{
					Labels: []string{"MEHRHOF-GENERATED"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auto.isMehrhofGenerated(tt.event)
			if result != tt.expected {
				t.Errorf("isMehrhofGenerated() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAutomation_HandleWebhook_Disabled(t *testing.T) {
	auto := New(Config{
		Settings: &storage.AutomationSettings{
			Enabled: false,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook/github", nil)
	w := httptest.NewRecorder()

	auto.HandleWebhook(w, req, "github")

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

func TestAutomation_HandleWebhook_ProviderNotEnabled(t *testing.T) {
	auto := New(Config{
		Settings: &storage.AutomationSettings{
			Enabled:   true,
			Providers: map[string]storage.ProviderAutoConfig{},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook/github", nil)
	w := httptest.NewRecorder()

	auto.HandleWebhook(w, req, "github")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestAutomation_HandleWebhook_NoParser(t *testing.T) {
	auto := New(Config{
		Settings: &storage.AutomationSettings{
			Enabled: true,
			Providers: map[string]storage.ProviderAutoConfig{
				"github": {Enabled: true},
			},
		},
	})
	// Don't register parser.

	req := httptest.NewRequest(http.MethodPost, "/webhook/github", strings.NewReader("{}"))
	w := httptest.NewRecorder()

	auto.HandleWebhook(w, req, "github")

	if w.Code != http.StatusNotImplemented {
		t.Errorf("Expected status 501, got %d", w.Code)
	}
}

// mockParser implements WebhookParser for testing.
type mockParser struct {
	event *WebhookEvent
	err   error
}

func (m *mockParser) Parse(r *http.Request, body []byte) (*WebhookEvent, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.event != nil {
		return m.event, nil
	}

	return &WebhookEvent{
		Type:     EventTypeIssueOpened,
		Provider: "github",
		Sender:   UserInfo{Login: "testuser", Type: "User"},
		Repository: RepositoryInfo{
			Owner:    "owner",
			FullName: "owner/repo",
		},
		Issue: &IssueInfo{Number: 1},
	}, nil
}

func (m *mockParser) ValidateSignature(r *http.Request, body []byte, secret string) error {
	return nil
}
