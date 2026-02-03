// Package vcs provides git operations for the mehrhof task automation tool.
//
// The Git type wraps common git operations needed for task management:
//   - Branch creation and switching
//   - Checkpoint creation for undo/redo functionality
//   - Worktree management for parallel task execution
//   - Status and diff operations
//
// Thread safety:
//   - Git methods are safe for concurrent use as they don't maintain mutable state.
//   - The Git value itself should not be copied after creation.
//
// Usage:
//
//	g, err := vcs.New("/path/to/repo")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	branch, _ := g.CurrentBranch()
package vcs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// Git porcelain v1 format constants
// Format: XY PATH where X=index status, Y=worktree status
// See: https://git-scm.com/docs/git-status#_short_format
const (
	gitStatusIndexPos   = 0 // Position of index (staged) status character
	gitStatusWorkDirPos = 1 // Position of working directory status character
	gitStatusPathStart  = 3 // Position where file path begins (after "XY ")
	gitStatusMinLength  = 4 // Minimum valid entry length (XY + space + at least 1 char)
)

// Git provides git operations for a repository.
type Git struct {
	repoRoot string

	// Version cache (per-instance to support multi-repo server environments)
	versionOnce   sync.Once
	versionCached *GitVersion
	versionErr    error
}

// New creates a Git instance for the given path.
func New(ctx context.Context, path string) (*Git, error) {
	root, err := findRepoRoot(ctx, path)
	if err != nil {
		return nil, err
	}

	return &Git{repoRoot: root}, nil
}

// Root returns the repository root path.
func (g *Git) Root() string {
	return g.repoRoot
}

// IsRepo checks if the path is inside a git repository.
func IsRepo(ctx context.Context, path string) bool {
	_, err := findRepoRoot(ctx, path)

	return err == nil
}

// findRepoRoot locates the git repository root.
func findRepoRoot(ctx context.Context, path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	out, err := runGitCommandContext(ctx, absPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}

	return strings.TrimSpace(out), nil
}

// CurrentBranch returns the current branch name.
func (g *Git) CurrentBranch(ctx context.Context) (string, error) {
	out, err := g.run(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("get current branch: %w", err)
	}

	return strings.TrimSpace(out), nil
}

// Status returns uncommitted changes.
func (g *Git) Status(ctx context.Context) ([]FileStatus, error) {
	out, err := g.run(ctx, "status", "--porcelain", "-z")
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}

	if out == "" {
		return nil, nil
	}

	var files []FileStatus
	entries := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	for _, entry := range entries {
		if len(entry) < gitStatusMinLength {
			continue
		}
		fs := FileStatus{
			Index:   entry[gitStatusIndexPos],
			WorkDir: entry[gitStatusWorkDirPos],
			Path:    strings.TrimSpace(entry[gitStatusPathStart:]),
		}
		files = append(files, fs)
	}

	return files, nil
}

// FileStatus represents a file's git status.
type FileStatus struct {
	Index   byte   // Status in index
	WorkDir byte   // Status in working directory
	Path    string // File path
}

// IsStaged returns true if the file is staged.
func (f FileStatus) IsStaged() bool {
	return f.Index != ' ' && f.Index != '?'
}

// IsModified returns true if the file is modified in working directory.
func (f FileStatus) IsModified() bool {
	return f.WorkDir == 'M' || f.WorkDir == 'D'
}

// HasChanges returns true if there are uncommitted changes.
func (g *Git) HasChanges(ctx context.Context) (bool, error) {
	files, err := g.Status(ctx)
	if err != nil {
		return false, err
	}

	return len(files) > 0, nil
}

// Add stages files for commit.
func (g *Git) Add(ctx context.Context, paths ...string) error {
	args := append([]string{"add"}, paths...)
	_, err := g.run(ctx, args...)
	if err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	return nil
}

// AddAll stages all changes.
func (g *Git) AddAll(ctx context.Context) error {
	return g.Add(ctx, "-A")
}

// CommitOptions configures commit behavior.
type CommitOptions struct {
	AllowEmpty bool // Create commit even with no changes
}

