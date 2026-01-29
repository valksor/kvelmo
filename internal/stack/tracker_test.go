package stack

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// mockPRFetcher implements provider.PRFetcher for testing.
type mockPRFetcher struct {
	prs map[int]*provider.PullRequest
}

func (m *mockPRFetcher) FetchPullRequest(_ context.Context, number int) (*provider.PullRequest, error) {
	if pr, ok := m.prs[number]; ok {
		return pr, nil
	}

	return nil, fmt.Errorf("PR #%d not found", number)
}

func (m *mockPRFetcher) FetchPullRequestDiff(_ context.Context, number int) (*provider.PullRequestDiff, error) {
	return nil, fmt.Errorf("PR diff #%d not found", number)
}

func TestNewTracker(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)
	tracker := NewTracker(storage)

	if tracker == nil {
		t.Fatal("expected tracker, got nil")
	}
	if tracker.pollInterval != DefaultPollInterval {
		t.Errorf("expected default poll interval %v, got %v", DefaultPollInterval, tracker.pollInterval)
	}
}

func TestTracker_SetPollInterval(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)
	tracker := NewTracker(storage)

	newInterval := 10 * time.Minute
	tracker.SetPollInterval(newInterval)

	if tracker.pollInterval != newInterval {
		t.Errorf("expected poll interval %v, got %v", newInterval, tracker.pollInterval)
	}
}

func TestPrStateToStackState(t *testing.T) {
	tests := []struct {
		prState  string
		expected StackState
	}{
		{"merged", StateMerged},
		{"MERGED", StateMerged},
		{"closed", StateAbandoned},
		{"CLOSED", StateAbandoned},
		{"declined", StateAbandoned},
		{"DECLINED", StateAbandoned},
		{"superseded", StateAbandoned},
		{"open", StatePendingReview},
		{"OPEN", StatePendingReview},
		{"opened", StatePendingReview},
		{"approved", StateApproved},
		{"unknown", StatePendingReview},
	}

	for _, tt := range tests {
		t.Run(tt.prState, func(t *testing.T) {
			got := prStateToStackState(tt.prState)
			if got != tt.expected {
				t.Errorf("prStateToStackState(%q) = %v, want %v", tt.prState, got, tt.expected)
			}
		})
	}
}

func TestTracker_Sync(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)

	// Create a stack with parent task having a PR
	s := NewStack("stack-1", "issue-100", "feature/auth", "main")
	s.Tasks[0].PRNumber = 100
	s.Tasks[0].State = StatePendingReview
	// Add child task without PR (simpler test case)
	s.AddTask("issue-101", "feature/oauth", "issue-100")

	_ = storage.AddStack(s)
	_ = storage.Save()

	// Create mock PR fetcher - only parent has PR
	mockFetcher := &mockPRFetcher{
		prs: map[int]*provider.PullRequest{
			100: {Number: 100, State: "merged"},
		},
	}

	tracker := NewTracker(storage)
	result, err := tracker.Sync(context.Background(), mockFetcher)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	// Should have 1 update (PR 100 merged)
	if len(result.UpdatedTasks) != 1 {
		t.Errorf("expected 1 update, got %d", len(result.UpdatedTasks))
	}

	if len(result.UpdatedTasks) > 0 {
		update := result.UpdatedTasks[0]
		if update.TaskID != "issue-100" {
			t.Errorf("expected task issue-100, got %s", update.TaskID)
		}
		if update.NewState != StateMerged {
			t.Errorf("expected state merged, got %s", update.NewState)
		}
		if update.ChildrenMarkedForRebase != 1 {
			t.Errorf("expected 1 child marked for rebase, got %d", update.ChildrenMarkedForRebase)
		}
	}

	// Reload and verify state
	_ = storage.Load()
	updatedStack := storage.GetStack("stack-1")
	task100 := updatedStack.GetTask("issue-100")
	if task100.State != StateMerged {
		t.Errorf("expected issue-100 state merged, got %s", task100.State)
	}

	// Child should be marked for rebase
	task101 := updatedStack.GetTask("issue-101")
	if task101.State != StateNeedsRebase {
		t.Errorf("expected issue-101 state needs-rebase, got %s", task101.State)
	}
}

func TestTracker_SyncWithChildPR(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)

	// Create a stack with both tasks having PRs
	s := NewStack("stack-1", "issue-100", "feature/auth", "main")
	s.Tasks[0].PRNumber = 100
	s.Tasks[0].State = StatePendingReview
	s.AddTask("issue-101", "feature/oauth", "issue-100")
	s.Tasks[1].PRNumber = 101
	s.Tasks[1].State = StatePendingReview

	_ = storage.AddStack(s)
	_ = storage.Save()

	// Parent PR merged, child still open
	mockFetcher := &mockPRFetcher{
		prs: map[int]*provider.PullRequest{
			100: {Number: 100, State: "merged"},
			101: {Number: 101, State: "open"},
		},
	}

	tracker := NewTracker(storage)
	result, err := tracker.Sync(context.Background(), mockFetcher)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	// Should have 1 update - parent merged
	// Child should be skipped because it's marked needs-rebase after parent merges
	if len(result.UpdatedTasks) != 1 {
		t.Errorf("expected 1 update, got %d", len(result.UpdatedTasks))
		for _, u := range result.UpdatedTasks {
			t.Logf("  update: %s %s -> %s", u.TaskID, u.OldState, u.NewState)
		}
	}

	// Verify states
	_ = storage.Load()
	updatedStack := storage.GetStack("stack-1")

	task100 := updatedStack.GetTask("issue-100")
	if task100.State != StateMerged {
		t.Errorf("expected issue-100 state merged, got %s", task100.State)
	}

	task101 := updatedStack.GetTask("issue-101")
	// Child should be needs-rebase, NOT reverted to pending-review by its open PR
	if task101.State != StateNeedsRebase {
		t.Errorf("expected issue-101 state needs-rebase, got %s", task101.State)
	}
}

func TestTracker_SyncNoChanges(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)

	s := NewStack("stack-1", "issue-100", "feature/auth", "main")
	s.Tasks[0].PRNumber = 100
	s.Tasks[0].State = StatePendingReview

	_ = storage.AddStack(s)
	_ = storage.Save()

	mockFetcher := &mockPRFetcher{
		prs: map[int]*provider.PullRequest{
			100: {Number: 100, State: "open"}, // Still open, no change
		},
	}

	tracker := NewTracker(storage)
	result, err := tracker.Sync(context.Background(), mockFetcher)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(result.UpdatedTasks) != 0 {
		t.Errorf("expected 0 updates, got %d", len(result.UpdatedTasks))
	}
}

func TestTracker_StartStopPolling(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)
	tracker := NewTracker(storage)

	if tracker.IsRunning() {
		t.Error("expected tracker to not be running initially")
	}

	// Start polling with short interval for testing
	tracker.SetPollInterval(100 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockFetcher := &mockPRFetcher{prs: map[int]*provider.PullRequest{}}
	tracker.StartPolling(ctx, mockFetcher, nil)

	if !tracker.IsRunning() {
		t.Error("expected tracker to be running after StartPolling")
	}

	// Stop polling
	tracker.StopPolling()

	// Give it a moment to stop
	time.Sleep(50 * time.Millisecond)

	if tracker.IsRunning() {
		t.Error("expected tracker to not be running after StopPolling")
	}
}
