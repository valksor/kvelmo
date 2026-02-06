package linear

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	providererrors "github.com/valksor/go-toolkit/errors"
)

type graphqlExchange struct {
	assert func(*testing.T, graphqlRequest, *http.Request)
	body   string
	status int
}

func newGraphQLTestServer(t *testing.T, exchanges []graphqlExchange) *httptest.Server {
	t.Helper()

	index := 0

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Helper()

		if index >= len(exchanges) {
			t.Fatalf("unexpected request #%d", index+1)
		}

		ex := exchanges[index]
		index++

		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		if ex.assert != nil {
			ex.assert(t, req, r)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(ex.status)
		_, _ = w.Write([]byte(ex.body))
	}))
}

func newTestClient(baseURL string) *Client {
	c := NewClient("test-token")
	c.baseURL = baseURL

	return c
}

func TestClientDoGraphQLRequestSuccess(t *testing.T) {
	server := newGraphQLTestServer(t, []graphqlExchange{
		{
			status: http.StatusOK,
			body:   `{"data":{"issue":{"id":"issue-1"}}}`,
			assert: func(t *testing.T, req graphqlRequest, r *http.Request) {
				t.Helper()

				if !strings.Contains(req.Query, "query TestIssue") {
					t.Fatalf("query was not sent, got %q", req.Query)
				}
				if req.Variables["issueId"] != "ENG-1" {
					t.Fatalf("issueId variable = %v, want ENG-1", req.Variables["issueId"])
				}
				if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
					t.Fatalf("authorization header = %q", got)
				}
			},
		},
	})
	defer server.Close()

	client := newTestClient(server.URL)
	var response struct {
		Issue struct {
			ID string `json:"id"`
		} `json:"issue"`
	}

	err := client.doGraphQLRequest(
		context.Background(),
		"query TestIssue($issueId: String!) { issue(id: $issueId) { id } }",
		map[string]any{"issueId": "ENG-1"},
		&response,
	)
	if err != nil {
		t.Fatalf("doGraphQLRequest returned error: %v", err)
	}
	if response.Issue.ID != "issue-1" {
		t.Fatalf("Issue.ID = %q, want issue-1", response.Issue.ID)
	}
}

func TestClientDoGraphQLRequestHTTPErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		want error
		code int
	}{
		{
			name: "unauthorized",
			code: http.StatusUnauthorized,
			want: providererrors.ErrUnauthorized,
		},
		{
			name: "not found",
			code: http.StatusNotFound,
			want: providererrors.ErrNotFound,
		},
		{
			name: "rate limited",
			code: http.StatusTooManyRequests,
			want: providererrors.ErrRateLimited,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newGraphQLTestServer(t, []graphqlExchange{
				{
					status: tt.code,
					body:   `{"error":"failed"}`,
				},
			})
			defer server.Close()

			client := newTestClient(server.URL)
			var out struct{}

			err := client.doGraphQLRequest(context.Background(), "query { viewer { id } }", nil, &out)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, expected wrapped %v", err, tt.want)
			}
		})
	}
}

func TestClientDoGraphQLRequestDecodeFailures(t *testing.T) {
	t.Run("invalid response json", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{status: http.StatusOK, body: `{not-json`},
		})
		defer server.Close()

		client := newTestClient(server.URL)
		var out struct{}
		err := client.doGraphQLRequest(context.Background(), "query { viewer { id } }", nil, &out)
		if err == nil || !strings.Contains(err.Error(), "decode response") {
			t.Fatalf("expected decode response error, got %v", err)
		}
	})

	t.Run("graphql errors", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{status: http.StatusOK, body: `{"errors":[{"message":"boom"}]}`},
		})
		defer server.Close()

		client := newTestClient(server.URL)
		var out struct{}
		err := client.doGraphQLRequest(context.Background(), "query { viewer { id } }", nil, &out)
		if err == nil || !strings.Contains(err.Error(), "graphql errors: boom") {
			t.Fatalf("expected graphql errors message, got %v", err)
		}
	})

	t.Run("invalid data shape", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{status: http.StatusOK, body: `{"data":"not-an-object"}`},
		})
		defer server.Close()

		client := newTestClient(server.URL)
		var out struct {
			Issue struct {
				ID string `json:"id"`
			} `json:"issue"`
		}
		err := client.doGraphQLRequest(context.Background(), "query { viewer { id } }", nil, &out)
		if err == nil || !strings.Contains(err.Error(), "decode data") {
			t.Fatalf("expected decode data error, got %v", err)
		}
	})
}

