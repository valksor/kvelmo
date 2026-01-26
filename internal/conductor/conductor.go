// Package conductor provides the main orchestration layer for the mehrhof task automation tool.
//
// The Conductor is a facade that combines workflow management, storage, VCS operations,
// and AI agent coordination. It manages the complete lifecycle of tasks from creation
// through planning, implementation, and completion.
//
// Key responsibilities:
//   - Task lifecycle management (start, plan, implement, review, finish)
//   - Git branch and worktree management for parallel tasks
//   - Agent selection and configuration (with per-step agent overrides)
//   - State machine orchestration via the workflow package
//   - Event publishing and subscription for component decoupling
//
// Thread safety:
//   - Most methods are not thread-safe and should be called from a single goroutine.
//   - GetActiveTask() and GetTaskWork() return copies to avoid data races.
//   - State changes are protected by an internal mutex.
//
// Usage:
//
//	c := conductor.New(
//	    conductor.WithWorkDir("/path/to/repo"),
//	    conductor.WithStdout(os.Stdout),
//	)
//	if err := c.Initialize(ctx); err != nil {
//	    log.Fatal(err)
//	}
package conductor

import (
	"context"
	"crypto/subtle"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/browser"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/plugin"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/vcs"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/eventbus"
)

// Conductor orchestrates the task automation workflow.
type Conductor struct {
	mu sync.RWMutex

	// Core components
	machine   *workflow.Machine
	eventBus  *eventbus.Bus
	workspace *storage.Workspace
	git       *vcs.Git

	// Registries
	providers *provider.Registry
	agents    *agent.Registry
	plugins   *plugin.Registry

	// Browser controller (lazy initialization)
	browser     browser.Controller
	browserOnce sync.Once

	// Memory system (initialized if enabled)
	memory *MemorySystem

	// ML system (initialized if enabled)
	ml *MLSystem

	// Workflow plugin adapters (for lifecycle management)
	workflowAdapters []*plugin.WorkflowAdapter

	// Current state
	activeTask *storage.ActiveTask
	taskWork   *storage.TaskWork

	// Configuration
	opts Options

	// Active agent
	activeAgent     agent.Agent
	taskAgentConfig *provider.AgentConfig // Agent config from task source (if any)

	// Session tracking (for conversation history and token usage)
	currentSession     *storage.Session
	currentSessionFile string
}

// New creates a new Conductor with the given options.
func New(opts ...Option) (*Conductor, error) {
	options := DefaultOptions()
	options.Apply(opts...)

	// Create event bus
	bus := eventbus.NewBus()

	// Create state machine
	machine := workflow.NewMachine(bus)

	// Create registries
	providerRegistry := provider.NewRegistry()
	agentRegistry := agent.NewRegistry()

	c := &Conductor{
		machine:   machine,
		eventBus:  bus,
		providers: providerRegistry,
		agents:    agentRegistry,
		opts:      options,
	}

	// Subscribe to state changes
	bus.Subscribe(events.TypeStateChanged, c.onStateChanged)

	return c, nil
}

// GetProviderRegistry returns the provider registry.
func (c *Conductor) GetProviderRegistry() *provider.Registry {
	return c.providers
}

// GetAgentRegistry returns the agent registry.
func (c *Conductor) GetAgentRegistry() *agent.Registry {
	return c.agents
}

// GetEventBus returns the event bus.
func (c *Conductor) GetEventBus() *eventbus.Bus {
	return c.eventBus
}

// GetWorkspace returns the workspace.
func (c *Conductor) GetWorkspace() *storage.Workspace {
	return c.workspace
}

// GetGit returns the git instance.
func (c *Conductor) GetGit() *vcs.Git {
	return c.git
}

// GetActiveTask returns the current active task.
// Returns a copy to avoid data races; the caller cannot modify the internal state.
// Note: ActiveTask only contains value types (strings, bool, time.Time),
// so a shallow copy is sufficient - no deep copy needed.
func (c *Conductor) GetActiveTask() *storage.ActiveTask {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.activeTask == nil {
		return nil
	}
	// Return a copy to prevent caller from mutating internal state without lock
	taskCopy := *c.activeTask

	return &taskCopy
}

// GetTaskWork returns the current task work
// Returns a copy to avoid data races; the caller cannot modify the internal state.
func (c *Conductor) GetTaskWork() *storage.TaskWork {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.taskWork == nil {
		return nil
	}
	// Return a copy to prevent caller from mutating internal state without lock
	workCopy := *c.taskWork

	return &workCopy
}

