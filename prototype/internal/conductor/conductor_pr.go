package conductor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/provider/token"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/pullrequest"
	"github.com/valksor/go-toolkit/workunit"
)

// finishWithPR creates a pull request instead of merging locally.
func (c *Conductor) finishWithPR(ctx context.Context, opts FinishOptions) (*pullrequest.PullRequest, error) {
	if c.git == nil {
		return nil, errors.New("git not available for PR creation")
	}

	if c.activeTask.Branch == "" {
		return nil, errors.New("no branch associated with task for PR creation")
	}

	taskID := c.activeTask.ID
	sourceBranch := c.activeTask.Branch

	// Get the default remote (don't assume "origin")
	remote, err := c.git.GetDefaultRemote(ctx)
	if err != nil {
		return nil, fmt.Errorf("get default remote: %w", err)
	}

	// Push the branch to remote first
	if err := c.git.PushBranch(ctx, sourceBranch, remote, true); err != nil {
		return nil, fmt.Errorf("push branch: %w", err)
	}

	// Resolve task source provider
	taskProvider, resolveErr := c.resolveTaskProvider(ctx)

	// Check if task source supports PRs
	var prCreator pullrequest.PRCreator
	if resolveErr == nil {
		prCreator, _ = taskProvider.(pullrequest.PRCreator)
	}

	// Fallback to git remote provider if task source doesn't support PRs
	usedRemoteFallback := false
	if prCreator == nil {
		remotePR, err := c.resolveRemotePRProvider(ctx)
		if err != nil {
			return nil, fmt.Errorf("cannot create PR: %w", err)
		}
		prCreator = remotePR
		usedRemoteFallback = true
	}

	// Load specifications for PR body
	specs, err := c.loadSpecificationsForPR()
	if err != nil {
		c.logError(fmt.Errorf("load specifications for PR: %w", err))
		// Continue without specs, not fatal
	}

	// Get diff stats
	diffStat := c.getDiffStats(ctx)

	// Determine target branch
	targetBranch := opts.TargetBranch
	if targetBranch == "" {
		targetBranch = c.resolveTargetBranch(ctx, "")
	}

	// Build PR options
	prOpts := pullrequest.PullRequestOptions{
		Title:        opts.PRTitle,
		Body:         opts.PRBody,
		SourceBranch: sourceBranch,
		TargetBranch: targetBranch,
		Draft:        opts.DraftPR,
	}

	// Generate title if not provided
	if prOpts.Title == "" {
		prOpts.Title = c.generatePRTitle()
	}

	// Generate body if not provided
	if prOpts.Body == "" {
		prOpts.Body = c.generatePRBody(specs, diffStat)
	}

	// Create the PR
	pr, err := prCreator.CreatePullRequest(ctx, prOpts)
	if err != nil {
		return nil, fmt.Errorf("create pull request: %w", err)
	}

	// Persist PR info to work metadata for later retrieval
	if c.taskWork != nil {
		c.taskWork.Metadata.PullRequest = &storage.PullRequestInfo{
			Number:    pr.Number,
			URL:       pr.URL,
			CreatedAt: time.Now(),
		}
		if saveErr := c.workspace.SaveWork(c.taskWork); saveErr != nil {
			c.logError(fmt.Errorf("save PR info to work: %w", saveErr))
			// Non-fatal: PR was created, just metadata save failed
		}
	}

	// Publish PRCreatedEvent
	c.eventBus.Publish(events.PRCreatedEvent{
		TaskID:   taskID,
		PRNumber: pr.Number,
		PRURL:    pr.URL,
	})

	// Post comment to issue ONLY if task source provider supports commenting
	// (remote fallback provider doesn't have issue context)
	if !usedRemoteFallback && resolveErr == nil {
		if commenter, ok := taskProvider.(workunit.Commenter); ok {
			issueID := c.taskWork.Metadata.ExternalKey
			if issueID != "" {
				comment := fmt.Sprintf("Pull request created: #%d\n%s\n\nThe PR includes all changes from branch `%s`.",
					pr.Number, pr.URL, sourceBranch)
				if _, err := commenter.AddComment(ctx, issueID, comment); err != nil {
					c.logError(fmt.Errorf("add PR comment to issue: %w", err))
					// Don't fail the PR creation
				}
			}
		}
	}

	return pr, nil
}

// resolveRemotePRProvider resolves a PR-capable provider from the git remote URL.
// This is used as a fallback when the task source provider doesn't support PRs.
func (c *Conductor) resolveRemotePRProvider(ctx context.Context) (pullrequest.PRCreator, error) {
	if c.git == nil {
		return nil, errors.New("no git available")
	}

	// Get the default remote
	remote, err := c.git.GetDefaultRemote(ctx)
	if err != nil {
		return nil, errors.New("no git remote configured")
	}

	// Get the remote URL
	remoteURL, err := c.git.RemoteURL(ctx, remote)
	if err != nil {
		return nil, fmt.Errorf("get remote URL: %w", err)
	}

	// Detect provider from URL
	providerName := provider.DetectProviderFromURL(remoteURL)
	if providerName == "" {
		return nil, fmt.Errorf("unsupported git remote: %s (only GitHub/GitLab supported)", remoteURL)
	}

	// Parse owner/repo from URL
	owner, repo, err := provider.ParseOwnerRepoFromURL(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("parse remote URL: %w", err)
	}

	c.logVerbosef("Task source provider doesn't support PRs, using git remote: %s (%s/%s)", providerName, owner, repo)

	// Build provider config from workspace settings
	workspaceCfg, _ := c.workspace.LoadConfig() // ignore error, use defaults
	cfg := buildProviderConfig(ctx, workspaceCfg, providerName)

	// Override owner/repo with URL-extracted values (remote URL is source of truth)
	if providerName == "gitlab" {
		// GitLab uses project_path for nested groups
		cfg.Set("project_path", owner+"/"+repo)
	} else {
		cfg.Set("owner", owner)
		cfg.Set("repo", repo)
	}

	// Create provider instance
	p, err := c.providers.Create(ctx, providerName, cfg)
	if err != nil {
		if errors.Is(err, token.ErrNoToken) {
			switch providerName {
			case "github":
				return nil, errors.New("GitHub token required: run 'gh auth login' or set github.token in config.yaml")
			case "gitlab":
				return nil, errors.New("GitLab token required: set gitlab.token in config.yaml")
			default:
				return nil, fmt.Errorf("%s token required: check config.yaml", providerName)
			}
		}

		return nil, fmt.Errorf("create %s provider: %w", providerName, err)
	}

	// Type-assert to PRCreator
	prCreator, ok := p.(pullrequest.PRCreator)
	if !ok {
		return nil, fmt.Errorf("%s provider does not support pull requests", providerName)
	}

	return prCreator, nil
}

