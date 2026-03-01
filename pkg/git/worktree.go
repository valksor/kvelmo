package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Worktree struct {
	Path   string
	Branch string
	Bare   bool
}

func (r *Repository) ListWorktrees(ctx context.Context) ([]Worktree, error) {
	out, err := r.run(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var worktrees []Worktree
	var current Worktree

	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = Worktree{}
			}

			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		} else if line == "bare" {
			current.Bare = true
		}
	}

	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

func (r *Repository) AddWorktree(ctx context.Context, path, branch string, create bool) error {
	args := []string{"worktree", "add"}
	if create {
		args = append(args, "-b", branch)
	}
	args = append(args, path)
	if !create {
		args = append(args, branch)
	}

	_, err := r.run(ctx, args...)

	return err
}

func (r *Repository) RemoveWorktree(ctx context.Context, path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)

	_, err := r.run(ctx, args...)

	return err
}

func (r *Repository) CreateTaskBranch(ctx context.Context, taskID string) (string, error) {
	branchName := "kvelmo/" + taskID

	// Check if branch exists
	_, err := r.run(ctx, "rev-parse", "--verify", branchName)
	if err == nil {
		// Branch exists, switch to it
		return branchName, r.SwitchBranch(ctx, branchName)
	}

	// Create new branch
	return branchName, r.CreateBranch(ctx, branchName)
}

func (r *Repository) CreateTaskWorktree(ctx context.Context, taskID, basePath string) (*Worktree, error) {
	branchName := "kvelmo/" + taskID
	worktreePath := filepath.Join(basePath, taskID)

	// Remove existing worktree if it exists
	if _, err := os.Stat(worktreePath); err == nil {
		_ = r.RemoveWorktree(ctx, worktreePath, true)
	}

	if err := r.AddWorktree(ctx, worktreePath, branchName, true); err != nil {
		return nil, fmt.Errorf("add worktree: %w", err)
	}

	return &Worktree{
		Path:   worktreePath,
		Branch: branchName,
	}, nil
}
