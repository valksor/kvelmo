package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConvertQueuedTask(t *testing.T) {
	// Test nil input
	result := convertQueuedTask(nil)
	if result != nil {
		t.Errorf("convertQueuedTask(nil) = %v, want nil", result)
	}
}

func TestConvertTasks(t *testing.T) {
	// Test nil input
	result := convertTasks(nil)
	if result == nil {
		t.Error("convertTasks(nil) should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("convertTasks(nil) length = %d, want 0", len(result))
	}
}

func TestParseQueueID(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		prefix   string
		expected string
	}{
		{
			name:     "valid queue ID",
			path:     "/api/v1/project/queue/abc123",
			prefix:   "/api/v1/project/queue/",
			expected: "abc123",
		},
		{
			name:     "empty after prefix",
			path:     "/api/v1/project/queue/",
			prefix:   "/api/v1/project/queue/",
			expected: "",
		},
		{
			name:     "complex ID",
			path:     "/api/v1/project/queue/my-queue-2024-01-15",
			prefix:   "/api/v1/project/queue/",
			expected: "my-queue-2024-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseQueueID(tt.path, tt.prefix)
			if got != tt.expected {
				t.Errorf("parseQueueID(%q, %q) = %q, want %q", tt.path, tt.prefix, got, tt.expected)
			}
		})
	}
}

