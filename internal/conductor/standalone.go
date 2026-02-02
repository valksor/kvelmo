package conductor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/progress"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/eventbus"
)

// StandaloneDiffMode specifies how to gather diffs for standalone operations.
type StandaloneDiffMode string

const (
	// DiffModeUncommitted includes all uncommitted changes (staged + unstaged).
	DiffModeUncommitted StandaloneDiffMode = "uncommitted"
	// DiffModeBranch compares current branch against base branch.
	DiffModeBranch StandaloneDiffMode = "branch"
	// DiffModeRange compares a specific commit range.
	DiffModeRange StandaloneDiffMode = "range"
	// DiffModeFiles compares specific files.
	DiffModeFiles StandaloneDiffMode = "files"
)

// StandaloneDiffOptions configures how to gather diffs for standalone review/simplify.
type StandaloneDiffOptions struct {
	Mode       StandaloneDiffMode // How to gather the diff
	BaseBranch string             // For branch mode (from flag, config, or auto-detect)
	Range      string             // For range mode (e.g., "HEAD~3..HEAD")
	Files      []string           // For files mode
	Context    int                // Lines of context (default: 3)
}

// StandaloneReviewOptions configures standalone code review behavior.
type StandaloneReviewOptions struct {
	StandaloneDiffOptions

	Agent            string // Optional agent override
	ApplyFixes       bool   // If true, apply suggested fixes to files
	CreateCheckpoint bool   // Create checkpoint before changes (only used if ApplyFixes)
}

// StandaloneSimplifyOptions configures standalone simplification behavior.
type StandaloneSimplifyOptions struct {
	StandaloneDiffOptions

	Agent            string // Optional agent override
	CreateCheckpoint bool   // Whether to create a checkpoint before changes
}

// StandaloneReviewResult contains the results of a standalone review.
type StandaloneReviewResult struct {
	Diff    string             // The diff that was reviewed
	Issues  []ReviewIssue      // Issues found during review (reuses existing type)
	Summary string             // AI-generated summary
	Verdict string             // "APPROVED" or "NEEDS_CHANGES"
	Usage   *agent.UsageStats  // Token usage
	Changes []agent.FileChange // File changes applied (only populated if ApplyFixes was true)
}

// StandaloneSimplifyResult contains the results of standalone simplification.
type StandaloneSimplifyResult struct {
	Diff    string             // The diff that was simplified
	Changes []agent.FileChange // Suggested file changes (reuses existing type)
	Summary string             // AI-generated summary
	Usage   *agent.UsageStats  // Token usage
}

// GetStandaloneDiff gathers a diff based on the provided options.
// This is the shared diff-gathering infrastructure used by both review and simplify.
func (c *Conductor) GetStandaloneDiff(ctx context.Context, opts StandaloneDiffOptions) (string, error) {
	if c.git == nil {
		return "", errors.New("git not available")
	}

	contextLines := opts.Context
	if contextLines == 0 {
		contextLines = 3
	}

	switch opts.Mode {
	case DiffModeUncommitted:
		return c.git.DiffUncommitted(ctx, contextLines)

	case DiffModeBranch:
		baseBranch := opts.BaseBranch
		// Check workspace config for default branch override
		if baseBranch == "" {
			cfg, err := c.workspace.LoadConfig()
			if err == nil && cfg.Git.DefaultBranch != "" {
				baseBranch = cfg.Git.DefaultBranch
			}
		}

		return c.git.DiffBranch(ctx, baseBranch, contextLines)

	case DiffModeRange:
		if opts.Range == "" {
			return "", errors.New("range is required for range mode")
		}

		return c.git.DiffRange(ctx, opts.Range, contextLines)

	case DiffModeFiles:
		if len(opts.Files) == 0 {
			return "", errors.New("files are required for files mode")
		}

		return c.git.DiffFiles(ctx, opts.Files, contextLines)

	default:
		// Default to uncommitted changes
		return c.git.DiffUncommitted(ctx, contextLines)
	}
}

