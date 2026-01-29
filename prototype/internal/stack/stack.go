// Package stack manages stacked features - dependent tasks with git branch relationships.
// This is different from TaskQueue which handles pre-submission planning.
// Stack tracks features already in progress with PRs and branch dependencies.
package stack

import (
	"time"
)

// StackState represents the state of a stacked task.
type StackState string

const (
	// StateActive indicates the task is being actively worked on.
	StateActive StackState = "active"
	// StatePendingReview indicates a PR is open and awaiting review.
	StatePendingReview StackState = "pending-review"
	// StateApproved indicates the PR has been approved.
	StateApproved StackState = "approved"
	// StateMerged indicates the PR has been merged to target branch.
	StateMerged StackState = "merged"
	// StateNeedsRebase indicates the parent was merged and this needs rebasing.
	StateNeedsRebase StackState = "needs-rebase"
	// StateConflict indicates a rebase was attempted but conflicts were found.
	StateConflict StackState = "conflict"
	// StateAbandoned indicates the stack was abandoned (PR closed without merge).
	StateAbandoned StackState = "abandoned"
)

// StackedTask represents a single task within a stack.
type StackedTask struct {
	ID         string     `yaml:"id"`                    // Task reference (e.g., "issue-123", "github:owner/repo#123")
	Branch     string     `yaml:"branch"`                // Git branch name
	State      StackState `yaml:"state"`                 // Current state
	PRNumber   int        `yaml:"pr_number,omitempty"`   // PR/MR number (0 if none)
	PRURL      string     `yaml:"pr_url,omitempty"`      // Full URL to PR/MR
	DependsOn  string     `yaml:"depends_on,omitempty"`  // Parent task ID (linear stack)
	BaseBranch string     `yaml:"base_branch,omitempty"` // Original base branch (main/master)
	MergedAt   *time.Time `yaml:"merged_at,omitempty"`   // When PR was merged
	UpdatedAt  time.Time  `yaml:"updated_at"`            // Last state update
}

// Stack represents a chain of dependent tasks.
type Stack struct {
	ID        string        `yaml:"id"`         // Unique stack identifier
	RootTask  string        `yaml:"root_task"`  // First task in the chain (no dependencies)
	Tasks     []StackedTask `yaml:"tasks"`      // All tasks in dependency order
	CreatedAt time.Time     `yaml:"created_at"` // When stack was created
	UpdatedAt time.Time     `yaml:"updated_at"` // Last modification
}

// NewStack creates a new stack with the given root task.
func NewStack(id, rootTaskID, branch, baseBranch string) *Stack {
	now := time.Now()

	return &Stack{
		ID:       id,
		RootTask: rootTaskID,
		Tasks: []StackedTask{
			{
				ID:         rootTaskID,
				Branch:     branch,
				State:      StateActive,
				BaseBranch: baseBranch,
				UpdatedAt:  now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddTask adds a dependent task to the stack.
func (s *Stack) AddTask(taskID, branch, dependsOn string) {
	now := time.Now()
	s.Tasks = append(s.Tasks, StackedTask{
		ID:        taskID,
		Branch:    branch,
		State:     StateActive,
		DependsOn: dependsOn,
		UpdatedAt: now,
	})
	s.UpdatedAt = now
}

// GetTask returns the task with the given ID, or nil if not found.
func (s *Stack) GetTask(taskID string) *StackedTask {
	for i := range s.Tasks {
		if s.Tasks[i].ID == taskID {
			return &s.Tasks[i]
		}
	}

	return nil
}

// GetChildren returns all tasks that depend on the given task ID.
func (s *Stack) GetChildren(taskID string) []StackedTask {
	var children []StackedTask
	for _, task := range s.Tasks {
		if task.DependsOn == taskID {
			children = append(children, task)
		}
	}

	return children
}

// GetTasksNeedingRebase returns all tasks in needs-rebase state.
func (s *Stack) GetTasksNeedingRebase() []StackedTask {
	var tasks []StackedTask
	for _, task := range s.Tasks {
		if task.State == StateNeedsRebase {
			tasks = append(tasks, task)
		}
	}

	return tasks
}

// MarkChildrenNeedsRebase marks all descendant tasks as needing rebase.
func (s *Stack) MarkChildrenNeedsRebase(parentID string) {
	now := time.Now()
	for i := range s.Tasks {
		if s.Tasks[i].DependsOn == parentID && s.Tasks[i].State != StateMerged {
			s.Tasks[i].State = StateNeedsRebase
			s.Tasks[i].UpdatedAt = now
			// Recursively mark children
			s.MarkChildrenNeedsRebase(s.Tasks[i].ID)
		}
	}
	s.UpdatedAt = now
}

// IsLinear returns true if the stack is a simple linear chain (no branches).
func (s *Stack) IsLinear() bool {
	// Count how many tasks depend on each task
	dependentCount := make(map[string]int)
	for _, task := range s.Tasks {
		if task.DependsOn != "" {
			dependentCount[task.DependsOn]++
		}
	}
	// Linear if no task has more than one dependent
	for _, count := range dependentCount {
		if count > 1 {
			return false
		}
	}

	return true
}

// TaskCount returns the number of tasks in the stack.
func (s *Stack) TaskCount() int {
	return len(s.Tasks)
}
