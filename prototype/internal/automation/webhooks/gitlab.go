package webhooks

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/valksor/go-mehrhof/internal/automation"
)

// GitLabParser parses GitLab webhook payloads.
type GitLabParser struct{}

// NewGitLabParser creates a new GitLab webhook parser.
func NewGitLabParser() *GitLabParser {
	return &GitLabParser{}
}

// ValidateSignature validates a GitLab webhook signature.
func (p *GitLabParser) ValidateSignature(r *http.Request, body []byte, secret string) error {
	return automation.ValidateGitLabSignature(r, secret)
}

// Parse parses a GitLab webhook request into a WebhookEvent.
func (p *GitLabParser) Parse(r *http.Request, body []byte) (*automation.WebhookEvent, error) {
	eventType := r.Header.Get("X-Gitlab-Event")

	if eventType == "" {
		return nil, errors.New("missing X-Gitlab-Event header")
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	event := &automation.WebhookEvent{
		ID:         getString(payload, "object_kind") + "-" + time.Now().Format("20060102150405"),
		Provider:   "gitlab",
		Timestamp:  time.Now(),
		RawPayload: payload,
	}

	// Parse repository/project.
	if project, ok := payload["project"].(map[string]any); ok {
		event.Repository = parseGitLabProject(project)
	}

	// Parse user/sender.
	if user, ok := payload["user"].(map[string]any); ok {
		event.Sender = parseGitLabUser(user)
	}

	// Parse event-specific data and determine event type.
	objectKind := getString(payload, "object_kind")
	switch objectKind {
	case "issue":
		event.Type = p.parseIssueEvent(event, payload)
	case "merge_request":
		event.Type = p.parseMergeRequestEvent(event, payload)
	case "note":
		event.Type = p.parseNoteEvent(event, payload)
	case "push":
		// Push events are typically not used for automation triggers.
		event.Type = automation.EventTypeUnknown
	default:
		event.Type = automation.EventTypeUnknown
	}

	return event, nil
}

// parseIssueEvent parses a GitLab issue event.
func (p *GitLabParser) parseIssueEvent(event *automation.WebhookEvent, payload map[string]any) automation.EventType {
	if attrs, ok := payload["object_attributes"].(map[string]any); ok {
		event.Issue = parseGitLabIssue(attrs, payload)
		event.Action = getString(attrs, "action")
	}

	switch event.Action {
	case "open":
		return automation.EventTypeIssueOpened
	case "close":
		return automation.EventTypeIssueClosed
	case "update":
		// Check if labels were changed.
		if changes, ok := payload["changes"].(map[string]any); ok {
			if _, hasLabels := changes["labels"]; hasLabels {
				return automation.EventTypeIssueLabeled
			}
		}

		return automation.EventTypeIssueEdited
	default:
		return automation.EventTypeUnknown
	}
}

// parseMergeRequestEvent parses a GitLab merge request event.
func (p *GitLabParser) parseMergeRequestEvent(event *automation.WebhookEvent, payload map[string]any) automation.EventType {
	if attrs, ok := payload["object_attributes"].(map[string]any); ok {
		event.PullRequest = parseGitLabMergeRequest(attrs, payload)
		event.Action = getString(attrs, "action")
	}

	switch event.Action {
	case "open":
		return automation.EventTypePROpened
	case "update", "reopen":
		return automation.EventTypePRUpdated
	case "close":
		return automation.EventTypePRClosed
	case "merge":
		return automation.EventTypePRMerged
	default:
		return automation.EventTypeUnknown
	}
}

// parseNoteEvent parses a GitLab note (comment) event.
func (p *GitLabParser) parseNoteEvent(event *automation.WebhookEvent, payload map[string]any) automation.EventType {
	if attrs, ok := payload["object_attributes"].(map[string]any); ok {
		event.Comment = &automation.CommentInfo{
			ID:      getInt64(attrs, "id"),
			Body:    getString(attrs, "note"),
			HTMLURL: getString(attrs, "url"),
		}

		noteableType := getString(attrs, "notable_type")
		switch noteableType {
		case "Issue":
			// Parse issue context.
			if issue, ok := payload["issue"].(map[string]any); ok {
				event.Issue = parseGitLabIssueSimple(issue)
			}

			return automation.EventTypeIssueComment
		case "MergeRequest":
			// Parse MR context.
			if mr, ok := payload["merge_request"].(map[string]any); ok {
				event.PullRequest = parseGitLabMRSimple(mr)
			}

			return automation.EventTypePRComment
		}
	}

	return automation.EventTypeUnknown
}

// parseGitLabProject parses a GitLab project object.
func parseGitLabProject(project map[string]any) automation.RepositoryInfo {
	// Parse namespace (owner).
	namespace := getString(project, "namespace")
	if namespace == "" {
		// Try path_with_namespace and extract.
		pathWithNamespace := getString(project, "path_with_namespace")
		if idx := lastIndex(pathWithNamespace, "/"); idx != -1 {
			namespace = pathWithNamespace[:idx]
		}
	}

	return automation.RepositoryInfo{
		Owner:         namespace,
		Name:          getString(project, "name"),
		FullName:      getString(project, "path_with_namespace"),
		DefaultBranch: getString(project, "default_branch"),
		CloneURL:      getString(project, "git_http_url"),
		HTMLURL:       getString(project, "web_url"),
	}
}

// parseGitLabUser parses a GitLab user object.
func parseGitLabUser(user map[string]any) automation.UserInfo {
	return automation.UserInfo{
		Login: getString(user, "username"),
		ID:    getInt64(user, "id"),
		Type:  "User", // GitLab doesn't distinguish user types in webhooks.
		Email: getString(user, "email"),
	}
}

// parseGitLabIssue parses GitLab issue from object_attributes.
func parseGitLabIssue(attrs map[string]any, payload map[string]any) *automation.IssueInfo {
	return &automation.IssueInfo{
		Number:  getInt(attrs, "iid"),
		Title:   getString(attrs, "title"),
		Body:    getString(attrs, "description"),
		State:   getString(attrs, "state"),
		Labels:  parseGitLabLabels(payload),
		HTMLURL: getString(attrs, "url"),
	}
}

// parseGitLabIssueSimple parses issue from simplified note context.
func parseGitLabIssueSimple(issue map[string]any) *automation.IssueInfo {
	return &automation.IssueInfo{
		Number:  getInt(issue, "iid"),
		Title:   getString(issue, "title"),
		Body:    getString(issue, "description"),
		State:   getString(issue, "state"),
		HTMLURL: getString(issue, "url"),
	}
}

// parseGitLabMergeRequest parses GitLab MR from object_attributes.
func parseGitLabMergeRequest(attrs map[string]any, payload map[string]any) *automation.PullRequestInfo {
	return &automation.PullRequestInfo{
		Number:     getInt(attrs, "iid"),
		Title:      getString(attrs, "title"),
		Body:       getString(attrs, "description"),
		State:      getString(attrs, "state"),
		Labels:     parseGitLabLabels(payload),
		HeadBranch: getString(attrs, "source_branch"),
		HeadSHA:    getString(attrs, "last_commit.id"),
		BaseBranch: getString(attrs, "target_branch"),
		HTMLURL:    getString(attrs, "url"),
		Draft:      getBool(attrs, "draft") || getBool(attrs, "work_in_progress"),
	}
}

// parseGitLabMRSimple parses MR from simplified note context.
func parseGitLabMRSimple(mr map[string]any) *automation.PullRequestInfo {
	return &automation.PullRequestInfo{
		Number:     getInt(mr, "iid"),
		Title:      getString(mr, "title"),
		Body:       getString(mr, "description"),
		State:      getString(mr, "state"),
		HeadBranch: getString(mr, "source_branch"),
		BaseBranch: getString(mr, "target_branch"),
		HTMLURL:    getString(mr, "url"),
	}
}

// parseGitLabLabels extracts labels from GitLab payload.
func parseGitLabLabels(payload map[string]any) []string {
	labels, ok := payload["labels"].([]any)
	if !ok {
		return nil
	}

	result := make([]string, 0, len(labels))
	for _, l := range labels {
		if label, ok := l.(map[string]any); ok {
			if title := getString(label, "title"); title != "" {
				result = append(result, title)
			}
		}
	}

	return result
}

// lastIndex returns the index of the last occurrence of substr in s, or -1.
func lastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}
