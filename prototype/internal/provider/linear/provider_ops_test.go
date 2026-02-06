package linear

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/provider"
)

func TestProviderListRequiresTeam(t *testing.T) {
	p := &Provider{
		client: NewClient("test-token"),
	}

	_, err := p.List(context.Background(), provider.ListOptions{})
	if !errors.Is(err, ErrTeamRequired) {
		t.Fatalf("expected ErrTeamRequired, got %v", err)
	}
}

func TestProviderListFiltersOffsetAndLimit(t *testing.T) {
	server := newGraphQLTestServer(t, []graphqlExchange{
		{
			status: http.StatusOK,
			body: `{
				"data":{
					"team":{
						"issues":{
							"nodes":[
								{"id":"1","identifier":"ENG-1","title":"First issue","description":"A","state":{"id":"s1","name":"Todo","type":"backlog"},"priority":2,"labels":{"nodes":[{"id":"l1","name":"bug","color":"red"}]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","url":"https://linear.app/eng/issue/ENG-1"},
								{"id":"2","identifier":"ENG-2","title":"Second issue","description":"B","state":{"id":"s2","name":"In Progress","type":"started"},"priority":3,"labels":{"nodes":[{"id":"l2","name":"bug","color":"red"},{"id":"l3","name":"backend","color":"blue"}]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","url":"https://linear.app/eng/issue/ENG-2"},
								{"id":"3","identifier":"ENG-3","title":"Third issue","description":"C","state":{"id":"s3","name":"Done","type":"completed"},"priority":1,"labels":{"nodes":[{"id":"l4","name":"frontend","color":"green"}]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","url":"https://linear.app/eng/issue/ENG-3"}
							],
							"pageInfo":{"hasNextPage":false,"endCursor":""}
						}
					}
				}
			}`,
		},
	})
	defer server.Close()

	p := &Provider{
		client: newTestClient(server.URL),
		team:   "ENG",
	}

	got, err := p.List(context.Background(), provider.ListOptions{
		Labels: []string{"bug"},
		Offset: 1,
		Limit:  1,
		Status: provider.StatusInProgress,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("List returned %d items, want 1", len(got))
	}
	if got[0].ExternalID != "ENG-2" {
		t.Fatalf("ExternalID = %q, want ENG-2", got[0].ExternalID)
	}
	if got[0].Status != provider.StatusInProgress {
		t.Fatalf("Status = %q, want in_progress", got[0].Status)
	}
}

func TestProviderSnapshotIncludesMetadataAndComments(t *testing.T) {
	server := newGraphQLTestServer(t, []graphqlExchange{
		{
			status: http.StatusOK,
			body: `{
				"data":{
					"issue":{
						"id":"i1",
						"identifier":"ENG-12",
						"title":"Snapshot title",
						"description":"Snapshot body",
						"state":{"id":"s1","name":"In Progress","type":"started"},
						"priority":2,
						"labels":{"nodes":[{"id":"l1","name":"backend","color":"blue"}]},
						"assignee":{"id":"u1","name":"Ada","email":"ada@example.com"},
						"createdAt":"2026-01-05T10:00:00Z",
						"updatedAt":"2026-01-06T10:00:00Z",
						"url":"https://linear.app/eng/issue/ENG-12",
						"team":{"key":"ENG","name":"Engineering"}
					}
				}
			}`,
		},
		{
			status: http.StatusOK,
			body: `{
				"data":{
					"issue":{
						"comments":{
							"nodes":[
								{"id":"c1","body":"Looks good","user":{"id":"u2","name":"Bob"},"createdAt":"2026-01-07T11:00:00Z","updatedAt":"2026-01-07T11:00:00Z"}
							]
						}
					}
				}
			}`,
		},
	})
	defer server.Close()

	p := &Provider{client: newTestClient(server.URL)}
	snapshot, err := p.Snapshot(context.Background(), "ENG-12")
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if snapshot.Type != ProviderName || snapshot.Ref != "ENG-12" {
		t.Fatalf("unexpected snapshot metadata: %#v", snapshot)
	}
	if len(snapshot.Files) != 1 || snapshot.Files[0].Path != "issue.md" {
		t.Fatalf("unexpected snapshot files: %#v", snapshot.Files)
	}

	content := snapshot.Files[0].Content
	for _, want := range []string{
		"# ENG-12",
		"## Snapshot title",
		"**Status:** In Progress",
		"**Team:** Engineering (ENG)",
		"## Description",
		"Snapshot body",
		"## Comments",
		"Looks good",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("snapshot content missing %q", want)
		}
	}
}

