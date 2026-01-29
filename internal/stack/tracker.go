package stack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// DefaultPollInterval is the default interval for PR status polling.
const DefaultPollInterval = 5 * time.Minute

// Tracker monitors PR status for stacked features.
type Tracker struct {
	storage      *Storage
	pollInterval time.Duration
	stopChan     chan struct{}
	running      bool
}

// NewTracker creates a new PR status tracker.
func NewTracker(storage *Storage) *Tracker {
	return &Tracker{
		storage:      storage,
		pollInterval: DefaultPollInterval,
		stopChan:     make(chan struct{}),
	}
}

// SetPollInterval sets the polling interval.
func (t *Tracker) SetPollInterval(d time.Duration) {
	t.pollInterval = d
}

// Sync synchronizes PR status for all stacked tasks.
// This is the manual sync operation called by `mehr sync`.
func (t *Tracker) Sync(ctx context.Context, prFetcher provider.PRFetcher) (*SyncResult, error) {
	if err := t.storage.Load(); err != nil {
		return nil, fmt.Errorf("load stacks: %w", err)
	}

	result := &SyncResult{
		UpdatedTasks: make([]TaskUpdate, 0),
	}

	stacks := t.storage.ListStacks()
	for _, s := range stacks {
		for i := range s.Tasks {
			task := &s.Tasks[i]

			// Skip tasks without PR numbers
			if task.PRNumber == 0 {
				continue
			}

			// Skip tasks in terminal or pending-action states
			// These states take precedence over PR status
			if task.State == StateMerged || task.State == StateAbandoned ||
				task.State == StateNeedsRebase || task.State == StateConflict {
				continue
			}

			// Fetch PR status
			pr, err := prFetcher.FetchPullRequest(ctx, task.PRNumber)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("fetch PR #%d: %w", task.PRNumber, err))

				continue
			}

			// Update task state based on PR status
			update := t.updateTaskFromPR(s, task, pr)
			if update != nil {
				result.UpdatedTasks = append(result.UpdatedTasks, *update)
			}
		}
	}

	// Save updated stacks
	if err := t.storage.Save(); err != nil {
		return result, fmt.Errorf("save stacks: %w", err)
	}

	return result, nil
}

// SyncResult contains the results of a sync operation.
type SyncResult struct {
	UpdatedTasks []TaskUpdate
	Errors       []error
}

// TaskUpdate represents a single task state update.
type TaskUpdate struct {
	TaskID                  string
	PRNumber                int
	OldState                StackState
	NewState                StackState
	MergedAt                *time.Time
	ChildrenMarkedForRebase int
}

// updateTaskFromPR updates a task's state based on PR information.
// Returns a TaskUpdate if the state changed, nil otherwise.
func (t *Tracker) updateTaskFromPR(s *Stack, task *StackedTask, pr *provider.PullRequest) *TaskUpdate {
	oldState := task.State
	newState := prStateToStackState(pr.State)

	// No change needed
	if oldState == newState {
		return nil
	}

	update := &TaskUpdate{
		TaskID:   task.ID,
		PRNumber: task.PRNumber,
		OldState: oldState,
		NewState: newState,
	}

	task.State = newState
	task.UpdatedAt = time.Now()

	// If merged, record merge time and mark children for rebase
	if newState == StateMerged {
		now := time.Now()
		task.MergedAt = &now
		update.MergedAt = &now

		// Mark all children as needing rebase
		children := s.GetChildren(task.ID)
		for _, child := range children {
			childTask := s.GetTask(child.ID)
			if childTask != nil && childTask.State != StateMerged {
				childTask.State = StateNeedsRebase
				childTask.UpdatedAt = now
				update.ChildrenMarkedForRebase++
			}
		}

		// Recursively mark grandchildren
		s.MarkChildrenNeedsRebase(task.ID)
	}

	return update
}

// prStateToStackState converts a PR state string to a StackState.
// Handles variations across providers (GitHub, GitLab, Bitbucket, etc.).
func prStateToStackState(prState string) StackState {
	normalized := strings.ToLower(prState)

	switch normalized {
	case "merged":
		return StateMerged
	case "closed", "declined", "superseded":
		return StateAbandoned
	case "open", "opened":
		return StatePendingReview
	case "approved":
		return StateApproved
	default:
		// Unknown state, assume pending review
		return StatePendingReview
	}
}

// StartPolling starts background polling for PR status updates.
// This is used in auto mode.
func (t *Tracker) StartPolling(ctx context.Context, prFetcher provider.PRFetcher, onUpdate func(*SyncResult)) {
	if t.running {
		return
	}

	t.running = true
	go t.pollLoop(ctx, prFetcher, onUpdate)
}

// StopPolling stops background polling.
func (t *Tracker) StopPolling() {
	if !t.running {
		return
	}

	close(t.stopChan)
	t.running = false
	t.stopChan = make(chan struct{}) // Reset for potential restart
}

// IsRunning returns true if the tracker is actively polling.
func (t *Tracker) IsRunning() bool {
	return t.running
}

func (t *Tracker) pollLoop(ctx context.Context, prFetcher provider.PRFetcher, onUpdate func(*SyncResult)) {
	ticker := time.NewTicker(t.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.running = false

			return
		case <-t.stopChan:
			return
		case <-ticker.C:
			result, err := t.Sync(ctx, prFetcher)
			if err != nil {
				// Log error but continue polling
				continue
			}

			if onUpdate != nil && len(result.UpdatedTasks) > 0 {
				onUpdate(result)
			}
		}
	}
}