// resolveTaskProvider resolves the provider from the task's original reference.
func (c *Conductor) resolveTaskProvider(ctx context.Context) (any, error) {
	if c.activeTask == nil || c.activeTask.Ref == "" {
		return nil, errors.New("no task reference available")
	}

	// Resolve provider from the stored reference
	resolveOpts := provider.ResolveOptions{
		DefaultProvider: c.opts.DefaultProvider,
	}

	// Load workspace config and build provider config
	workspaceCfg, _ := c.workspace.LoadConfig() // ignore error, use defaults
	scheme := parseScheme(c.activeTask.Ref)
	providerCfg := buildProviderConfig(ctx, workspaceCfg, scheme)

	p, _, err := c.providers.Resolve(ctx, c.activeTask.Ref, providerCfg, resolveOpts)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// loadSpecificationsForPR loads all specifications for PR body generation.
func (c *Conductor) loadSpecificationsForPR() ([]*storage.Specification, error) {
	if c.workspace == nil || c.activeTask == nil {
		return nil, nil
	}

	taskID := c.activeTask.ID
	specNums, err := c.workspace.ListSpecifications(taskID)
	if err != nil {
		return nil, err
	}

	var specs []*storage.Specification
	for _, num := range specNums {
		content, err := c.workspace.LoadSpecification(taskID, num)
		if err != nil {
			continue
		}
		specs = append(specs, &storage.Specification{
			Number:  num,
			Content: content,
		})
	}

	return specs, nil
}

// getDiffStats returns git diff stats for the current branch vs base.
func (c *Conductor) getDiffStats(ctx context.Context) string {
	if c.git == nil || c.taskWork == nil {
		return ""
	}

	baseBranch := c.taskWork.Git.BaseBranch
	if baseBranch == "" {
		baseBranch, _ = c.git.GetBaseBranch(ctx)
	}

	if baseBranch == "" {
		return ""
	}

	// git diff --stat base..HEAD
	stat, err := c.git.Diff(ctx, "--stat", baseBranch+"..HEAD")
	if err != nil {
		return ""
	}

	return stat
}

// generatePRTitle generates a PR title from task metadata.
func (c *Conductor) generatePRTitle() string {
	if c.taskWork == nil {
		return "Implementation"
	}

	var title string
	if c.taskWork.Metadata.ExternalKey != "" {
		title = fmt.Sprintf("[#%s] ", c.taskWork.Metadata.ExternalKey)
	}

	if c.taskWork.Metadata.Title != "" {
		title += c.taskWork.Metadata.Title
	} else {
		title += "Implementation"
	}

	return title
}

// generatePRBody generates a PR body with implementation summary.
func (c *Conductor) generatePRBody(specs []*storage.Specification, diffStat string) string {
	var parts []string

	// Summary section
	parts = append(parts, "## Summary\n")

	if c.taskWork != nil && c.taskWork.Metadata.Title != "" {
		parts = append(parts, fmt.Sprintf("Implementation for: %s\n", c.taskWork.Metadata.Title))
	}

	// Link to issue if this is a GitHub issue task
	if c.taskWork != nil && c.taskWork.Source.Type == "github" && c.taskWork.Metadata.ExternalKey != "" {
		parts = append(parts, fmt.Sprintf("Closes #%s\n", c.taskWork.Metadata.ExternalKey))
	}

	// Specifications section
	if len(specs) > 0 {
		parts = append(parts, "\n## Implementation Details\n")
		for _, spec := range specs {
			if spec.Title != "" {
				parts = append(parts, fmt.Sprintf("### %s\n", spec.Title))
			}
			// Include first 500 chars of spec content as summary
			content := spec.Content
			if len(content) > 500 {
				content = content[:500] + "..."
			}
			parts = append(parts, content+"\n")
		}
	}

	// Changes section
	if diffStat != "" {
		parts = append(parts, "\n## Changes\n")
		parts = append(parts, "```\n"+diffStat+"\n```\n")
	}

	// Test plan section
	parts = append(parts, "\n## Test Plan\n")
	parts = append(parts, "- [ ] Manual testing\n")
	parts = append(parts, "- [ ] Unit tests pass\n")
	parts = append(parts, "- [ ] Code review\n")

	// Footer
	parts = append(parts, "\n---\n")
	parts = append(parts, "*Generated by [Mehrhof](https://github.com/valksor/go-mehrhof)*\n")

	// Use strings.Builder for efficient concatenation
	var sb strings.Builder
	for _, p := range parts {
		sb.WriteString(p)
	}

	return sb.String()
}