func TestProjectPlanRequest(t *testing.T) {
	req := projectPlanRequest{
		Source:       "file:task.md",
		Title:        "Test Project",
		Instructions: "Custom instructions",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded projectPlanRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Source != req.Source {
		t.Errorf("Source = %q, want %q", decoded.Source, req.Source)
	}
	if decoded.Title != req.Title {
		t.Errorf("Title = %q, want %q", decoded.Title, req.Title)
	}
	if decoded.Instructions != req.Instructions {
		t.Errorf("Instructions = %q, want %q", decoded.Instructions, req.Instructions)
	}
}

func TestProjectTaskEditRequest(t *testing.T) {
	title := "Updated Title"
	priority := 1
	status := "ready"

	req := projectTaskEditRequest{
		Title:     &title,
		Priority:  &priority,
		Status:    &status,
		DependsOn: []string{"task-1", "task-2"},
		Labels:    []string{"backend", "urgent"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded projectTaskEditRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if *decoded.Title != title {
		t.Errorf("Title = %q, want %q", *decoded.Title, title)
	}
	if *decoded.Priority != priority {
		t.Errorf("Priority = %d, want %d", *decoded.Priority, priority)
	}
	if *decoded.Status != status {
		t.Errorf("Status = %q, want %q", *decoded.Status, status)
	}
	if len(decoded.DependsOn) != 2 {
		t.Errorf("DependsOn length = %d, want 2", len(decoded.DependsOn))
	}
	if len(decoded.Labels) != 2 {
		t.Errorf("Labels length = %d, want 2", len(decoded.Labels))
	}
}

func TestProjectReorderRequest(t *testing.T) {
	tests := []struct {
		name string
		req  projectReorderRequest
	}{
		{
			name: "auto reorder",
			req:  projectReorderRequest{Auto: true},
		},
		{
			name: "manual reorder before",
			req: projectReorderRequest{
				TaskID:      "task-3",
				Position:    "before",
				ReferenceID: "task-1",
			},
		},
		{
			name: "manual reorder after",
			req: projectReorderRequest{
				TaskID:      "task-3",
				Position:    "after",
				ReferenceID: "task-5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded projectReorderRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.Auto != tt.req.Auto {
				t.Errorf("Auto = %v, want %v", decoded.Auto, tt.req.Auto)
			}
			if decoded.TaskID != tt.req.TaskID {
				t.Errorf("TaskID = %q, want %q", decoded.TaskID, tt.req.TaskID)
			}
			if decoded.Position != tt.req.Position {
				t.Errorf("Position = %q, want %q", decoded.Position, tt.req.Position)
			}
			if decoded.ReferenceID != tt.req.ReferenceID {
				t.Errorf("ReferenceID = %q, want %q", decoded.ReferenceID, tt.req.ReferenceID)
			}
		})
	}
}

func TestProjectSubmitRequest(t *testing.T) {
	req := projectSubmitRequest{
		QueueID:    "queue-123",
		Provider:   "wrike",
		CreateEpic: true,
		Labels:     []string{"q1", "feature"},
		DryRun:     true,
		Mention:    "@manager please review",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded projectSubmitRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.QueueID != req.QueueID {
		t.Errorf("QueueID = %q, want %q", decoded.QueueID, req.QueueID)
	}
	if decoded.Provider != req.Provider {
		t.Errorf("Provider = %q, want %q", decoded.Provider, req.Provider)
	}
	if decoded.CreateEpic != req.CreateEpic {
		t.Errorf("CreateEpic = %v, want %v", decoded.CreateEpic, req.CreateEpic)
	}
	if decoded.DryRun != req.DryRun {
		t.Errorf("DryRun = %v, want %v", decoded.DryRun, req.DryRun)
	}
	if decoded.Mention != req.Mention {
		t.Errorf("Mention = %q, want %q", decoded.Mention, req.Mention)
	}
}

func TestProjectStartRequest(t *testing.T) {
	tests := []struct {
		name string
		req  projectStartRequest
	}{
		{
			name: "auto mode",
			req:  projectStartRequest{Auto: true},
		},
		{
			name: "specific task",
			req: projectStartRequest{
				QueueID: "queue-123",
				TaskID:  "task-1",
			},
		},
		{
			name: "next task",
			req: projectStartRequest{
				QueueID: "queue-123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded projectStartRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.Auto != tt.req.Auto {
				t.Errorf("Auto = %v, want %v", decoded.Auto, tt.req.Auto)
			}
			if decoded.QueueID != tt.req.QueueID {
				t.Errorf("QueueID = %q, want %q", decoded.QueueID, tt.req.QueueID)
			}
			if decoded.TaskID != tt.req.TaskID {
				t.Errorf("TaskID = %q, want %q", decoded.TaskID, tt.req.TaskID)
			}
		})
	}
}

func TestProjectTaskResponse(t *testing.T) {
	resp := projectTaskResponse{
		ID:          "task-1",
		Title:       "Implement feature",
		Description: "Detailed description",
		Status:      "ready",
		Priority:    1,
		DependsOn:   []string{"task-0"},
		Blocks:      []string{"task-2", "task-3"},
		Labels:      []string{"backend"},
		Assignee:    "user@example.com",
		ExternalID:  "PROJ-123",
		ExternalURL: "https://jira.example.com/browse/PROJ-123",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded projectTaskResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ID != resp.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, resp.ID)
	}
	if decoded.Status != resp.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, resp.Status)
	}
	if len(decoded.DependsOn) != 1 {
		t.Errorf("DependsOn length = %d, want 1", len(decoded.DependsOn))
	}
	if len(decoded.Blocks) != 2 {
		t.Errorf("Blocks length = %d, want 2", len(decoded.Blocks))
	}
}

func TestProjectPlanResponse(t *testing.T) {
	resp := projectPlanResponse{
		QueueID: "queue-abc123",
		Title:   "Test Project",
		Source:  "research:/workspace/docs",
		Tasks: []*projectTaskResponse{
			{
				ID:     "task-1",
				Title:  "Task 1",
				Status: "ready",
			},
		},
		Questions: []string{"What is the scope?"},
		Blockers:  []string{"Needs API access"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded projectPlanResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.QueueID != resp.QueueID {
		t.Errorf("QueueID = %q, want %q", decoded.QueueID, resp.QueueID)
	}
	if decoded.Title != resp.Title {
		t.Errorf("Title = %q, want %q", decoded.Title, resp.Title)
	}
	if decoded.Source != resp.Source {
		t.Errorf("Source = %q, want %q", decoded.Source, resp.Source)
	}
	if len(decoded.Tasks) != 1 {
		t.Errorf("Tasks length = %d, want 1", len(decoded.Tasks))
	}
	if len(decoded.Questions) != 1 {
		t.Errorf("Questions length = %d, want 1", len(decoded.Questions))
	}
	if len(decoded.Blockers) != 1 {
		t.Errorf("Blockers length = %d, want 1", len(decoded.Blockers))
	}
}

func TestProjectQueueSummary(t *testing.T) {
	summary := projectQueueSummary{
		ID:        "queue-xyz",
		Title:     "My Project",
		Source:    "dir:/workspace/specs",
		Status:    "draft",
		TaskCount: 5,
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded projectQueueSummary
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ID != summary.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, summary.ID)
	}
	if decoded.Title != summary.Title {
		t.Errorf("Title = %q, want %q", decoded.Title, summary.Title)
	}
	if decoded.Source != summary.Source {
		t.Errorf("Source = %q, want %q", decoded.Source, summary.Source)
	}
	if decoded.Status != summary.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, summary.Status)
	}
	if decoded.TaskCount != summary.TaskCount {
		t.Errorf("TaskCount = %d, want %d", decoded.TaskCount, summary.TaskCount)
	}
}

