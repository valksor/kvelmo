package bitbucket

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-toolkit/pullrequest"
	"github.com/valksor/go-toolkit/workunit"
)

// ──────────────────────────────────────────────────────────────────────────────
// Provider Interface Compliance
// ──────────────────────────────────────────────────────────────────────────────

// Compile-time interface checks for PR functionality.
var (
	_ pullrequest.PRFetcher        = (*Provider)(nil)
	_ pullrequest.PRCommenter      = (*Provider)(nil)
	_ pullrequest.PRCommentFetcher = (*Provider)(nil)
	_ pullrequest.PRCommentUpdater = (*Provider)(nil)
)

// ──────────────────────────────────────────────────────────────────────────────
// FetchPullRequest Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderFetchPullRequest(t *testing.T) {
	tests := []struct {
		name        string
		workspace   string
		repoSlug    string
		prNumber    int
		wantErr     bool
		errContains string
		validate    func(*testing.T, *pullrequest.PullRequest)
	}{
		{
			name:      "success: fetches PR details",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			prNumber:  42,
			wantErr:   false,
			validate: func(t *testing.T, pr *pullrequest.PullRequest) {
				t.Helper()

				if pr.Number != 42 {
					t.Errorf("Number = %d, want %d", pr.Number, 42)
				}
				if pr.Title != "Test PR" {
					t.Errorf("Title = %q, want %q", pr.Title, "Test PR")
				}
				if pr.State != "OPEN" {
					t.Errorf("State = %q, want %q", pr.State, "OPEN")
				}
				if pr.HeadBranch != "feature/test" {
					t.Errorf("HeadBranch = %q, want %q", pr.HeadBranch, "feature/test")
				}
				if pr.BaseBranch != "main" {
					t.Errorf("BaseBranch = %q, want %q", pr.BaseBranch, "main")
				}
			},
		},
		{
			name:        "error: workspace not configured",
			workspace:   "",
			repoSlug:    "myrepo",
			prNumber:    42,
			wantErr:     true,
			errContains: "not configured",
		},
		{
			name:        "error: repo not configured",
			workspace:   "myworkspace",
			repoSlug:    "",
			prNumber:    42,
			wantErr:     true,
			errContains: "not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				// Verify path contains pull requests
				if !strings.Contains(r.URL.Path, "/pullrequests/") {
					t.Errorf("expected path to contain /pullrequests/, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				response := PullRequest{
					ID:          42,
					Title:       "Test PR",
					Description: "This is a test PR",
					State:       "OPEN",
					Source: PRBranch{
						Branch: Branch{Name: "feature/test"},
					},
					Destination: PRBranch{
						Branch: Branch{Name: "main"},
					},
					Author: &User{
						Username:    "testuser",
						DisplayName: "Test User",
					},
					Links: Links{
						HTML: &Link{Href: "https://bitbucket.org/myworkspace/myrepo/pull-requests/42"},
					},
					CreatedOn: time.Now(),
					UpdatedOn: time.Now(),
				}
				respBytes, _ := json.Marshal(response)
				_, _ = w.Write(respBytes)
			}))
			defer server.Close()

			client := NewClient("testuser", "testpass", tt.workspace, tt.repoSlug)
			client.baseURL = server.URL

			p := &Provider{
				client: client,
				config: &Config{
					Workspace: tt.workspace,
					RepoSlug:  tt.repoSlug,
				},
			}

			pr, err := p.FetchPullRequest(context.Background(), tt.prNumber)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)

					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if pr == nil {
				t.Fatal("expected non-nil PullRequest on success")
			}

			if tt.validate != nil {
				tt.validate(t, pr)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// FetchPullRequestDiff Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderFetchPullRequestDiff(t *testing.T) {
	tests := []struct {
		name        string
		workspace   string
		repoSlug    string
		prNumber    int
		wantErr     bool
		errContains string
		validate    func(*testing.T, *pullrequest.PullRequestDiff)
	}{
		{
			name:      "success: fetches PR diff",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			prNumber:  42,
			wantErr:   false,
			validate: func(t *testing.T, diff *pullrequest.PullRequestDiff) {
				t.Helper()

				if diff.BaseBranch != "main" {
					t.Errorf("BaseBranch = %q, want %q", diff.BaseBranch, "main")
				}
				if diff.HeadBranch != "feature/test" {
					t.Errorf("HeadBranch = %q, want %q", diff.HeadBranch, "feature/test")
				}
			},
		},
		{
			name:        "error: workspace not configured",
			workspace:   "",
			repoSlug:    "myrepo",
			prNumber:    42,
			wantErr:     true,
			errContains: "not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				// First request gets PR details, second gets diff
				if strings.Contains(r.URL.Path, "/diff") {
					// Return raw diff
					_, _ = w.Write([]byte("diff --git a/file.txt b/file.txt\n+new line"))

					return
				}

				// Return PR details
				response := PullRequest{
					ID:    42,
					Title: "Test PR",
					State: "OPEN",
					Source: PRBranch{
						Branch: Branch{Name: "feature/test"},
					},
					Destination: PRBranch{
						Branch: Branch{Name: "main"},
					},
					Links: Links{
						HTML: &Link{Href: "https://bitbucket.org/myworkspace/myrepo/pull-requests/42"},
					},
					CreatedOn: time.Now(),
					UpdatedOn: time.Now(),
				}
				respBytes, _ := json.Marshal(response)
				_, _ = w.Write(respBytes)
			}))
			defer server.Close()

			client := NewClient("testuser", "testpass", tt.workspace, tt.repoSlug)
			client.baseURL = server.URL

			p := &Provider{
				client: client,
				config: &Config{
					Workspace: tt.workspace,
					RepoSlug:  tt.repoSlug,
				},
			}

			diff, err := p.FetchPullRequestDiff(context.Background(), tt.prNumber)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)

					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if diff == nil {
				t.Fatal("expected non-nil PullRequestDiff on success")
			}

			if tt.validate != nil {
				tt.validate(t, diff)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// AddPullRequestComment Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderAddPullRequestComment(t *testing.T) {
	tests := []struct {
		name        string
		workspace   string
		repoSlug    string
		prNumber    int
		body        string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *workunit.Comment)
	}{
		{
			name:      "success: adds PR comment",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			prNumber:  42,
			body:      "This is a test comment",
			wantErr:   false,
			validate: func(t *testing.T, comment *workunit.Comment) {
				t.Helper()

				if comment.Body != "This is a test comment" {
					t.Errorf("Body = %q, want %q", comment.Body, "This is a test comment")
				}
				if comment.ID == "" {
					t.Error("ID should not be empty")
				}
			},
		},
		{
			name:        "error: workspace not configured",
			workspace:   "",
			repoSlug:    "myrepo",
			prNumber:    42,
			body:        "Test",
			wantErr:     true,
			errContains: "not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				// Read and verify body
				body, _ := io.ReadAll(r.Body)
				if !strings.Contains(string(body), tt.body) {
					t.Errorf("request body should contain %q", tt.body)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)

				response := Comment{
					ID: 123,
					Content: &Content{
						Raw: tt.body,
					},
					User: &User{
						Username:    "testuser",
						DisplayName: "Test User",
					},
					CreatedOn: time.Now(),
					UpdatedOn: time.Now(),
				}
				respBytes, _ := json.Marshal(response)
				_, _ = w.Write(respBytes)
			}))
			defer server.Close()

			client := NewClient("testuser", "testpass", tt.workspace, tt.repoSlug)
			client.baseURL = server.URL

			p := &Provider{
				client: client,
				config: &Config{
					Workspace: tt.workspace,
					RepoSlug:  tt.repoSlug,
				},
			}

			comment, err := p.AddPullRequestComment(context.Background(), tt.prNumber, tt.body)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)

					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if comment == nil {
				t.Fatal("expected non-nil Comment on success")
			}

			if tt.validate != nil {
				tt.validate(t, comment)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// FetchPullRequestComments Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderFetchPullRequestComments(t *testing.T) {
	tests := []struct {
		name        string
		workspace   string
		repoSlug    string
		prNumber    int
		wantErr     bool
		errContains string
		wantCount   int
	}{
		{
			name:      "success: fetches PR comments",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			prNumber:  42,
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:        "error: workspace not configured",
			workspace:   "",
			repoSlug:    "myrepo",
			prNumber:    42,
			wantErr:     true,
			errContains: "not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				response := struct {
					Values []Comment `json:"values"`
					Next   string    `json:"next,omitempty"`
				}{
					Values: []Comment{
						{
							ID:        1,
							Content:   &Content{Raw: "First comment"},
							User:      &User{Username: "user1"},
							CreatedOn: time.Now(),
						},
						{
							ID:        2,
							Content:   &Content{Raw: "Second comment"},
							User:      &User{Username: "user2"},
							CreatedOn: time.Now(),
						},
					},
				}
				respBytes, _ := json.Marshal(response)
				_, _ = w.Write(respBytes)
			}))
			defer server.Close()

			client := NewClient("testuser", "testpass", tt.workspace, tt.repoSlug)
			client.baseURL = server.URL

			p := &Provider{
				client: client,
				config: &Config{
					Workspace: tt.workspace,
					RepoSlug:  tt.repoSlug,
				},
			}

			comments, err := p.FetchPullRequestComments(context.Background(), tt.prNumber)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)

					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(comments) != tt.wantCount {
				t.Errorf("got %d comments, want %d", len(comments), tt.wantCount)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// UpdatePullRequestComment Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderUpdatePullRequestComment(t *testing.T) {
	tests := []struct {
		name        string
		workspace   string
		repoSlug    string
		prNumber    int
		commentID   string
		newBody     string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *workunit.Comment)
	}{
		{
			name:      "success: updates PR comment",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			prNumber:  42,
			commentID: "123",
			newBody:   "Updated comment body",
			wantErr:   false,
			validate: func(t *testing.T, comment *workunit.Comment) {
				t.Helper()

				if comment.Body != "Updated comment body" {
					t.Errorf("Body = %q, want %q", comment.Body, "Updated comment body")
				}
			},
		},
		{
			name:        "error: workspace not configured",
			workspace:   "",
			repoSlug:    "myrepo",
			prNumber:    42,
			commentID:   "123",
			newBody:     "Test",
			wantErr:     true,
			errContains: "not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Errorf("expected PUT request, got %s", r.Method)
				}

				// Verify path contains comment ID
				if !strings.Contains(r.URL.Path, tt.commentID) {
					t.Errorf("expected path to contain comment ID %s", tt.commentID)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				response := Comment{
					ID: 123,
					Content: &Content{
						Raw: tt.newBody,
					},
					User: &User{
						Username: "testuser",
					},
					CreatedOn: time.Now(),
					UpdatedOn: time.Now(),
				}
				respBytes, _ := json.Marshal(response)
				_, _ = w.Write(respBytes)
			}))
			defer server.Close()

			client := NewClient("testuser", "testpass", tt.workspace, tt.repoSlug)
			client.baseURL = server.URL

			p := &Provider{
				client: client,
				config: &Config{
					Workspace: tt.workspace,
					RepoSlug:  tt.repoSlug,
				},
			}

			comment, err := p.UpdatePullRequestComment(context.Background(), tt.prNumber, tt.commentID, tt.newBody)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)

					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if comment == nil {
				t.Fatal("expected non-nil Comment on success")
			}

			if tt.validate != nil {
				tt.validate(t, comment)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Benchmark Tests
// ──────────────────────────────────────────────────────────────────────────────

func BenchmarkMapPRToProvider(b *testing.B) {
	pr := &PullRequest{
		ID:          42,
		Title:       "Test PR",
		Description: "This is a test PR description",
		State:       "OPEN",
		Source: PRBranch{
			Branch: Branch{Name: "feature/test"},
		},
		Destination: PRBranch{
			Branch: Branch{Name: "main"},
		},
		Author: &User{
			Username:    "testuser",
			DisplayName: "Test User",
		},
		Links: Links{
			HTML: &Link{Href: "https://bitbucket.org/workspace/repo/pull-requests/42"},
		},
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}

	b.ResetTimer()
	for range b.N {
		// Simulate mapping logic
		webURL := ""
		if pr.Links.HTML != nil {
			webURL = pr.Links.HTML.Href
		}
		author := ""
		if pr.Author != nil {
			author = pr.Author.Username
		}
		_ = &pullrequest.PullRequest{
			ID:         "42",
			URL:        webURL,
			Title:      pr.Title,
			State:      pr.State,
			Number:     pr.ID,
			Body:       pr.Description,
			HeadBranch: pr.Source.Branch.Name,
			BaseBranch: pr.Destination.Branch.Name,
			Author:     author,
			CreatedAt:  pr.CreatedOn,
			UpdatedAt:  pr.UpdatedOn,
		}
	}
}
