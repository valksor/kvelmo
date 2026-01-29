package stack

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStack(t *testing.T) {
	stack := NewStack("stack-1", "issue-100", "feature/auth", "main")

	if stack.ID != "stack-1" {
		t.Errorf("expected ID 'stack-1', got '%s'", stack.ID)
	}
	if stack.RootTask != "issue-100" {
		t.Errorf("expected RootTask 'issue-100', got '%s'", stack.RootTask)
	}
	if len(stack.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(stack.Tasks))
	}
	if stack.Tasks[0].State != StateActive {
		t.Errorf("expected state 'active', got '%s'", stack.Tasks[0].State)
	}
	if stack.Tasks[0].BaseBranch != "main" {
		t.Errorf("expected base branch 'main', got '%s'", stack.Tasks[0].BaseBranch)
	}
}

func TestStack_AddTask(t *testing.T) {
	stack := NewStack("stack-1", "issue-100", "feature/auth", "main")
	stack.AddTask("issue-101", "feature/oauth", "issue-100")

	if len(stack.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(stack.Tasks))
	}
	if stack.Tasks[1].DependsOn != "issue-100" {
		t.Errorf("expected depends_on 'issue-100', got '%s'", stack.Tasks[1].DependsOn)
	}
}

func TestStack_GetTask(t *testing.T) {
	stack := NewStack("stack-1", "issue-100", "feature/auth", "main")
	stack.AddTask("issue-101", "feature/oauth", "issue-100")

	tests := []struct {
		name    string
		taskID  string
		wantNil bool
	}{
		{"existing task", "issue-100", false},
		{"second task", "issue-101", false},
		{"non-existent", "issue-999", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := stack.GetTask(tt.taskID)
			if tt.wantNil && task != nil {
				t.Errorf("expected nil, got task %+v", task)
			}
			if !tt.wantNil && task == nil {
				t.Error("expected task, got nil")
			}
		})
	}
}

func TestStack_GetChildren(t *testing.T) {
	stack := NewStack("stack-1", "issue-100", "feature/auth", "main")
	stack.AddTask("issue-101", "feature/oauth", "issue-100")
	stack.AddTask("issue-102", "feature/oauth-google", "issue-101")

	children := stack.GetChildren("issue-100")
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if children[0].ID != "issue-101" {
		t.Errorf("expected child 'issue-101', got '%s'", children[0].ID)
	}
}

func TestStack_MarkChildrenNeedsRebase(t *testing.T) {
	stack := NewStack("stack-1", "issue-100", "feature/auth", "main")
	stack.AddTask("issue-101", "feature/oauth", "issue-100")
	stack.AddTask("issue-102", "feature/oauth-google", "issue-101")

	stack.MarkChildrenNeedsRebase("issue-100")

	task101 := stack.GetTask("issue-101")
	if task101.State != StateNeedsRebase {
		t.Errorf("expected task-101 state 'needs-rebase', got '%s'", task101.State)
	}

	task102 := stack.GetTask("issue-102")
	if task102.State != StateNeedsRebase {
		t.Errorf("expected task-102 state 'needs-rebase', got '%s'", task102.State)
	}
}

func TestStack_IsLinear(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Stack
		expected bool
	}{
		{
			name: "linear chain",
			setup: func() *Stack {
				s := NewStack("s1", "t1", "b1", "main")
				s.AddTask("t2", "b2", "t1")
				s.AddTask("t3", "b3", "t2")

				return s
			},
			expected: true,
		},
		{
			name: "branched (two children)",
			setup: func() *Stack {
				s := NewStack("s1", "t1", "b1", "main")
				s.AddTask("t2", "b2", "t1")
				s.AddTask("t3", "b3", "t1") // Both t2 and t3 depend on t1

				return s
			},
			expected: false,
		},
		{
			name: "single task",
			setup: func() *Stack {
				return NewStack("s1", "t1", "b1", "main")
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := tt.setup()
			if got := stack.IsLinear(); got != tt.expected {
				t.Errorf("IsLinear() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStorage_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)

	// Load empty storage
	if err := storage.Load(); err != nil {
		t.Fatalf("Load empty: %v", err)
	}
	if storage.StackCount() != 0 {
		t.Errorf("expected 0 stacks, got %d", storage.StackCount())
	}

	// Add a stack
	stack := NewStack("stack-1", "issue-100", "feature/auth", "main")
	if err := storage.AddStack(stack); err != nil {
		t.Fatalf("AddStack: %v", err)
	}

	// Save
	if err := storage.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, StacksDir, StacksFile)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("stacks file not created: %v", err)
	}

	// Load in new storage instance
	storage2 := NewStorage(tmpDir)
	if err := storage2.Load(); err != nil {
		t.Fatalf("Load after save: %v", err)
	}
	if storage2.StackCount() != 1 {
		t.Errorf("expected 1 stack after reload, got %d", storage2.StackCount())
	}
}

func TestStorage_GetStackByTask(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)

	stack := NewStack("stack-1", "issue-100", "feature/auth", "main")
	stack.AddTask("issue-101", "feature/oauth", "issue-100")
	_ = storage.AddStack(stack)

	tests := []struct {
		name    string
		taskID  string
		wantNil bool
		wantID  string
	}{
		{"root task", "issue-100", false, "stack-1"},
		{"child task", "issue-101", false, "stack-1"},
		{"unknown task", "issue-999", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := storage.GetStackByTask(tt.taskID)
			if tt.wantNil && got != nil {
				t.Errorf("expected nil, got stack %s", got.ID)
			}
			if !tt.wantNil {
				if got == nil {
					t.Error("expected stack, got nil")
				} else if got.ID != tt.wantID {
					t.Errorf("expected stack %s, got %s", tt.wantID, got.ID)
				}
			}
		})
	}
}

