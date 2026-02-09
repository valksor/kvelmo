package commands

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/vcs"
)

// mockGitCommitAPI implements GitCommitAPI for testing.
type mockGitCommitAPI struct {
	AddCalls    [][]string
	AddErr      error
	CommitCalls []string
	CommitHash  string
	CommitErr   error
	Branch      string
	BranchErr   error
	PushCalls   []struct{ Remote, Branch string }
	PushErr     error
}

func (m *mockGitCommitAPI) Add(ctx context.Context, files ...string) error {
	m.AddCalls = append(m.AddCalls, files)

	return m.AddErr
}

func (m *mockGitCommitAPI) Commit(ctx context.Context, message string) (string, error) {
	m.CommitCalls = append(m.CommitCalls, message)
	if m.CommitErr != nil {
		return "", m.CommitErr
	}
	if m.CommitHash == "" {
		return "abc12345", nil // Default hash
	}

	return m.CommitHash, nil
}

func (m *mockGitCommitAPI) CurrentBranch(ctx context.Context) (string, error) {
	if m.BranchErr != nil {
		return "", m.BranchErr
	}
	if m.Branch == "" {
		return "main", nil
	}

	return m.Branch, nil
}

func (m *mockGitCommitAPI) Push(ctx context.Context, remote, branch string) error {
	m.PushCalls = append(m.PushCalls, struct{ Remote, Branch string }{remote, branch})

	return m.PushErr
}

func TestCommitCommand_HasFlags(t *testing.T) {
	t.Parallel()

	if commitCmd.Flags().Lookup("push") == nil {
		t.Error("commitCmd missing --push flag")
	}
	if commitCmd.Flags().Lookup("all") == nil {
		t.Error("commitCmd missing --all flag")
	}
	if commitCmd.Flags().Lookup("dry-run") == nil {
		t.Error("commitCmd missing --dry-run flag")
	}
	if commitCmd.Flags().Lookup("note") == nil {
		t.Error("commitCmd missing --note flag")
	}
	if commitCmd.Flags().Lookup("agent-commit") == nil {
		t.Error("commitCmd missing --agent-commit flag")
	}
}

func TestAgentAdapter_ImplementsVCSAgent(t *testing.T) {
	t.Parallel()

	// Verify the adapter implements the vcs.Agent interface
	var _ vcs.Agent = (*agentAdapter)(nil)
}

func TestAgentAdapter_Run(t *testing.T) {
	t.Parallel()

	// Verify the adapter type is correctly defined
	// Full integration tests would require mocking the agent
	var _ *agentAdapter
}

// ──────────────────────────────────────────────────────────────────────────────
// executeCommits behavioral tests
// ──────────────────────────────────────────────────────────────────────────────

func TestExecuteCommits_NoChanges(t *testing.T) {
	t.Parallel()

	git := &mockGitCommitAPI{}
	var stdout bytes.Buffer

	err := executeCommits(context.Background(), git, nil, commitOptions{}, &stdout)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "No changes") {
		t.Errorf("expected 'No changes' message, got: %s", stdout.String())
	}
	if len(git.AddCalls) != 0 {
		t.Errorf("Add called %d times, want 0", len(git.AddCalls))
	}
}

func TestExecuteCommits_DryRun(t *testing.T) {
	t.Parallel()

	git := &mockGitCommitAPI{}
	groups := []commitGroup{
		{Files: []string{"file1.go", "file2.go"}, Message: "feat: add feature"},
		{Files: []string{"file3.go"}, Message: "fix: bug fix"},
	}
	var stdout bytes.Buffer

	err := executeCommits(context.Background(), git, groups, commitOptions{dryRun: true}, &stdout)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should NOT have called git operations
	if len(git.AddCalls) != 0 {
		t.Errorf("Add called %d times in dry-run, want 0", len(git.AddCalls))
	}
	if len(git.CommitCalls) != 0 {
		t.Errorf("Commit called %d times in dry-run, want 0", len(git.CommitCalls))
	}
	// Should show preview
	output := stdout.String()
	if !strings.Contains(output, "feat: add feature") {
		t.Errorf("output should contain commit message, got: %s", output)
	}
	if !strings.Contains(output, "dry run") {
		t.Errorf("output should indicate dry run, got: %s", output)
	}
}

