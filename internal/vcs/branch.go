package vcs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Branch represents a git branch.
type Branch struct {
	Name      string
	Remote    string // Empty for local branches
	IsCurrent bool
	Commit    string // HEAD commit hash
}

// CreateBranch creates and optionally checks out a new branch.
func (g *Git) CreateBranch(ctx context.Context, name string, base string) error {
	args := []string{"checkout", "-b", name}
	if base != "" {
		args = append(args, base)
	}
	_, err := g.run(ctx, args...)
	if err != nil {
		return fmt.Errorf("create branch %s: %w", name, err)
	}

	return nil
}

// CreateBranchNoCheckout creates a branch without checking it out.
func (g *Git) CreateBranchNoCheckout(ctx context.Context, name string, base string) error {
	args := []string{"branch", name}
	if base != "" {
		args = append(args, base)
	}
	_, err := g.run(ctx, args...)
	if err != nil {
		return fmt.Errorf("create branch %s: %w", name, err)
	}

	return nil
}

// DeleteBranch deletes a branch.
func (g *Git) DeleteBranch(ctx context.Context, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := g.run(ctx, "branch", flag, name)
	if err != nil {
		return fmt.Errorf("delete branch %s: %w", name, err)
	}

	return nil
}

// BranchExists checks if a branch exists.
func (g *Git) BranchExists(ctx context.Context, name string) bool {
	_, err := g.run(ctx, "rev-parse", "--verify", "refs/heads/"+name)

	return err == nil
}

// RemoteBranchExists checks if a remote branch exists.
func (g *Git) RemoteBranchExists(ctx context.Context, remote, name string) bool {
	_, err := g.run(ctx, "rev-parse", "--verify", fmt.Sprintf("refs/remotes/%s/%s", remote, name))

	return err == nil
}

// ListBranches returns all local branches.
func (g *Git) ListBranches(ctx context.Context) ([]Branch, error) {
	out, err := g.run(ctx, "branch", "-v", "--no-abbrev")
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	var branches []Branch
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		b := Branch{}
		if strings.HasPrefix(line, "* ") {
			b.IsCurrent = true
			line = line[2:]
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			b.Name = parts[0]
			b.Commit = parts[1]
		}

		branches = append(branches, b)
	}

	return branches, nil
}

// GetBaseBranch finds the base branch (usually main or master).
func (g *Git) GetBaseBranch(ctx context.Context) (string, error) {
	// 1. Prefer current branch upstream tracking branch.
	currentBranch, err := g.CurrentBranch(ctx)
	if err == nil && currentBranch != "" && currentBranch != "HEAD" {
		if tracking, trackErr := g.GetTrackingBranch(ctx, currentBranch); trackErr == nil {
			tracking = strings.TrimSpace(tracking)
			if tracking != "" {
				if idx := strings.Index(tracking, "/"); idx > 0 && idx < len(tracking)-1 {
					return tracking[idx+1:], nil
				}
			}
		}
	}

	remote, _ := g.GetDefaultRemote(ctx)

	// 2. Try remote HEAD symbolic ref.
	if remote != "" {
		out, refErr := g.run(ctx, "symbolic-ref", "--short", fmt.Sprintf("refs/remotes/%s/HEAD", remote))
		if refErr == nil {
			branch := strings.TrimSpace(out)
			branch = strings.TrimPrefix(branch, remote+"/")
			if branch != "" {
				return branch, nil
			}
		}
	}

	// 3. Try known candidate branch names (remote first, then local).
	candidates := []string{"dev-asc", "dev", "staging", "main", "master", "develop"}
	if remote != "" {
		for _, name := range candidates {
			if g.RemoteBranchExists(ctx, remote, name) {
				return name, nil
			}
		}
	}
	for _, name := range candidates {
		if g.BranchExists(ctx, name) {
			return name, nil
		}
	}

	// 4. Fall back to first local branch.
	branches, err := g.ListBranches(ctx)
	if err != nil {
		return "", err
	}
	if len(branches) > 0 {
		return branches[0].Name, nil
	}

	return "", errors.New("no base branch found")
}

// GetTrackingBranch returns the remote tracking branch for a local branch.
func (g *Git) GetTrackingBranch(ctx context.Context, name string) (string, error) {
	out, err := g.run(ctx, "rev-parse", "--abbrev-ref", name+"@{upstream}")
	if err != nil {
		return "", fmt.Errorf("no tracking branch for %s", name)
	}

	return strings.TrimSpace(out), nil
}