// GetActiveAgent returns the active agent.
func (c *Conductor) GetActiveAgent() agent.Agent {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.activeAgent
}

// GetMachine returns the state machine.
func (c *Conductor) GetMachine() *workflow.Machine {
	return c.machine
}

// GetStdout returns the configured stdout writer.
func (c *Conductor) GetStdout() io.Writer {
	return c.opts.Stdout
}

// GetStderr returns the configured stderr writer.
func (c *Conductor) GetStderr() io.Writer {
	return c.opts.Stderr
}

// GetBrowser returns the browser controller with lazy initialization.
// Returns nil if browser is not configured.
func (c *Conductor) GetBrowser(ctx context.Context) browser.Controller {
	if c.opts.BrowserConfig == nil {
		return nil
	}

	c.browserOnce.Do(func() {
		c.browser = browser.NewController(*c.opts.BrowserConfig)
		c.logVerbosef("browser controller initialized")
	})

	return c.browser
}

// logVerbosef logs a message if verbose mode is enabled.
func (c *Conductor) logVerbosef(format string, args ...any) {
	if c.opts.Verbose && c.opts.Stdout != nil {
		_, _ = fmt.Fprintf(c.opts.Stdout, format+"\n", args...)
	}
}

// SetImplementationOptions temporarily sets implementation options for the next implement call.
// This is used by the Web UI to pass component filter and parallel mode via query parameters.
func (c *Conductor) SetImplementationOptions(component, parallel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.opts.OnlyComponent = component
	c.opts.ParallelCount = parallel
}

// ClearImplementationOptions clears temporary implementation options.
// Should be called after implementation to reset to defaults.
func (c *Conductor) ClearImplementationOptions() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.opts.OnlyComponent = ""
	c.opts.ParallelCount = ""
}

// logError logs an error using the callback if configured.
func (c *Conductor) logError(err error) {
	if c.opts.OnError != nil {
		c.opts.OnError(err)
	}
}

