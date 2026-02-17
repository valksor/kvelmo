package asana

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-toolkit/workunit"
)

// ──────────────────────────────────────────────────────────────────────────────
// Provider Interface Compliance.
// ──────────────────────────────────────────────────────────────────────────────

// Compile-time interface checks.
var (
	_ workunit.WorkUnitCreator = (*Provider)(nil)
)

// ──────────────────────────────────────────────────────────────────────────────
// CreateWorkUnit
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderCreateWorkUnit(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Provider)
		opts        workunit.CreateWorkUnitOptions
		wantErr     bool
		errContains string
		validate    func(*testing.T, *workunit.WorkUnit)
	}{
		{
			name: "success: creates task with title and description",
			setup: func(p *Provider) {
				p.config.DefaultProject = "1234567890123456"
			},
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
				if wu.TaskType != "task" {
					t.Errorf("TaskType = %q, want %q", wu.TaskType, "task")
				}
			},
		},
		{
			name: "success: uses project from opts.ParentID",
			setup: func(p *Provider) {
				p.config.DefaultProject = "default-project"
			},
			opts: workunit.CreateWorkUnitOptions{
				Title:    "Test Task",
				ParentID: "custom-project",
			},
			wantErr: false,
			validate: func(t *testing.T, wu *workunit.WorkUnit) {
				t.Helper()

				if wu.Title != "Test Task" {
					t.Errorf("Title = %q, want %q", wu.Title, "Test Task")
				}
			},
		},
		{
			name: "success: uses default project when ParentID empty",
			setup: func(p *Provider) {
				p.config.DefaultProject = "default-project-123"
			},
			opts: workunit.CreateWorkUnitOptions{
				Title:    "Test Task",
				ParentID: "",
			},
			wantErr: false,
			validate: func(t *testing.T, wu *workunit.WorkUnit) {
				t.Helper()

				if wu.Title != "Test Task" {
					t.Errorf("Title = %q, want %q", wu.Title, "Test Task")
				}
			},
		},
		{
			name: "success: maps task to WorkUnit correctly",
			setup: func(p *Provider) {
				p.config.DefaultProject = "1234567890123456"
			},
			opts: workunit.CreateWorkUnitOptions{
				Title:       "Full Task",
				Description: "Complete description",
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
				if wu.Status == "" {
					t.Error("Status should not be empty")
				}
				if wu.Priority != workunit.PriorityNormal {
					t.Errorf("Priority = %v, want %v", wu.Priority, workunit.PriorityNormal)
				}
				if wu.TaskType != "task" {
					t.Errorf("TaskType = %q, want %q", wu.TaskType, "task")
				}
				if wu.Slug == "" {
					t.Error("Slug should not be empty")
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
				if !strings.HasPrefix(wu.Source.Reference, "asana:") {
					t.Errorf("Source.Reference should start with 'asana:', got %q", wu.Source.Reference)
				}
				if wu.Metadata == nil {
					t.Error("Metadata should not be nil")
				}
			},
		},
		{
			name: "error: no project configured (default empty, opts empty)",
			setup: func(p *Provider) {
				p.config.DefaultProject = ""
			},
			opts: workunit.CreateWorkUnitOptions{
				Title:    "Test Task",
				ParentID: "",
			},
			wantErr:     true,
			errContains: "no project specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/tasks") {
					t.Errorf("expected path to contain /tasks, got %s", r.URL.Path)
				}

				// Check authorization
				auth := r.Header.Get("Authorization")
				if !strings.HasPrefix(auth, "Bearer ") {
					t.Errorf("expected Authorization header to start with 'Bearer ', got %q", auth)
				}

				// Read request body to check if project is provided
				body, _ := io.ReadAll(r.Body)
				bodyStr := string(body)

				// For the error test case, check if projects field is missing or empty
				if tt.wantErr && tt.name == "error: no project configured (default empty, opts empty)" {
					hasProjects := strings.Contains(bodyStr, `"projects"`)
					isEmptyProjects := strings.Contains(bodyStr, `"projects":[]`) || strings.Contains(bodyStr, `"projects": []`)
					if !hasProjects || isEmptyProjects {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusBadRequest)
						_, _ = w.Write([]byte(`{"errors": [{"message": "project required"}]}`))

						return
					}
				}

				// Return success response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)

				// Build response based on request
				response := `{
					"data": {
						"gid": "9876543210987654",
						"name": "` + tt.opts.Title + `",
						"notes": "` + tt.opts.Description + `",
						"completed": false,
						"created_at": "` + time.Now().Format(time.RFC3339) + `",
						"modified_at": "` + time.Now().Format(time.RFC3339) + `",
						"permalink_url": "https://app.asana.com/0/1234567890123456/9876543210987654",
						"projects": [{"gid": "1234567890123456", "name": "Test Project"}],
						"tags": []
					}
				}`
				_, _ = w.Write([]byte(response))
			}))
			defer server.Close()

			// Create client with mock server
			client := NewClient("test-token", "")
			client.baseURL = server.URL

			// Create provider
			provider := &Provider{
				client: client,
				config: &Config{
					DefaultProject: "",
				},
			}

			// Apply test-specific setup
			tt.setup(provider)

			// Execute
			wu, err := provider.CreateWorkUnit(context.Background(), tt.opts)

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

func TestProviderCreateWorkUnitWithTags(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		response := `{
			"data": {
				"gid": "task123",
				"name": "Tagged Task",
				"notes": "Description",
				"completed": false,
				"created_at": "` + time.Now().Format(time.RFC3339) + `",
				"modified_at": "` + time.Now().Format(time.RFC3339) + `",
				"permalink_url": "https://app.asana.com/0/123/456",
				"projects": [],
				"tags": [
					{"gid": "tag1", "name": "urgent"},
					{"gid": "tag2", "name": "feature"}
				]
			}
		}`
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewClient("test-token", "")
	client.baseURL = server.URL

	p := &Provider{
		client: client,
		config: &Config{
			DefaultProject: "project123",
		},
	}

	wu, err := p.CreateWorkUnit(context.Background(), workunit.CreateWorkUnitOptions{
		Title: "Tagged Task",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(wu.Labels) != 2 {
		t.Errorf("Labels length = %d, want 2", len(wu.Labels))
	}

	expectedLabels := []string{"urgent", "feature"}
	for i, label := range wu.Labels {
		if label != expectedLabels[i] {
			t.Errorf("Labels[%d] = %q, want %q", i, label, expectedLabels[i])
		}
	}
}
