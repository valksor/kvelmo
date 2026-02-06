package linear

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
)

func TestInferTaskTypeFromLabelsAndLower(t *testing.T) {
	tests := []struct {
		want   string
		labels []string
	}{
		{labels: []string{"BUG"}, want: "fix"},
		{labels: []string{"enhancement"}, want: "feature"},
		{labels: []string{"Documentation"}, want: "docs"},
		{labels: []string{"refactor"}, want: "refactor"},
		{labels: []string{"test"}, want: "test"},
		{labels: []string{"ci"}, want: "ci"},
		{labels: []string{"misc"}, want: "issue"},
	}
	for _, tt := range tests {
		if got := inferTaskTypeFromLabels(tt.labels); got != tt.want {
			t.Fatalf("inferTaskTypeFromLabels(%v) = %q, want %q", tt.labels, got, tt.want)
		}
	}

	if got := lower("AbC-123_X"); got != "abc-123_x" {
		t.Fatalf("lower() = %q", got)
	}
}

func TestCreateWorkUnit(t *testing.T) {
	t.Run("team required", func(t *testing.T) {
		p := &Provider{client: NewClient("test-token")}
		_, err := p.CreateWorkUnit(context.Background(), provider.CreateWorkUnitOptions{Title: "x"})
		if !errors.Is(err, ErrTeamRequired) {
			t.Fatalf("expected ErrTeamRequired, got %v", err)
		}
	})

	t.Run("success maps work unit", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{
				status: http.StatusOK,
				body: `{
					"data":{
						"issueCreate":{
							"success":true,
							"issue":{
								"id":"i1",
								"identifier":"ENG-9",
								"title":"Created issue",
								"description":"Desc",
								"state":{"id":"s1","name":"Todo","type":"backlog"},
								"priority":2,
								"labels":{"nodes":[{"id":"l1","name":"bug","color":"red"}]},
								"assignee":{"id":"u1","name":"Ada","email":"ada@example.com"},
								"createdAt":"2026-01-01T00:00:00Z",
								"updatedAt":"2026-01-01T00:00:00Z",
								"url":"https://linear.app/eng/issue/ENG-9",
								"team":{"key":"ENG","name":"Engineering"}
							}
						}
					}
				}`,
				assert: func(t *testing.T, req graphqlRequest, _ *http.Request) {
					t.Helper()
					input, ok := req.Variables["input"].(map[string]any)
					if !ok {
						t.Fatalf("input type = %T", req.Variables["input"])
					}
					if input["title"] != "Created issue" {
						t.Fatalf("title = %v", input["title"])
					}
				},
			},
		})
		defer server.Close()

		p := &Provider{client: newTestClient(server.URL), team: "ENG"}
		wu, err := p.CreateWorkUnit(context.Background(), provider.CreateWorkUnitOptions{
			Title:       "Created issue",
			Description: "Desc",
			Priority:    provider.PriorityHigh,
			Labels:      []string{"bug"},
			Assignees:   []string{"u1"},
		})
		if err != nil {
			t.Fatalf("CreateWorkUnit error: %v", err)
		}
		if wu.ExternalID != "ENG-9" || wu.TaskType != "fix" {
			t.Fatalf("unexpected work unit: %#v", wu)
		}
	})
}