func TestHandleProjectPlan_NoConductor(t *testing.T) {
	s := &Server{config: Config{Conductor: nil}}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/project/plan", bytes.NewBufferString(`{"source":"file:test.md"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleProjectPlan(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleProjectPlan_InvalidJSON(t *testing.T) {
	s := &Server{config: Config{Conductor: nil}}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/project/plan", bytes.NewBufferString(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleProjectPlan(w, req)

	// Should fail with bad request since JSON is invalid
	if w.Code != http.StatusServiceUnavailable && w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d or %d", w.Code, http.StatusServiceUnavailable, http.StatusBadRequest)
	}
}

func TestHandleProjectQueues_NoConductor(t *testing.T) {
	s := &Server{config: Config{Conductor: nil}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/project/queues", nil)
	w := httptest.NewRecorder()

	s.handleProjectQueues(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleProjectTasks_NoConductor(t *testing.T) {
	s := &Server{config: Config{Conductor: nil}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/project/tasks", nil)
	w := httptest.NewRecorder()

	s.handleProjectTasks(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleProjectSubmit_NoConductor(t *testing.T) {
	s := &Server{config: Config{Conductor: nil}}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/project/submit", bytes.NewBufferString(`{"provider":"wrike"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleProjectSubmit(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleProjectSubmit_MissingProvider(t *testing.T) {
	s := &Server{config: Config{Conductor: nil}}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/project/submit", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleProjectSubmit(w, req)

	// Either ServiceUnavailable (no conductor) or BadRequest (missing provider)
	if w.Code != http.StatusServiceUnavailable && w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d or %d", w.Code, http.StatusServiceUnavailable, http.StatusBadRequest)
	}
}

func TestHandleProjectReorder_NoConductor(t *testing.T) {
	s := &Server{config: Config{Conductor: nil}}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/project/reorder", bytes.NewBufferString(`{"auto":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleProjectReorder(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleProjectStart_NoConductor(t *testing.T) {
	s := &Server{config: Config{Conductor: nil}}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/project/start", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleProjectStart(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleProjectUpload_NoConductor(t *testing.T) {
	s := &Server{config: Config{Conductor: nil}}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/project/upload", nil)
	w := httptest.NewRecorder()

	s.handleProjectUpload(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleProjectSource_NoConductor(t *testing.T) {
	s := &Server{config: Config{Conductor: nil}}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/project/source", bytes.NewBufferString(`{"type":"reference","value":"github:123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleProjectSource(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleProjectSource_InvalidType(t *testing.T) {
	s := &Server{config: Config{Conductor: nil}}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/project/source", bytes.NewBufferString(`{"type":"invalid","value":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleProjectSource(w, req)

	// Either ServiceUnavailable (no conductor) or BadRequest (invalid type)
	if w.Code != http.StatusServiceUnavailable && w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d or %d", w.Code, http.StatusServiceUnavailable, http.StatusBadRequest)
	}
}

func TestHandleProjectQueueRoute_EmptyID(t *testing.T) {
	s := &Server{config: Config{Conductor: nil}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/project/queue/", nil)
	w := httptest.NewRecorder()

	s.handleProjectQueueRoute(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleProjectTaskEditRoute_EmptyID(t *testing.T) {
	s := &Server{config: Config{Conductor: nil}}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/project/tasks/", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleProjectTaskEditRoute(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
