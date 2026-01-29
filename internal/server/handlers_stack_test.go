package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/valksor/go-mehrhof/internal/stack"
)

func TestGetStackStateIcon(t *testing.T) {
	tests := []struct {
		state    stack.StackState
		expected string
	}{
		{stack.StateMerged, "check"},
		{stack.StateNeedsRebase, "refresh"},
		{stack.StateConflict, "x-circle"},
		{stack.StatePendingReview, "clock"},
		{stack.StateApproved, "check-circle"},
		{stack.StateAbandoned, "slash"},
		{stack.StateActive, "play"},
		{stack.StackState("unknown"), "circle"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := getStackStateIcon(tt.state)
			if got != tt.expected {
				t.Errorf("getStackStateIcon(%q) = %q, want %q", tt.state, got, tt.expected)
			}
		})
	}
}

func TestHandleStackList_NoConductor(t *testing.T) {
	srv := &Server{
		config: Config{
			Mode: ModeProject,
			// No Conductor set
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stack", nil)
	w := httptest.NewRecorder()

	srv.handleStackList(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp["error"] != "conductor not initialized" {
		t.Errorf("expected 'conductor not initialized' error, got %q", resp["error"])
	}
}

func TestStackSyncResponse_Types(t *testing.T) {
	// Test response structure
	resp := stackSyncResponse{
		Success: true,
		Updated: 3,
		UpdatedTasks: []taskUpdate{
			{TaskID: "task-1", OldState: "pending-review", NewState: "merged", Children: 1},
		},
		Errors: []string{"error 1"},
	}

	if !resp.Success {
		t.Error("expected Success to be true")
	}
	if resp.Updated != 3 {
		t.Errorf("expected Updated=3, got %d", resp.Updated)
	}
	if len(resp.UpdatedTasks) != 1 {
		t.Errorf("expected 1 updated task, got %d", len(resp.UpdatedTasks))
	}
	if len(resp.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(resp.Errors))
	}
}

func TestRebaseRequest_Types(t *testing.T) {
	// Test request structure
	req := rebaseRequest{
		StackID: "stack-1",
		TaskID:  "task-1",
	}

	if req.StackID != "stack-1" {
		t.Errorf("expected StackID='stack-1', got %q", req.StackID)
	}
	if req.TaskID != "task-1" {
		t.Errorf("expected TaskID='task-1', got %q", req.TaskID)
	}
}

func TestRebaseResponse_Types(t *testing.T) {
	// Test response structure
	resp := rebaseResponse{
		Success: true,
		Rebased: 2,
		Results: []rebaseResult{
			{TaskID: "task-1", Branch: "feature/a", OldBase: "main", NewBase: "main"},
		},
		Failed: &failedRebaseInfo{
			TaskID:       "task-2",
			Branch:       "feature/b",
			OntoBase:     "main",
			IsConflict:   true,
			ConflictHint: "resolve manually",
		},
		Error: "rebase failed",
	}

	if !resp.Success {
		t.Error("expected Success to be true")
	}
	if resp.Rebased != 2 {
		t.Errorf("expected Rebased=2, got %d", resp.Rebased)
	}
	if len(resp.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Failed == nil {
		t.Error("expected Failed to be set")
	} else if !resp.Failed.IsConflict {
		t.Error("expected IsConflict to be true")
	}
}

func TestStackSummary_Types(t *testing.T) {
	summary := stackSummary{
		ID:          "stack-1",
		RootTask:    "task-100",
		TaskCount:   3,
		HasRebase:   true,
		HasConflict: false,
	}

	if summary.ID != "stack-1" {
		t.Errorf("expected ID='stack-1', got %q", summary.ID)
	}
	if !summary.HasRebase {
		t.Error("expected HasRebase to be true")
	}
	if summary.HasConflict {
		t.Error("expected HasConflict to be false")
	}
}
