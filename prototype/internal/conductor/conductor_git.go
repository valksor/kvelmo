package conductor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/naming"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// gitInfo holds git branch/worktree information created during task start.
type gitInfo struct {
	branchName    string
	baseBranch    string
	worktreePath  string
	commitPrefix  string // Resolved commit prefix (e.g., "[FEATURE-123]")
	branchPattern string // Template used to generate branch
}

// namingInfo holds resolved naming for a task.
type namingInfo struct {
	externalKey   string // User-facing key (e.g., "FEATURE-123")
	taskType      string // Task type (e.g., "feature", "fix")
	slug          string // URL-safe title slug
	branchName    string // Resolved branch name
	commitPrefix  string // Resolved commit prefix
	branchPattern string // Template used for branch
}

// resolveNaming resolves external key, branch name, and commit prefix from workUnit and options.
func (c *Conductor) resolveNaming(workUnit *provider.WorkUnit, taskID string) *namingInfo {
	// Load workspace config for templates
	cfg, _ := c.workspace.LoadConfig()

	// Resolve external key: CLI flag > workUnit > taskID fallback
	externalKey := c.opts.ExternalKey
	if externalKey == "" {
		externalKey = workUnit.ExternalKey
	}
	if externalKey == "" {
		externalKey = taskID
	}

	// Resolve task type
	taskType := workUnit.TaskType
	if taskType == "" {
		taskType = "task"
	}

	// Resolve title: CLI flag > workUnit
	title := workUnit.Title
	if c.opts.TitleOverride != "" {
		title = c.opts.TitleOverride
	}

	// Resolve slug: CLI flag > workUnit > generated from title
	slug := workUnit.Slug
	if slug == "" {
		slug = naming.Slugify(title, 50)
	}
	if c.opts.SlugOverride != "" {
		slug = c.opts.SlugOverride
	}

	// Resolve branch pattern template: CLI flag > workspace config
	branchPattern := c.opts.BranchPatternTemplate
	if branchPattern == "" {
		branchPattern = cfg.Git.BranchPattern
	}

	// Resolve commit prefix template: CLI flag > workspace config
	commitPrefixTemplate := c.opts.CommitPrefixTemplate
	if commitPrefixTemplate == "" {
		commitPrefixTemplate = cfg.Git.CommitPrefix
	}

	// Build template variables
	vars := naming.TemplateVars{
		Key:    externalKey,
		TaskID: taskID,
		Type:   taskType,
		Slug:   slug,
		Title:  title,
	}

	// Expand templates
	branchName := naming.ExpandTemplate(branchPattern, vars)
	branchName = naming.CleanBranchName(branchName)

	commitPrefix := naming.ExpandTemplate(commitPrefixTemplate, vars)

	return &namingInfo{
		externalKey:   externalKey,
		taskType:      taskType,
		slug:          slug,
		branchName:    branchName,
		commitPrefix:  commitPrefix,
		branchPattern: branchPattern,
	}
}

// generateUniqueBranchName generates a unique branch name by appending a numeric suffix
// if the base name already exists. Returns the first available name.
func (c *Conductor) generateUniqueBranchName(ctx context.Context, baseName string) string {
	if !c.git.BranchExists(ctx, baseName) {
		return baseName
	}

	// Try suffixes -2, -3, -4, etc. until we find an available name
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", baseName, i)
		if !c.git.BranchExists(ctx, candidate) {
			return candidate
		}
	}
}

