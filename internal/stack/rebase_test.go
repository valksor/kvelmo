package stack

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/valksor/go-mehrhof/internal/vcs"
)

func TestNewRebaser(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)
	// Can't create real git instance without a repo, so test with nil
	rebaser := NewRebaser(storage, nil)

	if rebaser == nil {
		t.Fatal("expected rebaser, got nil")
	}
	if rebaser.storage != storage {
		t.Error("rebaser storage not set correctly")
	}
}

func TestRebaseResult_Types(t *testing.T) {
	// Test that result types are properly initialized
	result := &RebaseResult{
		RebasedTasks:   make([]RebaseTaskResult, 0),
		SkippedTasks:   make([]SkippedTask, 0),
		OriginalBranch: "main",
	}

	if result.OriginalBranch != "main" {
		t.Errorf("expected original branch 'main', got %s", result.OriginalBranch)
	}
	if len(result.RebasedTasks) != 0 {
		t.Errorf("expected empty rebased tasks, got %d", len(result.RebasedTasks))
	}
}

func TestFailedRebase_Types(t *testing.T) {
	failed := &FailedRebase{
		TaskID:       "task-1",
		Branch:       "feature/task-1",
		OntoBase:     "main",
		Error:        errors.New("conflict"),
		IsConflict:   true,
		ConflictHint: "resolve manually",
	}

	if failed.TaskID != "task-1" {
		t.Errorf("expected task ID 'task-1', got %s", failed.TaskID)
	}
	if !failed.IsConflict {
		t.Error("expected IsConflict to be true")
	}
}

func TestGetTasksInRebaseOrder(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)
	rebaser := NewRebaser(storage, nil)

	// Create a stack with parent-child relationship
	s := NewStack("stack-1", "task-100", "feature/parent", "main")
	s.Tasks[0].State = StateNeedsRebase
	s.AddTask("task-101", "feature/child", "task-100")
	s.Tasks[1].State = StateNeedsRebase

	// Get tasks in rebase order
	ordered := rebaser.getTasksInRebaseOrder(s)

	if len(ordered) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(ordered))
	}

	// Parent should come before child
	if ordered[0].ID != "task-100" {
		t.Errorf("expected first task to be 'task-100', got %s", ordered[0].ID)
	}
	if ordered[1].ID != "task-101" {
		t.Errorf("expected second task to be 'task-101', got %s", ordered[1].ID)
	}
}

func TestGetTasksInRebaseOrder_OnlyChildNeedsRebase(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)
	rebaser := NewRebaser(storage, nil)

	// Create a stack where only child needs rebase
	s := NewStack("stack-1", "task-100", "feature/parent", "main")
	s.Tasks[0].State = StateMerged // Parent is merged
	s.AddTask("task-101", "feature/child", "task-100")
	s.Tasks[1].State = StateNeedsRebase // Only child needs rebase

	ordered := rebaser.getTasksInRebaseOrder(s)

	if len(ordered) != 1 {
		t.Fatalf("expected 1 task, got %d", len(ordered))
	}
	if ordered[0].ID != "task-101" {
		t.Errorf("expected task 'task-101', got %s", ordered[0].ID)
	}
}

func TestGetRebaseTarget(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)
	rebaser := NewRebaser(storage, nil)

	tests := []struct {
		name     string
		setup    func() (*Stack, StackedTask)
		expected string
	}{
		{
			name: "root task rebases onto target branch",
			setup: func() (*Stack, StackedTask) {
				s := NewStack("stack-1", "task-100", "feature/root", "main")

				return s, s.Tasks[0]
			},
			expected: "main",
		},
		{
			name: "child of merged parent rebases onto target branch",
			setup: func() (*Stack, StackedTask) {
				s := NewStack("stack-1", "task-100", "feature/parent", "main")
				s.Tasks[0].State = StateMerged
				s.AddTask("task-101", "feature/child", "task-100")

				return s, s.Tasks[1]
			},
			expected: "main",
		},
		{
			name: "child of active parent rebases onto parent branch",
			setup: func() (*Stack, StackedTask) {
				s := NewStack("stack-1", "task-100", "feature/parent", "main")
				s.Tasks[0].State = StateActive
				s.AddTask("task-101", "feature/child", "task-100")

				return s, s.Tasks[1]
			},
			expected: "feature/parent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, task := tt.setup()
			target := rebaser.getRebaseTarget(s, task)
			if target != tt.expected {
				t.Errorf("expected target %q, got %q", tt.expected, target)
			}
		})
	}
}

func TestRebaseTask_StackNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)
	rebaser := NewRebaser(storage, nil)

	// Save empty storage
	_ = storage.Save()

	_, err := rebaser.RebaseTask(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestRebaseTask_TaskNotNeedingRebase(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)

	// Create a stack with active task (not needing rebase)
	s := NewStack("stack-1", "task-100", "feature/test", "main")
	s.Tasks[0].State = StateActive
	_ = storage.AddStack(s)
	_ = storage.Save()

	rebaser := NewRebaser(storage, nil)

	_, err := rebaser.RebaseTask(context.Background(), "task-100")
	if err == nil {
		t.Error("expected error for task not needing rebase")
	}
}

func TestRebaseAll_EmptyStack(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)

	// Create a stack with no tasks needing rebase
	s := NewStack("stack-1", "task-100", "feature/test", "main")
	s.Tasks[0].State = StateMerged
	_ = storage.AddStack(s)
	_ = storage.Save()

	// Create git repo for testing
	gitDir := initTestGitRepo(t)
	git, err := vcs.New(context.Background(), gitDir)
	if err != nil {
		t.Fatalf("create git: %v", err)
	}

	// Use storage in git directory
	gitStorage := NewStorage(gitDir)
	_ = gitStorage.AddStack(s)
	_ = gitStorage.Save()

	rebaser := NewRebaser(gitStorage, git)

	result, err := rebaser.RebaseAll(context.Background(), "stack-1")
	if err != nil {
		t.Fatalf("RebaseAll: %v", err)
	}

	if len(result.RebasedTasks) != 0 {
		t.Errorf("expected 0 rebased tasks, got %d", len(result.RebasedTasks))
	}
}

func TestRebaseAll_StackNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir)
	_ = storage.Save()

	rebaser := NewRebaser(storage, nil)

	_, err := rebaser.RebaseAll(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent stack")
	}
}

// initTestGitRepo creates a temporary git repository for testing.
func initTestGitRepo(t *testing.T) string {
	t.Helper()

	ctx := context.Background()
	dir := t.TempDir()

	// Initialize git repo
	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	// Configure git user for commits
	cmd = exec.CommandContext(ctx, "git", "config", "user.email", "test@test.com")
	cmd.Dir = dir
	_ = cmd.Run()

	cmd = exec.CommandContext(ctx, "git", "config", "user.name", "Test User")
	cmd.Dir = dir
	_ = cmd.Run()

	// Create initial commit
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cmd = exec.CommandContext(ctx, "git", "add", ".")
	cmd.Dir = dir
	_ = cmd.Run()

	cmd = exec.CommandContext(ctx, "git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	return dir
}

func TestErrRebaseConflict(t *testing.T) {
	// Verify the error can be wrapped and checked
	wrappedErr := fmt.Errorf("rebase failed: %w", ErrRebaseConflict)

	if !errors.Is(wrappedErr, ErrRebaseConflict) {
		t.Error("expected errors.Is to detect ErrRebaseConflict")
	}
}
