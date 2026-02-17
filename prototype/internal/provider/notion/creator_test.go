package notion

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-toolkit/workunit"
)

// ──────────────────────────────────────────────────────────────────────────────
// Provider Interface Compliance
// ──────────────────────────────────────────────────────────────────────────────

// Compile-time interface checks.
var _ workunit.WorkUnitCreator = (*Provider)(nil)

// ──────────────────────────────────────────────────────────────────────────────
// CreateWorkUnit Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderCreateWorkUnit(t *testing.T) {
	tests := []struct {
		name        string
		databaseID  string
		opts        workunit.CreateWorkUnitOptions
		wantErr     bool
		errContains string
		validate    func(*testing.T, *workunit.WorkUnit)
	}{
		{
			name:       "success: creates page with title and description",
			databaseID: "a1b2c3d4e5f678901234567890abcdef",
			opts: workunit.CreateWorkUnitOptions{
				Title:       "Test Task",
				Description: "Test description",
			},
			wantErr: false,
			validate: func(t *testing.T, wu *workunit.WorkUnit) {
				t.Helper()

				if wu.Title != "Test Task" {
					t.Errorf("Title = %q, want %q", wu.Title, "Test Task")
				}
				if wu.Description != "Test description" {
					t.Errorf("Description = %q, want %q", wu.Description, "Test description")
				}
				if wu.Provider != ProviderName {
					t.Errorf("Provider = %q, want %q", wu.Provider, ProviderName)
				}
				if wu.TaskType != "page" {
					t.Errorf("TaskType = %q, want %q", wu.TaskType, "page")
				}
			},
		},
		{
			name:       "success: creates page with labels",
			databaseID: "a1b2c3d4e5f678901234567890abcdef",
			opts: workunit.CreateWorkUnitOptions{
				Title:  "Labeled Task",
				Labels: []string{"bug", "urgent"},
			},
			wantErr: false,
			validate: func(t *testing.T, wu *workunit.WorkUnit) {
				t.Helper()

				if len(wu.Labels) != 2 {
					t.Errorf("Labels length = %d, want 2", len(wu.Labels))
				}
			},
		},
		{
			name:       "success: creates page with assignee",
			databaseID: "a1b2c3d4e5f678901234567890abcdef",
			opts: workunit.CreateWorkUnitOptions{
				Title:     "Assigned Task",
				Assignees: []string{"user-123"},
			},
			wantErr: false,
			validate: func(t *testing.T, wu *workunit.WorkUnit) {
				t.Helper()

				if wu.Title != "Assigned Task" {
					t.Errorf("Title = %q, want %q", wu.Title, "Assigned Task")
				}
			},
		},
		{
			name:       "success: uses status from CustomFields",
			databaseID: "a1b2c3d4e5f678901234567890abcdef",
			opts: workunit.CreateWorkUnitOptions{
				Title: "Status Task",
				CustomFields: map[string]any{
					"status": workunit.StatusInProgress,
				},
			},
			wantErr: false,
			validate: func(t *testing.T, wu *workunit.WorkUnit) {
				t.Helper()

				if wu.Status != workunit.StatusInProgress {
					t.Errorf("Status = %q, want %q", wu.Status, workunit.StatusInProgress)
				}
			},
		},
		{
			name:       "success: defaults to StatusOpen when no status",
			databaseID: "a1b2c3d4e5f678901234567890abcdef",
			opts: workunit.CreateWorkUnitOptions{
				Title: "Default Status Task",
			},
			wantErr: false,
			validate: func(t *testing.T, wu *workunit.WorkUnit) {
				t.Helper()

				if wu.Status != workunit.StatusOpen {
					t.Errorf("Status = %q, want %q", wu.Status, workunit.StatusOpen)
				}
			},
		},
		{
			name:       "success: maps WorkUnit fields correctly",
			databaseID: "a1b2c3d4e5f678901234567890abcdef",
			opts: workunit.CreateWorkUnitOptions{
				Title:       "Full Task",
				Description: "Complete description",
				Priority:    workunit.PriorityHigh,
			},
			wantErr: false,
			validate: func(t *testing.T, wu *workunit.WorkUnit) {
				t.Helper()

				// Verify all required fields are set
				if wu.ID == "" {
					t.Error("ID should not be empty")
				}
				if wu.ExternalID == "" {
					t.Error("ExternalID should not be empty")
				}
				if wu.ExternalKey == "" {
					t.Error("ExternalKey should not be empty")
				}
				if wu.Provider != ProviderName {
					t.Errorf("Provider = %q, want %q", wu.Provider, ProviderName)
				}
				if wu.Priority != workunit.PriorityHigh {
					t.Errorf("Priority = %v, want %v", wu.Priority, workunit.PriorityHigh)
				}
				if wu.TaskType != "page" {
					t.Errorf("TaskType = %q, want %q", wu.TaskType, "page")
				}
				if wu.CreatedAt.IsZero() {
					t.Error("CreatedAt should be set")
				}
				if wu.UpdatedAt.IsZero() {
					t.Error("UpdatedAt should be set")
				}
				if wu.Source.Type != ProviderName {
					t.Errorf("Source.Type = %q, want %q", wu.Source.Type, ProviderName)
				}
			},
		},
		{
			name:       "error: no database configured",
			databaseID: "",
			opts: workunit.CreateWorkUnitOptions{
				Title: "Test Task",
			},
			wantErr:     true,
			errContains: "database_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				// Verify path
				if !strings.Contains(r.URL.Path, "/v1/pages") {
					t.Errorf("expected path to contain /v1/pages, got %s", r.URL.Path)
				}

				// Check authorization
				auth := r.Header.Get("Authorization")
				if !strings.HasPrefix(auth, "Bearer ") {
					t.Errorf("expected Authorization header to start with 'Bearer ', got %q", auth)
				}

				// Check Notion-Version header
				notionVersion := r.Header.Get("Notion-Version")
				if notionVersion == "" {
					t.Error("expected Notion-Version header to be set")
				}

				// Parse request body
				body, _ := io.ReadAll(r.Body)
				var createReq CreatePageInput
				if err := json.Unmarshal(body, &createReq); err != nil {
					t.Errorf("failed to parse request body: %v", err)
				}

				// Verify request contains title
				if createReq.Properties["Name"].Title == nil {
					t.Error("expected Name property to contain title")
				}

				// Return success response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				now := time.Now().Format(time.RFC3339)
				response := Page{
					ID:             "page-123456789012345678901234",
					CreatedTime:    time.Now(),
					LastEditedTime: time.Now(),
					URL:            "https://www.notion.so/Test-page-123456789012345678901234",
					Properties: map[string]Property{
						"Name": MakeTitleProperty(tt.opts.Title),
					},
				}
				respBytes, _ := json.Marshal(response)
				_, _ = w.Write(respBytes)
				_ = now // silence unused
			}))
			defer server.Close()

			// Create client with mock server
			client := NewClient("test-token")
			client.baseURL = server.URL

			// Create provider
			p := &Provider{
				client:              client,
				databaseID:          tt.databaseID,
				statusProperty:      "Status",
				descriptionProperty: "Description",
				labelsProperty:      "Tags",
			}

			// Execute
			wu, err := p.CreateWorkUnit(context.Background(), tt.opts)

			// Verify error cases
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

			// Verify success case
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if wu == nil {
				t.Fatal("expected non-nil WorkUnit on success")
			}

			// Run validation function
			if tt.validate != nil {
				tt.validate(t, wu)
			}
		})
	}
}