func TestDependenciesOperations(t *testing.T) {
	t.Run("client nil guard", func(t *testing.T) {
		p := &Provider{}
		if err := p.CreateDependency(context.Background(), "ENG-1", "ENG-2"); err == nil {
			t.Fatalf("expected client nil error")
		}
		_, err := p.GetDependencies(context.Background(), "ENG-2")
		if err == nil {
			t.Fatalf("expected client nil error")
		}
	})

	t.Run("create and read dependencies", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{
				status: http.StatusOK,
				body:   `{"data":{"issue":{"id":"i2","identifier":"ENG-2","title":"x","description":"Body","state":{"id":"s1","name":"Todo","type":"backlog"},"priority":3,"labels":{"nodes":[]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","url":"https://linear.app"}}}`,
			},
			{
				status: http.StatusOK,
				body:   `{"data":{"issueUpdate":{"success":true}}}`,
			},
			{
				status: http.StatusOK,
				body:   `{"data":{"issue":{"id":"i2","identifier":"ENG-2","title":"x","description":"**Depends on:** ENG-1, ENG-3","state":{"id":"s1","name":"Todo","type":"backlog"},"priority":3,"labels":{"nodes":[]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","url":"https://linear.app"}}}`,
			},
		})
		defer server.Close()

		p := &Provider{client: newTestClient(server.URL)}
		if err := p.CreateDependency(context.Background(), "ENG-1", "ENG-2"); err != nil {
			t.Fatalf("CreateDependency error: %v", err)
		}
		deps, err := p.GetDependencies(context.Background(), "ENG-2")
		if err != nil {
			t.Fatalf("GetDependencies error: %v", err)
		}
		if len(deps) != 2 || deps[0] != "ENG-1" || deps[1] != "ENG-3" {
			t.Fatalf("deps = %#v", deps)
		}
	})

	t.Run("existing dependency does not update", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{
				status: http.StatusOK,
				body:   `{"data":{"issue":{"id":"i2","identifier":"ENG-2","title":"x","description":"**Depends on:** ENG-1","state":{"id":"s1","name":"Todo","type":"backlog"},"priority":3,"labels":{"nodes":[]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","url":"https://linear.app"}}}`,
			},
		})
		defer server.Close()

		p := &Provider{client: newTestClient(server.URL)}
		if err := p.CreateDependency(context.Background(), "ENG-1", "ENG-2"); err != nil {
			t.Fatalf("CreateDependency should no-op, got %v", err)
		}
	})
}

func TestSubtasksOperations(t *testing.T) {
	t.Run("FetchParent returns ErrNotASubtask", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{
				status: http.StatusOK,
				body:   `{"data":{"issue":{"id":"c1","identifier":"ENG-10","title":"Child","description":"","state":{"id":"s1","name":"Todo","type":"backlog"},"priority":3,"labels":{"nodes":[]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","url":"https://linear.app"}}}`,
			},
		})
		defer server.Close()

		p := &Provider{client: newTestClient(server.URL)}
		_, err := p.FetchParent(context.Background(), "ENG-10")
		if !errors.Is(err, ErrNotASubtask) {
			t.Fatalf("expected ErrNotASubtask, got %v", err)
		}
	})

	t.Run("FetchParent and FetchSubtasks success", func(t *testing.T) {
		now := time.Now().UTC().Format(time.RFC3339)
		server := newGraphQLTestServer(t, []graphqlExchange{
			{
				status: http.StatusOK,
				body:   `{"data":{"issue":{"id":"child-1","identifier":"ENG-11","title":"Child","description":"","state":{"id":"s1","name":"Todo","type":"backlog"},"priority":3,"labels":{"nodes":[]},"createdAt":"` + now + `","updatedAt":"` + now + `","url":"https://linear.app","parent":{"id":"parent-1"}}}}`,
			},
			{
				status: http.StatusOK,
				body:   `{"data":{"issue":{"id":"parent-1","identifier":"ENG-1","title":"Parent","description":"","state":{"id":"s2","name":"In Progress","type":"started"},"priority":2,"labels":{"nodes":[{"id":"l1","name":"backend","color":"blue"}]},"createdAt":"` + now + `","updatedAt":"` + now + `","url":"https://linear.app","team":{"key":"ENG","name":"Engineering"}}}}`,
			},
			{
				status: http.StatusOK,
				body:   `{"data":{"issue":{"children":{"nodes":[{"id":"child-1","identifier":"ENG-11","title":"Child","description":"","state":{"id":"s1","name":"Todo","type":"backlog"},"priority":3,"labels":{"nodes":[]},"createdAt":"` + now + `","updatedAt":"` + now + `","url":"https://linear.app","team":{"key":"ENG","name":"Engineering"}}]}}}}`,
			},
		})
		defer server.Close()

		p := &Provider{client: newTestClient(server.URL)}
		parent, err := p.FetchParent(context.Background(), "ENG-11")
		if err != nil {
			t.Fatalf("FetchParent error: %v", err)
		}
		if parent.ExternalID != "ENG-1" {
			t.Fatalf("parent external id = %q", parent.ExternalID)
		}

		subtasks, err := p.FetchSubtasks(context.Background(), "ENG-1")
		if err != nil {
			t.Fatalf("FetchSubtasks error: %v", err)
		}
		if len(subtasks) != 1 || subtasks[0].TaskType != "subtask" {
			t.Fatalf("unexpected subtasks: %#v", subtasks)
		}
	})
}