func TestStorage_UpdateStack(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)

	stack := NewStack("stack-1", "issue-100", "feature/auth", "main")
	_ = storage.AddStack(stack)

	err := storage.UpdateStack("stack-1", func(s *Stack) {
		task := s.GetTask("issue-100")
		task.State = StatePendingReview
		task.PRNumber = 123
	})
	if err != nil {
		t.Fatalf("UpdateStack: %v", err)
	}

	updated := storage.GetStack("stack-1")
	task := updated.GetTask("issue-100")
	if task.State != StatePendingReview {
		t.Errorf("expected state 'pending-review', got '%s'", task.State)
	}
	if task.PRNumber != 123 {
		t.Errorf("expected PR 123, got %d", task.PRNumber)
	}
}

func TestStorage_DuplicateStack(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)

	stack1 := NewStack("stack-1", "issue-100", "feature/auth", "main")
	stack2 := NewStack("stack-1", "issue-200", "feature/other", "main")

	if err := storage.AddStack(stack1); err != nil {
		t.Fatalf("AddStack first: %v", err)
	}

	err := storage.AddStack(stack2)
	if err == nil {
		t.Error("expected error for duplicate stack, got nil")
	}
}

func TestStackedTask_Times(t *testing.T) {
	now := time.Now()
	task := StackedTask{
		ID:        "issue-100",
		Branch:    "feature/test",
		State:     StateMerged,
		MergedAt:  &now,
		UpdatedAt: now,
	}

	if task.MergedAt == nil {
		t.Error("expected MergedAt to be set")
	}
	if task.MergedAt.IsZero() {
		t.Error("expected non-zero MergedAt")
	}
}

func TestStorage_RemoveStack(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)

	stack := NewStack("stack-1", "issue-100", "feature/auth", "main")
	_ = storage.AddStack(stack)

	if storage.StackCount() != 1 {
		t.Errorf("expected 1 stack, got %d", storage.StackCount())
	}

	removed := storage.RemoveStack("stack-1")
	if !removed {
		t.Error("expected RemoveStack to return true")
	}
	if storage.StackCount() != 0 {
		t.Errorf("expected 0 stacks after removal, got %d", storage.StackCount())
	}

	// Try removing non-existent
	removed = storage.RemoveStack("stack-999")
	if removed {
		t.Error("expected RemoveStack to return false for non-existent")
	}
}

func TestStorage_ListStacks(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)

	stack1 := NewStack("stack-1", "issue-100", "feature/auth", "main")
	stack2 := NewStack("stack-2", "issue-200", "feature/other", "main")
	_ = storage.AddStack(stack1)
	_ = storage.AddStack(stack2)

	stacks := storage.ListStacks()
	if len(stacks) != 2 {
		t.Errorf("expected 2 stacks, got %d", len(stacks))
	}
}

func TestStack_GetTasksNeedingRebase(t *testing.T) {
	stack := NewStack("stack-1", "issue-100", "feature/auth", "main")
	stack.AddTask("issue-101", "feature/oauth", "issue-100")
	stack.AddTask("issue-102", "feature/oauth-google", "issue-101")

	// Initially no tasks need rebase
	tasks := stack.GetTasksNeedingRebase()
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks needing rebase, got %d", len(tasks))
	}

	// Mark root as merged, children need rebase
	root := stack.GetTask("issue-100")
	root.State = StateMerged
	stack.MarkChildrenNeedsRebase("issue-100")

	tasks = stack.GetTasksNeedingRebase()
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks needing rebase, got %d", len(tasks))
	}
}

func TestStack_TaskCount(t *testing.T) {
	stack := NewStack("stack-1", "issue-100", "feature/auth", "main")
	if stack.TaskCount() != 1 {
		t.Errorf("expected 1 task, got %d", stack.TaskCount())
	}

	stack.AddTask("issue-101", "feature/oauth", "issue-100")
	if stack.TaskCount() != 2 {
		t.Errorf("expected 2 tasks, got %d", stack.TaskCount())
	}
}

func TestStorage_UpdateStackNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)

	err := storage.UpdateStack("nonexistent", func(s *Stack) {})
	if err == nil {
		t.Error("expected error for non-existent stack")
	}
}