func TestProviderCreateWorkUnitRequestBody(t *testing.T) {
	// Test that the request body is properly formed
	var capturedBody CreatePageInput

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := Page{
			ID:             "page-123456789012345678901234",
			CreatedTime:    time.Now(),
			LastEditedTime: time.Now(),
			URL:            "https://www.notion.so/Test-page-123456789012345678901234",
		}
		respBytes, _ := json.Marshal(response)
		_, _ = w.Write(respBytes)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	p := &Provider{
		client:              client,
		databaseID:          "db-123",
		statusProperty:      "Status",
		descriptionProperty: "Description",
		labelsProperty:      "Tags",
	}

	opts := workunit.CreateWorkUnitOptions{
		Title:       "Test Title",
		Description: "Test Description",
		Labels:      []string{"label1", "label2"},
		Assignees:   []string{"user-1"},
		CustomFields: map[string]any{
			"status": workunit.StatusInProgress,
		},
	}

	_, err := p.CreateWorkUnit(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify parent is set correctly
	if capturedBody.Parent.Type != "database_id" {
		t.Errorf("Parent.Type = %q, want %q", capturedBody.Parent.Type, "database_id")
	}
	if capturedBody.Parent.DatabaseID != "db-123" {
		t.Errorf("Parent.DatabaseID = %q, want %q", capturedBody.Parent.DatabaseID, "db-123")
	}

	// Verify title property
	nameProp, ok := capturedBody.Properties["Name"]
	if !ok {
		t.Fatal("expected Name property to be set")
	}
	if nameProp.Title == nil || len(nameProp.Title.Title) == 0 {
		t.Fatal("expected Name property to have title text")
	}
	if nameProp.Title.Title[0].PlainText != "Test Title" {
		t.Errorf("Name title = %q, want %q", nameProp.Title.Title[0].PlainText, "Test Title")
	}

	// Verify description property
	descProp, ok := capturedBody.Properties["Description"]
	if !ok {
		t.Fatal("expected Description property to be set")
	}
	if descProp.RichText == nil {
		t.Fatal("expected Description property to have rich text")
	}

	// Verify status property
	statusProp, ok := capturedBody.Properties["Status"]
	if !ok {
		t.Fatal("expected Status property to be set")
	}
	if statusProp.Status == nil {
		t.Fatal("expected Status property to have status")
	}
	if statusProp.Status.Name != "In Progress" {
		t.Errorf("Status = %q, want %q", statusProp.Status.Name, "In Progress")
	}

	// Verify labels/tags property
	tagsProp, ok := capturedBody.Properties["Tags"]
	if !ok {
		t.Fatal("expected Tags property to be set")
	}
	if tagsProp.MultiSelect == nil || len(tagsProp.MultiSelect.Options) != 2 {
		t.Fatal("expected Tags property to have 2 options")
	}

	// Verify assignee property
	assigneeProp, ok := capturedBody.Properties["Assignee"]
	if !ok {
		t.Fatal("expected Assignee property to be set")
	}
	if assigneeProp.People == nil || len(assigneeProp.People.People) == 0 {
		t.Fatal("expected Assignee property to have people")
	}
	if assigneeProp.People.People[0].ID != "user-1" {
		t.Errorf("Assignee ID = %q, want %q", assigneeProp.People.People[0].ID, "user-1")
	}
}

func TestProviderCreateWorkUnitAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"object":"error","status":400,"code":"validation_error","message":"Invalid request"}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	p := &Provider{
		client:              client,
		databaseID:          "db-123",
		statusProperty:      "Status",
		descriptionProperty: "Description",
		labelsProperty:      "Tags",
	}

	_, err := p.CreateWorkUnit(context.Background(), workunit.CreateWorkUnitOptions{
		Title: "Test",
	})

	if err == nil {
		t.Fatal("expected error on API failure")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Benchmark Tests
// ──────────────────────────────────────────────────────────────────────────────

func BenchmarkCreateWorkUnitOptionsMapping(b *testing.B) {
	opts := workunit.CreateWorkUnitOptions{
		Title:       "Benchmark Task",
		Description: "This is a benchmark test task",
		Labels:      []string{"benchmark", "test", "performance"},
		Assignees:   []string{"user-1", "user-2"},
		Priority:    workunit.PriorityHigh,
		CustomFields: map[string]any{
			"status": workunit.StatusInProgress,
		},
	}

	p := &Provider{
		databaseID:          "db-123",
		statusProperty:      "Status",
		descriptionProperty: "Description",
		labelsProperty:      "Tags",
	}

	b.ResetTimer()
	for range b.N {
		// Test the mapping logic without making HTTP calls
		status := workunit.StatusOpen
		if opts.CustomFields != nil {
			if s, ok := opts.CustomFields["status"].(workunit.Status); ok {
				status = s
			}
		}

		properties := map[string]Property{
			"Name":           MakeTitleProperty(opts.Title),
			p.statusProperty: MakeStatusProperty(mapProviderStatusToNotion(status)),
			p.labelsProperty: MakeMultiSelectProperty(opts.Labels),
			"Description":    MakeRichTextProperty(opts.Description),
		}
		_ = properties
	}
}
