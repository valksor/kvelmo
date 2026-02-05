package conductor

import (
	"context"
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestCheckActiveTaskConflict_NoActiveTask(t *testing.T) {
	c := &Conductor{}

	conflict := c.CheckActiveTaskConflict(context.Background())

	if conflict != nil {
		t.Errorf("expected nil conflict when no active task, got %+v", conflict)
	}
}

func TestCheckActiveTaskConflict_ActiveTaskExists(t *testing.T) {
	c := &Conductor{
		activeTask: &storage.ActiveTask{
			ID:     "test-123",
			Branch: "feature/test-task",
		},
		taskWork: &storage.TaskWork{
			Metadata: storage.WorkMetadata{
				Title: "Test Task Title",
			},
		},
	}

	conflict := c.CheckActiveTaskConflict(context.Background())

	if conflict == nil {
		t.Fatal("expected conflict info when active task exists")
	}
	if conflict.ActiveTaskID != "test-123" {
		t.Errorf("ActiveTaskID = %q, want %q", conflict.ActiveTaskID, "test-123")
	}
	if conflict.ActiveTaskTitle != "Test Task Title" {
		t.Errorf("ActiveTaskTitle = %q, want %q", conflict.ActiveTaskTitle, "Test Task Title")
	}
	if conflict.ActiveBranch != "feature/test-task" {
		t.Errorf("ActiveBranch = %q, want %q", conflict.ActiveBranch, "feature/test-task")
	}
	if conflict.UsingWorktree {
		t.Error("UsingWorktree = true, want false (no worktree path set)")
	}
}

func TestCheckActiveTaskConflict_ActiveTaskWithWorktree(t *testing.T) {
	c := &Conductor{
		activeTask: &storage.ActiveTask{
			ID:           "test-456",
			Branch:       "feature/parallel-task",
			WorktreePath: "/path/to/worktree",
		},
	}

	conflict := c.CheckActiveTaskConflict(context.Background())

	if conflict == nil {
		t.Fatal("expected conflict info")
	}
	if !conflict.UsingWorktree {
		t.Error("UsingWorktree = false, want true (worktree path is set)")
	}
}

func TestCheckActiveTaskConflict_WorktreeModeNoConflict(t *testing.T) {
	c := &Conductor{
		activeTask: &storage.ActiveTask{
			ID: "test-789",
		},
		opts: Options{
			UseWorktree: true, // Worktree mode enabled
		},
	}

	conflict := c.CheckActiveTaskConflict(context.Background())

	if conflict != nil {
		t.Errorf("expected nil conflict when using worktree mode, got %+v", conflict)
	}
}

func TestCheckActiveTaskConflict_NoTaskWorkMetadata(t *testing.T) {
	c := &Conductor{
		activeTask: &storage.ActiveTask{
			ID:     "test-no-work",
			Branch: "feature/orphan",
		},
		// taskWork is nil - no work metadata loaded
	}

	conflict := c.CheckActiveTaskConflict(context.Background())

	if conflict == nil {
		t.Fatal("expected conflict info")
	}
	if conflict.ActiveTaskID != "test-no-work" {
		t.Errorf("ActiveTaskID = %q, want %q", conflict.ActiveTaskID, "test-no-work")
	}
	if conflict.ActiveTaskTitle != "" {
		t.Errorf("ActiveTaskTitle = %q, want empty (no taskWork)", conflict.ActiveTaskTitle)
	}
}
