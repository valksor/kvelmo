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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
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
	"github.com/valksor/go-toolkit/providerconfig"
	"github.com/valksor/go-toolkit/pullrequest"
	"github.com/valksor/go-toolkit/workunit"
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

	// Library system (documentation collections)
	library        *LibrarySystem
	libraryInitErr error // Stored initialization error for better UX

	// Workflow plugin adapters (for lifecycle management)
	workflowAdapters []*plugin.WorkflowAdapter

	// Current state
	activeTask         *storage.ActiveTask
	taskWork           *storage.TaskWork
	planningInProgress bool // Guard against concurrent/sequential planning calls

	// Configuration
	opts Options

	// Active agent
	activeAgent     agent.Agent
	taskAgentConfig *workunit.AgentConfig // Agent config from task source (if any)
	agentOverride   string                // Temporary agent override (for Web UI API)

	// Last PR creation result, accessible after Finish().
	lastPRResult *pullrequest.PullRequest

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
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.workspace
}

// GetGit returns the git instance.
func (c *Conductor) GetGit() *vcs.Git {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.git
}

// CodeDir returns the directory where code lives (for agents, linters, git operations).
// Uses workspace.CodeRoot() when available, falls back to git root, then opts.WorkDir.
func (c *Conductor) CodeDir() string {
	if c.workspace != nil {
		return c.workspace.CodeRoot()
	}
	if c.git != nil {
		return c.git.Root()
	}

	return c.opts.WorkDir
}

// TasksDir returns the directory for storing task source files.
// Respects storage.save_in_project config for consistent storage location.
func (c *Conductor) TasksDir() string {
	if c.workspace == nil {
		return ""
	}

	cfg, _ := c.workspace.LoadConfig()

	return c.workspace.TasksDir(cfg)
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

// ClearStaleTask clears an active task that no longer has valid work data.
// This handles the case where task files were deleted externally while the
// server was running. Returns true if a stale task was cleared.
// Only clears the task if the work directory is confirmed missing (os.ErrNotExist),
// not on transient errors like permission denied or I/O errors.
func (c *Conductor) ClearStaleTask() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return false
	}

	// Check if work directory still exists
	if c.workspace != nil {
		_, err := c.workspace.LoadWork(c.activeTask.ID)
		if err == nil {
			// Work exists, task is not stale
			return false
		}

		// Only clear if work is definitely missing, not on transient errors
		if !errors.Is(err, os.ErrNotExist) {
			// Transient error (permission denied, I/O error, etc.) - don't clear
			slog.Warn("unable to verify task work, keeping task active",
				"id", c.activeTask.ID, "error", err)

			return false
		}
	}

	// Work doesn't exist - clear stale state
	slog.Info("clearing stale active task", "id", c.activeTask.ID)

	if c.workspace != nil {
		if err := c.workspace.ClearActiveTask(); err != nil {
			slog.Warn("failed to clear active task file", "id", c.activeTask.ID, "error", err)
		}
	}

	c.activeTask = nil
	c.taskWork = nil

	return true
}

