package youtrack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
)

func TestIssueToWorkUnitAndHelpers(t *testing.T) {
	p := &Provider{}
	nowMillis := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC).UnixMilli()

	issue := &Issue{
		ID:          "internal-1",
		IDReadable:  "ABC-123",
		Summary:     "Fix parser bug",
		Description: "Details",
		Created:     nowMillis,
		Updated:     nowMillis + 1000,
		Project: Project{
			Name:      "Main",
			ShortName: "ABC",
		},
		CustomFields: []CustomField{
			{Name: "Assignee", Value: map[string]interface{}{"id": "u1", "name": "Ada"}},
			{Name: "Priority", Value: map[string]interface{}{"name": "High"}},
			{Name: "State", Value: map[string]interface{}{"name": "In Progress"}},
			{Name: "Type", Value: map[string]interface{}{"name": "Bug"}},
		},
		Tags: []Tag{
			{ID: "t1", Name: "backend"},
			{ID: "t2", Name: "urgent"},
		},
		Subtasks: []IssueLink{
			{IDReadable: "ABC-124"},
		},
	}

	comments := []Comment{
		{
			ID:      "c1",
			Text:    "Looks good",
			Created: nowMillis,
			Updated: nowMillis,
			Author: User{
				ID:       "u2",
				FullName: "Bob",
			},
		},
	}
	attachments := []Attachment{
		{
			ID:       "a1",
			Name:     "report.txt",
			URL:      "https://example.com/report.txt",
			MimeType: "text/plain",
			Size:     42,
			Created:  nowMillis,
		},
	}

	wu := p.issueToWorkUnit(issue, comments, attachments)
	if wu.ID != "ABC-123" || wu.ExternalID != "internal-1" {
		t.Fatalf("unexpected IDs: %#v", wu)
	}
	if wu.Status != provider.StatusInProgress {
		t.Fatalf("status = %q, want in_progress", wu.Status)
	}
	if wu.Priority != provider.PriorityHigh {
		t.Fatalf("priority = %q, want high", wu.Priority)
	}
	if len(wu.Labels) != 2 || wu.Labels[0] != "backend" {
		t.Fatalf("labels = %#v", wu.Labels)
	}
	if len(wu.Assignees) != 1 || wu.Assignees[0].Name != "Ada" {
		t.Fatalf("assignees = %#v", wu.Assignees)
	}
	if len(wu.Attachments) != 1 || wu.Attachments[0].ID != "a1" {
		t.Fatalf("attachments = %#v", wu.Attachments)
	}
	if len(wu.Comments) != 1 || wu.Comments[0].Body != "Looks good" {
		t.Fatalf("comments = %#v", wu.Comments)
	}
	if len(wu.Subtasks) != 1 || wu.Subtasks[0] != "ABC-124" {
		t.Fatalf("subtasks = %#v", wu.Subtasks)
	}
	if wu.TaskType != "bug" {
		t.Fatalf("task type = %q, want bug", wu.TaskType)
	}
}

func TestMapStatusAndTaskTypeFallbacks(t *testing.T) {
	p := &Provider{}

	if got := p.mapStatus(&Issue{Resolved: 1}); got != provider.StatusDone {
		t.Fatalf("resolved issue status = %q, want done", got)
	}
	if got := p.mapStatus(&Issue{CustomFields: []CustomField{{Name: "State", Value: map[string]interface{}{"name": "Review"}}}}); got != provider.StatusReview {
		t.Fatalf("state-derived status = %q, want review", got)
	}
	if got := p.mapStatus(&Issue{}); got != provider.StatusOpen {
		t.Fatalf("default status = %q, want open", got)
	}

	if got := p.inferTaskType(&Issue{}); got != "issue" {
		t.Fatalf("default task type = %q, want issue", got)
	}
}

func TestFormatIssueMarkdown(t *testing.T) {
	nowMillis := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC).UnixMilli()
	issue := &Issue{
		IDReadable:  "ABC-7",
		Summary:     "Title",
		Description: "Body",
		Created:     nowMillis,
		Updated:     nowMillis,
		Project: Project{
			Name:      "Main",
			ShortName: "ABC",
		},
		Reporter: User{FullName: "Ada"},
		Tags: []Tag{
			{Name: "backend"},
		},
		CustomFields: []CustomField{
			{Name: "Type", Value: map[string]interface{}{"name": "Task"}},
		},
	}
	comments := []Comment{
		{Text: "visible", Created: nowMillis, Author: User{FullName: "Bob"}},
		{Text: "hidden", Created: nowMillis, Author: User{FullName: "Eve"}, Deleted: true},
	}

	md := formatIssueMarkdown(issue, comments)
	for _, want := range []string{
		"# ABC-7",
		"## Title",
		"**Project:** Main (ABC)",
		"**Reporter:** Ada",
		"**Tags:** backend",
		"### Description",
		"Body",
		"#### Bob",
		"visible",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q", want)
		}
	}
	if strings.Contains(md, "hidden") {
		t.Fatalf("deleted comment should be omitted")
	}
}

func TestProviderCommentsAndTagsOperations(t *testing.T) {
	t.Run("FetchComments filters deleted and AddComment maps response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/comments"):
				_, _ = w.Write([]byte(`{"data":[{"id":"c1","text":"keep","author":{"id":"u1","name":"Ada"},"created":1730000000000,"updated":1730000000000,"deleted":false},{"id":"c2","text":"drop","author":{"id":"u2","name":"Bob"},"created":1730000000000,"updated":1730000000000,"deleted":true}]}`))
			case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments"):
				_, _ = w.Write([]byte(`{"data":[{"id":"c3","text":"new","author":{"id":"u3","name":"Cat"},"created":1730000000000}]}`))
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		}))
		defer server.Close()

		p := &Provider{client: newYouTrackTestClient(server.URL)}
		comments, err := p.FetchComments(context.Background(), "ABC-1")
		if err != nil {
			t.Fatalf("FetchComments error: %v", err)
		}
		if len(comments) != 1 || comments[0].Body != "keep" {
			t.Fatalf("unexpected filtered comments: %#v", comments)
		}

		comment, err := p.AddComment(context.Background(), "ABC-1", "new")
		if err != nil {
			t.Fatalf("AddComment error: %v", err)
		}
		if comment.ID != "c3" || comment.Author.Name != "Cat" {
			t.Fatalf("unexpected mapped comment: %#v", comment)
		}
	})

	t.Run("AddLabels and RemoveLabels tolerate per-tag failures", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/tags"):
				_, _ = w.Write([]byte(`{"data":[{"id":"t1","name":"bug"},{"id":"t2","name":"old"}]}`))
			case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/tags"):
				if strings.Contains(r.URL.RawQuery, "fields") {
					_, _ = w.Write([]byte(`{"data":[{"id":"t3","name":"new"}]}`))

					return
				}
				t.Fatalf("unexpected add tag query: %s", r.URL.RawQuery)
			case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/tags/t1"):
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"fail"}`))
			case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/tags/t2"):
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		}))
		defer server.Close()

		p := &Provider{client: newYouTrackTestClient(server.URL)}
		if err := p.AddLabels(context.Background(), "ABC-1", []string{"bug", "new"}); err != nil {
			t.Fatalf("AddLabels error: %v", err)
		}
		// RemoveLabels should continue even if one delete fails
		if err := p.RemoveLabels(context.Background(), "ABC-1", []string{"bug", "old"}); err != nil {
			t.Fatalf("RemoveLabels error: %v", err)
		}
	})
}
