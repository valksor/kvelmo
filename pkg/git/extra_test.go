package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteBranch(t *testing.T) {
	ctx := context.Background()
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	// Create branch from current
	if err := repo.CreateBranch(ctx, "to-delete"); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	// Switch back to main/master so we can delete to-delete
	mainBranch := "main"
	if err := repo.SwitchBranch(ctx, mainBranch); err != nil {
		// Try master
		mainBranch = "master"
		if err := repo.SwitchBranch(ctx, mainBranch); err != nil {
			t.Skipf("could not switch to main/master: %v", err)
		}
	}

	if err := repo.DeleteBranch(ctx, "to-delete"); err != nil {
		t.Errorf("DeleteBranch() error = %v", err)
	}
}

func TestDeleteBranch_Nonexistent(t *testing.T) {
	ctx := context.Background()
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	err = repo.DeleteBranch(ctx, "nonexistent-branch-xyz")
	if err == nil {
		t.Error("DeleteBranch() on nonexistent branch should return error")
	}
}

func TestStash_StashPop(t *testing.T) {
	ctx := context.Background()
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	// Make a change
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("stashable change"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Stash
	if err := repo.Stash(ctx); err != nil {
		t.Fatalf("Stash() error = %v", err)
	}

	// Verify no uncommitted changes
	hasChanges, err := repo.HasUncommittedChanges(ctx)
	if err != nil {
		t.Fatalf("HasUncommittedChanges after stash: %v", err)
	}
	if hasChanges {
		t.Error("should have no uncommitted changes after Stash()")
	}

	// Pop
	if err := repo.StashPop(ctx); err != nil {
		t.Fatalf("StashPop() error = %v", err)
	}

	// Should have changes back
	hasChanges, err = repo.HasUncommittedChanges(ctx)
	if err != nil {
		t.Fatalf("HasUncommittedChanges after stash pop: %v", err)
	}
	if !hasChanges {
		t.Error("should have uncommitted changes after StashPop()")
	}
}

func TestStashPop_Empty(t *testing.T) {
	ctx := context.Background()
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	// Pop with no stash should return error
	err = repo.StashPop(ctx)
	if err == nil {
		t.Error("StashPop() with empty stash should return error")
	}
}

func TestPushDefault_NoRemote(t *testing.T) {
	ctx := context.Background()
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	// No remote configured → should return error
	err = repo.PushDefault(ctx)
	if err == nil {
		t.Error("PushDefault() with no remote should return error")
	}
}

func TestListWorktrees(t *testing.T) {
	ctx := context.Background()
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	wts, err := repo.ListWorktrees(ctx)
	if err != nil {
		t.Fatalf("ListWorktrees() error = %v", err)
	}
	// Should have at least the main worktree
	if len(wts) == 0 {
		t.Error("ListWorktrees() should return at least one worktree (main)")
	}
	// Main worktree should have the right path.
	// On macOS, /var is a symlink to /private/var, so we must resolve both
	// paths before comparing to avoid mismatches.
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", dir, err)
	}
	found := false
	for _, wt := range wts {
		resolvedWtPath, err := filepath.EvalSymlinks(wt.Path)
		if err != nil {
			// If path doesn't exist or can't be resolved, skip
			continue
		}
		if resolvedWtPath == resolvedDir {
			found = true
		}
	}
	if !found {
		t.Errorf("ListWorktrees() did not include main worktree %q", dir)
	}
}

func TestCreateTaskBranch(t *testing.T) {
	ctx := context.Background()
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	branchName, err := repo.CreateTaskBranch(ctx, "task-123")
	if err != nil {
		t.Fatalf("CreateTaskBranch() error = %v", err)
	}
	if branchName != "kvelmo/task-123" {
		t.Errorf("branchName = %q, want kvelmo/task-123", branchName)
	}
}

func TestCreateTaskBranch_Idempotent(t *testing.T) {
	ctx := context.Background()
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	// Create the branch
	if _, err := repo.CreateTaskBranch(ctx, "task-idem"); err != nil {
		t.Fatalf("first CreateTaskBranch() error = %v", err)
	}

	// Switch back to main/master
	for _, branch := range []string{"main", "master"} {
		if err := repo.SwitchBranch(ctx, branch); err == nil {
			break
		}
	}

	// Create again — should not error (branch exists, just switches)
	if _, err := repo.CreateTaskBranch(ctx, "task-idem"); err != nil {
		t.Fatalf("second CreateTaskBranch() error = %v", err)
	}
}

func TestAddRemoveWorktree(t *testing.T) {
	ctx := context.Background()
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	wtPath := filepath.Join(t.TempDir(), "my-worktree")

	// Add a new worktree with a new branch
	if err := repo.AddWorktree(ctx, wtPath, "wt-branch", true); err != nil {
		t.Fatalf("AddWorktree() error = %v", err)
	}

	// Verify the worktree path exists
	if _, err := os.Stat(wtPath); err != nil {
		t.Errorf("worktree path %q should exist: %v", wtPath, err)
	}

	// Remove the worktree
	if err := repo.RemoveWorktree(ctx, wtPath, false); err != nil {
		t.Fatalf("RemoveWorktree() error = %v", err)
	}
}

func TestCreateTaskWorktree(t *testing.T) {
	ctx := context.Background()
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	basePath := t.TempDir()

	wt, err := repo.CreateTaskWorktree(ctx, "task-wt-1", basePath)
	if err != nil {
		t.Fatalf("CreateTaskWorktree() error = %v", err)
	}
	if wt == nil {
		t.Fatal("CreateTaskWorktree() returned nil worktree")
	}
	if wt.Branch != "kvelmo/task-wt-1" {
		t.Errorf("Branch = %q, want kvelmo/task-wt-1", wt.Branch)
	}
	if _, err := os.Stat(wt.Path); err != nil {
		t.Errorf("worktree path %q should exist: %v", wt.Path, err)
	}
}