func TestClientGetIssueNotFound(t *testing.T) {
	server := newGraphQLTestServer(t, []graphqlExchange{
		{
			status: http.StatusOK,
			body:   `{"data":{"issue":null}}`,
		},
	})
	defer server.Close()

	client := newTestClient(server.URL)
	issue, err := client.GetIssue(context.Background(), "ENG-404")
	if !errors.Is(err, providererrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got issue=%v err=%v", issue, err)
	}
}

func TestClientListIssuesReturnsPartialResultsOnPagingError(t *testing.T) {
	server := newGraphQLTestServer(t, []graphqlExchange{
		{
			status: http.StatusOK,
			body: `{
				"data":{
					"team":{
						"issues":{
							"nodes":[{"id":"i1","identifier":"ENG-1","title":"One"}],
							"pageInfo":{"hasNextPage":true,"endCursor":"cursor-1"}
						}
					}
				}
			}`,
			assert: func(t *testing.T, req graphqlRequest, _ *http.Request) {
				t.Helper()
				if req.Variables["after"] != "" {
					t.Fatalf("first page after=%v, want empty", req.Variables["after"])
				}
			},
		},
		{
			status: http.StatusInternalServerError,
			body:   `{"error":"temporary failure"}`,
			assert: func(t *testing.T, req graphqlRequest, _ *http.Request) {
				t.Helper()
				if req.Variables["after"] != "cursor-1" {
					t.Fatalf("second page after=%v, want cursor-1", req.Variables["after"])
				}
			},
		},
	})
	defer server.Close()

	client := newTestClient(server.URL)
	issues, err := client.ListIssues(context.Background(), "ENG", ListFilters{})
	if err != nil {
		t.Fatalf("ListIssues returned unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("ListIssues len = %d, want 1", len(issues))
	}
	if issues[0].Identifier != "ENG-1" {
		t.Fatalf("Identifier = %q, want ENG-1", issues[0].Identifier)
	}
}

func TestClientMutationFailurePaths(t *testing.T) {
	t.Run("CreateIssue unsuccessful mutation", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{
				status: http.StatusOK,
				body:   `{"data":{"issueCreate":{"success":false,"issue":null}}}`,
			},
		})
		defer server.Close()

		client := newTestClient(server.URL)
		issue, err := client.CreateIssue(context.Background(), CreateIssueInput{
			Title:  "title",
			TeamID: "team-id",
		})
		if err == nil || !strings.Contains(err.Error(), "failed to create issue") {
			t.Fatalf("expected create failure, got issue=%v err=%v", issue, err)
		}
	})

	t.Run("UpdateIssue unsuccessful mutation", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{
				status: http.StatusOK,
				body:   `{"data":{"issueUpdate":{"success":false,"issue":null}}}`,
			},
		})
		defer server.Close()

		client := newTestClient(server.URL)
		issue, err := client.UpdateIssue(context.Background(), "issue-id", UpdateIssueInput{})
		if err == nil || !strings.Contains(err.Error(), "failed to update issue") {
			t.Fatalf("expected update failure, got issue=%v err=%v", issue, err)
		}
	})

	t.Run("AddComment unsuccessful mutation", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{
				status: http.StatusOK,
				body:   `{"data":{"commentCreate":{"success":false,"comment":null}}}`,
			},
		})
		defer server.Close()

		client := newTestClient(server.URL)
		comment, err := client.AddComment(context.Background(), "issue-id", "test body")
		if err == nil || !strings.Contains(err.Error(), "failed to create comment") {
			t.Fatalf("expected comment failure, got comment=%v err=%v", comment, err)
		}
	})

	t.Run("UpdateIssueDescription unsuccessful mutation", func(t *testing.T) {
		server := newGraphQLTestServer(t, []graphqlExchange{
			{
				status: http.StatusOK,
				body:   `{"data":{"issueUpdate":{"success":false}}}`,
			},
		})
		defer server.Close()

		client := newTestClient(server.URL)
		err := client.UpdateIssueDescription(context.Background(), "issue-id", "new description")
		if err == nil || !strings.Contains(err.Error(), "failed to update issue description") {
			t.Fatalf("expected description failure, got %v", err)
		}
	})
}

func TestHTTPErrorMethods(t *testing.T) {
	err := &httpError{code: http.StatusForbidden, message: "forbidden"}
	if err.HTTPStatusCode() != http.StatusForbidden {
		t.Fatalf("HTTPStatusCode() = %d, want %d", err.HTTPStatusCode(), http.StatusForbidden)
	}
	if got := err.Error(); got != "HTTP 403: forbidden" {
		t.Fatalf("Error() = %q, want %q", got, "HTTP 403: forbidden")
	}
}
