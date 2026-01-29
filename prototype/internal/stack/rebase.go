package stack

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/valksor/go-mehrhof/internal/vcs"
)

// ErrRebaseConflict indicates a rebase failed due to merge conflicts.
var ErrRebaseConflict = errors.New("rebase conflict")

// Rebaser handles rebasing stacked tasks after parent branches merge.
type Rebaser struct {
	storage *Storage
	git     *vcs.Git
}

// NewRebaser creates a new Rebaser.
func NewRebaser(storage *Storage, git *vcs.Git) *Rebaser {
	return &Rebaser{
		storage: storage,
		git:     git,
	}
}

// RebaseResult contains the results of a rebase operation.
type RebaseResult struct {
	RebasedTasks   []RebaseTaskResult
	SkippedTasks   []SkippedTask
	FailedTask     *FailedRebase
	OriginalBranch string
}

// RebaseTaskResult represents a successful rebase of a single task.
type RebaseTaskResult struct {
	TaskID   string
	Branch   string
	OldBase  string
	NewBase  string
	Rebased  bool
	Duration time.Duration
}

// SkippedTask represents a task that was skipped during rebase.
type SkippedTask struct {
	TaskID string
	Branch string
	Reason string
}

// FailedRebase contains details about a failed rebase.
type FailedRebase struct {
	TaskID       string
	Branch       string
	OntoBase     string
	Error        error
	IsConflict   bool
	ConflictHint string
}

// RebaseAll rebases all tasks in the stack that need rebasing.
// Returns after first conflict - abort is automatic.
func (r *Rebaser) RebaseAll(ctx context.Context, stackID string) (*RebaseResult, error) {
	if err := r.storage.Load(); err != nil {
		return nil, fmt.Errorf("load stacks: %w", err)
	}

	s := r.storage.GetStack(stackID)
	if s == nil {
		return nil, fmt.Errorf("stack not found: %s", stackID)
	}

	// Get current branch to restore later
	originalBranch, err := r.git.CurrentBranch(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current branch: %w", err)
	}

	result := &RebaseResult{
		RebasedTasks:   make([]RebaseTaskResult, 0),
		SkippedTasks:   make([]SkippedTask, 0),
		OriginalBranch: originalBranch,
	}

	// Get tasks in dependency order (parents first)
	tasksToRebase := r.getTasksInRebaseOrder(s)

	for _, task := range tasksToRebase {
		taskResult, err := r.rebaseTask(ctx, s, task)
		if err != nil {
			// Check if it's a conflict
			if errors.Is(err, ErrRebaseConflict) {
				result.FailedTask = &FailedRebase{
					TaskID:       task.ID,
					Branch:       task.Branch,
					OntoBase:     r.getRebaseTarget(s, task),
					Error:        err,
					IsConflict:   true,
					ConflictHint: "Resolve conflicts manually or run 'mehr stack rebase --continue' after fixing",
				}
			} else {
				result.FailedTask = &FailedRebase{
					TaskID:   task.ID,
					Branch:   task.Branch,
					OntoBase: r.getRebaseTarget(s, task),
					Error:    err,
				}
			}

			// Try to restore original branch
			_ = r.git.Checkout(ctx, originalBranch)

			return result, err
		}

		if taskResult != nil {
			result.RebasedTasks = append(result.RebasedTasks, *taskResult)
		}
	}

	// Restore original branch
	if err := r.git.Checkout(ctx, originalBranch); err != nil {
		return result, fmt.Errorf("restore original branch %s: %w", originalBranch, err)
	}

	// Save updated states
	if err := r.storage.Save(); err != nil {
		return result, fmt.Errorf("save stacks: %w", err)
	}

	return result, nil
}

// RebaseTask rebases a single task.
func (r *Rebaser) RebaseTask(ctx context.Context, taskID string) (*RebaseResult, error) {
	if err := r.storage.Load(); err != nil {
		return nil, fmt.Errorf("load stacks: %w", err)
	}

	s := r.storage.GetStackByTask(taskID)
	if s == nil {
		return nil, fmt.Errorf("task not in any stack: %s", taskID)
	}

	task := s.GetTask(taskID)
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	if task.State != StateNeedsRebase {
		return nil, fmt.Errorf("task %s does not need rebasing (state: %s)", taskID, task.State)
	}

	// Get current branch to restore later
	originalBranch, err := r.git.CurrentBranch(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current branch: %w", err)
	}

	result := &RebaseResult{
		RebasedTasks:   make([]RebaseTaskResult, 0),
		SkippedTasks:   make([]SkippedTask, 0),
		OriginalBranch: originalBranch,
	}

	taskResult, err := r.rebaseTask(ctx, s, *task)
	if err != nil {
		if errors.Is(err, ErrRebaseConflict) {
			result.FailedTask = &FailedRebase{
				TaskID:       task.ID,
				Branch:       task.Branch,
				OntoBase:     r.getRebaseTarget(s, *task),
				Error:        err,
				IsConflict:   true,
				ConflictHint: "Resolve conflicts manually, then run 'mehr stack rebase --continue'",
			}
		} else {
			result.FailedTask = &FailedRebase{
				TaskID:   task.ID,
				Branch:   task.Branch,
				OntoBase: r.getRebaseTarget(s, *task),
				Error:    err,
			}
		}
		// Try to restore original branch
		_ = r.git.Checkout(ctx, originalBranch)

		return result, err
	}

	if taskResult != nil {
		result.RebasedTasks = append(result.RebasedTasks, *taskResult)
	}

	// Restore original branch
	if err := r.git.Checkout(ctx, originalBranch); err != nil {
		return result, fmt.Errorf("restore original branch %s: %w", originalBranch, err)
	}

	// Save updated states
	if err := r.storage.Save(); err != nil {
		return result, fmt.Errorf("save stacks: %w", err)
	}

	return result, nil
}

