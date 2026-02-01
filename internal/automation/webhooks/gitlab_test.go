package webhooks

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/valksor/go-mehrhof/internal/automation"
)

func TestGitLabParser_Parse_IssueOpened(t *testing.T) {
	parser := NewGitLabParser()

	body := []byte(`{
		"object_kind": "issue",
		"object_attributes": {
			"iid": 42,
			"title": "Test GitLab Issue",
			"description": "This is a test issue",
			"state": "opened",
			"action": "open",
			"url": "https://gitlab.com/group/project/-/issues/42"
		},
		"project": {
			"name": "project",
			"namespace": "group",
			"path_with_namespace": "group/project",
			"default_branch": "main",
			"git_http_url": "https://gitlab.com/group/project.git",
			"web_url": "https://gitlab.com/group/project"
		},
		"user": {
			"username": "testuser",
			"id": 12345,
			"email": "test@example.com"
		},
		"labels": [{"title": "bug"}, {"title": "priority::high"}]
	}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Gitlab-Event", "Issue Hook")

	event, err := parser.Parse(req, body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if event.Type != automation.EventTypeIssueOpened {
		t.Errorf("Expected EventTypeIssueOpened, got %v", event.Type)
	}

	if event.Provider != "gitlab" {
		t.Errorf("Expected provider 'gitlab', got %s", event.Provider)
	}

	if event.Action != "open" {
		t.Errorf("Expected action 'open', got %s", event.Action)
	}

	if event.Issue == nil {
		t.Fatal("Expected issue to be set")
	}

	if event.Issue.Number != 42 {
		t.Errorf("Expected issue number 42, got %d", event.Issue.Number)
	}

	if event.Issue.Title != "Test GitLab Issue" {
		t.Errorf("Expected title 'Test GitLab Issue', got %s", event.Issue.Title)
	}

	if len(event.Issue.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(event.Issue.Labels))
	}

	if event.Repository.FullName != "group/project" {
		t.Errorf("Expected repo 'group/project', got %s", event.Repository.FullName)
	}

	if event.Sender.Login != "testuser" {
		t.Errorf("Expected sender 'testuser', got %s", event.Sender.Login)
	}
}

func TestGitLabParser_Parse_MergeRequestOpened(t *testing.T) {
	parser := NewGitLabParser()

	body := []byte(`{
		"object_kind": "merge_request",
		"object_attributes": {
			"iid": 99,
			"title": "Test MR",
			"description": "This is a test MR",
			"state": "opened",
			"action": "open",
			"source_branch": "feature-branch",
			"target_branch": "main",
			"url": "https://gitlab.com/group/project/-/merge_requests/99",
			"draft": false
		},
		"project": {
			"name": "project",
			"namespace": "group",
			"path_with_namespace": "group/project"
		},
		"user": {
			"username": "developer",
			"id": 67890
		},
		"labels": [{"title": "ready-for-review"}]
	}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Gitlab-Event", "Merge Request Hook")

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

	if event.PullRequest.Number != 99 {
		t.Errorf("Expected MR number 99, got %d", event.PullRequest.Number)
	}

	if event.PullRequest.HeadBranch != "feature-branch" {
		t.Errorf("Expected head branch 'feature-branch', got %s", event.PullRequest.HeadBranch)
	}

	if event.PullRequest.BaseBranch != "main" {
		t.Errorf("Expected base branch 'main', got %s", event.PullRequest.BaseBranch)
	}
}

func TestGitLabParser_Parse_NoteOnIssue(t *testing.T) {
	parser := NewGitLabParser()

	body := []byte(`{
		"object_kind": "note",
		"object_attributes": {
			"id": 555,
			"note": "@mehrhof fix this please",
			"notable_type": "Issue",
			"url": "https://gitlab.com/group/project/-/issues/42#note_555"
		},
		"issue": {
			"iid": 42,
			"title": "Test Issue",
			"description": "Issue description",
			"state": "opened"
		},
		"project": {
			"name": "project",
			"path_with_namespace": "group/project"
		},
		"user": {
			"username": "commenter",
			"id": 11111
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Gitlab-Event", "Note Hook")

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

	if event.Comment.Body != "@mehrhof fix this please" {
		t.Errorf("Expected comment body '@mehrhof fix this please', got %s", event.Comment.Body)
	}

	if event.Issue == nil {
		t.Fatal("Expected issue to be set")
	}

	if event.Issue.Number != 42 {
		t.Errorf("Expected issue number 42, got %d", event.Issue.Number)
	}
}

func TestGitLabParser_Parse_NoteOnMR(t *testing.T) {
	parser := NewGitLabParser()

	body := []byte(`{
		"object_kind": "note",
		"object_attributes": {
			"id": 666,
			"note": "@mehrhof review",
			"notable_type": "MergeRequest",
			"url": "https://gitlab.com/group/project/-/merge_requests/99#note_666"
		},
		"merge_request": {
			"iid": 99,
			"title": "Test MR",
			"description": "MR description",
			"state": "opened",
			"source_branch": "feature",
			"target_branch": "main"
		},
		"project": {
			"name": "project",
			"path_with_namespace": "group/project"
		},
		"user": {
			"username": "reviewer",
			"id": 22222
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Gitlab-Event", "Note Hook")

	event, err := parser.Parse(req, body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if event.Type != automation.EventTypePRComment {
		t.Errorf("Expected EventTypePRComment, got %v", event.Type)
	}

	if event.PullRequest == nil {
		t.Fatal("Expected pull_request to be set")
	}

	if event.PullRequest.Number != 99 {
		t.Errorf("Expected MR number 99, got %d", event.PullRequest.Number)
	}
}

func TestGitLabParser_Parse_IssueClosed(t *testing.T) {
	parser := NewGitLabParser()

	body := []byte(`{
		"object_kind": "issue",
		"object_attributes": {
			"iid": 42,
			"title": "Test Issue",
			"state": "closed",
			"action": "close"
		},
		"project": {
			"path_with_namespace": "group/project"
		},
		"user": {
			"username": "closer"
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Gitlab-Event", "Issue Hook")

	event, err := parser.Parse(req, body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if event.Type != automation.EventTypeIssueClosed {
		t.Errorf("Expected EventTypeIssueClosed, got %v", event.Type)
	}
}

func TestGitLabParser_Parse_MRMerged(t *testing.T) {
	parser := NewGitLabParser()

	body := []byte(`{
		"object_kind": "merge_request",
		"object_attributes": {
			"iid": 99,
			"title": "Test MR",
			"state": "merged",
			"action": "merge",
			"source_branch": "feature",
			"target_branch": "main"
		},
		"project": {
			"path_with_namespace": "group/project"
		},
		"user": {
			"username": "merger"
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Gitlab-Event", "Merge Request Hook")

	event, err := parser.Parse(req, body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if event.Type != automation.EventTypePRMerged {
		t.Errorf("Expected EventTypePRMerged, got %v", event.Type)
	}
}

func TestGitLabParser_Parse_IssueLabeled(t *testing.T) {
	parser := NewGitLabParser()

	body := []byte(`{
		"object_kind": "issue",
		"object_attributes": {
			"iid": 42,
			"title": "Test Issue",
			"state": "opened",
			"action": "update"
		},
		"changes": {
			"labels": {
				"previous": [{"title": "bug"}],
				"current": [{"title": "bug"}, {"title": "mehr-fix"}]
			}
		},
		"project": {
			"path_with_namespace": "group/project"
		},
		"user": {
			"username": "labeler"
		},
		"labels": [{"title": "bug"}, {"title": "mehr-fix"}]
	}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Gitlab-Event", "Issue Hook")

	event, err := parser.Parse(req, body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if event.Type != automation.EventTypeIssueLabeled {
		t.Errorf("Expected EventTypeIssueLabeled, got %v", event.Type)
	}
}

func TestGitLabParser_Parse_MissingHeader(t *testing.T) {
	parser := NewGitLabParser()

	body := []byte(`{"object_kind": "issue"}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	// No X-Gitlab-Event header.

	_, err := parser.Parse(req, body)
	if err == nil {
		t.Error("Expected error for missing X-Gitlab-Event header")
	}
}

func TestGitLabParser_Parse_InvalidJSON(t *testing.T) {
	parser := NewGitLabParser()

	body := []byte(`not valid json`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Gitlab-Event", "Issue Hook")

	_, err := parser.Parse(req, body)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestGitLabParser_Parse_PushEvent(t *testing.T) {
	parser := NewGitLabParser()

	body := []byte(`{
		"object_kind": "push",
		"ref": "refs/heads/main",
		"project": {
			"path_with_namespace": "group/project"
		},
		"user_name": "pusher"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Gitlab-Event", "Push Hook")

	event, err := parser.Parse(req, body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if event.Type != automation.EventTypeUnknown {
		t.Errorf("Expected EventTypeUnknown for push, got %v", event.Type)
	}
}

func TestParseGitLabProject(t *testing.T) {
	project := map[string]any{
		"name":                "my-project",
		"namespace":           "my-group",
		"path_with_namespace": "my-group/my-project",
		"default_branch":      "main",
		"git_http_url":        "https://gitlab.com/my-group/my-project.git",
		"web_url":             "https://gitlab.com/my-group/my-project",
	}

	repo := parseGitLabProject(project)

	if repo.Name != "my-project" {
		t.Errorf("Expected name 'my-project', got %s", repo.Name)
	}

	if repo.Owner != "my-group" {
		t.Errorf("Expected owner 'my-group', got %s", repo.Owner)
	}

	if repo.FullName != "my-group/my-project" {
		t.Errorf("Expected full name 'my-group/my-project', got %s", repo.FullName)
	}

	if repo.DefaultBranch != "main" {
		t.Errorf("Expected default branch 'main', got %s", repo.DefaultBranch)
	}
}

func TestParseGitLabUser(t *testing.T) {
	user := map[string]any{
		"username": "testuser",
		"id":       float64(12345),
		"email":    "test@example.com",
	}

	info := parseGitLabUser(user)

	if info.Login != "testuser" {
		t.Errorf("Expected login 'testuser', got %s", info.Login)
	}

	if info.ID != 12345 {
		t.Errorf("Expected ID 12345, got %d", info.ID)
	}

	if info.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", info.Email)
	}

	if info.Type != "User" {
		t.Errorf("Expected type 'User', got %s", info.Type)
	}
}

func TestLastIndex(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected int
	}{
		{
			name:     "found",
			s:        "group/subgroup/project",
			substr:   "/",
			expected: 14,
		},
		{
			name:     "not_found",
			s:        "project",
			substr:   "/",
			expected: -1,
		},
		{
			name:     "at_end",
			s:        "foo/bar/",
			substr:   "/",
			expected: 7,
		},
		{
			name:     "at_start",
			s:        "/foo/bar",
			substr:   "/",
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lastIndex(tt.s, tt.substr); got != tt.expected {
				t.Errorf("lastIndex(%q, %q) = %d, want %d", tt.s, tt.substr, got, tt.expected)
			}
		})
	}
}