// createBranchOrWorktree creates a git branch or worktree for the task.
func (c *Conductor) createBranchOrWorktree(ctx context.Context, taskID string, ni *namingInfo) (*gitInfo, error) {
	if c.git == nil || !c.opts.CreateBranch {
		return &gitInfo{}, nil
	}

	baseBranch, _ := c.git.CurrentBranch(ctx)
	// Generate unique branch name (adds suffix if already exists)
	branchName := c.generateUniqueBranchName(ctx, ni.branchName)

	if c.opts.UseWorktree {
		worktreePath := c.git.GetWorktreePath(taskID)
		if err := c.git.EnsureWorktreesDir(); err != nil {
			return nil, fmt.Errorf("create worktrees directory: %w", err)
		}
		if err := c.git.CreateWorktreeNewBranch(ctx, worktreePath, branchName, baseBranch); err != nil {
			return nil, fmt.Errorf("create worktree: %w", err)
		}

		return &gitInfo{
			branchName:    branchName,
			baseBranch:    baseBranch,
			worktreePath:  worktreePath,
			commitPrefix:  ni.commitPrefix,
			branchPattern: ni.branchPattern,
		}, nil
	}

	// Create and checkout the branch
	if err := c.git.CreateBranch(ctx, branchName, baseBranch); err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}

	// Switch to the new branch
	if err := c.git.Checkout(ctx, branchName); err != nil {
		// Try to clean up the branch we just created
		_ = c.git.DeleteBranch(ctx, branchName, false)

		return nil, fmt.Errorf("checkout branch: %w", err)
	}

	// Pop stash if we stashed earlier AND auto-pop is enabled (only for regular branches, not worktrees)
	if c.opts.StashChanges {
		if c.opts.AutoPopStash {
			if err := c.popStashIfExists(ctx); err != nil {
				return nil, fmt.Errorf("failed to restore stashed changes: %w", err)
			}
			c.publishProgress("Restored stashed changes", 10)
		} else {
			c.publishProgress("Stashed changes preserved (use 'git stash pop' to restore)", 10)
		}
	}

	return &gitInfo{
		branchName:    branchName,
		baseBranch:    baseBranch,
		commitPrefix:  ni.commitPrefix,
		branchPattern: ni.branchPattern,
	}, nil
}

// resolveTargetBranch determines the target branch for merging.
func (c *Conductor) resolveTargetBranch(ctx context.Context, requested string) string {
	if requested != "" {
		return requested
	}

	// Use the stored base branch from when task was started
	if c.taskWork != nil && c.taskWork.Git.BaseBranch != "" {
		return c.taskWork.Git.BaseBranch
	}

	// Fallback to detecting base branch
	baseBranch, err := c.git.GetBaseBranch(ctx)
	if err == nil && baseBranch != "" {
		return baseBranch
	}

	// Last resort: use current branch (better than empty string)
	currentBranch, _ := c.git.CurrentBranch(ctx)

	return currentBranch
}

// generateFinalCommitMessage uses the agent to generate a structured commit message
// based on specifications and git changes.
func (c *Conductor) generateFinalCommitMessage(ctx context.Context) (string, error) {
	if c.activeTask == nil {
		return "", errors.New("no active task")
	}

	taskID := c.activeTask.ID
	title := c.taskWork.Metadata.Title

	// Get spec paths
	specNumbers, err := c.workspace.ListSpecifications(taskID)
	if err != nil {
		return "", fmt.Errorf("list specifications: %w", err)
	}

	var specPaths []string
	for _, num := range specNumbers {
		specPaths = append(specPaths, fmt.Sprintf("specification-%d.md", num))
	}

	// Gather specs content
	specSnapshot, err := c.workspace.GatherSpecificationsContent(taskID)
	if err != nil {
		specSnapshot = fmt.Sprintf("Error gathering specs: %v", err)
	}

	// Get git diff information
	var diffStat, stagedFiles, stagedDiff string
	if c.git != nil {
		// Get diff stat
		if changes, err := c.git.GetChangeSummary(ctx); err == nil {
			var parts []string
			if len(changes.Added) > 0 {
				parts = append(parts, "Added: "+strings.Join(changes.Added, ", "))
			}
			if len(changes.Modified) > 0 {
				parts = append(parts, "Modified: "+strings.Join(changes.Modified, ", "))
			}
			if len(changes.Deleted) > 0 {
				parts = append(parts, "Deleted: "+strings.Join(changes.Deleted, ", "))
			}
			diffStat = strings.Join(parts, "\n")
		}

		// Get staged files
		if output, err := c.git.Diff(ctx, "--cached", "--name-only"); err == nil {
			stagedFiles = strings.TrimSpace(output)
		}

		// Get staged diff (truncated to 100k chars)
		const maxStagedDiffChars = 100000
		if output, err := c.git.Diff(ctx, "--cached"); err == nil {
			if len(output) > maxStagedDiffChars {
				stagedDiff = output[:maxStagedDiffChars] + "\n[staged diff omitted due to size; relying on staged file list and spec snapshot]"
			} else {
				stagedDiff = output
			}
		}
	}

	// Build prompt
	prompt := buildFinishPrompt(taskID, title, specPaths, specSnapshot, diffStat, stagedFiles, stagedDiff)

	// Get agent for commit message generation
	agent, err := c.GetAgentForStep(ctx, workflow.StepImplementing)
	if err != nil {
		return "", fmt.Errorf("get agent for commit message: %w", err)
	}

	// Execute prompt
	response, err := agent.Run(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("execute agent: %w", err)
	}

	// Extract commit message from response
	commitMsg := strings.TrimSpace(response.Summary)
	if commitMsg == "" && len(response.Messages) > 0 {
		commitMsg = strings.TrimSpace(response.Messages[0])
	}

	if commitMsg == "" {
		return "", errors.New("agent returned empty commit message")
	}

	return commitMsg, nil
}