// syncActiveTaskLocked reconciles in-memory active task state with the on-disk
// .active_task file. This handles the case where another process (e.g., CLI)
// modifies the active task while the Web UI server is running.
//
// Must be called with c.mu held in write mode (Lock, not RLock).
func (c *Conductor) syncActiveTaskLocked() {
	if c.workspace == nil {
		return
	}

	diskHasTask := c.workspace.HasActiveTask()

	// Case 1: Task abandoned/finished externally (file deleted, memory still has it)
	if !diskHasTask && c.activeTask != nil {
		slog.Info("syncing stale active task: cleared externally", "id", c.activeTask.ID)
		c.activeTask = nil
		c.taskWork = nil
		c.machine.Reset()

		return
	}

	// Case 2: Task started externally (file exists, memory has nothing)
	if diskHasTask && c.activeTask == nil {
		active, err := c.workspace.LoadActiveTask()
		if err != nil {
			slog.Warn("sync active task: failed to load from disk", "error", err)

			return
		}

		slog.Info("syncing active task: loaded from disk", "id", active.ID)

		c.activeTask = active

		work, err := c.workspace.LoadWork(active.ID)
		if err == nil {
			c.taskWork = work
			c.machine.SetWorkUnit(c.buildWorkUnit())
		}

		return
	}

	// Case 3: Both exist but IDs differ (task changed externally)
	if diskHasTask && c.activeTask != nil {
		active, err := c.workspace.LoadActiveTask()
		if err != nil {
			slog.Warn("sync active task: failed to reload from disk", "error", err)

			return
		}

		if active.ID != c.activeTask.ID {
			slog.Info("syncing active task: changed externally",
				"old_id", c.activeTask.ID, "new_id", active.ID)

			c.activeTask = active
			c.machine.Reset()

			work, err := c.workspace.LoadWork(active.ID)
			if err == nil {
				c.taskWork = work
				c.machine.SetWorkUnit(c.buildWorkUnit())
			} else {
				c.taskWork = nil
			}
		}
	}
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

// dispatchWithRetry attempts a state machine transition with one retry.
// If both attempts fail, it transitions to StateFailed for user recovery via 'mehr reset'.
// Returns nil on success, error on failure (after transitioning to failed state).
func (c *Conductor) dispatchWithRetry(ctx context.Context, event workflow.Event) error {
	err := c.machine.Dispatch(ctx, event)
	if err == nil {
		return nil
	}

	// Retry once after a brief pause
	slog.Warn("state dispatch failed, retrying", "event", event, "error", err)
	time.Sleep(100 * time.Millisecond)

	err = c.machine.Dispatch(ctx, event)
	if err == nil {
		return nil
	}

	// Both attempts failed - transition to visible error state for user recovery
	slog.Error("state dispatch failed after retry", "event", event, "error", err)

	// Attempt to reach failed state (recoverable via 'mehr reset')
	_ = c.machine.Dispatch(ctx, workflow.EventAbort)

	// Sync activeTask state if we have one
	if c.activeTask != nil {
		c.activeTask.State = string(c.machine.State())
		if saveErr := c.workspace.SaveActiveTask(c.activeTask); saveErr != nil {
			slog.Error("failed to save task state after dispatch error", "error", saveErr)
		}
	}

	return fmt.Errorf("state transition failed (use 'mehr reset' to recover): %w", err)
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

// SetAgent sets a temporary agent override for the next operation.
// This is used by the Web UI to allow per-request agent selection.
// The override is cleared after the operation completes (call ClearAgent).
func (c *Conductor) SetAgent(agent string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.agentOverride = agent
}

// ClearAgent clears the temporary agent override.
// Should be called after the operation that used the override.
func (c *Conductor) ClearAgent() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.agentOverride = ""
}

// ─────────────────────────────────────────────────────────────────────────────
// Methods for taskrunner.ConductorRef interface support
// ─────────────────────────────────────────────────────────────────────────────

// AddNote adds a note to the active task's notes.md file.
// This method is used by the taskrunner to send messages to running tasks.
func (c *Conductor) AddNote(ctx context.Context, message string) error {
	c.mu.RLock()
	activeTask := c.activeTask
	workspace := c.workspace
	c.mu.RUnlock()

	if activeTask == nil {
		return errors.New("no active task")
	}

	if workspace == nil {
		return errors.New("workspace not initialized")
	}

	return workspace.AppendNote(activeTask.ID, message, activeTask.State)
}

// GetTaskID returns the active task's ID.
// Returns empty string if no task is active.
func (c *Conductor) GetTaskID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.activeTask == nil {
		return ""
	}

	return c.activeTask.ID
}

// GetWorktreePath returns the worktree path if using worktrees.
// Returns empty string if not using worktrees.
func (c *Conductor) GetWorktreePath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.activeTask == nil {
		return ""
	}

	return c.activeTask.WorktreePath
}

// LastPRResult returns the number and URL of the last PR created by Finish().
// Returns (0, "") if no PR was created.
func (c *Conductor) LastPRResult() (int, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastPRResult != nil {
		return c.lastPRResult.Number, c.lastPRResult.URL
	}

	return 0, ""
}

// Close performs cleanup for the conductor.
// This should be called when the conductor is no longer needed.
func (c *Conductor) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Flush any pending usage data
	if c.workspace != nil {
		_ = c.workspace.FlushUsage()
	}

	// Disconnect browser if initialized
	if c.browser != nil {
		_ = c.browser.Disconnect()
	}

	// Plugin adapters don't have cleanup - they share processes with the registry

	return nil
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
	providerCfg := providerconfig.NewConfig()
	if opts.Token != "" {
		providerCfg = providerCfg.Set("token", opts.Token)
	}

	instance, err := factory(ctx, providerCfg)
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}

	// 3. Check if provider implements required interfaces
	prFetcher, ok := instance.(pullrequest.PRFetcher)
	if !ok {
		return nil, fmt.Errorf("provider %q does not implement PRFetcher", opts.Provider)
	}

	prCommenter, ok := instance.(pullrequest.PRCommenter)
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
	var ourPreviousComment *workunit.Comment

	if prCommentFetcher, ok := instance.(pullrequest.PRCommentFetcher); ok {
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

	// 7. Get agent for review (prefer step-based resolution)
	var agentInst agent.Agent
	agentInst, err = c.GetAgentForStep(ctx, workflow.StepPRReview)
	if err != nil {
		return nil, fmt.Errorf("get agent for pr_review step: %w", err)
	}
	c.logVerbosef("Using agent for pr_review step")

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
		if prCommentUpdater, ok := instance.(pullrequest.PRCommentUpdater); ok {
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
