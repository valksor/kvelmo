package wrike

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider/token"
)

// ──────────────────────────────────────────────────────────────────────────────
// NewClient tests
// ──────────────────────────────────────────────────────────────────────────────

func TestNewClient(t *testing.T) {
	c := NewClient("test-token", "")

	if c.httpClient == nil {
		t.Error("httpClient is nil")
	}
	if c.token != "test-token" {
		t.Errorf("token = %q, want %q", c.token, "test-token")
	}
	if c.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, defaultBaseURL)
	}
}

func TestNewClientWithCustomHost(t *testing.T) {
	customHost := "https://custom.wrike.com/api/v4"
	c := NewClient("test-token", customHost)

	if c.baseURL != customHost {
		t.Errorf("baseURL = %q, want %q", c.baseURL, customHost)
	}
}

func TestNewClientWithTrailingSlashHost(t *testing.T) {
	customHost := "https://custom.wrike.com/api/v4/"
	c := NewClient("test-token", customHost)

	// Trailing slash should be removed
	if c.baseURL != "https://custom.wrike.com/api/v4" {
		t.Errorf("baseURL = %q, want %q (trailing slash removed)", c.baseURL, "https://custom.wrike.com/api/v4")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// ResolveToken tests
// ──────────────────────────────────────────────────────────────────────────────

func TestResolveToken(t *testing.T) {
	t.Run("config token is used", func(t *testing.T) {
		token, err := ResolveToken("config-token")
		if err != nil {
			t.Fatalf("ResolveToken error = %v", err)
		}
		if token != "config-token" {
			t.Errorf("token = %q, want %q", token, "config-token")
		}
	})

	t.Run("empty config token returns ErrNoToken", func(t *testing.T) {
		_, err := ResolveToken("")
		if !errors.Is(err, token.ErrNoToken) {
			t.Errorf("error = %v, want %v", err, token.ErrNoToken)
		}
	})

	t.Run("${VAR} syntax is passed through as-is (expansion happens at config layer)", func(t *testing.T) {
		// Note: ${VAR} expansion happens at the config loading layer, not in ResolveToken
		token, err := ResolveToken("${TEST_WRIKE_TOKEN}")
		if err != nil {
			t.Fatalf("ResolveToken error = %v", err)
		}
		/* #nosec G101 -- Test placeholder, not a real credential */
		const expectedToken = "${TEST_WRIKE_TOKEN}"
		if token != expectedToken {
			t.Errorf("token = %q, want %q (${VAR} is passed through)", token, expectedToken)
		}
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// HTTP Mock tests for API calls
// ──────────────────────────────────────────────────────────────────────────────

// setupMockServer creates a test server with a custom handler.
func setupMockServer(t *testing.T, handler http.HandlerFunc) (*Client, func()) {
	t.Helper()

	server := httptest.NewServer(handler)

	c := &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    server.URL,
		token:      "test-token",
	}

	return c, func() { server.Close() }
}

func TestGetTask(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check request
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/tasks/IEAAJTASKID" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Errorf("missing or incorrect Authorization header")
			}

			// Send response
			response := taskResponse{
				Data: []Task{
					{
						ID:          "IEAAJTASKID",
						Title:       "Test Task",
						Description: "Test description",
						Status:      "Active",
						Priority:    "High",
						Permalink:   "https://www.wrike.com/open.htm?id=1234567890",
						CreatedDate: time.Now(),
						UpdatedDate: time.Now(),
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		task, err := client.GetTask(context.Background(), "IEAAJTASKID")
		if err != nil {
			t.Fatalf("GetTask error = %v", err)
		}
		if task.ID != "IEAAJTASKID" {
			t.Errorf("task.ID = %q, want %q", task.ID, "IEAAJTASKID")
		}
		if task.Title != "Test Task" {
			t.Errorf("task.Title = %q, want %q", task.Title, "Test Task")
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Not Found"})
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		_, err := client.GetTask(context.Background(), "INVALID")
		if err == nil {
			t.Error("expected error for not found task")
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		_, err := client.GetTask(context.Background(), "IEAAJTASKID")
		if err == nil {
			t.Error("expected error for unauthorized")
		}
	})

	t.Run("empty response data", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := taskResponse{
				Data: []Task{},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		_, err := client.GetTask(context.Background(), "IEAAJTASKID")
		if !errors.Is(err, ErrTaskNotFound) {
			t.Errorf("error = %v, want %v", err, ErrTaskNotFound)
		}
	})
}

func TestGetTaskByPermalink(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check that the endpoint is the standard task endpoint with extracted numeric ID
			if r.URL.Path != "/tasks/1234567890" {
				t.Errorf("unexpected path: %s, want /tasks/1234567890", r.URL.Path)
			}

			response := taskResponse{
				Data: []Task{
					{
						ID:          "IEAAJTASKID",
						Title:       "Task by Permalink",
						Description: "Description",
						Status:      "Active",
						Priority:    "Normal",
						Permalink:   "https://www.wrike.com/open.htm?id=1234567890",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		task, err := client.GetTaskByPermalink(context.Background(), "https://www.wrike.com/open.htm?id=1234567890")
		if err != nil {
			t.Fatalf("GetTaskByPermalink error = %v", err)
		}
		if task.Title != "Task by Permalink" {
			t.Errorf("task.Title = %q, want %q", task.Title, "Task by Permalink")
		}
	})

	t.Run("invalid permalink format", func(t *testing.T) {
		client := &Client{token: "test"}
		_, err := client.GetTaskByPermalink(context.Background(), "not-a-permalink")
		if err == nil {
			t.Error("expected error for invalid permalink format")
		}
	})
}

func TestGetComments(t *testing.T) {
	t.Run("success with comments", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := commentsResponse{
				Data: []Comment{
					{
						ID:          "IEAAJCOMMENT1",
						Text:        "First comment",
						AuthorID:    "USER1",
						AuthorName:  "Author One",
						CreatedDate: time.Now(),
						UpdatedDate: time.Now(),
					},
					{
						ID:          "IEAAJCOMMENT2",
						Text:        "Second comment",
						AuthorID:    "USER2",
						AuthorName:  "Author Two",
						CreatedDate: time.Now(),
						UpdatedDate: time.Now(),
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		comments, err := client.GetComments(context.Background(), "IEAAJTASKID")
		if err != nil {
			t.Fatalf("GetComments error = %v", err)
		}
		if len(comments) != 2 {
			t.Errorf("len(comments) = %d, want 2", len(comments))
		}
	})

	t.Run("empty comments", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := commentsResponse{
				Data: []Comment{},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		comments, err := client.GetComments(context.Background(), "IEAAJTASKID")
		if err != nil {
			t.Fatalf("GetComments error = %v", err)
		}
		if len(comments) != 0 {
			t.Errorf("len(comments) = %d, want 0", len(comments))
		}
	})

	t.Run("API error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		_, err := client.GetComments(context.Background(), "IEAAJTASKID")
		if err == nil {
			t.Error("expected error for API failure")
		}
	})
}

func TestGetAttachments(t *testing.T) {
	t.Run("success with attachments", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := attachmentsResponse{
				Data: []Attachment{
					{
						ID:          "IEAAJATTACH1",
						Name:        "document.pdf",
						CreatedDate: time.Now(),
						Size:        1024,
					},
					{
						ID:          "IEAAJATTACH2",
						Name:        "image.png",
						CreatedDate: time.Now(),
						Size:        2048,
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		attachments, err := client.GetAttachments(context.Background(), "IEAAJTASKID")
		if err != nil {
			t.Fatalf("GetAttachments error = %v", err)
		}
		if len(attachments) != 2 {
			t.Errorf("len(attachments) = %d, want 2", len(attachments))
		}
	})

	t.Run("empty attachments", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := attachmentsResponse{
				Data: []Attachment{},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		attachments, err := client.GetAttachments(context.Background(), "IEAAJTASKID")
		if err != nil {
			t.Fatalf("GetAttachments error = %v", err)
		}
		if len(attachments) != 0 {
			t.Errorf("len(attachments) = %d, want 0", len(attachments))
		}
	})
}

func TestDownloadAttachment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check path
			if r.URL.Path != "/attachments/IEAAJATTACH1/download" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			// Set Content-Disposition header
			w.Header().Set("Content-Disposition", "attachment; filename=\"test.pdf\"")
			_, _ = w.Write([]byte("test file content"))
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		rc, disposition, err := client.DownloadAttachment(context.Background(), "IEAAJATTACH1")
		if err != nil {
			t.Fatalf("DownloadAttachment error = %v", err)
		}
		defer func() { _ = rc.Close() }()

		if disposition == "" {
			t.Error("Content-Disposition header is empty")
		}

		// Read content to verify
		content := make([]byte, 100)
		n, _ := rc.Read(content)
		if n == 0 {
			t.Error("received empty content")
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		rc, _, err := client.DownloadAttachment(context.Background(), "INVALID")
		if err == nil {
			_ = rc.Close()
			t.Error("expected error for not found attachment")
		}
	})
}

func TestPostComment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}

			// Parse request body
			var req map[string]string
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}
			if req["text"] != "Test comment" {
				t.Errorf("request text = %q, want %q", req["text"], "Test comment")
			}

			response := commentResponse{
				Data: []Comment{
					{
						ID:          "IEAAJNEWCOMMENT",
						Text:        "Test comment",
						AuthorID:    "USER1",
						AuthorName:  "Test User",
						CreatedDate: time.Now(),
						UpdatedDate: time.Now(),
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		comment, err := client.PostComment(context.Background(), "IEAAJTASKID", "Test comment")
		if err != nil {
			t.Fatalf("PostComment error = %v", err)
		}
		if comment.ID != "IEAAJNEWCOMMENT" {
			t.Errorf("comment.ID = %q, want %q", comment.ID, "IEAAJNEWCOMMENT")
		}
		if comment.Text != "Test comment" {
			t.Errorf("comment.Text = %q, want %q", comment.Text, "Test comment")
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		_, err := client.PostComment(context.Background(), "IEAAJTASKID", "Test")
		if err == nil {
			t.Error("expected error for unauthorized")
		}
	})

	t.Run("empty response data", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := commentResponse{
				Data: []Comment{},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		_, err := client.PostComment(context.Background(), "IEAAJTASKID", "Test")
		if err == nil {
			t.Error("expected error for empty response")
		}
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// doRequestWithRetry tests
// ──────────────────────────────────────────────────────────────────────────────

func TestDoRequestWithRetry(t *testing.T) {
	t.Run("retry on rate limit with eventual success", func(t *testing.T) {
		attempts := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 2 {
				// First attempt: rate limited
				w.WriteHeader(http.StatusTooManyRequests)

				return
			}
			// Second attempt: success
			response := taskResponse{
				Data: []Task{
					{
						ID:          "IEAAJTASKID",
						Title:       "Test Task",
						Description: "Description",
						Status:      "Active",
						Priority:    "Normal",
						Permalink:   "https://www.wrike.com/open.htm?id=1234567890",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		// Use a short timeout for testing
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var response taskResponse
		err := client.doRequestWithRetry(ctx, http.MethodGet, "/tasks/IEAAJTASKID", nil, &response)
		if err != nil {
			t.Fatalf("doRequestWithRetry error = %v", err)
		}

		if attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", attempts)
		}

		if len(response.Data) != 1 {
			t.Errorf("expected 1 task, got %d", len(response.Data))
		}
	})

	t.Run("non-retryable error fails immediately", func(t *testing.T) {
		attempts := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusUnauthorized)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var response taskResponse
		err := client.doRequestWithRetry(ctx, http.MethodGet, "/tasks/IEAAJTASKID", nil, &response)
		if err == nil {
			t.Error("expected error for unauthorized")
		}

		if attempts != 1 {
			t.Errorf("expected 1 attempt for non-retryable error, got %d", attempts)
		}
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var response taskResponse
		err := client.doRequestWithRetry(ctx, http.MethodGet, "/tasks/IEAAJTASKID", nil, &response)
		if err == nil {
			t.Error("expected error after max retries")
		}
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// GetTasksInFolder tests
// ──────────────────────────────────────────────────────────────────────────────

func TestGetTasksInFolder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/folders/IEAAJFOLDERID/tasks" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			response := taskResponse{
				Data: []Task{
					{
						ID:          "IEAAJTASK1",
						Title:       "Task 1",
						Description: "Description 1",
						Status:      "Active",
						Priority:    "Normal",
						Permalink:   "https://www.wrike.com/open.htm?id=1234567890",
					},
					{
						ID:          "IEAAJTASK2",
						Title:       "Task 2",
						Description: "Description 2",
						Status:      "Completed",
						Priority:    "High",
						Permalink:   "https://www.wrike.com/open.htm?id=1234567891",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		tasks, err := client.GetTasksInFolder(context.Background(), "IEAAJFOLDERID")
		if err != nil {
			t.Fatalf("GetTasksInFolder error = %v", err)
		}

		if len(tasks) != 2 {
			t.Errorf("got %d tasks, want 2", len(tasks))
		}

		if tasks[0].Title != "Task 1" {
			t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "Task 1")
		}
	})

	t.Run("API error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		_, err := client.GetTasksInFolder(context.Background(), "IEAAJFOLDERID")
		if err == nil {
			t.Error("expected error for not found")
		}
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// GetTasksInSpace tests
// ──────────────────────────────────────────────────────────────────────────────

func TestGetTasksInSpace(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/spaces/IEAAJSPACEID/tasks" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			response := taskResponse{
				Data: []Task{
					{
						ID:          "IEAAJTASK1",
						Title:       "Space Task 1",
						Description: "Description 1",
						Status:      "Active",
						Priority:    "Normal",
						Permalink:   "https://www.wrike.com/open.htm?id=1234567890",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		tasks, err := client.GetTasksInSpace(context.Background(), "IEAAJSPACEID")
		if err != nil {
			t.Fatalf("GetTasksInSpace error = %v", err)
		}

		if len(tasks) != 1 {
			t.Errorf("got %d tasks, want 1", len(tasks))
		}

		if tasks[0].Title != "Space Task 1" {
			t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "Space Task 1")
		}
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// GetComments with pagination tests
// ──────────────────────────────────────────────────────────────────────────────

// ──────────────────────────────────────────────────────────────────────────────
// GetFolderByPermalink tests
// ──────────────────────────────────────────────────────────────────────────────

func TestGetFolderByPermalink(t *testing.T) {
	t.Run("success - folder", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check request
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			// Verify permalink query parameter is URL-encoded
			if !strings.Contains(r.URL.RawQuery, "permalink=") {
				t.Errorf("missing permalink query parameter: %s", r.URL.RawQuery)
			}
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Errorf("missing or incorrect Authorization header")
			}

			// Send folder response (without project)
			response := folderResponse{
				Data: []Folder{
					{
						ID:        "IEAAJFOLDERID",
						Title:     "Test Folder",
						ChildIDs:  []string{"IEAAJCHILD1", "IEAAJCHILD2"},
						Scope:     "WsFolder",
						Permalink: "https://www.wrike.com/open.htm?id=1635167041",
						Project:   nil, // Not a project
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		folder, err := client.GetFolderByPermalink(context.Background(), "1635167041")
		if err != nil {
			t.Fatalf("GetFolderByPermalink error = %v", err)
		}
		if folder.ID != "IEAAJFOLDERID" {
			t.Errorf("folder.ID = %q, want %q", folder.ID, "IEAAJFOLDERID")
		}
		if folder.Title != "Test Folder" {
			t.Errorf("folder.Title = %q, want %q", folder.Title, "Test Folder")
		}
		if folder.Scope != "WsFolder" {
			t.Errorf("folder.Scope = %q, want %q", folder.Scope, "WsFolder")
		}
		if folder.Project != nil {
			t.Error("folder.Project should be nil for a folder")
		}
	})

	t.Run("success - project (folder with project properties)", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Send project response (folder with project field)
			response := folderResponse{
				Data: []Folder{
					{
						ID:        "IEAAJPROJECTID",
						Title:     "Test Project",
						ChildIDs:  []string{},
						Scope:     "WsProject",
						Permalink: "https://www.wrike.com/open.htm?id=4352950154",
						Project: &FolderProject{
							AuthorID:    "IEAAJAUTHOR",
							OwnerIDs:    []string{"IEAAJOWNER1", "IEAAJOWNER2"},
							Status:      "Green",
							CreatedDate: time.Now(),
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		folder, err := client.GetFolderByPermalink(context.Background(), "4352950154")
		if err != nil {
			t.Fatalf("GetFolderByPermalink error = %v", err)
		}
		if folder.ID != "IEAAJPROJECTID" {
			t.Errorf("folder.ID = %q, want %q", folder.ID, "IEAAJPROJECTID")
		}
		if folder.Title != "Test Project" {
			t.Errorf("folder.Title = %q, want %q", folder.Title, "Test Project")
		}
		if folder.Scope != "WsProject" {
			t.Errorf("folder.Scope = %q, want %q", folder.Scope, "WsProject")
		}
		if folder.Project == nil {
			t.Fatal("folder.Project should not be nil for a project")
		}
		if folder.Project.Status != "Green" {
			t.Errorf("folder.Project.Status = %q, want %q", folder.Project.Status, "Green")
		}
		if len(folder.Project.OwnerIDs) != 2 {
			t.Errorf("folder.Project.OwnerIDs length = %d, want 2", len(folder.Project.OwnerIDs))
		}
	})

	t.Run("not found - empty response data", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := folderResponse{
				Data: []Folder{},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		_, err := client.GetFolderByPermalink(context.Background(), "9999999999")
		if !errors.Is(err, ErrFolderNotFound) {
			t.Errorf("error = %v, want %v", err, ErrFolderNotFound)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		_, err := client.GetFolderByPermalink(context.Background(), "1234567890")
		if err == nil {
			t.Error("expected error for unauthorized")
		}
	})

	t.Run("API error - server error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		_, err := client.GetFolderByPermalink(context.Background(), "1234567890")
		if err == nil {
			t.Error("expected error for server error")
		}
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// GetComments with pagination tests
// ──────────────────────────────────────────────────────────────────────────────

func TestGetCommentsPagination(t *testing.T) {
	t.Run("paginated comments", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// First page with nextPage link
				response := commentsResponse{
					Data: []Comment{
						{
							ID:          "IEAAJCOMMENT1",
							Text:        "First comment",
							AuthorID:    "USER1",
							AuthorName:  "Author One",
							CreatedDate: time.Now(),
							UpdatedDate: time.Now(),
						},
					},
					NextPage: "/tasks/IEAAJTASKID/comments?page=2",
				}
				_ = json.NewEncoder(w).Encode(response)
			} else {
				// Second page without nextPage
				response := commentsResponse{
					Data: []Comment{
						{
							ID:          "IEAAJCOMMENT2",
							Text:        "Second comment",
							AuthorID:    "USER2",
							AuthorName:  "Author Two",
							CreatedDate: time.Now(),
							UpdatedDate: time.Now(),
						},
					},
				}
				_ = json.NewEncoder(w).Encode(response)
			}
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		comments, err := client.GetComments(context.Background(), "IEAAJTASKID")
		if err != nil {
			t.Fatalf("GetComments error = %v", err)
		}

		if callCount != 2 {
			t.Errorf("expected 2 API calls for pagination, got %d", callCount)
		}

		if len(comments) != 2 {
			t.Errorf("got %d comments, want 2", len(comments))
		}

		if comments[0].ID != "IEAAJCOMMENT1" {
			t.Errorf("comments[0].ID = %q, want %q", comments[0].ID, "IEAAJCOMMENT1")
		}

		if comments[1].ID != "IEAAJCOMMENT2" {
			t.Errorf("comments[1].ID = %q, want %q", comments[1].ID, "IEAAJCOMMENT2")
		}
	})

	t.Run("single page comments", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := commentsResponse{
				Data: []Comment{
					{
						ID:          "IEAAJCOMMENT1",
						Text:        "Only comment",
						AuthorID:    "USER1",
						AuthorName:  "Author One",
						CreatedDate: time.Now(),
						UpdatedDate: time.Now(),
					},
				},
				// No nextPage - this is the last page
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		comments, err := client.GetComments(context.Background(), "IEAAJTASKID")
		if err != nil {
			t.Fatalf("GetComments error = %v", err)
		}

		if len(comments) != 1 {
			t.Errorf("got %d comments, want 1", len(comments))
		}
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// GetTasks tests (multiple task IDs)
// ──────────────────────────────────────────────────────────────────────────────

func TestGetTasks(t *testing.T) {
	t.Run("success with multiple IDs", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify comma-separated IDs in path are NOT URL-encoded
			// The path should be /tasks/ID1,ID2,ID3 (not /tasks/ID1%2CID2%2CID3)
			expectedPath := "/tasks/IEAAJTASK1,IEAAJTASK2,IEAAJTASK3"
			if r.URL.Path != expectedPath {
				t.Errorf("unexpected path: %q, want %q", r.URL.Path, expectedPath)
			}

			response := taskResponse{
				Data: []Task{
					{ID: "IEAAJTASK1", Title: "Task 1", Status: "Active"},
					{ID: "IEAAJTASK2", Title: "Task 2", Status: "Completed"},
					{ID: "IEAAJTASK3", Title: "Task 3", Status: "Active"},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		tasks, err := client.GetTasks(context.Background(), []string{"IEAAJTASK1", "IEAAJTASK2", "IEAAJTASK3"})
		if err != nil {
			t.Fatalf("GetTasks error = %v", err)
		}
		if len(tasks) != 3 {
			t.Errorf("got %d tasks, want 3", len(tasks))
		}
	})

	t.Run("IDs with special characters", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// IDs like MAAAAAED-KwL have dashes and underscores
			// These should be escaped individually, but comma should remain literal
			if !strings.Contains(r.URL.Path, ",") {
				t.Errorf("path should contain literal comma: %s", r.URL.Path)
			}

			response := taskResponse{
				Data: []Task{
					{ID: "MAAAAAED-KwL", Title: "Subtask 1", Status: "Active"},
					{ID: "MAAAAAED-K_R", Title: "Subtask 2", Status: "Active"},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		tasks, err := client.GetTasks(context.Background(), []string{"MAAAAAED-KwL", "MAAAAAED-K_R"})
		if err != nil {
			t.Fatalf("GetTasks error = %v", err)
		}
		if len(tasks) != 2 {
			t.Errorf("got %d tasks, want 2", len(tasks))
		}
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// GetTaskByPermalinkParam tests (two-step resolution)
// ──────────────────────────────────────────────────────────────────────────────

func TestGetTaskByPermalinkParam(t *testing.T) {
	t.Run("two-step resolution: permalink -> API ID -> full task", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// First call: permalink resolution (minimal data)
				if !strings.Contains(r.URL.RawQuery, "permalink=") {
					t.Errorf("first call should use permalink query: %s", r.URL.RawQuery)
				}

				response := taskResponse{
					Data: []Task{
						{
							ID:        "IEAAJAPIID",
							Title:     "Minimal Task", // Minimal data from permalink endpoint
							Status:    "Active",
							Permalink: "https://www.wrike.com/open.htm?id=1234567890",
							// Note: No SubTaskIDs, Description, etc.
						},
					},
				}
				_ = json.NewEncoder(w).Encode(response)
			} else {
				// Second call: direct task fetch (full data)
				if r.URL.Path != "/tasks/IEAAJAPIID" {
					t.Errorf("second call should fetch by API ID: %s", r.URL.Path)
				}

				response := taskResponse{
					Data: []Task{
						{
							ID:          "IEAAJAPIID",
							Title:       "Full Task",
							Description: "Full description",
							Status:      "Active",
							Permalink:   "https://www.wrike.com/open.htm?id=1234567890",
							SubTaskIDs:  []string{"SUB1", "SUB2", "SUB3"},
							ParentIDs:   []string{"PARENT1"},
							AuthorIDs:   []string{"AUTHOR1"},
						},
					},
				}
				_ = json.NewEncoder(w).Encode(response)
			}
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		task, err := client.GetTaskByPermalinkParam(context.Background(), "https://www.wrike.com/open.htm?id=1234567890")
		if err != nil {
			t.Fatalf("GetTaskByPermalinkParam error = %v", err)
		}

		// Verify two calls were made
		if callCount != 2 {
			t.Errorf("expected 2 API calls (permalink + direct), got %d", callCount)
		}

		// Verify full task data is returned
		if task.Description != "Full description" {
			t.Errorf("task.Description = %q, want %q", task.Description, "Full description")
		}
		if len(task.SubTaskIDs) != 3 {
			t.Errorf("task.SubTaskIDs length = %d, want 3", len(task.SubTaskIDs))
		}
		if len(task.ParentIDs) != 1 {
			t.Errorf("task.ParentIDs length = %d, want 1", len(task.ParentIDs))
		}
	})

	t.Run("permalink not found", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := taskResponse{Data: []Task{}}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		_, err := client.GetTaskByPermalinkParam(context.Background(), "https://www.wrike.com/open.htm?id=9999999999")
		if !errors.Is(err, ErrTaskNotFound) {
			t.Errorf("error = %v, want %v", err, ErrTaskNotFound)
		}
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Task struct with optional fields tests
// ──────────────────────────────────────────────────────────────────────────────

// ──────────────────────────────────────────────────────────────────────────────
// GetCustomFields tests
// ──────────────────────────────────────────────────────────────────────────────

func TestGetCustomFields(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/customfields" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			response := customFieldsResponse{
				Data: []CustomFieldDefinition{
					{ID: "IEAAJCF1", Title: "Days since creation", Type: "Duration"},
					{ID: "IEAAJCF2", Title: "Story Points", Type: "Numeric"},
					{ID: "IEAAJCF3", Title: "Sprint", Type: "DropDown"},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		fields, err := client.GetCustomFields(context.Background())
		if err != nil {
			t.Fatalf("GetCustomFields error = %v", err)
		}
		if len(fields) != 3 {
			t.Errorf("got %d fields, want 3", len(fields))
		}
		if fields[0].Title != "Days since creation" {
			t.Errorf("fields[0].Title = %q, want %q", fields[0].Title, "Days since creation")
		}
	})
}

func TestTaskStructOptionalFields(t *testing.T) {
	t.Run("parse all optional fields", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := taskResponse{
				Data: []Task{
					{
						ID:               "IEAAJTASKID",
						Title:            "Task with all fields",
						Description:      "Full description",
						BriefDescription: "Brief...",
						Status:           "Active",
						Priority:         "High",
						Permalink:        "https://www.wrike.com/open.htm?id=1234567890",
						SubTaskIDs:       []string{"SUB1", "SUB2"},
						ParentIDs:        []string{"PARENT1"},
						SuperParentIDs:   []string{"SUPER1"},
						SuperTaskIDs:     []string{"SUPERTASK1"},
						DependencyIDs:    []string{"DEP1"},
						ResponsibleIDs:   []string{"USER1", "USER2"},
						AuthorIDs:        []string{"AUTHOR1"},
						SharedIDs:        []string{"SHARED1"},
						AttachmentCount:  5,
						HasAttachments:   true,
						Recurrent:        false,
						CustomFields: []CustomField{
							{ID: "CF1", Value: "custom value"},
						},
						Metadata: []MetadataItem{
							{Key: "key1", Value: "value1"},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		})

		client, cleanup := setupMockServer(t, handler)
		defer cleanup()

		task, err := client.GetTask(context.Background(), "IEAAJTASKID")
		if err != nil {
			t.Fatalf("GetTask error = %v", err)
		}

		// Verify all optional fields are parsed
		if task.Description != "Full description" {
			t.Errorf("task.Description = %q, want %q", task.Description, "Full description")
		}
		if task.BriefDescription != "Brief..." {
			t.Errorf("task.BriefDescription = %q, want %q", task.BriefDescription, "Brief...")
		}
		if len(task.SubTaskIDs) != 2 {
			t.Errorf("task.SubTaskIDs length = %d, want 2", len(task.SubTaskIDs))
		}
		if len(task.ParentIDs) != 1 {
			t.Errorf("task.ParentIDs length = %d, want 1", len(task.ParentIDs))
		}
		if len(task.SuperParentIDs) != 1 {
			t.Errorf("task.SuperParentIDs length = %d, want 1", len(task.SuperParentIDs))
		}
		if len(task.SuperTaskIDs) != 1 {
			t.Errorf("task.SuperTaskIDs length = %d, want 1", len(task.SuperTaskIDs))
		}
		if len(task.DependencyIDs) != 1 {
			t.Errorf("task.DependencyIDs length = %d, want 1", len(task.DependencyIDs))
		}
		if len(task.ResponsibleIDs) != 2 {
			t.Errorf("task.ResponsibleIDs length = %d, want 2", len(task.ResponsibleIDs))
		}
		if len(task.AuthorIDs) != 1 {
			t.Errorf("task.AuthorIDs length = %d, want 1", len(task.AuthorIDs))
		}
		if len(task.SharedIDs) != 1 {
			t.Errorf("task.SharedIDs length = %d, want 1", len(task.SharedIDs))
		}
		if task.AttachmentCount != 5 {
			t.Errorf("task.AttachmentCount = %d, want 5", task.AttachmentCount)
		}
		if !task.HasAttachments {
			t.Error("task.HasAttachments should be true")
		}
		if task.Recurrent {
			t.Error("task.Recurrent should be false")
		}
		if len(task.CustomFields) != 1 {
			t.Errorf("task.CustomFields length = %d, want 1", len(task.CustomFields))
		}
		if task.CustomFields[0].Value != "custom value" {
			t.Errorf("task.CustomFields[0].Value = %q, want %q", task.CustomFields[0].Value, "custom value")
		}
		if len(task.Metadata) != 1 {
			t.Errorf("task.Metadata length = %d, want 1", len(task.Metadata))
		}
		if task.Metadata[0].Key != "key1" {
			t.Errorf("task.Metadata[0].Key = %q, want %q", task.Metadata[0].Key, "key1")
		}
	})
}
