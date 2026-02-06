package vcs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo initializes a git repository for testing.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()

	if err := runGit(ctx, dir, "init"); err != nil {
		t.Skipf("git not available: %v", err)
	}
	if err := runGit(ctx, dir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("git config: %v", err)
	}
	if err := runGit(ctx, dir, "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config: %v", err)
	}

	// Create initial commit
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := runGit(ctx, dir, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGit(ctx, dir, "commit", "-m", "initial"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	return dir
}

func runGit(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE=2020-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2020-01-01T00:00:00Z",
	)

	return cmd.Run()
}

func TestCreateBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = g.CreateBranch(ctx, "feature/test", "")
	if err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	// Should have switched to new branch
	current, err := g.CurrentBranch(ctx)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if current != "feature/test" {
		t.Errorf("CurrentBranch = %q, want %q", current, "feature/test")
	}

	// Branch should exist
	if !g.BranchExists(ctx, "feature/test") {
		t.Error("feature/test branch should exist")
	}
}

func TestCreateBranchWithBase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Get current branch name first
	baseBranch, _ := g.CurrentBranch(ctx)

	err = g.CreateBranch(ctx, "feature/from-base", baseBranch)
	if err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	if !g.BranchExists(ctx, "feature/from-base") {
		t.Error("branch should exist")
	}
}

func TestCreateBranchNoCheckout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	originalBranch, _ := g.CurrentBranch(ctx)

	err = g.CreateBranchNoCheckout(ctx, "feature/no-checkout", "")
	if err != nil {
		t.Fatalf("CreateBranchNoCheckout: %v", err)
	}

	// Should NOT have switched branches
	current, _ := g.CurrentBranch(ctx)
	if current != originalBranch {
		t.Errorf("should still be on %s, but on %s", originalBranch, current)
	}

	// Branch should exist
	if !g.BranchExists(ctx, "feature/no-checkout") {
		t.Error("branch should exist")
	}
}

func TestDeleteBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Create branch first
	if err := g.CreateBranchNoCheckout(ctx, "to-delete", ""); err != nil {
		t.Fatalf("CreateBranchNoCheckout: %v", err)
	}
	if !g.BranchExists(ctx, "to-delete") {
		t.Fatal("branch should exist before delete")
	}

	err = g.DeleteBranch(ctx, "to-delete", false)
	if err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}

	if g.BranchExists(ctx, "to-delete") {
		t.Error("branch should not exist after delete")
	}
}

func TestDeleteBranchForce(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	// Create branch with a commit
	if err := g.CreateBranch(ctx, "force-delete", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := g.Add(ctx, filepath.Join(dir, "new.txt")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := g.Commit(ctx, "new commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Switch back to base
	if err := g.Checkout(ctx, baseBranch); err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	// Force delete should work
	err = g.DeleteBranch(ctx, "force-delete", true)
	if err != nil {
		t.Errorf("force delete should work: %v", err)
	}
}

func TestBranchExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	if !g.BranchExists(ctx, baseBranch) {
		t.Errorf("%s should exist", baseBranch)
	}

	if g.BranchExists(ctx, "nonexistent-branch") {
		t.Error("nonexistent-branch should not exist")
	}
}

func TestRemoteBranchExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// No remote configured, should return false
	if g.RemoteBranchExists(ctx, "origin", "main") {
		t.Error("remote branch should not exist without remote")
	}
}

func TestListBranches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Create some branches
	if err := g.CreateBranchNoCheckout(ctx, "branch-a", ""); err != nil {
		t.Fatalf("CreateBranchNoCheckout: %v", err)
	}
	if err := g.CreateBranchNoCheckout(ctx, "branch-b", ""); err != nil {
		t.Fatalf("CreateBranchNoCheckout: %v", err)
	}

	branches, err := g.ListBranches(ctx)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}

	if len(branches) < 3 {
		t.Errorf("expected at least 3 branches, got %d", len(branches))
	}

	// Check one is marked current
	hasCurrent := false
	for _, b := range branches {
		if b.IsCurrent {
			hasCurrent = true

			break
		}
	}
	if !hasCurrent {
		t.Error("no branch marked as current")
	}
}

func TestGetBaseBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	base, err := g.GetBaseBranch(ctx)
	if err != nil {
		t.Fatalf("GetBaseBranch: %v", err)
	}

	// Should be main or master (depends on git version)
	if base != "main" && base != "master" {
		t.Errorf("base branch = %q, expected main or master", base)
	}
}

