package conductor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/vcs"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/naming"
	"github.com/valksor/go-toolkit/slug"
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
	// Sanitize external key for use in branch names - removes/replaces git-invalid chars.
	externalKey := c.opts.ExternalKey
	if externalKey == "" {
		externalKey = workUnit.ExternalKey
	}
	if externalKey == "" {
		externalKey = taskID
	}
	// Slugify external key if it contains spaces/special chars (e.g., from empty provider).
	// This ensures valid git branch names when {key} is used in branch patterns.
	if strings.ContainsAny(externalKey, " ,!?@#$%^&*()[]{}|\\<>") {
		externalKey = slug.Slugify(externalKey, 50)
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
	branchSlug := workUnit.Slug
	if branchSlug == "" {
		branchSlug = slug.Slugify(title, 50)
	}
	if c.opts.SlugOverride != "" {
		branchSlug = c.opts.SlugOverride
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
		Slug:   branchSlug,
		Title:  title,
	}

	// Expand templates
	branchName := naming.ExpandTemplate(branchPattern, vars)
	branchName = naming.CleanBranchName(branchName)

	commitPrefix := naming.ExpandTemplate(commitPrefixTemplate, vars)

	return &namingInfo{
		externalKey:   externalKey,
		taskType:      taskType,
		slug:          branchSlug,
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
	if c.git == nil || c.opts.NoBranch {
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

// GenerateCommitMessageForGroup generates a commit message for a specific change group.
// This is GENERIC - works in ANY repo, using AI to analyze the actual changes.
// Used by the `mehr commit` command to create logical commits.
func (c *Conductor) GenerateCommitMessageForGroup(ctx context.Context, group vcs.FileGroup, note string, previousAttempts []storage.CommitAttempt) string {
	if c.git == nil {
		return "Changes"
	}

	// Get diffs for context - AI sees actual changes
	diffs, _ := c.git.DiffFiles(ctx, group.Files, 3)

	// Build prompt with REAL changes from ANY repo
	var prompt strings.Builder
	prompt.WriteString("Generate a git commit message for these changes.\n\n")

	// Previous attempts for context
	if len(previousAttempts) > 0 {
		prompt.WriteString(fmt.Sprintf("Context: This is attempt #%d of refining commit grouping.\n", len(previousAttempts)+1))
		if note != "" {
			prompt.WriteString(fmt.Sprintf("User feedback: %s\n\n", note))
		}
	}

	// The actual file changes - AI figures out what happened
	prompt.WriteString("Files changed:\n")
	for _, f := range group.Files {
		prompt.WriteString(fmt.Sprintf("  %s\n", f))
	}

	// Include actual diffs if available (for better messages)
	if diffs != "" {
		// Limit diff size to avoid overwhelming the AI
		maxDiffLen := 2000
		if len(diffs) > maxDiffLen {
			diffs = diffs[:maxDiffLen] + "\n... (truncated)"
		}
		prompt.WriteString("\nDiff preview:\n")
		prompt.WriteString(diffs)
		prompt.WriteString("\n")
	}

	// CRITICAL: Get existing commits from THIS repo to match the style
	existingCommits, _ := c.git.Log(ctx, "-20", "--format=%B")

	prompt.WriteString(`
Generate a git commit message.

Existing commit messages from this repository (for style matching):
`)
	prompt.WriteString(existingCommits)
	prompt.WriteString(`

---

Files changed:
`)
	for _, f := range group.Files {
		prompt.WriteString(fmt.Sprintf("  %s\n", f))
	}

	// Include actual diffs if available (for better messages)
	if diffs != "" {
		// Limit diff size to avoid overwhelming the AI
		maxDiffLen := 2000
		if len(diffs) > maxDiffLen {
			diffs = diffs[:maxDiffLen] + "\n... (truncated)"
		}
		prompt.WriteString("\nDiff preview:\n")
		prompt.WriteString(diffs)
		prompt.WriteString("\n")
	}

	prompt.WriteString(`
---

Generate a commit message that MATCHES THE STYLE of existing commits above.

Analyze the existing commits and:
- Match their format (emoji prefixes? conventional commits? Co-Authored-By?)
- Match their length (short one-liners vs detailed multi-line?)
- Match their tone (casual vs formal?)

Then write a commit message for the new changes following that SAME style.
`)

	// Get agent and generate
	commitAgent, err := c.GetAgentForStep(ctx, workflow.StepCheckpointing)
	if err != nil {
		c.logError(fmt.Errorf("get agent for commit message: %w", err))

		return "Changes"
	}

	response, err := commitAgent.Run(ctx, prompt.String())
	if err != nil {
		c.logError(fmt.Errorf("generate commit message: %w", err))

		return "Changes"
	}

	if len(response.Messages) > 0 {
		return strings.TrimSpace(response.Messages[0])
	}

	return "Changes"
}

// SetCommitGroupAgent sets the AI agent for commit grouping.
// Used by the `mehr commit` command to enable AI-based file grouping.
func (c *Conductor) SetCommitGroupAgent(agent vcs.Agent) {
	// This would be stored on a ChangeAnalyzer instance
	// For now, the CLI command will create the analyzer and set the agent directly
}