// Commit creates a commit with the given message.
// Optional CommitOptions can be provided to modify behavior.
func (g *Git) Commit(ctx context.Context, message string, opts ...CommitOptions) (string, error) {
	args := []string{"commit"}

	// Apply options if provided
	if len(opts) > 0 && opts[0].AllowEmpty {
		args = append(args, "--allow-empty")
	}

	args = append(args, "-m", message)

	if _, err := g.run(ctx, args...); err != nil {
		return "", fmt.Errorf("git commit: %w", err)
	}

	// Get the commit hash
	out, err := g.run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("get commit hash: %w", err)
	}

	return strings.TrimSpace(out), nil
}

// Checkout switches to a branch.
func (g *Git) Checkout(ctx context.Context, ref string) error {
	_, err := g.run(ctx, "checkout", ref)
	if err != nil {
		return fmt.Errorf("git checkout %s: %w", ref, err)
	}

	return nil
}

// Diff returns diff output.
func (g *Git) Diff(ctx context.Context, args ...string) (string, error) {
	cmdArgs := append([]string{"diff"}, args...)

	return g.run(ctx, cmdArgs...)
}

// Log returns commit logs.
func (g *Git) Log(ctx context.Context, args ...string) (string, error) {
	cmdArgs := append([]string{"log"}, args...)

	return g.run(ctx, cmdArgs...)
}

// RevParse resolves a git reference.
func (g *Git) RevParse(ctx context.Context, ref string) (string, error) {
	out, err := g.run(ctx, "rev-parse", ref)
	if err != nil {
		return "", fmt.Errorf("rev-parse %s: %w", ref, err)
	}

	return strings.TrimSpace(out), nil
}

// GetCommitMessage returns the message for a commit.
func (g *Git) GetCommitMessage(ctx context.Context, ref string) (string, error) {
	out, err := g.run(ctx, "log", "-1", "--format=%B", ref)
	if err != nil {
		return "", fmt.Errorf("get commit message: %w", err)
	}

	return strings.TrimSpace(out), nil
}

// GetCommitAuthor returns the author of a commit.
func (g *Git) GetCommitAuthor(ctx context.Context, ref string) (string, error) {
	out, err := g.run(ctx, "log", "-1", "--format=%an <%ae>", ref)
	if err != nil {
		return "", fmt.Errorf("get commit author: %w", err)
	}

	return strings.TrimSpace(out), nil
}

// ResetHard resets to a ref, discarding all changes.
func (g *Git) ResetHard(ctx context.Context, ref string) error {
	_, err := g.run(ctx, "reset", "--hard", ref)
	if err != nil {
		return fmt.Errorf("reset hard to %s: %w", ref, err)
	}

	return nil
}

// ResetSoft resets to a ref, keeping changes staged.
func (g *Git) ResetSoft(ctx context.Context, ref string) error {
	_, err := g.run(ctx, "reset", "--soft", ref)
	if err != nil {
		return fmt.Errorf("reset soft to %s: %w", ref, err)
	}

	return nil
}

// Clean removes untracked files.
func (g *Git) Clean(ctx context.Context, force bool) error {
	args := []string{"clean", "-d"}
	if force {
		args = append(args, "-f")
	}
	_, err := g.run(ctx, args...)

	return err
}

// Stash saves changes to stash (including untracked files).
func (g *Git) Stash(ctx context.Context, message string) error {
	args := []string{"stash", "push", "-u"}
	if message != "" {
		args = append(args, "-m", message)
	}
	_, err := g.run(ctx, args...)
	if err != nil {
		return fmt.Errorf("git stash: %w", err)
	}

	return nil
}

// StashPop applies and removes the top stash entry.
func (g *Git) StashPop(ctx context.Context) error {
	_, err := g.run(ctx, "stash", "pop")
	if err != nil {
		return fmt.Errorf("git stash pop: %w", err)
	}

	return nil
}

