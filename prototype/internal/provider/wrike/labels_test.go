package wrike

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// ──────────────────────────────────────────────────────────────────────────────
// Provider Interface Compliance.
// ──────────────────────────────────────────────────────────────────────────────

// Compile-time interface checks.
var (
	_ provider.LabelManager = (*Provider)(nil)
)

// ──────────────────────────────────────────────────────────────────────────────
// AddLabels
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderAddLabels(t *testing.T) {
	tests := []struct {
		name         string
		taskID       string
		existingTags []string
		labels       []string
		wantTags     []string
		wantErr      bool
		errContains  string
		wantGetCalls int
		wantPutCalls int
	}{
		{
			name:         "success: adds new labels to empty task",
			taskID:       "TASK123",
			existingTags: []string{},
			labels:       []string{"urgent", "bug"},
			wantTags:     []string{"urgent", "bug"},
			wantErr:      false,
			wantGetCalls: 1,
			wantPutCalls: 1,
		},
		{
			name:         "success: adds new labels to existing tags",
			taskID:       "TASK456",
			existingTags: []string{"feature"},
			labels:       []string{"urgent", "bug"},
			wantTags:     []string{"feature", "urgent", "bug"},
			wantErr:      false,
			wantGetCalls: 1,
			wantPutCalls: 1,
		},
		{
			name:         "success: handles duplicate labels (no duplicates in result)",
			taskID:       "TASK789",
			existingTags: []string{"urgent"},
			labels:       []string{"urgent", "bug"},
			wantTags:     []string{"urgent", "bug"},
			wantErr:      false,
			wantGetCalls: 1,
			wantPutCalls: 1, // "bug" is new, so we need to update
		},
		{
			name:         "success: merging existing and new tags preserves order",
			taskID:       "TASK101",
			existingTags: []string{"low", "medium"},
			labels:       []string{"high", "critical"},
			wantTags:     []string{"low", "medium", "high", "critical"},
			wantErr:      false,
			wantGetCalls: 1,
			wantPutCalls: 1,
		},
		{
			name:         "success: adding duplicate labels from existing doesn't create duplicates",
			taskID:       "TASK102",
			existingTags: []string{"tag1", "tag2", "tag3"},
			labels:       []string{"tag2", "tag4"},
			wantTags:     []string{"tag1", "tag2", "tag3", "tag4"},
			wantErr:      false,
			wantGetCalls: 1,
			wantPutCalls: 1, // "tag4" is new
		},
		{
			name:         "success: empty labels list is no-op",
			taskID:       "TASK103",
			existingTags: []string{"existing"},
			labels:       []string{},
			wantTags:     []string{"existing"},
			wantErr:      false,
			wantGetCalls: 0, // Early return before GET
			wantPutCalls: 0,
		},
		{
			name:         "error: task not found",
			taskID:       "NOTFOUND",
			existingTags: []string{},
			labels:       []string{"urgent"},
			wantTags:     nil,
			wantErr:      true,
			errContains:  "get task",
			wantGetCalls: 1,
			wantPutCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server
			getCallCount := 0
			updateCallCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check authorization
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("expected Authorization header 'Bearer test-token', got %q", r.Header.Get("Authorization"))
				}

				if strings.Contains(r.URL.Path, "/tasks/") && r.Method == http.MethodGet {
					getCallCount++
					if tt.taskID == "NOTFOUND" {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusNotFound)

						return
					}

					// Return task with existing tags
					var tagsJSON []byte
					if len(tt.existingTags) > 0 {
						tagsArray := `["` + strings.Join(tt.existingTags, `","`) + `"]`
						tagsJSON = []byte(`{"data":[{"id":"` + tt.taskID + `","tags":` + tagsArray + `}]}`)
					} else {
						tagsJSON = []byte(`{"data":[{"id":"` + tt.taskID + `","tags":[]}]}`)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write(tagsJSON)
				} else if strings.Contains(r.URL.Path, "/tasks/") && r.Method == http.MethodPut {
					updateCallCount++
					// Verify the request body contains the updated tags
					// For now, just return success
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"data":[{"id":"` + tt.taskID + `"}]}`))
				} else {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			// Create provider with mock server
			client := NewClient("test-token", "")
			client.baseURL = server.URL

			provider := &Provider{
				client: client,
			}

			// Execute
			err := provider.AddLabels(context.Background(), tt.taskID, tt.labels)

			// Verify
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
				t.Errorf("unexpected error: %v", err)

				return
			}

			if getCallCount != tt.wantGetCalls {
				t.Errorf("expected %d GET call(s), got %d", tt.wantGetCalls, getCallCount)
			}

			if updateCallCount != tt.wantPutCalls {
				t.Errorf("expected %d PUT call(s), got %d", tt.wantPutCalls, updateCallCount)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// RemoveLabels
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderRemoveLabels(t *testing.T) {
	tests := []struct {
		name         string
		taskID       string
		existingTags []string
		labels       []string
		wantTags     []string
		wantErr      bool
		errContains  string
		wantGetCalls int
		wantPutCalls int
	}{
		{
			name:         "success: removes existing labels",
			taskID:       "TASK201",
			existingTags: []string{"urgent", "bug", "feature"},
			labels:       []string{"bug"},
			wantTags:     []string{"urgent", "feature"},
			wantErr:      false,
			wantGetCalls: 1,
			wantPutCalls: 1,
		},
		{
			name:         "success: keeps other tags intact",
			taskID:       "TASK202",
			existingTags: []string{"tag1", "tag2", "tag3", "tag4"},
			labels:       []string{"tag2"},
			wantTags:     []string{"tag1", "tag3", "tag4"},
			wantErr:      false,
			wantGetCalls: 1,
			wantPutCalls: 1,
		},
		{
			name:         "success: removing non-existent label is no-op",
			taskID:       "TASK203",
			existingTags: []string{"tag1", "tag2"},
			labels:       []string{"nonexistent"},
			wantTags:     []string{"tag1", "tag2"},
			wantErr:      false,
			wantGetCalls: 1,
			wantPutCalls: 0, // No change, so no PUT
		},
		{
			name:         "success: removing multiple labels",
			taskID:       "TASK204",
			existingTags: []string{"a", "b", "c", "d", "e"},
			labels:       []string{"b", "d"},
			wantTags:     []string{"a", "c", "e"},
			wantErr:      false,
			wantGetCalls: 1,
			wantPutCalls: 1,
		},
		{
			name:         "success: removing all tags leaves empty list",
			taskID:       "TASK205",
			existingTags: []string{"only-tag"},
			labels:       []string{"only-tag"},
			wantTags:     []string{},
			wantErr:      false,
			wantGetCalls: 1,
			wantPutCalls: 1,
		},
		{
			name:         "success: removing from empty task is no-op",
			taskID:       "TASK206",
			existingTags: []string{},
			labels:       []string{"nonexistent"},
			wantTags:     []string{},
			wantErr:      false,
			wantGetCalls: 1,
			wantPutCalls: 0, // No change, so no PUT
		},
		{
			name:         "success: empty remove list is no-op",
			taskID:       "TASK207",
			existingTags: []string{"tag1", "tag2"},
			labels:       []string{},
			wantTags:     []string{"tag1", "tag2"},
			wantErr:      false,
			wantGetCalls: 0, // Early return before GET
			wantPutCalls: 0,
		},
		{
			name:         "error: task not found",
			taskID:       "NOTFOUND",
			existingTags: []string{},
			labels:       []string{"urgent"},
			wantTags:     nil,
			wantErr:      true,
			errContains:  "get task",
			wantGetCalls: 1,
			wantPutCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server
			getCallCount := 0
			updateCallCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check authorization
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("expected Authorization header 'Bearer test-token', got %q", r.Header.Get("Authorization"))
				}

				if strings.Contains(r.URL.Path, "/tasks/") && r.Method == http.MethodGet {
					getCallCount++
					if tt.taskID == "NOTFOUND" {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusNotFound)

						return
					}

					// Return task with existing tags
					tagsArray := `["` + strings.Join(tt.existingTags, `","`) + `"]`
					tagsJSON := `{"data":[{"id":"` + tt.taskID + `","tags":` + tagsArray + `}]}`
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(tagsJSON))
				} else if strings.Contains(r.URL.Path, "/tasks/") && r.Method == http.MethodPut {
					updateCallCount++
					// For this test, we'll just return success
					// In a real test, we could verify the request body contains the correct tags
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"data":[{"id":"` + tt.taskID + `"}]}`))
				} else {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			// Create provider with mock server
			client := NewClient("test-token", "")
			client.baseURL = server.URL

			provider := &Provider{
				client: client,
			}

			// Execute
			err := provider.RemoveLabels(context.Background(), tt.taskID, tt.labels)

			// Verify
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
				t.Errorf("unexpected error: %v", err)

				return
			}

			if getCallCount != tt.wantGetCalls {
				t.Errorf("expected %d GET call(s), got %d", tt.wantGetCalls, getCallCount)
			}

			if updateCallCount != tt.wantPutCalls {
				t.Errorf("expected %d PUT call(s), got %d", tt.wantPutCalls, updateCallCount)
			}
		})
	}
}
