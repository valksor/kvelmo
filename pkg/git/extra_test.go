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

func TestCreateTaskWorktree_InjectsGuardrails(t *testing.T) {
	ctx := context.Background()
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	basePath := t.TempDir()

	wt, err := repo.CreateTaskWorktree(ctx, "task-guardrails", basePath)
	if err != nil {
		t.Fatalf("CreateTaskWorktree() error = %v", err)
	}

	// Check that .claude/CLAUDE.md was created
	claudeMdPath := filepath.Join(wt.Path, ".claude", "CLAUDE.md")
	content, err := os.ReadFile(claudeMdPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", claudeMdPath, err)
	}

	// Verify guardrails content
	contentStr := string(content)
	if !contains(contentStr, "Branch Guardrails") {
		t.Error("CLAUDE.md should contain 'Branch Guardrails'")
	}
	if !contains(contentStr, wt.Branch) {
		t.Errorf("CLAUDE.md should contain branch name %q", wt.Branch)
	}
	if !contains(contentStr, "DO NOT run git checkout") {
		t.Error("CLAUDE.md should contain git checkout warning")
	}
}

func TestInjectBranchGuardrails_Idempotent(t *testing.T) {
	// Test that calling injectBranchGuardrails twice doesn't duplicate content
	tmpDir := t.TempDir()
	branch := "kvelmo/test-branch"

	// First injection
	if err := injectBranchGuardrails(tmpDir, branch); err != nil {
		t.Fatalf("first injectBranchGuardrails() error = %v", err)
	}

	claudeMdPath := filepath.Join(tmpDir, ".claude", "CLAUDE.md")
	content1, err := os.ReadFile(claudeMdPath)
	if err != nil {
		t.Fatalf("failed to read first CLAUDE.md: %v", err)
	}

	// Second injection (should be no-op)
	if err := injectBranchGuardrails(tmpDir, branch); err != nil {
		t.Fatalf("second injectBranchGuardrails() error = %v", err)
	}

	content2, err := os.ReadFile(claudeMdPath)
	if err != nil {
		t.Fatalf("failed to read second CLAUDE.md: %v", err)
	}

	// Content should be identical (no duplication)
	if string(content1) != string(content2) {
		t.Errorf("CLAUDE.md content changed after second injection\nfirst len=%d, second len=%d", len(content1), len(content2))
	}

	// Guardrails should appear exactly once
	count := countOccurrences(string(content2), "Branch Guardrails")
	if count != 1 {
		t.Errorf("CLAUDE.md contains %d 'Branch Guardrails', want 1", count)
	}
}

func TestInjectBranchGuardrails_AppendsToExisting(t *testing.T) {
	tmpDir := t.TempDir()
	branch := "kvelmo/test-branch"

	// Create .claude directory with existing content
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}

	existingContent := "# Existing Project Rules\n\nSome existing rules here.\n"
	claudeMdPath := filepath.Join(claudeDir, "CLAUDE.md")
	if err := os.WriteFile(claudeMdPath, []byte(existingContent), 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Inject guardrails
	if err := injectBranchGuardrails(tmpDir, branch); err != nil {
		t.Fatalf("injectBranchGuardrails() error = %v", err)
	}

	content, err := os.ReadFile(claudeMdPath)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	contentStr := string(content)

	// Should contain both existing content and guardrails
	if !contains(contentStr, "Existing Project Rules") {
		t.Error("CLAUDE.md should preserve existing content")
	}
	if !contains(contentStr, "Branch Guardrails") {
		t.Error("CLAUDE.md should contain guardrails")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}

	return count
}