// rebaseTask performs the actual rebase for a single task.
// Returns nil result (without error) if task doesn't need rebasing.
func (r *Rebaser) rebaseTask(ctx context.Context, s *Stack, task StackedTask) (*RebaseTaskResult, error) {
	// Skip tasks that don't need rebasing
	if task.State != StateNeedsRebase {
		return nil, nil //nolint:nilnil // Intentional: nil result means task was skipped (not an error)
	}

	// Verify branch exists
	if !r.git.BranchExists(ctx, task.Branch) {
		return nil, fmt.Errorf("branch %s does not exist", task.Branch)
	}

	// Determine rebase target
	target := r.getRebaseTarget(s, task)
	if target == "" {
		return nil, fmt.Errorf("cannot determine rebase target for task %s", task.ID)
	}

	// Switch to task branch
	if err := r.git.Checkout(ctx, task.Branch); err != nil {
		return nil, fmt.Errorf("switch to branch %s: %w", task.Branch, err)
	}

	start := time.Now()

	// Get old base for reporting
	oldBase := task.BaseBranch
	if oldBase == "" && task.DependsOn != "" {
		if parentTask := s.GetTask(task.DependsOn); parentTask != nil {
			oldBase = parentTask.Branch
		}
	}

	// Perform rebase
	if err := r.git.RebaseBranch(ctx, target); err != nil {
		// Abort the rebase to leave clean state
		_ = r.git.AbortRebase(ctx)

		return nil, fmt.Errorf("%w: rebasing %s onto %s: %w", ErrRebaseConflict, task.Branch, target, err)
	}

	duration := time.Since(start)

	// Update task state
	taskPtr := s.GetTask(task.ID)
	if taskPtr != nil {
		taskPtr.State = StateActive
		taskPtr.UpdatedAt = time.Now()
		// Update base branch to reflect new base
		taskPtr.BaseBranch = target
	}

	return &RebaseTaskResult{
		TaskID:   task.ID,
		Branch:   task.Branch,
		OldBase:  oldBase,
		NewBase:  target,
		Rebased:  true,
		Duration: duration,
	}, nil
}

// getTasksInRebaseOrder returns tasks that need rebasing in dependency order.
// Parents are returned before children to ensure valid rebase targets.
func (r *Rebaser) getTasksInRebaseOrder(s *Stack) []StackedTask {
	needsRebase := s.GetTasksNeedingRebase()
	if len(needsRebase) == 0 {
		return nil
	}

	// Build map for quick lookup
	taskMap := make(map[string]StackedTask)
	for _, t := range needsRebase {
		taskMap[t.ID] = t
	}

	// Simple topological sort
	var ordered []StackedTask
	visited := make(map[string]bool)

	var visit func(taskID string)
	visit = func(taskID string) {
		if visited[taskID] {
			return
		}
		visited[taskID] = true

		task, ok := taskMap[taskID]
		if !ok {
			return
		}

		// Visit parent first if it also needs rebasing
		if task.DependsOn != "" {
			if _, parentNeedsRebase := taskMap[task.DependsOn]; parentNeedsRebase {
				visit(task.DependsOn)
			}
		}

		ordered = append(ordered, task)
	}

	for taskID := range taskMap {
		visit(taskID)
	}

	return ordered
}

// getRebaseTarget determines what branch a task should be rebased onto.
func (r *Rebaser) getRebaseTarget(s *Stack, task StackedTask) string {
	// Get the target branch from root task's base branch
	targetBranch := r.getStackTargetBranch(s)

	// If task depends on another task, rebase onto that task's new base
	if task.DependsOn != "" {
		parentTask := s.GetTask(task.DependsOn)
		if parentTask != nil {
			// If parent is merged, rebase onto the target branch (e.g., main)
			if parentTask.State == StateMerged {
				return targetBranch
			}
			// Otherwise rebase onto parent's branch
			return parentTask.Branch
		}
	}

	// Root task rebases onto target branch
	return targetBranch
}

// getStackTargetBranch returns the target branch for the stack.
// This is the base branch of the root task (e.g., "main", "master").
func (r *Rebaser) getStackTargetBranch(s *Stack) string {
	// Find root task and get its base branch
	rootTask := s.GetTask(s.RootTask)
	if rootTask != nil && rootTask.BaseBranch != "" {
		return rootTask.BaseBranch
	}

	// Fallback: find first task with a base branch
	for _, task := range s.Tasks {
		if task.BaseBranch != "" {
			return task.BaseBranch
		}
	}

	// Default fallback
	return "main"
}
