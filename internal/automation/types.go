package automation

import (
	"time"
)

// EventType categorizes webhook events from providers.
type EventType string

const (
	// Issue events.
	EventTypeIssueOpened  EventType = "issue_opened"
	EventTypeIssueClosed  EventType = "issue_closed"
	EventTypeIssueLabeled EventType = "issue_labeled"
	EventTypeIssueEdited  EventType = "issue_edited"

	// PR/MR events.
	EventTypePROpened  EventType = "pr_opened"
	EventTypePRUpdated EventType = "pr_updated"
	EventTypePRClosed  EventType = "pr_closed"
	EventTypePRMerged  EventType = "pr_merged"

	// Comment events.
	EventTypeIssueComment EventType = "issue_comment"
	EventTypePRComment    EventType = "pr_comment"

	// Unknown event.
	EventTypeUnknown EventType = "unknown"
)

// WorkflowType determines which workflow to run for a job.
type WorkflowType string

const (
	// WorkflowTypeIssueFix runs the full issue-to-PR workflow.
	WorkflowTypeIssueFix WorkflowType = "issue_fix"

	// WorkflowTypePRReview runs PR/MR review workflow.
	WorkflowTypePRReview WorkflowType = "pr_review"

	// WorkflowTypeCommand runs a specific command from a comment.
	WorkflowTypeCommand WorkflowType = "command"
)

// JobStatus tracks the lifecycle of a webhook job.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// RepositoryInfo contains repository metadata from a webhook.
type RepositoryInfo struct {
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
	CloneURL      string `json:"clone_url"`
	HTMLURL       string `json:"html_url"`
}

// UserInfo contains user metadata from a webhook.
type UserInfo struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
	Type  string `json:"type"` // "User", "Bot", "Organization"
	Email string `json:"email,omitempty"`
}

// IssueInfo contains issue metadata from a webhook.
type IssueInfo struct {
	Number  int      `json:"number"`
	Title   string   `json:"title"`
	Body    string   `json:"body"`
	State   string   `json:"state"`
	Labels  []string `json:"labels"`
	HTMLURL string   `json:"html_url"`
}

// PullRequestInfo contains PR/MR metadata from a webhook.
type PullRequestInfo struct {
	Number     int      `json:"number"`
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	State      string   `json:"state"`
	Labels     []string `json:"labels"`
	HeadBranch string   `json:"head_branch"`
	HeadSHA    string   `json:"head_sha"`
	BaseBranch string   `json:"base_branch"`
	HTMLURL    string   `json:"html_url"`
	Draft      bool     `json:"draft"`
}

// CommentInfo contains comment metadata from a webhook.
type CommentInfo struct {
	ID      int64  `json:"id"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
}

// WebhookEvent represents a normalized webhook event from any provider.
type WebhookEvent struct {
	// Delivery metadata.
	ID        string    `json:"id"`        // Unique delivery ID
	Provider  string    `json:"provider"`  // "github" or "gitlab"
	Type      EventType `json:"type"`      // Normalized event type
	Action    string    `json:"action"`    // Raw action (opened, edited, created, etc.)
	Timestamp time.Time `json:"timestamp"` // When event was received

	// Repository context.
	Repository RepositoryInfo `json:"repository"`

	// Actor who triggered the event.
	Sender UserInfo `json:"sender"`

	// Event-specific data (only one will be populated based on Type).
	Issue       *IssueInfo       `json:"issue,omitempty"`
	PullRequest *PullRequestInfo `json:"pull_request,omitempty"`
	Comment     *CommentInfo     `json:"comment,omitempty"`

	// Raw payload for provider-specific handling.
	RawPayload map[string]any `json:"raw_payload,omitempty"`
}

// ProviderReference returns the provider-specific reference string for the event.
// Examples: "github:owner/repo#123" or "gitlab:group/project#456".
func (e *WebhookEvent) ProviderReference() string {
	var number int
	switch {
	case e.Issue != nil:
		number = e.Issue.Number
	case e.PullRequest != nil:
		number = e.PullRequest.Number
	default:
		return ""
	}

	return e.Provider + ":" + e.Repository.FullName + "#" + itoa(number)
}

// itoa converts int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}

	return string(buf[i:])
}

// JobResult contains the outcome of a completed job.
type JobResult struct {
	Success        bool          `json:"success"`
	PRNumber       int           `json:"pr_number,omitempty"`
	PRURL          string        `json:"pr_url,omitempty"`
	CommentsPosted int           `json:"comments_posted"`
	ErrorMessage   string        `json:"error_message,omitempty"`
	Duration       time.Duration `json:"duration"`
}

// WebhookJob represents a queued automation job.
type WebhookJob struct {
	// Job identity.
	ID string `json:"id"`

	// Source event.
	Event *WebhookEvent `json:"event"`

	// Workflow configuration.
	WorkflowType WorkflowType `json:"workflow_type"`
	Command      string       `json:"command,omitempty"` // For WorkflowTypeCommand

	// Execution state.
	Status       JobStatus `json:"status"`
	Priority     int       `json:"priority"`
	Attempts     int       `json:"attempts"`
	MaxAttempts  int       `json:"max_attempts"`
	WorktreePath string    `json:"worktree_path,omitempty"`
	Error        string    `json:"error,omitempty"`

	// Timestamps.
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Result (populated on completion).
	Result *JobResult `json:"result,omitempty"`
}

// IsTerminal returns true if the job is in a terminal state.
func (j *WebhookJob) IsTerminal() bool {
	return j.Status == JobStatusCompleted ||
		j.Status == JobStatusFailed ||
		j.Status == JobStatusCancelled
}

// CanRetry returns true if the job can be retried.
func (j *WebhookJob) CanRetry() bool {
	return j.Status == JobStatusFailed && j.Attempts < j.MaxAttempts
}

// QueueStatus represents the current state of the job queue.
type QueueStatus struct {
	Enabled       bool `json:"enabled"`
	Running       bool `json:"running"`
	Workers       int  `json:"workers"`
	PendingJobs   int  `json:"pending_jobs"`
	RunningJobs   int  `json:"running_jobs"`
	CompletedJobs int  `json:"completed_jobs"`
	FailedJobs    int  `json:"failed_jobs"`
	CancelledJobs int  `json:"cancelled_jobs"`
}