func TestExecuteCommits_CreatesCommits(t *testing.T) {
	t.Parallel()

	git := &mockGitCommitAPI{CommitHash: "abc12345def"}
	groups := []commitGroup{
		{Files: []string{"file1.go", "file2.go"}, Message: "feat: add feature"},
		{Files: []string{"file3.go"}, Message: "fix: bug fix"},
	}
	var stdout bytes.Buffer

	err := executeCommits(context.Background(), git, groups, commitOptions{}, &stdout)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should have called Add twice (once per group)
	if len(git.AddCalls) != 2 {
		t.Errorf("Add called %d times, want 2", len(git.AddCalls))
	}
	// Verify first Add call had correct files
	if len(git.AddCalls[0]) != 2 || git.AddCalls[0][0] != "file1.go" {
		t.Errorf("first Add call files = %v, want [file1.go file2.go]", git.AddCalls[0])
	}
	// Should have called Commit twice
	if len(git.CommitCalls) != 2 {
		t.Errorf("Commit called %d times, want 2", len(git.CommitCalls))
	}
	if git.CommitCalls[0] != "feat: add feature" {
		t.Errorf("first Commit message = %q, want %q", git.CommitCalls[0], "feat: add feature")
	}
	// Should NOT have pushed (no --push flag)
	if len(git.PushCalls) != 0 {
		t.Errorf("Push called %d times, want 0", len(git.PushCalls))
	}
}

func TestExecuteCommits_PushesAfterCommit(t *testing.T) {
	t.Parallel()

	git := &mockGitCommitAPI{Branch: "feature-branch"}
	groups := []commitGroup{
		{Files: []string{"file1.go"}, Message: "feat: add feature"},
	}
	var stdout bytes.Buffer

	err := executeCommits(context.Background(), git, groups, commitOptions{push: true}, &stdout)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should have pushed
	if len(git.PushCalls) != 1 {
		t.Fatalf("Push called %d times, want 1", len(git.PushCalls))
	}
	if git.PushCalls[0].Remote != "origin" {
		t.Errorf("Push remote = %q, want %q", git.PushCalls[0].Remote, "origin")
	}
	if git.PushCalls[0].Branch != "feature-branch" {
		t.Errorf("Push branch = %q, want %q", git.PushCalls[0].Branch, "feature-branch")
	}
	if !strings.Contains(stdout.String(), "Pushed") {
		t.Errorf("output should contain 'Pushed', got: %s", stdout.String())
	}
}

func TestExecuteCommits_PropagatesAddError(t *testing.T) {
	t.Parallel()

	addErr := errors.New("permission denied")
	git := &mockGitCommitAPI{AddErr: addErr}
	groups := []commitGroup{
		{Files: []string{"file1.go"}, Message: "feat: add feature"},
	}

	err := executeCommits(context.Background(), git, groups, commitOptions{}, nil)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, addErr) {
		t.Errorf("error = %v, want wrapped %v", err, addErr)
	}
}

func TestExecuteCommits_PropagatesCommitError(t *testing.T) {
	t.Parallel()

	commitErr := errors.New("pre-commit hook failed")
	git := &mockGitCommitAPI{CommitErr: commitErr}
	groups := []commitGroup{
		{Files: []string{"file1.go"}, Message: "feat: add feature"},
	}

	err := executeCommits(context.Background(), git, groups, commitOptions{}, nil)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, commitErr) {
		t.Errorf("error = %v, want wrapped %v", err, commitErr)
	}
}

func TestExecuteCommits_PropagatesPushError(t *testing.T) {
	t.Parallel()

	pushErr := errors.New("remote rejected")
	git := &mockGitCommitAPI{PushErr: pushErr}
	groups := []commitGroup{
		{Files: []string{"file1.go"}, Message: "feat: add feature"},
	}

	err := executeCommits(context.Background(), git, groups, commitOptions{push: true}, nil)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, pushErr) {
		t.Errorf("error = %v, want wrapped %v", err, pushErr)
	}
}

func TestExecuteCommits_PropagatesBranchError(t *testing.T) {
	t.Parallel()

	branchErr := errors.New("not on any branch")
	git := &mockGitCommitAPI{BranchErr: branchErr}
	groups := []commitGroup{
		{Files: []string{"file1.go"}, Message: "feat: add feature"},
	}

	err := executeCommits(context.Background(), git, groups, commitOptions{push: true}, nil)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, branchErr) {
		t.Errorf("error = %v, want wrapped %v", err, branchErr)
	}
}
