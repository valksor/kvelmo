package webhooks

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/valksor/go-mehrhof/internal/automation"
)

func TestGitHubParser_Parse_IssueOpened(t *testing.T) {
	parser := NewGitHubParser()

	body := []byte(`{
		"action": "opened",
		"issue": {
			"number": 123,
			"title": "Test Issue",
			"body": "This is a test issue",
			"state": "open",
			"labels": [{"name": "bug"}, {"name": "enhancement"}],
			"html_url": "https://github.com/owner/repo/issues/123"
		},
		"repository": {
			"name": "repo",
			"full_name": "owner/repo",
			"default_branch": "main",
			"clone_url": "https://github.com/owner/repo.git",
			"html_url": "https://github.com/owner/repo",
			"owner": {"login": "owner"}
		},
		"sender": {
			"login": "testuser",
			"id": 12345,
			"type": "User"
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-GitHub-Event", "issues")     //nolint:canonicalheader // GitHub uses this casing
	req.Header.Set("X-GitHub-Delivery", "abc-123") //nolint:canonicalheader // GitHub uses this casing

	event, err := parser.Parse(req, body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if event.Type != automation.EventTypeIssueOpened {
		t.Errorf("Expected EventTypeIssueOpened, got %v", event.Type)
	}

	if event.Provider != "github" {
		t.Errorf("Expected provider 'github', got %s", event.Provider)
	}

	if event.ID != "abc-123" {
		t.Errorf("Expected ID 'abc-123', got %s", event.ID)
	}

	if event.Action != "opened" {
		t.Errorf("Expected action 'opened', got %s", event.Action)
	}

	if event.Issue == nil {
		t.Fatal("Expected issue to be set")
	}

	if event.Issue.Number != 123 {
		t.Errorf("Expected issue number 123, got %d", event.Issue.Number)
	}

	if event.Issue.Title != "Test Issue" {
		t.Errorf("Expected title 'Test Issue', got %s", event.Issue.Title)
	}

	if len(event.Issue.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(event.Issue.Labels))
	}

	if event.Repository.FullName != "owner/repo" {
		t.Errorf("Expected repo 'owner/repo', got %s", event.Repository.FullName)
	}

	if event.Sender.Login != "testuser" {
		t.Errorf("Expected sender 'testuser', got %s", event.Sender.Login)
	}
}

func TestGitHubParser_Parse_PullRequestOpened(t *testing.T) {
	parser := NewGitHubParser()

	body := []byte(`{
		"action": "opened",
		"pull_request": {
			"number": 456,
			"title": "Test PR",
			"body": "This is a test PR",
			"state": "open",
			"html_url": "https://github.com/owner/repo/pull/456",
			"draft": false,
			"head": {"ref": "feature-branch", "sha": "abc123"},
			"base": {"ref": "main"},
			"labels": [{"name": "ready-for-review"}]
		},
		"repository": {
			"name": "repo",
			"full_name": "owner/repo",
			"owner": {"login": "owner"}
		},
		"sender": {
			"login": "developer",
			"id": 67890,
			"type": "User"
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-GitHub-Event", "pull_request") //nolint:canonicalheader // GitHub uses this casing
	req.Header.Set("X-GitHub-Delivery", "pr-456")    //nolint:canonicalheader // GitHub uses this casing

	event, err := parser.Parse(req, body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if event.Type != automation.EventTypePROpened {
		t.Errorf("Expected EventTypePROpened, got %v", event.Type)
	}

	if event.PullRequest == nil {
		t.Fatal("Expected pull_request to be set")
	}

	if event.PullRequest.Number != 456 {
		t.Errorf("Expected PR number 456, got %d", event.PullRequest.Number)
	}

	if event.PullRequest.HeadBranch != "feature-branch" {
		t.Errorf("Expected head branch 'feature-branch', got %s", event.PullRequest.HeadBranch)
	}

	if event.PullRequest.BaseBranch != "main" {
		t.Errorf("Expected base branch 'main', got %s", event.PullRequest.BaseBranch)
	}

	if event.PullRequest.HeadSHA != "abc123" {
		t.Errorf("Expected head SHA 'abc123', got %s", event.PullRequest.HeadSHA)
	}
}

func TestGitHubParser_Parse_IssueComment(t *testing.T) {
	parser := NewGitHubParser()

	body := []byte(`{
		"action": "created",
		"comment": {
			"id": 789,
			"body": "@mehrhof fix this issue",
			"html_url": "https://github.com/owner/repo/issues/123#comment-789"
		},
		"issue": {
			"number": 123,
			"title": "Test Issue",
			"state": "open"
		},
		"repository": {
			"name": "repo",
			"full_name": "owner/repo",
			"owner": {"login": "owner"}
		},
		"sender": {
			"login": "commenter",
			"id": 11111,
			"type": "User"
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-GitHub-Event", "issue_comment")  //nolint:canonicalheader // GitHub uses this casing
	req.Header.Set("X-GitHub-Delivery", "comment-789") //nolint:canonicalheader // GitHub uses this casing

	event, err := parser.Parse(req, body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if event.Type != automation.EventTypeIssueComment {
		t.Errorf("Expected EventTypeIssueComment, got %v", event.Type)
	}

	if event.Comment == nil {
		t.Fatal("Expected comment to be set")
	}

	if event.Comment.Body != "@mehrhof fix this issue" {
		t.Errorf("Expected comment body '@mehrhof fix this issue', got %s", event.Comment.Body)
	}

	if event.Issue == nil {
		t.Fatal("Expected issue to be set")
	}

	if event.Issue.Number != 123 {
		t.Errorf("Expected issue number 123, got %d", event.Issue.Number)
	}
}

func TestGitHubParser_Parse_PRComment(t *testing.T) {
	parser := NewGitHubParser()

	// Issue comment on a PR (has pull_request field).
	body := []byte(`{
		"action": "created",
		"comment": {
			"id": 999,
			"body": "@mehrhof review",
			"html_url": "https://github.com/owner/repo/pull/456#comment-999"
		},
		"issue": {
			"number": 456,
			"title": "Test PR",
			"state": "open",
			"pull_request": {}
		},
		"repository": {
			"name": "repo",
			"full_name": "owner/repo",
			"owner": {"login": "owner"}
		},
		"sender": {
			"login": "reviewer",
			"id": 22222,
			"type": "User"
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-GitHub-Event", "issue_comment")  //nolint:canonicalheader // GitHub uses this casing
	req.Header.Set("X-GitHub-Delivery", "comment-999") //nolint:canonicalheader // GitHub uses this casing

	event, err := parser.Parse(req, body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should be PR comment since issue has pull_request field.
	if event.Type != automation.EventTypePRComment {
		t.Errorf("Expected EventTypePRComment, got %v", event.Type)
	}
}

func TestGitHubParser_Parse_MissingHeader(t *testing.T) {
	parser := NewGitHubParser()

	body := []byte(`{"action": "opened"}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	// No X-GitHub-Event header.

	_, err := parser.Parse(req, body)
	if err == nil {
		t.Error("Expected error for missing X-GitHub-Event header")
	}
}

func TestGitHubParser_Parse_InvalidJSON(t *testing.T) {
	parser := NewGitHubParser()

	body := []byte(`not valid json`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-GitHub-Event", "issues") //nolint:canonicalheader // GitHub uses this casing

	_, err := parser.Parse(req, body)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestGitHubParser_Parse_PingEvent(t *testing.T) {
	parser := NewGitHubParser()

	body := []byte(`{"zen": "Keep it logically awesome.", "hook_id": 12345}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-GitHub-Event", "ping") //nolint:canonicalheader // GitHub uses this casing

	event, err := parser.Parse(req, body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if event.Type != automation.EventTypeUnknown {
		t.Errorf("Expected EventTypeUnknown for ping, got %v", event.Type)
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("getString", func(t *testing.T) {
		m := map[string]any{"key": "value", "number": 123}

		if got := getString(m, "key"); got != "value" {
			t.Errorf("Expected 'value', got %s", got)
		}

		if got := getString(m, "number"); got != "" {
			t.Errorf("Expected empty string for non-string, got %s", got)
		}

		if got := getString(m, "missing"); got != "" {
			t.Errorf("Expected empty string for missing key, got %s", got)
		}
	})

	t.Run("getInt", func(t *testing.T) {
		m := map[string]any{"float": 42.0, "int": 42, "int64": int64(42), "str": "42"}

		if got := getInt(m, "float"); got != 42 {
			t.Errorf("Expected 42 from float, got %d", got)
		}

		if got := getInt(m, "int"); got != 42 {
			t.Errorf("Expected 42 from int, got %d", got)
		}

		if got := getInt(m, "int64"); got != 42 {
			t.Errorf("Expected 42 from int64, got %d", got)
		}

		if got := getInt(m, "str"); got != 0 {
			t.Errorf("Expected 0 from string, got %d", got)
		}
	})

	t.Run("getBool", func(t *testing.T) {
		m := map[string]any{"true": true, "false": false, "str": "true"}

		if got := getBool(m, "true"); !got {
			t.Error("Expected true")
		}

		if got := getBool(m, "false"); got {
			t.Error("Expected false")
		}

		if got := getBool(m, "str"); got {
			t.Error("Expected false for non-bool")
		}

		if got := getBool(m, "missing"); got {
			t.Error("Expected false for missing key")
		}
	})
}