// ReviewStandalone performs a standalone code review without requiring an active task.
// It reviews code changes based on the provided diff options.
// If ApplyFixes is true, the agent will also apply suggested fixes to the files.
func (c *Conductor) ReviewStandalone(ctx context.Context, opts StandaloneReviewOptions) (*StandaloneReviewResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Gather the diff
	diff, err := c.GetStandaloneDiff(ctx, opts.StandaloneDiffOptions)
	if err != nil {
		return nil, fmt.Errorf("gather diff: %w", err)
	}

	if diff == "" {
		return nil, errors.New("nothing to review: no changes found")
	}

	// Create checkpoint if applying fixes and checkpoint is requested
	if opts.ApplyFixes && opts.CreateCheckpoint && c.git != nil {
		c.publishProgress("Creating checkpoint...", 0)
		checkpointMsg := "Pre-review-fix checkpoint " + time.Now().Format("2006-01-02 15:04")
		if _, err := c.git.CreateCheckpoint(ctx, "standalone-review-fix", checkpointMsg); err != nil {
			c.logError(fmt.Errorf("create checkpoint: %w", err))
		}
	}

	// Create progress tracker
	var statusLine *progress.StatusLine
	if !c.opts.DryRun {
		action := "Reviewing"
		if opts.ApplyFixes {
			action = "Reviewing and fixing"
		}
		statusLine = progress.NewStatusLine(action)
		defer statusLine.Done()
	}

	// Get agent for review (use implementing step if applying fixes to get edit permissions)
	step := workflow.StepReviewing
	if opts.ApplyFixes {
		step = workflow.StepImplementing
	}
	reviewAgent, err := c.getStandaloneAgent(ctx, opts.Agent, step)
	if err != nil {
		return nil, fmt.Errorf("get review agent: %w", err)
	}

	// Determine working directory
	workingDir := c.CodeDir()

	// Build review prompt (different prompt if applying fixes)
	workspaceCfg, _ := c.workspace.LoadConfig()
	customInstructions := buildCombinedInstructions(workspaceCfg, "reviewing")
	var prompt string
	if opts.ApplyFixes {
		prompt = buildStandaloneReviewWithFixesPrompt(workingDir, diff, opts.Mode, customInstructions)
	} else {
		prompt = buildStandaloneReviewPrompt(workingDir, diff, opts.Mode, customInstructions)
	}

	// Run agent with streaming
	progressMsg := "Agent reviewing changes..."
	if opts.ApplyFixes {
		progressMsg = "Agent reviewing and fixing..."
	}
	c.publishProgress(progressMsg, 20)
	var transcriptBuilder strings.Builder
	response, err := reviewAgent.RunWithCallback(ctx, prompt, func(event agent.Event) error {
		c.eventBus.PublishRaw(eventbus.Event{
			Type: events.TypeAgentMessage,
			Data: map[string]any{"event": event},
		})
		if statusLine != nil {
			_ = statusLine.OnEvent(event)
		}
		if event.Text != "" {
			transcriptBuilder.WriteString(event.Text)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("agent review: %w", err)
	}

	// Build result
	result := &StandaloneReviewResult{
		Diff:    diff,
		Summary: response.Summary,
		Usage:   response.Usage,
	}

	// Include file changes if applying fixes
	if opts.ApplyFixes && len(response.Files) > 0 {
		result.Changes = response.Files
	}

	// Parse the response for issues and verdict
	if len(response.Messages) > 0 {
		fullResponse := strings.Join(response.Messages, "\n")
		result.Issues = parseReviewIssues(fullResponse)
		result.Verdict = parseStandaloneReviewVerdict(fullResponse)
		if result.Summary == "" {
			result.Summary = extractStandaloneReviewSummary(fullResponse)
		}
	}

	completeMsg := "Review complete"
	if opts.ApplyFixes && len(result.Changes) > 0 {
		completeMsg = fmt.Sprintf("Review complete, %d files modified", len(result.Changes))
	}
	c.publishProgress(completeMsg, 100)

	return result, nil
}

// SimplifyStandalone performs standalone code simplification without requiring an active task.
// It simplifies code based on the provided diff options.
func (c *Conductor) SimplifyStandalone(ctx context.Context, opts StandaloneSimplifyOptions) (*StandaloneSimplifyResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Gather the diff
	diff, err := c.GetStandaloneDiff(ctx, opts.StandaloneDiffOptions)
	if err != nil {
		return nil, fmt.Errorf("gather diff: %w", err)
	}

	if diff == "" {
		return nil, errors.New("nothing to simplify: no changes found")
	}

	// Create checkpoint if requested and git is available
	if opts.CreateCheckpoint && c.git != nil {
		c.publishProgress("Creating checkpoint...", 0)
		checkpointMsg := "Pre-simplify checkpoint " + time.Now().Format("2006-01-02 15:04")
		// Use a temporary task ID for standalone checkpoints
		if _, err := c.git.CreateCheckpoint(ctx, "standalone-simplify", checkpointMsg); err != nil {
			c.logError(fmt.Errorf("create checkpoint: %w", err))
		}
	}

	// Create progress tracker
	var statusLine *progress.StatusLine
	if !c.opts.DryRun {
		statusLine = progress.NewStatusLine("Simplifying")
		defer statusLine.Done()
	}

	// Get agent for simplification
	simplifyAgent, err := c.getStandaloneAgent(ctx, opts.Agent, workflow.StepImplementing)
	if err != nil {
		return nil, fmt.Errorf("get simplify agent: %w", err)
	}

	// Determine working directory
	workingDir := c.CodeDir()

	// Build simplify prompt
	workspaceCfg, _ := c.workspace.LoadConfig()
	customInstructions := buildCombinedInstructions(workspaceCfg, "simplifying")
	prompt := buildStandaloneSimplifyPrompt(workingDir, diff, opts.Mode, customInstructions)

	// Run agent with streaming
	c.publishProgress("Agent simplifying code...", 20)
	var transcriptBuilder strings.Builder
	response, err := simplifyAgent.RunWithCallback(ctx, prompt, func(event agent.Event) error {
		c.eventBus.PublishRaw(eventbus.Event{
			Type: events.TypeAgentMessage,
			Data: map[string]any{"event": event},
		})
		if statusLine != nil {
			_ = statusLine.OnEvent(event)
		}
		if event.Text != "" {
			transcriptBuilder.WriteString(event.Text)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("agent simplify: %w", err)
	}

	// Build result
	result := &StandaloneSimplifyResult{
		Diff:    diff,
		Summary: response.Summary,
		Usage:   response.Usage,
		Changes: response.Files,
	}

	c.publishProgress("Simplification complete", 100)

	return result, nil
}

// getStandaloneAgent gets an agent for standalone operations.
// It uses the provided agent name if specified, otherwise falls back to the default for the step.
func (c *Conductor) getStandaloneAgent(ctx context.Context, agentName string, step workflow.Step) (agent.Agent, error) {
	if agentName != "" {
		// Try to get the specific agent
		a, err := c.agents.Get(agentName)
		if err != nil {
			return nil, fmt.Errorf("get agent %s: %w", agentName, err)
		}

		return a, nil
	}

	// Fall back to step-specific agent
	return c.GetAgentForStep(ctx, step)
}

// parseStandaloneReviewVerdict extracts the verdict from the review response.
func parseStandaloneReviewVerdict(response string) string {
	response = strings.ToUpper(response)
	if strings.Contains(response, "APPROVED") {
		return "APPROVED"
	}
	if strings.Contains(response, "NEEDS_CHANGES") || strings.Contains(response, "CHANGES_REQUESTED") {
		return "NEEDS_CHANGES"
	}

	return "COMMENT"
}

// extractStandaloneReviewSummary extracts a summary from the review response.
func extractStandaloneReviewSummary(response string) string {
	// Look for ## Summary section
	summaryIdx := strings.Index(response, "## Summary")
	if summaryIdx == -1 {
		summaryIdx = strings.Index(response, "# Summary")
	}
	if summaryIdx == -1 {
		// Return first few lines as summary
		lines := strings.Split(response, "\n")
		var summary []string
		for i, line := range lines {
			if i >= 3 {
				break
			}
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				summary = append(summary, line)
			}
		}

		return strings.Join(summary, " ")
	}

	// Extract text after Summary heading until next heading
	rest := response[summaryIdx:]
	lines := strings.Split(rest, "\n")
	var summary []string
	for i, line := range lines {
		if i == 0 {
			continue // Skip the heading itself
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "##") || strings.HasPrefix(line, "# ") {
			break
		}
		if line != "" {
			summary = append(summary, line)
		}
	}

	return strings.Join(summary, " ")
}