// SetTrackingBranch sets the remote tracking branch.
func (g *Git) SetTrackingBranch(ctx context.Context, local, remote, branch string) error {
	_, err := g.run(ctx, "branch", "-u", fmt.Sprintf("%s/%s", remote, branch), local)

	return err
}

// RenameBranch renames a branch.
func (g *Git) RenameBranch(ctx context.Context, oldName, newName string) error {
	_, err := g.run(ctx, "branch", "-m", oldName, newName)
	if err != nil {
		return fmt.Errorf("rename branch %s to %s: %w", oldName, newName, err)
	}

	return nil
}

// MergeBranch merges a branch into the current branch.
func (g *Git) MergeBranch(ctx context.Context, name string, noFF bool) error {
	args := []string{"merge", name}
	if noFF {
		args = append(args, "--no-ff")
	}
	_, err := g.run(ctx, args...)

	return err
}

// MergeSquash performs a squash merge.
func (g *Git) MergeSquash(ctx context.Context, name string) error {
	_, err := g.run(ctx, "merge", "--squash", name)

	return err
}

// RebaseBranch rebases current branch onto another.
func (g *Git) RebaseBranch(ctx context.Context, onto string) error {
	_, err := g.run(ctx, "rebase", onto)

	return err
}

// AbortRebase aborts an in-progress rebase.
func (g *Git) AbortRebase(ctx context.Context) error {
	_, err := g.run(ctx, "rebase", "--abort")

	return err
}

// ContinueRebase continues a rebase after resolving conflicts.
func (g *Git) ContinueRebase(ctx context.Context) error {
	_, err := g.run(ctx, "rebase", "--continue")

	return err
}

// GetBranchCommitCount returns the number of commits in branch ahead of base.
func (g *Git) GetBranchCommitCount(ctx context.Context, branch, base string) (int, error) {
	out, err := g.run(ctx, "rev-list", "--count", fmt.Sprintf("%s..%s", base, branch))
	if err != nil {
		return 0, err
	}

	var count int
	_, err = fmt.Sscanf(strings.TrimSpace(out), "%d", &count)

	return count, err
}

// GetMergeBase returns the common ancestor of two branches.
func (g *Git) GetMergeBase(ctx context.Context, a, b string) (string, error) {
	out, err := g.run(ctx, "merge-base", a, b)
	if err != nil {
		return "", fmt.Errorf("merge-base %s %s: %w", a, b, err)
	}

	return strings.TrimSpace(out), nil
}

// IsMerged checks if a branch has been merged into another.
func (g *Git) IsMerged(ctx context.Context, branch, into string) (bool, error) {
	mergeBase, err := g.GetMergeBase(ctx, branch, into)
	if err != nil {
		return false, err
	}

	branchHead, err := g.RevParse(ctx, branch)
	if err != nil {
		return false, err
	}

	return mergeBase == branchHead, nil
}

// GetAheadBehind returns commits ahead and behind remote.
func (g *Git) GetAheadBehind(ctx context.Context, local, remote string) (int, int, error) {
	out, err := g.run(ctx, "rev-list", "--left-right", "--count", fmt.Sprintf("%s...%s", local, remote))
	if err != nil {
		return 0, 0, err
	}

	var ahead, behind int
	_, err = fmt.Sscanf(strings.TrimSpace(out), "%d\t%d", &ahead, &behind)

	return ahead, behind, err
}

// PushBranch pushes a branch to remote, optionally setting upstream.
func (g *Git) PushBranch(ctx context.Context, branch, remote string, setUpstream bool) error {
	args := []string{"push", remote, branch}
	if setUpstream {
		args = []string{"push", "-u", remote, branch}
	}
	_, err := g.run(ctx, args...)

	return err
}

// ForcePushBranch force pushes a branch (use with caution).
func (g *Git) ForcePushBranch(ctx context.Context, branch, remote string) error {
	_, err := g.run(ctx, "push", "--force-with-lease", remote, branch)

	return err
}

// GitVersion holds parsed git version information.
type GitVersion struct {
	Major int
	Minor int
	Patch int
	Raw   string
}

// versionRegex matches git version strings like "git version 2.43.0" or "git version 2.43.0.windows.1".
var versionRegex = regexp.MustCompile(`^git version (\d+)\.(\d+)\.(\d+)`)

// GetGitVersion returns the installed git version (cached per Git instance).
// Caching is per-instance to support multi-repo server environments where
// different repositories may have different Git versions available.
func (g *Git) GetGitVersion(ctx context.Context) (*GitVersion, error) {
	g.versionOnce.Do(func() {
		out, err := g.run(ctx, "version")
		if err != nil {
			g.versionErr = fmt.Errorf("get git version: %w", err)

			return
		}

		g.versionCached, g.versionErr = parseGitVersion(strings.TrimSpace(out))
	})

	return g.versionCached, g.versionErr
}