// StashList returns all stash entries.
func (g *Git) StashList(ctx context.Context) ([]string, error) {
	output, err := g.run(ctx, "stash", "list")
	if err != nil {
		return nil, fmt.Errorf("git stash list: %w", err)
	}

	// Parse output - each line is a stash entry
	// Format: stash@{0}: message
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}

	return lines, nil
}

// run executes a git command in the repo root with context.
func (g *Git) run(ctx context.Context, args ...string) (string, error) {
	return runGitCommandContext(ctx, g.repoRoot, args...)
}

// runGitCommandContext executes a git command with context.
func runGitCommandContext(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}

		return "", fmt.Errorf("%s", strings.TrimSpace(errMsg))
	}

	return stdout.String(), nil
}

// Config represents git configuration.
type Config struct {
	Key   string
	Value string
}

// GetConfig reads a git config value.
func (g *Git) GetConfig(ctx context.Context, key string) (string, error) {
	out, err := g.run(ctx, "config", "--get", key)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

// SetConfig sets a git config value.
func (g *Git) SetConfig(ctx context.Context, key, value string) error {
	_, err := g.run(ctx, "config", key, value)

	return err
}

// RemoteURL returns the URL for a remote.
func (g *Git) RemoteURL(ctx context.Context, name string) (string, error) {
	out, err := g.run(ctx, "remote", "get-url", name)
	if err != nil {
		return "", fmt.Errorf("get remote URL %s: %w", name, err)
	}

	return strings.TrimSpace(out), nil
}

// GetDefaultRemote returns the default remote for the repository.
// It checks in order: current branch's tracking remote, "origin" if it exists,
// then falls back to the first available remote.
func (g *Git) GetDefaultRemote(ctx context.Context) (string, error) {
	// Try to get the remote for the current branch's tracking configuration
	currentBranch, err := g.CurrentBranch(ctx)
	if err == nil && currentBranch != "" && currentBranch != "HEAD" {
		remote, err := g.run(ctx, "config", "--get", "branch."+currentBranch+".remote")
		if err == nil {
			remote = strings.TrimSpace(remote)
			if remote != "" {
				return remote, nil
			}
		}
	}

	// Check if "origin" remote exists (most common case)
	if _, err := g.run(ctx, "remote", "get-url", "origin"); err == nil {
		return "origin", nil
	}

	// Fall back to the first available remote
	out, err := g.run(ctx, "remote")
	if err != nil {
		return "", fmt.Errorf("list remotes: %w", err)
	}

	remotes := strings.Fields(out)
	if len(remotes) > 0 {
		return remotes[0], nil
	}

	return "", errors.New("no remote configured")
}

// Fetch fetches from a remote.
func (g *Git) Fetch(ctx context.Context, remote string, args ...string) error {
	cmdArgs := append([]string{"fetch", remote}, args...)
	_, err := g.run(ctx, cmdArgs...)

	return err
}

// Pull pulls from a remote.
func (g *Git) Pull(ctx context.Context, remote, branch string) error {
	_, err := g.run(ctx, "pull", remote, branch)

	return err
}

// Push pushes to a remote.
func (g *Git) Push(ctx context.Context, remote, branch string, args ...string) error {
	cmdArgs := append([]string{"push", remote, branch}, args...)
	_, err := g.run(ctx, cmdArgs...)

	return err
}

// IsWorktree returns true if the current repository is a git worktree
// (as opposed to the main repository). Worktrees have a .git file
// pointing to the main repo, while main repos have a .git directory.
func (g *Git) IsWorktree() bool {
	gitPath := filepath.Join(g.repoRoot, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	// Worktrees have .git as a file, main repos have it as a directory
	return !info.IsDir()
}

// GetMainWorktreePath returns the path to the main repository when called
// from within a worktree. Returns an error if not in a worktree.
func (g *Git) GetMainWorktreePath(ctx context.Context) (string, error) {
	if !g.IsWorktree() {
		return "", errors.New("not in a worktree")
	}

	// git rev-parse --git-common-dir returns the shared .git directory
	// e.g., /path/to/main-repo/.git
	out, err := g.run(ctx, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", fmt.Errorf("get git common dir: %w", err)
	}

	gitCommonDir := strings.TrimSpace(out)

	// Handle relative paths (git may return relative path)
	if !filepath.IsAbs(gitCommonDir) {
		gitCommonDir = filepath.Join(g.repoRoot, gitCommonDir)
	}

	// Clean up the path (resolve .., symlinks, etc.)
	gitCommonDir, err = filepath.Abs(gitCommonDir)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}

	// The main repo root is the parent of the .git directory
	mainRepoRoot := filepath.Dir(gitCommonDir)

	return mainRepoRoot, nil
}

// DiffUncommitted returns the diff of all uncommitted changes (both staged and unstaged).
// context is the number of context lines (default 3 if 0).
func (g *Git) DiffUncommitted(ctx context.Context, contextLines int) (string, error) {
	if contextLines == 0 {
		contextLines = 3
	}

	// Get staged changes
	staged, err := g.run(ctx, "diff", "--cached", fmt.Sprintf("-U%d", contextLines))
	if err != nil {
		return "", fmt.Errorf("diff staged: %w", err)
	}

	// Get unstaged changes
	unstaged, err := g.run(ctx, "diff", fmt.Sprintf("-U%d", contextLines))
	if err != nil {
		return "", fmt.Errorf("diff unstaged: %w", err)
	}

	// Combine both diffs
	var result strings.Builder
	if staged != "" {
		result.WriteString("# Staged changes\n")
		result.WriteString(staged)
	}
	if unstaged != "" {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString("# Unstaged changes\n")
		result.WriteString(unstaged)
	}

	return result.String(), nil
}

// DiffBranch returns the diff between the current branch and a base branch.
// If baseBranch is empty, it uses the detected default branch.
// context is the number of context lines (default 3 if 0).
func (g *Git) DiffBranch(ctx context.Context, baseBranch string, contextLines int) (string, error) {
	if contextLines == 0 {
		contextLines = 3
	}

	// Detect base branch if not provided
	if baseBranch == "" {
		var err error
		baseBranch, err = g.DetectDefaultBranch(ctx)
		if err != nil {
			return "", fmt.Errorf("detect default branch: %w", err)
		}
	}

	// Get current branch
	currentBranch, err := g.CurrentBranch(ctx)
	if err != nil {
		return "", fmt.Errorf("get current branch: %w", err)
	}

	// Use three-dot diff to show only changes in current branch since divergence
	out, err := g.run(ctx, "diff", fmt.Sprintf("-U%d", contextLines), baseBranch+"..."+currentBranch)
	if err != nil {
		return "", fmt.Errorf("diff branch %s...%s: %w", baseBranch, currentBranch, err)
	}

	return out, nil
}

// DiffRange returns the diff for a commit range (e.g., "HEAD~3..HEAD").
// context is the number of context lines (default 3 if 0).
func (g *Git) DiffRange(ctx context.Context, rangeSpec string, contextLines int) (string, error) {
	if contextLines == 0 {
		contextLines = 3
	}

	out, err := g.run(ctx, "diff", fmt.Sprintf("-U%d", contextLines), rangeSpec)
	if err != nil {
		return "", fmt.Errorf("diff range %s: %w", rangeSpec, err)
	}

	return out, nil
}

// DiffFiles returns the diff for specific files.
// context is the number of context lines (default 3 if 0).
func (g *Git) DiffFiles(ctx context.Context, files []string, contextLines int) (string, error) {
	if contextLines == 0 {
		contextLines = 3
	}

	args := []string{"diff", fmt.Sprintf("-U%d", contextLines), "--"}
	args = append(args, files...)

	out, err := g.run(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("diff files: %w", err)
	}

	return out, nil
}

// DetectDefaultBranch detects the repository's default branch.
// Detection order:
//  1. Try to get from remote HEAD symbolic ref (origin/HEAD)
//  2. Check if common branch names exist (main, master, develop)
//  3. Fall back to first branch
func (g *Git) DetectDefaultBranch(ctx context.Context) (string, error) {
	// Try to get from origin/HEAD
	out, err := g.run(ctx, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err == nil {
		branch := strings.TrimSpace(out)
		// Remove "origin/" prefix if present
		branch = strings.TrimPrefix(branch, "origin/")
		if branch != "" {
			return branch, nil
		}
	}

	// Fall back to checking common branch names
	candidates := []string{"main", "master", "develop"}
	for _, name := range candidates {
		if g.BranchExists(ctx, name) {
			return name, nil
		}
	}

	// Try remote branches
	remote, _ := g.GetDefaultRemote(ctx)
	if remote != "" {
		for _, name := range candidates {
			if g.RemoteBranchExists(ctx, remote, name) {
				return name, nil
			}
		}
	}

	// Fall back to first branch
	branches, err := g.ListBranches(ctx)
	if err != nil {
		return "", fmt.Errorf("list branches: %w", err)
	}
	if len(branches) > 0 {
		return branches[0].Name, nil
	}

	return "", errors.New("no default branch found")
}

// RepoInfo contains detected information about the repository.
// Used to provide context to AI for commit grouping (GENERIC - works in any repo).
type RepoInfo struct {
	Language   string   // "go", "python", "javascript", etc.
	Frameworks []string // ["react", "nextjs"], ["django"], etc.
	RootDirs   []string // Top-level directories present
	BuildFiles []string // "package.json", "go.mod", "requirements.txt", etc.
}

// GetRepoInfo detects repository information by examining files.
// This is GENERIC - works in ANY repository type.
func (g *Git) GetRepoInfo(ctx context.Context) (RepoInfo, error) {
	var info RepoInfo

	// Check for build/config files to detect language/framework
	buildFiles := []string{
		"go.mod", "go.sum",
		"package.json", "yarn.lock", "pnpm-lock.yaml", "package-lock.json",
		"requirements.txt", "pyproject.toml", "setup.py", "Pipfile",
		"Cargo.toml", "Cargo.lock",
		"pom.xml", "build.gradle", "build.gradle.kts",
		"Gemfile", "Gemfile.lock",
		"composer.json",
		"mix.exs",
	}

	for _, file := range buildFiles {
		fullPath := filepath.Join(g.repoRoot, file)
		if _, err := os.Stat(fullPath); err == nil {
			info.BuildFiles = append(info.BuildFiles, file)

			// Detect language from build files
			switch file {
			case "go.mod", "go.sum":
				info.Language = "go"
			case "package.json", "yarn.lock", "pnpm-lock.yaml", "package-lock.json":
				info.Language = "javascript"
				// Could detect frameworks by reading package.json
			case "requirements.txt", "pyproject.toml", "setup.py", "Pipfile":
				info.Language = "python"
			case "Cargo.toml", "Cargo.lock":
				info.Language = "rust"
			case "pom.xml", "build.gradle", "build.gradle.kts":
				info.Language = "java"
			case "Gemfile", "Gemfile.lock":
				info.Language = "ruby"
			case "composer.json":
				info.Language = "php"
			case "mix.exs":
				info.Language = "elixir"
			}
		}
	}

	// Get root directories (non-hidden, top-level)
	entries, err := os.ReadDir(g.repoRoot)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && e.Name() != "node_modules" {
				info.RootDirs = append(info.RootDirs, e.Name())
			}
		}
	}

	return info, nil
}

// HashChangedFiles creates a hash of the current changed files.
// Used for detecting if files changed between dry-run and commit.
func (g *Git) HashChangedFiles(ctx context.Context, includeUnstaged bool) (string, error) {
	status, err := g.Status(ctx)
	if err != nil {
		return "", err
	}

	var files []string
	for _, f := range status {
		if includeUnstaged {
			if f.Index != ' ' || f.WorkDir != ' ' {
				files = append(files, f.Path)
			}
		} else {
			if f.Index != ' ' && f.Index != '?' {
				files = append(files, f.Path)
			}
		}
	}

	slices.Sort(files)
	h := sha256.Sum256([]byte(strings.Join(files, "|")))

	return hex.EncodeToString(h[:]), nil
}