func TestGetBaseBranch_UsesTrackingBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)

	remoteDir := filepath.Join(t.TempDir(), "origin.git")
	if err := os.MkdirAll(remoteDir, 0o755); err != nil {
		t.Fatalf("create remote dir: %v", err)
	}
	if err := runGit(ctx, remoteDir, "init", "--bare"); err != nil {
		t.Fatalf("init bare remote: %v", err)
	}

	// Configure remote and publish a staging branch.
	if err := runGit(ctx, dir, "remote", "add", "origin", remoteDir); err != nil {
		t.Fatalf("add remote: %v", err)
	}

	baseBranch, err := runGitCommandContext(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		t.Fatalf("get base branch: %v", err)
	}
	baseBranch = strings.TrimSpace(baseBranch)
	if _, err := runGitCommandContext(ctx, dir, "push", "-u", "origin", baseBranch); err != nil {
		t.Fatalf("push base branch: %v", err)
	}

	if err := runGit(ctx, dir, "checkout", "-b", "staging"); err != nil {
		t.Fatalf("create staging branch: %v", err)
	}
	if _, err := runGitCommandContext(ctx, dir, "push", "-u", "origin", "staging"); err != nil {
		t.Fatalf("push staging branch: %v", err)
	}

	// Work branch tracks origin/staging.
	if err := runGit(ctx, dir, "checkout", "-b", "feature/tracking"); err != nil {
		t.Fatalf("create feature branch: %v", err)
	}
	if err := runGit(ctx, dir, "branch", "--set-upstream-to=origin/staging", "feature/tracking"); err != nil {
		t.Fatalf("set upstream: %v", err)
	}

	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got, err := g.GetBaseBranch(ctx)
	if err != nil {
		t.Fatalf("GetBaseBranch: %v", err)
	}
	if got != "staging" {
		t.Fatalf("GetBaseBranch = %q, want %q", got, "staging")
	}
}

func TestRenameBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := g.CreateBranchNoCheckout(ctx, "old-name", ""); err != nil {
		t.Fatalf("CreateBranchNoCheckout: %v", err)
	}
	if !g.BranchExists(ctx, "old-name") {
		t.Fatal("branch should exist before rename")
	}

	err = g.RenameBranch(ctx, "old-name", "new-name")
	if err != nil {
		t.Fatalf("RenameBranch: %v", err)
	}

	if g.BranchExists(ctx, "old-name") {
		t.Error("old name should not exist")
	}
	if !g.BranchExists(ctx, "new-name") {
		t.Error("new name should exist")
	}
}

func TestGetTrackingBranch_NoRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	_, err = g.GetTrackingBranch(ctx, baseBranch)
	if err == nil {
		t.Error("GetTrackingBranch should fail without remote")
	}
}

func TestCheckout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	if err := g.CreateBranchNoCheckout(ctx, "other", ""); err != nil {
		t.Fatalf("CreateBranchNoCheckout: %v", err)
	}
	err = g.Checkout(ctx, "other")
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	current, _ := g.CurrentBranch(ctx)
	if current != "other" {
		t.Errorf("current = %q, want %q", current, "other")
	}

	// Checkout back
	err = g.Checkout(ctx, baseBranch)
	if err != nil {
		t.Fatalf("Checkout back: %v", err)
	}
}

func TestMergeBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	// Create feature branch with a commit
	if err := g.CreateBranch(ctx, "feature/merge-test", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := g.Add(ctx, filepath.Join(dir, "feature.txt")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := g.Commit(ctx, "feature commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Switch back to base and merge
	if err := g.Checkout(ctx, baseBranch); err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	err = g.MergeBranch(ctx, "feature/merge-test", false) // noFF=false for fast-forward merge
	if err != nil {
		t.Fatalf("MergeBranch: %v", err)
	}

	// Verify file exists after merge
	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); os.IsNotExist(err) {
		t.Error("feature.txt should exist after merge")
	}
}

func TestMergeSquash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	// Create feature branch with multiple commits
	if err := g.CreateBranch(ctx, "feature/squash-test", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("1"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := g.Add(ctx, filepath.Join(dir, "file1.txt")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := g.Commit(ctx, "commit 1"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("2"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := g.Add(ctx, filepath.Join(dir, "file2.txt")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := g.Commit(ctx, "commit 2"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Switch back to base and squash merge
	if err := g.Checkout(ctx, baseBranch); err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	err = g.MergeSquash(ctx, "feature/squash-test")
	if err != nil {
		t.Fatalf("MergeSquash: %v", err)
	}

	// Files should be staged but not committed
	status, _ := g.Status(ctx)
	if len(status) == 0 {
		t.Error("should have staged files after squash")
	}
}

func TestSetTrackingBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	// This should fail without a remote - SetTrackingBranch(local, remote, branch)
	err = g.SetTrackingBranch(ctx, baseBranch, "origin", "main")
	if err == nil {
		t.Error("SetTrackingBranch should fail without remote")
	}
}

func TestGetBranchCommitCount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	// Create branch with additional commits
	if err := g.CreateBranch(ctx, "feature/count-test", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f1.txt"), []byte("1"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := g.Add(ctx, filepath.Join(dir, "f1.txt")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := g.Commit(ctx, "commit 1"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f2.txt"), []byte("2"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := g.Add(ctx, filepath.Join(dir, "f2.txt")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := g.Commit(ctx, "commit 2"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	count, err := g.GetBranchCommitCount(ctx, "feature/count-test", baseBranch)
	if err != nil {
		t.Fatalf("GetBranchCommitCount: %v", err)
	}

	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestGetMergeBase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	// Get the current HEAD commit
	baseCommit, _ := g.RevParse(ctx, "HEAD")

	// Create feature branch
	if err := g.CreateBranch(ctx, "feature/merge-base-test", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("1"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := g.Add(ctx, filepath.Join(dir, "f.txt")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := g.Commit(ctx, "feature commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	mergeBase, err := g.GetMergeBase(ctx, "feature/merge-base-test", baseBranch)
	if err != nil {
		t.Fatalf("GetMergeBase: %v", err)
	}

	// Merge base should be the original base commit
	if mergeBase != baseCommit {
		t.Errorf("mergeBase = %q, want %q", mergeBase, baseCommit)
	}
}

func TestIsMerged(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	// Create and merge a branch
	if err := g.CreateBranch(ctx, "feature/merged", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "merged.txt"), []byte("1"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := g.Add(ctx, filepath.Join(dir, "merged.txt")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := g.Commit(ctx, "merged commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	if err := g.Checkout(ctx, baseBranch); err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	if err := g.MergeBranch(ctx, "feature/merged", false); err != nil {
		t.Fatalf("MergeBranch: %v", err)
	}

	merged, err := g.IsMerged(ctx, "feature/merged", baseBranch)
	if err != nil {
		t.Fatalf("IsMerged: %v", err)
	}

	if !merged {
		t.Error("branch should be merged")
	}

	// Create an unmerged branch
	if err := g.CreateBranchNoCheckout(ctx, "feature/unmerged", ""); err != nil {
		t.Fatalf("CreateBranchNoCheckout: %v", err)
	}
	unmerged, err := g.IsMerged(ctx, "feature/unmerged", baseBranch)
	if err != nil {
		t.Fatalf("IsMerged (unmerged): %v", err)
	}

	// Note: a branch with no additional commits IS considered merged
	// because all its commits are reachable from the target
	if !unmerged {
		t.Log("empty branch is technically merged (all commits reachable)")
	}
}

func TestGetAheadBehind(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	// Create feature branch with commits
	if err := g.CreateBranch(ctx, "feature/ahead-test", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a1.txt"), []byte("1"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := g.Add(ctx, filepath.Join(dir, "a1.txt")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := g.Commit(ctx, "commit 1"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a2.txt"), []byte("2"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := g.Add(ctx, filepath.Join(dir, "a2.txt")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := g.Commit(ctx, "commit 2"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	ahead, behind, err := g.GetAheadBehind(ctx, "feature/ahead-test", baseBranch)
	if err != nil {
		t.Fatalf("GetAheadBehind: %v", err)
	}

	if ahead != 2 {
		t.Errorf("ahead = %d, want 2", ahead)
	}
	if behind != 0 {
		t.Errorf("behind = %d, want 0", behind)
	}
}

func TestPushBranch_NoRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	// Push should fail without a remote
	err = g.PushBranch(ctx, baseBranch, "origin", false)
	if err == nil {
		t.Error("PushBranch should fail without remote")
	}
}

func TestForcePushBranch_NoRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	// Force push should fail without a remote
	err = g.ForcePushBranch(ctx, baseBranch, "origin")
	if err == nil {
		t.Error("ForcePushBranch should fail without remote")
	}
}

func TestParseGitVersion(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantMajor int
		wantMinor int
		wantPatch int
		wantErr   bool
	}{
		{
			name:      "standard version",
			raw:       "git version 2.43.0",
			wantMajor: 2,
			wantMinor: 43,
			wantPatch: 0,
		},
		{
			name:      "windows version",
			raw:       "git version 2.43.0.windows.1",
			wantMajor: 2,
			wantMinor: 43,
			wantPatch: 0,
		},
		{
			name:      "old version",
			raw:       "git version 2.25.1",
			wantMajor: 2,
			wantMinor: 25,
			wantPatch: 1,
		},
		{
			name:      "version 2.38.0 (minimum for merge-tree)",
			raw:       "git version 2.38.0",
			wantMajor: 2,
			wantMinor: 38,
			wantPatch: 0,
		},
		{
			name:      "apple git version",
			raw:       "git version 2.39.3 (Apple Git-145)",
			wantMajor: 2,
			wantMinor: 39,
			wantPatch: 3,
		},
		{
			name:    "invalid format",
			raw:     "not a version",
			wantErr: true,
		},
		{
			name:    "empty string",
			raw:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := parseGitVersion(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseGitVersion(%q) should return error", tt.raw)
				}

				return
			}
			if err != nil {
				t.Fatalf("parseGitVersion(%q) error: %v", tt.raw, err)
			}
			if v.Major != tt.wantMajor || v.Minor != tt.wantMinor || v.Patch != tt.wantPatch {
				t.Errorf("parseGitVersion(%q) = %d.%d.%d, want %d.%d.%d",
					tt.raw, v.Major, v.Minor, v.Patch, tt.wantMajor, tt.wantMinor, tt.wantPatch)
			}
		})
	}
}

func TestGitVersion_AtLeast(t *testing.T) {
	tests := []struct {
		name   string
		v      GitVersion
		major  int
		minor  int
		patch  int
		expect bool
	}{
		{
			name:   "exact match",
			v:      GitVersion{Major: 2, Minor: 38, Patch: 0},
			major:  2,
			minor:  38,
			patch:  0,
			expect: true,
		},
		{
			name:   "higher major",
			v:      GitVersion{Major: 3, Minor: 0, Patch: 0},
			major:  2,
			minor:  38,
			patch:  0,
			expect: true,
		},
		{
			name:   "higher minor",
			v:      GitVersion{Major: 2, Minor: 43, Patch: 0},
			major:  2,
			minor:  38,
			patch:  0,
			expect: true,
		},
		{
			name:   "higher patch",
			v:      GitVersion{Major: 2, Minor: 38, Patch: 5},
			major:  2,
			minor:  38,
			patch:  0,
			expect: true,
		},
		{
			name:   "lower major",
			v:      GitVersion{Major: 1, Minor: 99, Patch: 99},
			major:  2,
			minor:  0,
			patch:  0,
			expect: false,
		},
		{
			name:   "lower minor",
			v:      GitVersion{Major: 2, Minor: 37, Patch: 99},
			major:  2,
			minor:  38,
			patch:  0,
			expect: false,
		},
		{
			name:   "lower patch",
			v:      GitVersion{Major: 2, Minor: 38, Patch: 0},
			major:  2,
			minor:  38,
			patch:  1,
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.v.AtLeast(tt.major, tt.minor, tt.patch)
			if got != tt.expect {
				t.Errorf("GitVersion{%d.%d.%d}.AtLeast(%d, %d, %d) = %v, want %v",
					tt.v.Major, tt.v.Minor, tt.v.Patch, tt.major, tt.minor, tt.patch, got, tt.expect)
			}
		})
	}
}

func TestGetGitVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	v, err := g.GetGitVersion(ctx)
	if err != nil {
		t.Fatalf("GetGitVersion: %v", err)
	}

	// Git should be at least version 2.x
	if v.Major < 2 {
		t.Errorf("Git version %d.%d.%d is too old", v.Major, v.Minor, v.Patch)
	}

	// Version should be cached (same pointer)
	v2, _ := g.GetGitVersion(ctx)
	if v != v2 {
		t.Error("GetGitVersion should return cached version")
	}
}

func TestCheckRebaseConflicts_NoConflicts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Check git version first
	v, err := g.GetGitVersion(ctx)
	if err != nil {
		t.Fatalf("GetGitVersion: %v", err)
	}
	if !v.AtLeast(2, 38, 0) {
		t.Skipf("git %d.%d.%d is too old for merge-tree --write-tree (requires 2.38+)", v.Major, v.Minor, v.Patch)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	// Create a feature branch
	if err := g.CreateBranch(ctx, "feature/no-conflict", baseBranch); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	// Add a new file on feature branch
	if err := os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := runGit(ctx, dir, "add", "feature.txt"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGit(ctx, dir, "commit", "-m", "add feature"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	// Go back to base and add a different file (non-conflicting)
	if err := runGit(ctx, dir, "checkout", baseBranch); err != nil {
		t.Fatalf("git checkout: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := runGit(ctx, dir, "add", "base.txt"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGit(ctx, dir, "commit", "-m", "add base file"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	// Check for conflicts when rebasing feature onto base
	// Both branches have diverged but with different files, so no conflicts
	info, err := g.CheckRebaseConflicts(ctx, "feature/no-conflict", baseBranch)
	if err != nil {
		t.Fatalf("CheckRebaseConflicts: %v", err)
	}

	if info.Unavailable {
		t.Errorf("CheckRebaseConflicts should be available: %s", info.UnavailableReason)
	}
	if info.HasConflicts {
		t.Errorf("CheckRebaseConflicts should not detect conflicts, got: %v", info.ConflictingFiles)
	}
}

func TestCheckRebaseConflicts_WithConflicts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	dir := initTestRepo(t)
	g, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Check git version first
	v, err := g.GetGitVersion(ctx)
	if err != nil {
		t.Fatalf("GetGitVersion: %v", err)
	}
	if !v.AtLeast(2, 38, 0) {
		t.Skipf("git %d.%d.%d is too old for merge-tree --write-tree (requires 2.38+)", v.Major, v.Minor, v.Patch)
	}

	baseBranch, _ := g.CurrentBranch(ctx)

	// Create a feature branch
	if err := g.CreateBranch(ctx, "feature/conflict", baseBranch); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	// Modify README.md on feature branch
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Feature changes\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := runGit(ctx, dir, "add", "README.md"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGit(ctx, dir, "commit", "-m", "modify readme on feature"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	// Go back to base and make conflicting changes
	if err := runGit(ctx, dir, "checkout", baseBranch); err != nil {
		t.Fatalf("git checkout: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Base changes\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := runGit(ctx, dir, "add", "README.md"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGit(ctx, dir, "commit", "-m", "modify readme on base"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	// Check for conflicts when rebasing feature onto base
	info, err := g.CheckRebaseConflicts(ctx, "feature/conflict", baseBranch)
	if err != nil {
		t.Fatalf("CheckRebaseConflicts: %v", err)
	}

	if info.Unavailable {
		t.Errorf("CheckRebaseConflicts should be available: %s", info.UnavailableReason)
	}
	if !info.HasConflicts {
		t.Error("CheckRebaseConflicts should detect conflicts")
	}
	// README.md should be in conflicting files
	found := false
	for _, f := range info.ConflictingFiles {
		if f == "README.md" {
			found = true

			break
		}
	}
	if !found && len(info.ConflictingFiles) == 0 {
		// Some git versions may not output detailed conflict info, just check HasConflicts
		t.Log("No conflicting files listed, but HasConflicts is true")
	}
}

func TestParseConflictingFiles(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []string
	}{
		{
			name:   "single conflict",
			output: "CONFLICT (content): Merge conflict in README.md\n",
			want:   []string{"README.md"},
		},
		{
			name:   "multiple conflicts",
			output: "CONFLICT (content): Merge conflict in file1.go\nCONFLICT (content): Merge conflict in file2.go\n",
			want:   []string{"file1.go", "file2.go"},
		},
		{
			name:   "no conflicts",
			output: "Auto-merging file.go\n",
			want:   nil,
		},
		{
			name:   "empty output",
			output: "",
			want:   nil,
		},
		{
			name:   "duplicate files",
			output: "CONFLICT (content): Merge conflict in file.go\nCONFLICT (modify/delete): Merge conflict in file.go\n",
			want:   []string{"file.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseConflictingFiles(tt.output)
			if len(got) != len(tt.want) {
				t.Errorf("parseConflictingFiles() = %v, want %v", got, tt.want)

				return
			}
			for i, f := range got {
				if f != tt.want[i] {
					t.Errorf("parseConflictingFiles()[%d] = %q, want %q", i, f, tt.want[i])
				}
			}
		})
	}
}