// RunPRReview performs an AI-powered review of a pull/merge request.
// This is a standalone operation that does not require an active task or workspace.
// It supports incremental/differential reviews that only comment on new issues.
func (c *Conductor) RunPRReview(ctx context.Context, opts PRReviewOptions) (*PRReviewResult, error) {
	// 1. Get provider from registry
	c.mu.RLock()
	_, factory, ok := c.providers.Get(opts.Provider)
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("provider %q not found", opts.Provider)
	}

	// 2. Create provider instance with minimal config
	// Token is handled via environment variables or provider-specific config
	providerCfg := provider.NewConfig()
	if opts.Token != "" {
		providerCfg = providerCfg.Set("token", opts.Token)
	}

	instance, err := factory(ctx, providerCfg)
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}

	// 3. Check if provider implements required interfaces
	prFetcher, ok := instance.(provider.PRFetcher)
	if !ok {
		return nil, fmt.Errorf("provider %q does not implement PRFetcher", opts.Provider)
	}

	prCommenter, ok := instance.(provider.PRCommenter)
	if !ok {
		return nil, fmt.Errorf("provider %q does not implement PRCommenter", opts.Provider)
	}

	// 4. Fetch PR details and diff
	c.logVerbosef("Fetching PR #%d from %s...", opts.PRNumber, opts.Provider)
	pr, err := prFetcher.FetchPullRequest(ctx, opts.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("fetch PR: %w", err)
	}

	// Validate PR
	if pr.Number != opts.PRNumber {
		return nil, fmt.Errorf("PR number mismatch: requested %d, got %d", opts.PRNumber, pr.Number)
	}

	// Skip closed/merged PRs
	if pr.State == "closed" || pr.State == "merged" {
		return &PRReviewResult{
			Skipped: true,
			Reason:  fmt.Sprintf("PR is %s - skipping review", pr.State),
		}, nil
	}

	c.logVerbosef("Fetching PR diff...")
	diff, err := prFetcher.FetchPullRequestDiff(ctx, opts.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("fetch PR diff: %w", err)
	}

	// Check if diff is empty
	if diff.Patch == "" {
		return &PRReviewResult{
			Skipped: true,
			Reason:  "No diff found - PR may have no changes",
		}, nil
	}

	// 5. Fetch existing comments to find our previous review (if any)
	var prevState *PRReviewState
	var ourPreviousComment *provider.Comment

	if prCommentFetcher, ok := instance.(provider.PRCommentFetcher); ok {
		c.logVerbosef("Fetching existing PR comments...")
		comments, err := prCommentFetcher.FetchPullRequestComments(ctx, opts.PRNumber)
		if err != nil {
			// Log warning but continue - this is acceptable for first run
			c.logVerbosef("Warning: could not fetch existing comments (first run?): %v", err)
		} else {
			// Empty string for botUsername - we accept any comment with our marker
			// For stronger validation, the bot username should be passed in opts
			ourPreviousComment, prevState, _ = FindOurPreviousComment(comments, "")
		}
	}

	// 6. Check if this is a re-run on same commit (skip if no changes)
	if prevState != nil && prevState.CommitSHA == pr.HeadSHA {
		// Additional checks for edge cases - verify diff hasn't changed either
		currentDiffHash := hashDiffPatch(diff.Patch)
		if prevState.ReviewedDiffHash != "" {
			// Use constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(currentDiffHash), []byte(prevState.ReviewedDiffHash)) == 1 {
				c.logVerbosef("Skipping review: no new commits or diff changes since last review")

				return &PRReviewResult{
					Skipped: true,
					Reason:  "No new commits or diff changes since last review",
				}, nil
			}
		} else {
			// Old state without diff hash - skip based on commit SHA only
			c.logVerbosef("Skipping review: no new commits since last review")

			return &PRReviewResult{
				Skipped: true,
				Reason:  "No new commits since last review",
			}, nil
		}
	}

	// 7. Get agent for review
	c.logVerbosef("Getting agent '%s'...", opts.AgentName)
	c.mu.RLock()
	agentInst, err := c.agents.Get(opts.AgentName)
	c.mu.RUnlock()
	if err != nil {
		return nil, fmt.Errorf("get agent %q: %w", opts.AgentName, err)
	}

	// 8. Build review prompt
	c.logVerbosef("Building review prompt...")
	prompt := buildPRReviewPrompt(pr, diff, prevState, opts.Scope, c.workspace)

	// 9. Run AI review with timeout
	// Add timeout context (default 10 minutes)
	const defaultReviewTimeout = 10 * time.Minute
	reviewCtx, cancel := context.WithTimeout(ctx, defaultReviewTimeout)
	defer cancel()

	c.logVerbosef("Running AI review (timeout: %v)...", defaultReviewTimeout)
	response, err := agentInst.Run(reviewCtx, prompt)
	if err != nil {
		return nil, fmt.Errorf("run agent: %w", err)
	}

	// 10. Parse response
	c.logVerbosef("Parsing AI response...")
	currentReview := parsePRReview(response.Summary)

	// 11. Compute delta
	delta := ComputeReviewDelta(prevState, currentReview)
	c.logVerbosef("Delta: %d new issues, %d fixed, %d unchanged",
		len(delta.NewIssues), len(delta.FixedIssues), len(delta.Unchanged))

	// 12. Build new state
	newState := BuildPRReviewState(pr, diff, currentReview)

	// 13. Format review comment
	commentBody := FormatReviewComment(currentReview, delta, opts)

	// 14. Embed state in comment
	commentBody = EmbedStateInComment(commentBody, newState)

	// 15. Post or update comment
	c.logVerbosef("Posting comment to PR...")
	if ourPreviousComment != nil && opts.UpdateExisting {
		// Update existing comment
		if prCommentUpdater, ok := instance.(provider.PRCommentUpdater); ok {
			_, err = prCommentUpdater.UpdatePullRequestComment(ctx, opts.PRNumber, ourPreviousComment.ID, commentBody)
			if err != nil {
				return nil, fmt.Errorf("update comment: %w", err)
			}
			c.logVerbosef("Updated existing comment")
		} else {
			// Can't update, post new comment instead
			_, err = prCommenter.AddPullRequestComment(ctx, opts.PRNumber, commentBody)
			if err != nil {
				return nil, fmt.Errorf("add comment: %w", err)
			}
			c.logVerbosef("Posted new comment (update not supported)")
		}
	} else {
		// Post new comment
		_, err = prCommenter.AddPullRequestComment(ctx, opts.PRNumber, commentBody)
		if err != nil {
			return nil, fmt.Errorf("add comment: %w", err)
		}
		c.logVerbosef("Posted new comment")
	}

	// 16. Build result
	result := &PRReviewResult{
		CommentsPosted: 1,
		URL:            pr.URL,
	}

	// If we found new issues, count them
	if len(delta.NewIssues) > 0 {
		result.CommentsPosted += len(delta.NewIssues)
	}

	c.logVerbosef("Review completed successfully")

	return result, nil
}