// GenerateCommitMessagePreview generates a commit message preview for display.
// Returns empty string if generation fails (caller should handle gracefully).
func (c *Conductor) GenerateCommitMessagePreview(ctx context.Context) (string, error) {
	return c.generateFinalCommitMessage(ctx)
}

// performMerge handles the merge operation (squash or regular).
func (c *Conductor) performMerge(ctx context.Context, opts FinishOptions) error {
	targetBranch := c.resolveTargetBranch(ctx, opts.TargetBranch)
	currentBranch := c.activeTask.Branch
	taskID := c.activeTask.ID

	// Checkout target branch
	if err := c.git.Checkout(ctx, targetBranch); err != nil {
		return fmt.Errorf("checkout target: %w", err)
	}

	// Merge (squash or regular)
	if opts.SquashMerge {
		if err := c.git.MergeSquash(ctx, currentBranch); err != nil {
			_ = c.git.Checkout(ctx, currentBranch)

			return fmt.Errorf("squash merge: %w", err)
		}
		// Use pre-generated commit message, or generate one, or fallback to simple message
		var msg string
		if opts.CommitMessage != "" {
			msg = opts.CommitMessage
		} else if generatedMsg, err := c.generateFinalCommitMessage(ctx); err == nil {
			msg = generatedMsg
		} else {
			// Fallback to simple message if generation fails
			prefix := c.taskWork.Git.CommitPrefix
			if prefix == "" {
				prefix = fmt.Sprintf("(%s)", taskID)
			}
			msg = fmt.Sprintf("%s merged from %s", prefix, currentBranch)
		}
		if _, err := c.git.Commit(ctx, msg); err != nil {
			_ = c.git.Checkout(ctx, currentBranch)

			return fmt.Errorf("commit merge: %w", err)
		}
	} else {
		if err := c.git.MergeBranch(ctx, currentBranch, true); err != nil {
			_ = c.git.Checkout(ctx, currentBranch)

			return fmt.Errorf("merge: %w", err)
		}
	}

	return nil
}

// cleanupAfterMerge removes the branch and worktree after successful merge
// NOTE: Errors are logged but not returned intentionally.
// The merge succeeded, so cleanup failures should not undo the user's work.
func (c *Conductor) cleanupAfterMerge(ctx context.Context, opts FinishOptions) {
	currentBranch := c.activeTask.Branch
	targetBranch := c.resolveTargetBranch(ctx, opts.TargetBranch)
	taskID := c.activeTask.ID

	if !opts.DeleteBranch || currentBranch == targetBranch {
		return
	}

	// Checkpoint deletion is best-effort; ignore errors
	_ = c.git.DeleteAllCheckpoints(ctx, taskID)

	// If using worktree, remove it first
	if worktreePath := c.activeTask.WorktreePath; worktreePath != "" {
		if err := c.git.RemoveWorktree(ctx, worktreePath, true); err != nil {
			c.logError(fmt.Errorf("remove worktree: %w", err))
		}
	}

	if err := c.git.DeleteBranch(ctx, currentBranch, true); err != nil {
		c.logError(fmt.Errorf("delete branch: %w", err))
	}
}

// popStashIfExists pops the stash if it exists, handles errors gracefully.
func (c *Conductor) popStashIfExists(ctx context.Context) error {
	err := c.git.StashPop(ctx)
	if err != nil {
		// If no stash exists, that's OK (might have been empty or only untracked files)
		// Git returns "No stash entries found" when stash list is empty
		if strings.Contains(err.Error(), "No stash") {
			return nil
		}

		return err
	}

	return nil
}
