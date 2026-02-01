package webhooks

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/valksor/go-mehrhof/internal/automation"
)

// GitHubParser parses GitHub webhook payloads.
type GitHubParser struct{}

// NewGitHubParser creates a new GitHub webhook parser.
func NewGitHubParser() *GitHubParser {
	return &GitHubParser{}
}

// ValidateSignature validates a GitHub webhook signature.
func (p *GitHubParser) ValidateSignature(r *http.Request, body []byte, secret string) error {
	return automation.ValidateGitHubSignature(r, body, secret)
}

// Parse parses a GitHub webhook request into a WebhookEvent.
func (p *GitHubParser) Parse(r *http.Request, body []byte) (*automation.WebhookEvent, error) {
	eventType := r.Header.Get("X-GitHub-Event")     //nolint:canonicalheader // GitHub uses this casing
	deliveryID := r.Header.Get("X-GitHub-Delivery") //nolint:canonicalheader // GitHub uses this casing

	if eventType == "" {
		return nil, errors.New("missing X-GitHub-Event header")
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	event := &automation.WebhookEvent{
		ID:         deliveryID,
		Provider:   "github",
		Action:     getString(payload, "action"),
		Timestamp:  time.Now(),
		RawPayload: payload,
	}

	// Parse repository.
	if repo, ok := payload["repository"].(map[string]any); ok {
		event.Repository = parseGitHubRepository(repo)
	}

	// Parse sender.
	if sender, ok := payload["sender"].(map[string]any); ok {
		event.Sender = parseGitHubUser(sender)
	}

	// Parse event-specific data and determine event type.
	switch eventType {
	case "issues":
		event.Type = p.parseIssueEvent(event, payload)
	case "pull_request":
		event.Type = p.parsePullRequestEvent(event, payload)
	case "issue_comment":
		event.Type = p.parseIssueCommentEvent(event, payload)
	case "pull_request_review_comment":
		event.Type = automation.EventTypePRComment
		p.parseComment(event, payload)
	case "ping":
		// Ping event for webhook registration verification.
		event.Type = automation.EventTypeUnknown
	default:
		event.Type = automation.EventTypeUnknown
	}

	return event, nil
}

// parseIssueEvent parses an issue event.
func (p *GitHubParser) parseIssueEvent(event *automation.WebhookEvent, payload map[string]any) automation.EventType {
	if issue, ok := payload["issue"].(map[string]any); ok {
		event.Issue = parseGitHubIssue(issue)
	}

	switch event.Action {
	case "opened":
		return automation.EventTypeIssueOpened
	case "closed":
		return automation.EventTypeIssueClosed
	case "labeled":
		return automation.EventTypeIssueLabeled
	case "edited":
		return automation.EventTypeIssueEdited
	default:
		return automation.EventTypeUnknown
	}
}

// parsePullRequestEvent parses a pull request event.
func (p *GitHubParser) parsePullRequestEvent(event *automation.WebhookEvent, payload map[string]any) automation.EventType {
	if pr, ok := payload["pull_request"].(map[string]any); ok {
		event.PullRequest = parseGitHubPullRequest(pr)
	}

	switch event.Action {
	case "opened":
		return automation.EventTypePROpened
	case "synchronize", "edited", "reopened":
		return automation.EventTypePRUpdated
	case "closed":
		if event.PullRequest != nil && getString(payload, "merged") == "true" {
			return automation.EventTypePRMerged
		}

		return automation.EventTypePRClosed
	default:
		return automation.EventTypeUnknown
	}
}

// parseIssueCommentEvent parses an issue comment event.
func (p *GitHubParser) parseIssueCommentEvent(event *automation.WebhookEvent, payload map[string]any) automation.EventType {
	// Parse the comment.
	p.parseComment(event, payload)

	// Parse issue context if present.
	if issue, ok := payload["issue"].(map[string]any); ok {
		event.Issue = parseGitHubIssue(issue)

		// Check if this is a PR (issue comments on PRs have pull_request field).
		if _, isPR := issue["pull_request"]; isPR {
			return automation.EventTypePRComment
		}
	}

	return automation.EventTypeIssueComment
}

// parseComment parses comment from payload.
func (p *GitHubParser) parseComment(event *automation.WebhookEvent, payload map[string]any) {
	if comment, ok := payload["comment"].(map[string]any); ok {
		event.Comment = &automation.CommentInfo{
			ID:      getInt64(comment, "id"),
			Body:    getString(comment, "body"),
			HTMLURL: getString(comment, "html_url"),
		}
	}
}

// parseGitHubRepository parses a GitHub repository object.
func parseGitHubRepository(repo map[string]any) automation.RepositoryInfo {
	owner := ""
	if ownerObj, ok := repo["owner"].(map[string]any); ok {
		owner = getString(ownerObj, "login")
	}

	return automation.RepositoryInfo{
		Owner:         owner,
		Name:          getString(repo, "name"),
		FullName:      getString(repo, "full_name"),
		DefaultBranch: getString(repo, "default_branch"),
		CloneURL:      getString(repo, "clone_url"),
		HTMLURL:       getString(repo, "html_url"),
	}
}

// parseGitHubUser parses a GitHub user object.
func parseGitHubUser(user map[string]any) automation.UserInfo {
	return automation.UserInfo{
		Login: getString(user, "login"),
		ID:    getInt64(user, "id"),
		Type:  getString(user, "type"),
		Email: getString(user, "email"),
	}
}

// parseGitHubIssue parses a GitHub issue object.
func parseGitHubIssue(issue map[string]any) *automation.IssueInfo {
	return &automation.IssueInfo{
		Number:  getInt(issue, "number"),
		Title:   getString(issue, "title"),
		Body:    getString(issue, "body"),
		State:   getString(issue, "state"),
		Labels:  parseLabels(issue),
		HTMLURL: getString(issue, "html_url"),
	}
}

// parseGitHubPullRequest parses a GitHub pull request object.
func parseGitHubPullRequest(pr map[string]any) *automation.PullRequestInfo {
	headBranch := ""
	headSHA := ""
	if head, ok := pr["head"].(map[string]any); ok {
		headBranch = getString(head, "ref")
		headSHA = getString(head, "sha")
	}

	baseBranch := ""
	if base, ok := pr["base"].(map[string]any); ok {
		baseBranch = getString(base, "ref")
	}

	return &automation.PullRequestInfo{
		Number:     getInt(pr, "number"),
		Title:      getString(pr, "title"),
		Body:       getString(pr, "body"),
		State:      getString(pr, "state"),
		Labels:     parseLabels(pr),
		HeadBranch: headBranch,
		HeadSHA:    headSHA,
		BaseBranch: baseBranch,
		HTMLURL:    getString(pr, "html_url"),
		Draft:      getBool(pr, "draft"),
	}
}

// parseLabels extracts label names from a GitHub issue or PR.
func parseLabels(obj map[string]any) []string {
	labels, ok := obj["labels"].([]any)
	if !ok {
		return nil
	}

	result := make([]string, 0, len(labels))
	for _, l := range labels {
		if label, ok := l.(map[string]any); ok {
			if name := getString(label, "name"); name != "" {
				result = append(result, name)
			}
		}
	}

	return result
}

// Helper functions for safe type extraction.

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}

	return ""
}

func getInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	}

	return 0
}

//nolint:unparam // key is parameterized for consistency with getString/getInt
func getInt64(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	}

	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}

	return false
}
