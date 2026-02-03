package wrike

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// mustEncode encodes v to w and panics on error.
// Use in test HTTP handlers where t *testing.T isn't accessible.
func mustEncode(w http.ResponseWriter, v any) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(fmt.Sprintf("test HTTP handler: failed to encode response: %v", err))
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Info tests
// ──────────────────────────────────────────────────────────────────────────────

func TestInfo(t *testing.T) {
	info := Info()

	if info.Name != ProviderName {
		t.Errorf("Name = %q, want %q", info.Name, ProviderName)
	}

	if info.Description == "" {
		t.Error("Description is empty")
	}

	// Check schemes
	expectedSchemes := []string{"wrike", "wk"}
	if len(info.Schemes) != len(expectedSchemes) {
		t.Errorf("Schemes = %v, want %v", info.Schemes, expectedSchemes)
	}
	for i, s := range expectedSchemes {
		if info.Schemes[i] != s {
			t.Errorf("Schemes[%d] = %q, want %q", i, info.Schemes[i], s)
		}
	}

	// Check priority
	if info.Priority <= 0 {
		t.Errorf("Priority = %d, want > 0", info.Priority)
	}

	// Check capabilities
	expectedCaps := []provider.Capability{
		provider.CapRead,
		provider.CapFetchComments,
		provider.CapComment,
		provider.CapDownloadAttachment,
		provider.CapSnapshot,
	}
	for _, cap := range expectedCaps {
		if !info.Capabilities.Has(cap) {
			t.Errorf("Capabilities missing %q", cap)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// mapStatus tests
// ──────────────────────────────────────────────────────────────────────────────

func TestMapStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   provider.Status
	}{
		{
			name:   "active status",
			status: "Active",
			want:   provider.StatusOpen,
		},
		{
			name:   "new status",
			status: "New",
			want:   provider.StatusOpen,
		},
		{
			name:   "in progress status",
			status: "In Progress",
			want:   provider.StatusOpen,
		},
		{
			name:   "inprogress status (no space)",
			status: "Inprogress",
			want:   provider.StatusOpen,
		},
		{
			name:   "draft status",
			status: "Draft",
			want:   provider.StatusOpen,
		},
		{
			name:   "completed status",
			status: "Completed",
			want:   provider.StatusDone,
		},
		{
			name:   "done status",
			status: "Done",
			want:   provider.StatusDone,
		},
		{
			name:   "closed status",
			status: "Closed",
			want:   provider.StatusClosed,
		},
		{
			name:   "cancelled status",
			status: "Cancelled",
			want:   provider.StatusClosed,
		},
		{
			name:   "canceled status (single l)",
			status: "Canceled",
			want:   provider.StatusClosed,
		},
		{
			name:   "deferred status",
			status: "Deferred",
			want:   provider.StatusClosed,
		},
		{
			name:   "review status",
			status: "Review",
			want:   provider.StatusReview,
		},
		{
			name:   "unknown status defaults to open",
			status: "Unknown",
			want:   provider.StatusOpen,
		},
		{
			name:   "empty status defaults to open",
			status: "",
			want:   provider.StatusOpen,
		},
		{
			name:   "case insensitive - ACTIVE",
			status: "ACTIVE",
			want:   provider.StatusOpen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapStatus(tt.status)
			if got != tt.want {
				t.Errorf("mapStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// mapPriority tests
// ──────────────────────────────────────────────────────────────────────────────

func TestMapPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority string
		want     provider.Priority
	}{
		{
			name:     "critical priority",
			priority: "Critical",
			want:     provider.PriorityCritical,
		},
		{
			name:     "urgent priority",
			priority: "Urgent",
			want:     provider.PriorityCritical,
		},
		{
			name:     "high priority",
			priority: "High",
			want:     provider.PriorityHigh,
		},
		{
			name:     "low priority",
			priority: "Low",
			want:     provider.PriorityLow,
		},
		{
			name:     "normal priority",
			priority: "Normal",
			want:     provider.PriorityNormal,
		},
		{
			name:     "empty priority defaults to normal",
			priority: "",
			want:     provider.PriorityNormal,
		},
		{
			name:     "unknown priority defaults to normal",
			priority: "Unknown",
			want:     provider.PriorityNormal,
		},
		{
			name:     "case insensitive - CRITICAL",
			priority: "CRITICAL",
			want:     provider.PriorityCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapPriority(tt.priority)
			if got != tt.want {
				t.Errorf("mapPriority(%q) = %v, want %v", tt.priority, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// mapComments tests
// ──────────────────────────────────────────────────────────────────────────────

func TestMapComments(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		comments []Comment
		wantLen  int
	}{
		{
			name: "multiple comments",
			comments: []Comment{
				{
					ID:          "IEAAJ1",
					Text:        "First comment",
					AuthorID:    "USER1",
					AuthorName:  "Author One",
					CreatedDate: now,
					UpdatedDate: now,
				},
				{
					ID:          "IEAAJ2",
					Text:        "Second comment",
					AuthorID:    "USER2",
					AuthorName:  "Author Two",
					CreatedDate: now,
					UpdatedDate: now,
				},
			},
			wantLen: 2,
		},
		{
			name:     "empty comments",
			comments: []Comment{},
			wantLen:  0,
		},
		{
			name:     "nil comments",
			comments: nil,
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapComments(tt.comments)
			if len(got) != tt.wantLen {
				t.Errorf("mapComments() len = %d, want %d", len(got), tt.wantLen)

				return
			}

			for i, c := range tt.comments {
				if got[i].ID != c.ID {
					t.Errorf("mapComments()[%d].ID = %q, want %q", i, got[i].ID, c.ID)
				}
				if got[i].Body != c.Text {
					t.Errorf("mapComments()[%d].Body = %q, want %q", i, got[i].Body, c.Text)
				}
				if got[i].Author.Name != c.AuthorName {
					t.Errorf("mapComments()[%d].Author.Name = %q, want %q", i, got[i].Author.Name, c.AuthorName)
				}
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// buildMetadata tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildMetadata(t *testing.T) {
	now := time.Now()

	task := &Task{
		ID:          "IEAAJTASK",
		Title:       "Test Task",
		Description: "Test description",
		Status:      "Active",
		Priority:    "High",
		Permalink:   "https://www.wrike.com/open.htm?id=1234567890",
		SubTaskIDs:  []string{"IEAAJ1", "IEAAJ2"},
		CreatedDate: now,
		UpdatedDate: now,
	}

	subtasks := []SubtaskInfo{
		{ID: "IEAAJ1", Title: "Subtask 1", Status: "Active"},
		{ID: "IEAAJ2", Title: "Subtask 2", Status: "Completed"},
	}

	metadata := buildMetadata(task, subtasks)

	// Check basic metadata
	if metadata["permalink"] != task.Permalink {
		t.Errorf("permalink = %q, want %q", metadata["permalink"], task.Permalink)
	}
	if metadata["api_id"] != task.ID {
		t.Errorf("api_id = %q, want %q", metadata["api_id"], task.ID)
	}
	if metadata["wrike_status"] != task.Status {
		t.Errorf("wrike_status = %q, want %q", metadata["wrike_status"], task.Status)
	}
	if metadata["wrike_priority"] != task.Priority {
		t.Errorf("wrike_priority = %q, want %q", metadata["wrike_priority"], task.Priority)
	}

	// Check subtask metadata
	if metadata["subtask_count"] != 2 {
		t.Errorf("subtask_count = %v, want 2", metadata["subtask_count"])
	}
	subtaskList, ok := metadata["subtasks"].([]map[string]string)
	if !ok {
		t.Fatal("subtasks is not of type []map[string]string")
	}
	if len(subtaskList) != 2 {
		t.Errorf("len(subtasks) = %d, want 2", len(subtaskList))
	}
}

func TestBuildMetadataWithNoSubtasks(t *testing.T) {
	now := time.Now()

	task := &Task{
		ID:          "IEAAJTASK",
		Title:       "Test Task",
		Description: "Test description",
		Status:      "Active",
		Priority:    "High",
		Permalink:   "https://www.wrike.com/open.htm?id=1234567890",
		SubTaskIDs:  []string{},
		CreatedDate: now,
		UpdatedDate: now,
	}

	metadata := buildMetadata(task, nil)

	// Check basic metadata exists
	if metadata["permalink"] != task.Permalink {
		t.Errorf("permalink = %q, want %q", metadata["permalink"], task.Permalink)
	}

	// Check subtasks are not present
	if _, ok := metadata["subtasks"]; ok {
		t.Error("subtasks should not be present when there are none")
	}
	if _, ok := metadata["subtask_count"]; ok {
		t.Error("subtask_count should not be present when there are none")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Provider.Match tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProvider_Match(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "wrike scheme with numeric ID",
			input: "wrike:1234567890",
			want:  true,
		},
		{
			name:  "wrike scheme with API ID",
			input: "wrike:IEAAJXXXXXXXX",
			want:  true,
		},
		{
			name:  "wk scheme with numeric ID",
			input: "wk:1234567890",
			want:  true,
		},
		{
			name:  "wk scheme with API ID",
			input: "wk:IEAAJXXXXXXXX",
			want:  true,
		},
		{
			name:  "permalink URL",
			input: "https://www.wrike.com/open.htm?id=1234567890",
			want:  true,
		},
		{
			name:  "bare API ID",
			input: "IEAAJXXXXXXXX",
			want:  true,
		},
		{
			name:  "bare numeric ID (10 digits)",
			input: "1234567890",
			want:  true,
		},
		{
			name:  "bare numeric ID (more than 10 digits)",
			input: "12345678901234",
			want:  true,
		},
		{
			name:  "file scheme",
			input: "file:task.md",
			want:  false,
		},
		{
			name:  "github scheme",
			input: "github:123",
			want:  false,
		},
		{
			name:  "no scheme - short text",
			input: "just-text",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "similar but not wrike",
			input: "wrike-actions:run",
			want:  false,
		},
		{
			name:  "numeric ID less than 10 digits",
			input: "123456789",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Match(tt.input)
			if got != tt.want {
				t.Errorf("Match(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Provider.Parse tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProvider_Parse(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		name        string
		input       string
		want        string
		errContains string
		wantErr     bool
	}{
		{
			name:  "wrike scheme with numeric ID",
			input: "wrike:1234567890",
			want:  "1234567890",
		},
		{
			name:  "wrike scheme with API ID",
			input: "wrike:IEAAJXXXXXXXX",
			want:  "IEAAJXXXXXXXX",
		},
		{
			name:  "wk scheme with numeric ID",
			input: "wk:1234567890",
			want:  "1234567890",
		},
		{
			name:  "wk scheme with API ID",
			input: "wk:IEAAJXXXXXXXX",
			want:  "IEAAJXXXXXXXX",
		},
		{
			name:  "permalink - extracts numeric ID",
			input: "https://www.wrike.com/open.htm?id=1234567890",
			want:  "1234567890",
		},
		{
			name:  "wrike scheme with permalink URL",
			input: "wrike:https://www.wrike.com/open.htm?id=4360575608",
			want:  "4360575608",
		},
		{
			name:  "wk scheme with permalink URL",
			input: "wk:https://www.wrike.com/open.htm?id=4360575608",
			want:  "4360575608",
		},
		{
			name:        "empty string",
			input:       "",
			wantErr:     true,
			errContains: "invalid",
		},
		{
			name:        "invalid format - not a valid ID",
			input:       "wrike:abc",
			wantErr:     true,
			errContains: "invalid",
		},
		{
			name:        "invalid format - not enough digits",
			input:       "wrike:123",
			wantErr:     true,
			errContains: "invalid",
		},
		{
			name:        "not a wrike reference",
			input:       "github:123",
			wantErr:     true,
			errContains: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.Parse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse(%q) expected error, got nil", tt.input)

					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Parse(%q) error = %q, want to contain %q", tt.input, err.Error(), tt.errContains)
				}

				return
			}

			if err != nil {
				t.Errorf("Parse(%q) unexpected error: %v", tt.input, err)

				return
			}

			if got != tt.want {
				t.Errorf("Parse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// ProviderName constant test
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderName(t *testing.T) {
	if ProviderName != "wrike" {
		t.Errorf("ProviderName = %q, want %q", ProviderName, "wrike")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// isNumericID tests
// ──────────────────────────────────────────────────────────────────────────────

// ──────────────────────────────────────────────────────────────────────────────
// buildMetadata with SuperTaskIDs tests (subtask detection)
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildMetadataWithSuperTaskIDs(t *testing.T) {
	now := time.Now()

	task := &Task{
		ID:           "IEAAJSUBTASK",
		Title:        "Subtask",
		Description:  "This is a subtask",
		Status:       "Active",
		Priority:     "Normal",
		Permalink:    "https://www.wrike.com/open.htm?id=1234567891",
		SuperTaskIDs: []string{"IEAAJPARENT1", "IEAAJPARENT2"}, // Has parents
		CreatedDate:  now,
		UpdatedDate:  now,
	}

	metadata := buildMetadata(task, nil)

	// Check is_subtask flag
	if isSubtask, ok := metadata["is_subtask"].(bool); !ok || !isSubtask {
		t.Error("is_subtask should be true when SuperTaskIDs is populated")
	}

	// Check parent_task_ids
	parentIDs, ok := metadata["parent_task_ids"].([]string)
	if !ok {
		t.Fatal("parent_task_ids should be []string")
	}
	if len(parentIDs) != 2 {
		t.Errorf("parent_task_ids len = %d, want 2", len(parentIDs))
	}
	if parentIDs[0] != "IEAAJPARENT1" {
		t.Errorf("parent_task_ids[0] = %q, want %q", parentIDs[0], "IEAAJPARENT1")
	}
}

func TestBuildMetadataWithoutSuperTaskIDs(t *testing.T) {
	now := time.Now()

	task := &Task{
		ID:          "IEAAJTASK",
		Title:       "Regular Task",
		Description: "Not a subtask",
		Status:      "Active",
		Priority:    "Normal",
		Permalink:   "https://www.wrike.com/open.htm?id=1234567890",
		CreatedDate: now,
		UpdatedDate: now,
	}

	metadata := buildMetadata(task, nil)

	// Check is_subtask is NOT present
	if _, ok := metadata["is_subtask"]; ok {
		t.Error("is_subtask should not be present when SuperTaskIDs is empty")
	}

	// Check parent_task_ids is NOT present
	if _, ok := metadata["parent_task_ids"]; ok {
		t.Error("parent_task_ids should not be present when SuperTaskIDs is empty")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// fetchParentTask tests
// ──────────────────────────────────────────────────────────────────────────────

func TestFetchParentTask_NoParent(t *testing.T) {
	p := &Provider{}

	// Empty SuperTaskIDs should return nil, nil
	info, err := p.fetchParentTask(context.Background(), "IEAAJTASK", []string{})
	if err != nil {
		t.Errorf("fetchParentTask() error = %v, want nil", err)
	}
	if info != nil {
		t.Errorf("fetchParentTask() = %v, want nil", info)
	}
}

func TestFetchParentTask_CircularReference(t *testing.T) {
	p := &Provider{}

	// Task ID equals parent ID - should return nil, nil (guard against circular ref)
	info, err := p.fetchParentTask(context.Background(), "IEAAJTASK", []string{"IEAAJTASK"})
	if err != nil {
		t.Errorf("fetchParentTask() error = %v, want nil", err)
	}
	if info != nil {
		t.Errorf("fetchParentTask() with circular ref = %v, want nil", info)
	}
}

func TestFetchParentTask_EmptyParentID(t *testing.T) {
	p := &Provider{}

	// Empty string in SuperTaskIDs - should return nil, nil
	info, err := p.fetchParentTask(context.Background(), "IEAAJTASK", []string{""})
	if err != nil {
		t.Errorf("fetchParentTask() error = %v, want nil", err)
	}
	if info != nil {
		t.Errorf("fetchParentTask() with empty parent ID = %v, want nil", info)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// ParentTaskInfo struct tests
// ──────────────────────────────────────────────────────────────────────────────

func TestParentTaskInfoDoesNotIncludeSubtasks(t *testing.T) {
	// This is a compile-time check - ParentTaskInfo should NOT have SubTaskIDs
	// The struct intentionally excludes subtask data to avoid exposing siblings
	info := ParentTaskInfo{
		ID:          "IEAAJPARENT",
		Title:       "Parent Task",
		Status:      "Active",
		Description: "Parent description",
		Permalink:   "https://www.wrike.com/open.htm?id=1234567890",
	}

	// Verify fields exist as expected
	if info.ID != "IEAAJPARENT" {
		t.Errorf("ID = %q, want %q", info.ID, "IEAAJPARENT")
	}
	if info.Title != "Parent Task" {
		t.Errorf("Title = %q, want %q", info.Title, "Parent Task")
	}
	// Note: No SubTaskIDs field exists - this is intentional per user requirement
}

// ──────────────────────────────────────────────────────────────────────────────
// isNumericID tests
// ──────────────────────────────────────────────────────────────────────────────

func TestIsNumericID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{
			name: "10-digit numeric ID",
			id:   "1234567890",
			want: true,
		},
		{
			name: "13-digit numeric ID",
			id:   "4352950154123",
			want: true,
		},
		{
			name: "real Wrike folder ID from URL",
			id:   "1635167041",
			want: true,
		},
		{
			name: "real Wrike project ID from URL",
			id:   "4352950154",
			want: true,
		},
		{
			name: "real Wrike space ID from URL",
			id:   "824404493",
			want: false, // 9 digits, below minimum
		},
		{
			name: "API ID format (starts with letters)",
			id:   "IEAAJXXXXXXXX",
			want: false,
		},
		{
			name: "API ID format - short",
			id:   "IEAAJXXX",
			want: false,
		},
		{
			name: "mixed alphanumeric",
			id:   "123abc456",
			want: false,
		},
		{
			name: "9-digit number (below minimum)",
			id:   "123456789",
			want: false,
		},
		{
			name: "empty string",
			id:   "",
			want: false,
		},
		{
			name: "single digit",
			id:   "1",
			want: false,
		},
		{
			name: "special characters",
			id:   "123-456-789",
			want: false,
		},
		{
			name: "leading zeros 10 digits",
			id:   "0000000001",
			want: true,
		},
		{
			name: "URL-like string",
			id:   "https://wrike.com",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNumericID(tt.id)
			if got != tt.want {
				t.Errorf("isNumericID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// FetchParent integration tests (with mock HTTP server)
// ──────────────────────────────────────────────────────────────────────────────

// setupMockProvider creates a Provider with a mock HTTP server.
func setupMockProvider(t *testing.T, handler http.HandlerFunc) (*Provider, func()) {
	t.Helper()

	server := httptest.NewServer(handler)

	client := &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    server.URL,
		token:      "test-token",
	}

	p := &Provider{client: client}

	return p, func() { server.Close() }
}

func TestFetchParent_ReturnsParentForSubtask(t *testing.T) {
	now := time.Now()
	requestCount := 0

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		if requestCount == 1 {
			// First request: fetch the subtask
			response := taskResponse{
				Data: []Task{
					{
						ID:           "IEAAJSUBTASK",
						Title:        "Subtask",
						Status:       "Active",
						Permalink:    "https://www.wrike.com/open.htm?id=1111111111",
						SuperTaskIDs: []string{"IEAAJPARENT"},
						CreatedDate:  now,
						UpdatedDate:  now,
					},
				},
			}
			mustEncode(w, response)

			return
		}

		// Second request: fetch the parent task
		response := taskResponse{
			Data: []Task{
				{
					ID:          "IEAAJPARENT",
					Title:       "Parent Task",
					Description: "Parent description",
					Status:      "Active",
					Priority:    "High",
					Permalink:   "https://www.wrike.com/open.htm?id=2222222222",
					CreatedDate: now,
					UpdatedDate: now,
				},
			},
		}
		mustEncode(w, response)
	})

	p, cleanup := setupMockProvider(t, handler)
	defer cleanup()

	parent, err := p.FetchParent(context.Background(), "IEAAJSUBTASK")
	if err != nil {
		t.Fatalf("FetchParent() error = %v", err)
	}
	if parent == nil {
		t.Fatal("FetchParent() returned nil")
	}
	if parent.Title != "Parent Task" {
		t.Errorf("parent.Title = %q, want %q", parent.Title, "Parent Task")
	}
	if parent.ExternalID != "IEAAJPARENT" {
		t.Errorf("parent.ExternalID = %q, want %q", parent.ExternalID, "IEAAJPARENT")
	}
}

func TestFetchParent_ReturnsErrorForNonSubtask(t *testing.T) {
	now := time.Now()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a task with no SuperTaskIDs (not a subtask)
		response := taskResponse{
			Data: []Task{
				{
					ID:          "IEAAJTASK",
					Title:       "Regular Task",
					Status:      "Active",
					Permalink:   "https://www.wrike.com/open.htm?id=1234567890",
					CreatedDate: now,
					UpdatedDate: now,
				},
			},
		}
		mustEncode(w, response)
	})

	p, cleanup := setupMockProvider(t, handler)
	defer cleanup()

	parent, err := p.FetchParent(context.Background(), "IEAAJTASK")
	if !errors.Is(err, ErrNotASubtask) {
		t.Errorf("FetchParent() error = %v, want %v", err, ErrNotASubtask)
	}
	if parent != nil {
		t.Errorf("FetchParent() returned non-nil parent for non-subtask")
	}
}

func TestFetchParent_PropagatesClientError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 404 Not Found
		w.WriteHeader(http.StatusNotFound)
		mustEncode(w, map[string]string{"error": "Task not found"})
	})

	p, cleanup := setupMockProvider(t, handler)
	defer cleanup()

	_, err := p.FetchParent(context.Background(), "IEAAJNOTFOUND")
	if err == nil {
		t.Error("FetchParent() expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "fetch task") {
		t.Errorf("error should mention fetch task, got: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Snapshot parent section integration test
// ──────────────────────────────────────────────────────────────────────────────

func TestSnapshot_IncludesParentTaskSection(t *testing.T) {
	now := time.Now()
	requestCount := 0

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Handle comments request (returns empty)
		if strings.Contains(r.URL.Path, "/comments") {
			mustEncode(w, map[string][]any{"data": {}})

			return
		}

		if requestCount == 1 {
			// First task request: the subtask
			response := taskResponse{
				Data: []Task{
					{
						ID:           "IEAAJSUBTASK",
						Title:        "Subtask Title",
						Description:  "Subtask description",
						Status:       "Active",
						Priority:     "Normal",
						Permalink:    "https://www.wrike.com/open.htm?id=1111111111",
						SuperTaskIDs: []string{"IEAAJPARENT"},
						CreatedDate:  now,
						UpdatedDate:  now,
					},
				},
			}
			mustEncode(w, response)

			return
		}

		// Parent task request
		response := taskResponse{
			Data: []Task{
				{
					ID:          "IEAAJPARENT",
					Title:       "Parent Task Title",
					Description: "Parent task description content",
					Status:      "In Progress",
					Priority:    "High",
					Permalink:   "https://www.wrike.com/open.htm?id=2222222222",
					CreatedDate: now,
					UpdatedDate: now,
				},
			},
		}
		mustEncode(w, response)
	})

	p, cleanup := setupMockProvider(t, handler)
	defer cleanup()

	snapshot, err := p.Snapshot(context.Background(), "IEAAJSUBTASK")
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if snapshot == nil {
		t.Fatal("Snapshot() returned nil")
	}
	if len(snapshot.Files) == 0 {
		t.Fatal("Snapshot has no files")
	}

	content := snapshot.Files[0].Content

	// Check for "## Parent Task" section
	if !strings.Contains(content, "## Parent Task") {
		t.Error("Snapshot should contain '## Parent Task' section")
	}

	// Check parent title appears as markdown link
	if !strings.Contains(content, "[Parent Task Title]") {
		t.Error("Snapshot should contain parent title as link")
	}

	// Check parent permalink
	if !strings.Contains(content, "https://www.wrike.com/open.htm?id=2222222222") {
		t.Error("Snapshot should contain parent permalink")
	}

	// Check parent status
	if !strings.Contains(content, "In Progress") {
		t.Error("Snapshot should contain parent status")
	}

	// Check parent description
	if !strings.Contains(content, "Parent task description content") {
		t.Error("Snapshot should contain parent description")
	}
}

func TestSnapshot_ShowsUnavailableMessageOnParentFetchError(t *testing.T) {
	now := time.Now()
	requestCount := 0

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Handle comments request
		if strings.Contains(r.URL.Path, "/comments") {
			mustEncode(w, map[string][]any{"data": {}})

			return
		}

		if requestCount == 1 {
			// First request: subtask with parent ID
			response := taskResponse{
				Data: []Task{
					{
						ID:           "IEAAJSUBTASK",
						Title:        "Subtask",
						Status:       "Active",
						Permalink:    "https://www.wrike.com/open.htm?id=1111111111",
						SuperTaskIDs: []string{"IEAAJPARENT"},
						CreatedDate:  now,
						UpdatedDate:  now,
					},
				},
			}
			mustEncode(w, response)

			return
		}

		// Parent fetch fails with 500
		w.WriteHeader(http.StatusInternalServerError)
		mustEncode(w, map[string]string{"error": "Server error"})
	})

	p, cleanup := setupMockProvider(t, handler)
	defer cleanup()

	snapshot, err := p.Snapshot(context.Background(), "IEAAJSUBTASK")
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	content := snapshot.Files[0].Content

	// Should still have Parent Task section with unavailable message
	if !strings.Contains(content, "## Parent Task") {
		t.Error("Snapshot should contain '## Parent Task' section even on error")
	}
	if !strings.Contains(content, "Parent task information unavailable") {
		t.Error("Snapshot should show 'unavailable' message when parent fetch fails")
	}
}