// parseGitVersion parses a git version string like "git version 2.43.0".
func parseGitVersion(raw string) (*GitVersion, error) {
	matches := versionRegex.FindStringSubmatch(raw)
	if matches == nil {
		return nil, fmt.Errorf("cannot parse git version: %q", raw)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return &GitVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
		Raw:   raw,
	}, nil
}

// AtLeast returns true if the git version is at least major.minor.patch.
func (v *GitVersion) AtLeast(major, minor, patch int) bool {
	if v.Major != major {
		return v.Major > major
	}
	if v.Minor != minor {
		return v.Minor > minor
	}

	return v.Patch >= patch
}

// ConflictInfo holds details about potential rebase conflicts.
type ConflictInfo struct {
	HasConflicts      bool     // true if conflicts were detected
	ConflictingFiles  []string // list of files with conflicts
	Unavailable       bool     // true if conflict detection is unavailable (Git too old)
	UnavailableReason string   // reason why detection is unavailable
}

// CheckRebaseConflicts detects if rebasing branch onto target would result in conflicts.
// Uses git merge-tree which doesn't modify the working directory.
// Requires Git 2.38+; returns Unavailable=true on older versions.
func (g *Git) CheckRebaseConflicts(ctx context.Context, branch, onto string) (*ConflictInfo, error) {
	// Check git version (merge-tree --write-tree requires 2.38+)
	version, err := g.GetGitVersion(ctx)
	if err != nil {
		return &ConflictInfo{
			Unavailable:       true,
			UnavailableReason: fmt.Sprintf("cannot determine git version: %v", err),
		}, nil
	}

	if !version.AtLeast(2, 38, 0) {
		return &ConflictInfo{
			Unavailable:       true,
			UnavailableReason: fmt.Sprintf("git %d.%d.%d is too old; merge-tree --write-tree requires Git 2.38+", version.Major, version.Minor, version.Patch),
		}, nil
	}

	// Run git merge-tree --write-tree <onto> <branch>
	// With --write-tree, git calculates the merge-base automatically
	// This simulates merging 'branch' into 'onto'
	// Exit code 0 = no conflicts, exit code 1 = conflicts found
	// Note: We need the output even on non-zero exit, so we use runMergeTree
	out, exitCode, err := g.runMergeTree(ctx, onto, branch)
	if err != nil {
		// Unexpected error (not exit code related)
		return nil, fmt.Errorf("merge-tree: %w", err)
	}

	if exitCode == 0 {
		// No conflicts - output is just the tree hash
		return &ConflictInfo{
			HasConflicts:     false,
			ConflictingFiles: nil,
		}, nil
	}

	// Exit code 1 indicates conflicts - parse the output for details
	// merge-tree outputs "CONFLICT (content): Merge conflict in <file>" markers
	conflictingFiles := parseConflictingFiles(out)

	return &ConflictInfo{
		HasConflicts:     true,
		ConflictingFiles: conflictingFiles,
	}, nil
}

// runMergeTree executes git merge-tree and returns stdout even on non-zero exit.
// Returns (stdout, exitCode, error). Error is only set for actual execution failures.
func (g *Git) runMergeTree(ctx context.Context, onto, branch string) (string, int, error) {
	cmd := exec.CommandContext(ctx, "git", "merge-tree", "--write-tree", onto, branch)
	cmd.Dir = g.repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Get exit code
	exitCode := 0
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			// Actual execution error (not exit code related)
			return "", 0, err
		}
	}

	// Combine stdout and stderr for full output (merge-tree outputs to both)
	output := stdout.String() + stderr.String()

	return output, exitCode, nil
}

// parseConflictingFiles extracts conflicting file paths from git merge-tree output.
func parseConflictingFiles(output string) []string {
	var files []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		// Look for lines like "CONFLICT (content): Merge conflict in path/to/file"
		if strings.HasPrefix(line, "CONFLICT") {
			// Extract file path after "Merge conflict in " or "merge conflict in "
			if idx := strings.Index(strings.ToLower(line), "merge conflict in "); idx != -1 {
				file := strings.TrimSpace(line[idx+len("merge conflict in "):])
				if file != "" && !seen[file] {
					files = append(files, file)
					seen[file] = true
				}
			}
		}
	}

	return files
}