func TestProviderStatusAndLabels(t *testing.T) {
	t.Run("UpdateStatus maps to state name", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{
				status: http.StatusOK,
				body:   `{"data":{"issue":{"id":"issue-1","identifier":"ENG-1","title":"t","description":"","state":{"id":"s1","name":"Todo","type":"backlog"},"priority":3,"labels":{"nodes":[]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","url":"https://linear.app"}}}`,
			},
			{
				status: http.StatusOK,
				body:   `{"data":{"issueUpdate":{"success":true,"issue":{"id":"issue-1"}}}}`,
				assert: func(t *testing.T, req graphqlRequest, _ *http.Request) {
					t.Helper()
					input, ok := req.Variables["input"].(map[string]any)
					if !ok {
						t.Fatalf("input variable type = %T, want map[string]any", req.Variables["input"])
					}
					if input["stateId"] != "In Review" {
						t.Fatalf("stateId = %v, want In Review", input["stateId"])
					}
				},
			},
		})
		defer server.Close()

		p := &Provider{client: newTestClient(server.URL)}
		if err := p.UpdateStatus(context.Background(), "ENG-1", provider.StatusReview); err != nil {
			t.Fatalf("UpdateStatus returned error: %v", err)
		}
	})

	t.Run("AddLabels merges existing IDs and new labels", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{
				status: http.StatusOK,
				body:   `{"data":{"issue":{"id":"issue-1","identifier":"ENG-1","title":"t","description":"","state":{"id":"s1","name":"Todo","type":"backlog"},"priority":3,"labels":{"nodes":[{"id":"l1","name":"bug","color":"red"}]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","url":"https://linear.app"}}}`,
			},
			{
				status: http.StatusOK,
				body:   `{"data":{"issueUpdate":{"success":true,"issue":{"id":"issue-1"}}}}`,
				assert: func(t *testing.T, req graphqlRequest, _ *http.Request) {
					t.Helper()
					input, ok := req.Variables["input"].(map[string]any)
					if !ok {
						t.Fatalf("input variable type = %T, want map[string]any", req.Variables["input"])
					}
					labelIDs, ok := input["labelIds"].([]any)
					if !ok {
						t.Fatalf("labelIds type = %T, want []any", input["labelIds"])
					}
					if len(labelIDs) != 2 {
						t.Fatalf("labelIds len = %d, want 2", len(labelIDs))
					}
				},
			},
		})
		defer server.Close()

		p := &Provider{client: newTestClient(server.URL)}
		if err := p.AddLabels(context.Background(), "ENG-1", []string{"bug", "ops"}); err != nil {
			t.Fatalf("AddLabels returned error: %v", err)
		}
	})

	t.Run("RemoveLabels keeps non-removed IDs", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{
				status: http.StatusOK,
				body:   `{"data":{"issue":{"id":"issue-1","identifier":"ENG-1","title":"t","description":"","state":{"id":"s1","name":"Todo","type":"backlog"},"priority":3,"labels":{"nodes":[{"id":"l1","name":"bug","color":"red"},{"id":"l2","name":"ops","color":"blue"}]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","url":"https://linear.app"}}}`,
			},
			{
				status: http.StatusOK,
				body:   `{"data":{"issueUpdate":{"success":true,"issue":{"id":"issue-1"}}}}`,
				assert: func(t *testing.T, req graphqlRequest, _ *http.Request) {
					t.Helper()
					input, ok := req.Variables["input"].(map[string]any)
					if !ok {
						t.Fatalf("input variable type = %T, want map[string]any", req.Variables["input"])
					}
					labelIDs, ok := input["labelIds"].([]any)
					if !ok {
						t.Fatalf("labelIds type = %T, want []any", input["labelIds"])
					}
					if len(labelIDs) != 1 || labelIDs[0] != "l2" {
						t.Fatalf("labelIds = %v, want [l2]", labelIDs)
					}
				},
			},
		})
		defer server.Close()

		p := &Provider{client: newTestClient(server.URL)}
		if err := p.RemoveLabels(context.Background(), "ENG-1", []string{"bug"}); err != nil {
			t.Fatalf("RemoveLabels returned error: %v", err)
		}
	})
}

func TestProviderComments(t *testing.T) {
	t.Run("AddComment maps response", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{
				status: http.StatusOK,
				body:   `{"data":{"issue":{"id":"issue-1","identifier":"ENG-1","title":"x","description":"","state":{"id":"s1","name":"Todo","type":"backlog"},"priority":3,"labels":{"nodes":[]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","url":"https://linear.app"}}}`,
			},
			{
				status: http.StatusOK,
				body:   `{"data":{"commentCreate":{"success":true,"comment":{"id":"c1","body":"hello","user":{"id":"u1","name":"Ada"},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}}}`,
			},
		})
		defer server.Close()

		p := &Provider{client: newTestClient(server.URL)}
		comment, err := p.AddComment(context.Background(), "ENG-1", "hello")
		if err != nil {
			t.Fatalf("AddComment returned error: %v", err)
		}
		if comment.ID != "c1" || comment.Author.Name != "Ada" {
			t.Fatalf("unexpected comment mapping: %#v", comment)
		}
	})

	t.Run("FetchComments maps list", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{
				status: http.StatusOK,
				body:   `{"data":{"issue":{"id":"issue-1","identifier":"ENG-1","title":"x","description":"","state":{"id":"s1","name":"Todo","type":"backlog"},"priority":3,"labels":{"nodes":[]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","url":"https://linear.app"}}}`,
			},
			{
				status: http.StatusOK,
				body:   `{"data":{"issue":{"comments":{"nodes":[{"id":"c1","body":"first","user":{"id":"u1","name":"Ada"},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}]}}}}`,
			},
		})
		defer server.Close()

		p := &Provider{client: newTestClient(server.URL)}
		comments, err := p.FetchComments(context.Background(), "ENG-1")
		if err != nil {
			t.Fatalf("FetchComments returned error: %v", err)
		}
		if len(comments) != 1 || comments[0].Body != "first" {
			t.Fatalf("unexpected comments: %#v", comments)
		}
	})
}

func TestGetLabelIDsPlaceholder(t *testing.T) {
	ids, err := GetLabelIDs(context.Background(), nil, "ENG", []string{"bug"})
	if err == nil || !strings.Contains(err.Error(), "not fully implemented") {
		t.Fatalf("expected placeholder error, got ids=%v err=%v", ids, err)
	}
	if len(ids) != 1 || ids[0] != "bug" {
		t.Fatalf("unexpected ids: %v", ids)
	}
}
