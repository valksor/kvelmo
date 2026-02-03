package conductor

import (
	"context"
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/stack"
	"github.com/valksor/go-toolkit/eventbus"
)

// AutoRebaseResult contains the outcome of an auto-rebase attempt.
type AutoRebaseResult struct {
	Attempted    bool                 // Whether rebase was attempted
	Skipped      bool                 // Whether rebase was skipped (config, no children, etc.)
	SkipReason   string               // Reason for skipping
	Preview      *stack.RebasePreview // Preview results (nil if skipped before preview)
	Executed     bool                 // Whether rebase was actually executed
	Result       *stack.RebaseResult  // Rebase execution result (nil if not executed)
	UserDeclined bool                 // User declined confirmation
	HasConflicts bool                 // Preview showed conflicts
	Unavailable  bool                 // Conflict detection unavailable (Git too old)
}

// tryAutoRebase attempts to auto-rebase child tasks after PR creation.
// Returns nil if auto-rebase is not applicable or disabled.
// This method respects CLAUDE.md Tier 3 policy - rebase requires explicit user confirmation.
func (c *Conductor) tryAutoRebase(ctx context.Context, taskID string, opts FinishOptions) *AutoRebaseResult {
	result := &AutoRebaseResult{}

	// Check if explicitly disabled via flag
	if opts.SkipAutoRebase {
		result.Skipped = true
		result.SkipReason = "auto-rebase disabled via --no-auto-rebase flag"

		return result
	}

	// Load workspace config to check auto_rebase setting
	cfg, err := c.workspace.LoadConfig()
	if err != nil {
		c.logError(fmt.Errorf("load config for auto-rebase: %w", err))
		result.Skipped = true
		result.SkipReason = "failed to load config"

		return result
	}

	// Check config: auto_rebase must be "on_finish" or ForceAutoRebase flag must be set
	autoRebaseEnabled := cfg.Stack != nil && cfg.Stack.AutoRebase == "on_finish"
	if !autoRebaseEnabled && !opts.ForceAutoRebase {
		result.Skipped = true
		result.SkipReason = "auto-rebase not configured (set stack.auto_rebase: on_finish)"

		return result
	}

	// Non-interactive modes cannot use auto-rebase (requires user confirmation per Tier 3)
	if c.opts.AutoMode || c.opts.SkipAgentQuestions {
		result.Skipped = true
		result.SkipReason = "auto-rebase requires interactive mode for user confirmation"

		return result
	}

	// Check if git is available
	if c.git == nil {
		result.Skipped = true
		result.SkipReason = "git not available"

		return result
	}

	// Create stack storage and check if task is in a stack
	stackStorage := stack.NewStorage(c.workspace.Root())
	if err := stackStorage.Load(); err != nil {
		c.logError(fmt.Errorf("load stacks: %w", err))
		result.Skipped = true
		result.SkipReason = "failed to load stacks"

		return result
	}

	taskStack := stackStorage.GetStackByTask(taskID)
	if taskStack == nil {
		result.Skipped = true
		result.SkipReason = "task is not part of a stack"

		return result
	}

	// Check for children that might need rebasing
	children := taskStack.GetChildren(taskID)
	if len(children) == 0 {
		result.Skipped = true
		result.SkipReason = "no dependent tasks in stack"

		return result
	}

	// Note: We don't mark children as needing rebase here.
	// State mutation happens only AFTER successful rebase execution (below).
	// This ensures preview/confirmation failures don't leave state inconsistent.

	// Create rebaser and preview
	rebaser := stack.NewRebaser(stackStorage, c.git)
	preview, err := rebaser.PreviewRebase(ctx, taskStack.ID)
	if err != nil {
		c.logError(fmt.Errorf("preview rebase: %w", err))
		result.Attempted = true
		result.SkipReason = fmt.Sprintf("preview failed: %v", err)

		return result
	}

	result.Attempted = true
	result.Preview = preview

	// Display preview to user
	c.displayRebasePreview(preview)

	// Check for unavailable detection (Git too old)
	if preview.Unavailable {
		result.Unavailable = true
		c.publishStackMessage(fmt.Sprintf("Conflict detection unavailable: %s\nRun 'mehr stack rebase' manually to rebase children.", preview.UnavailableReason))

		return result
	}

	// Check for conflicts
	if preview.HasConflicts {
		result.HasConflicts = true

		// Respect block_on_conflicts config (default: true)
		blockOnConflicts := cfg.Stack == nil || cfg.Stack.BlockOnConflicts
		if blockOnConflicts {
			c.publishStackMessage(fmt.Sprintf("Auto-rebase blocked: %d task(s) have conflicts.\nResolve conflicts manually, then run 'mehr stack rebase'.", preview.ConflictCount))

			return result
		}

		// User configured block_on_conflicts: false - warn but allow proceeding
		c.publishStackMessage(fmt.Sprintf("⚠ Warning: %d task(s) have conflicts. Proceeding anyway (block_on_conflicts: false).", preview.ConflictCount))
	}

	// No tasks need rebasing
	if len(preview.Tasks) == 0 {
		result.Skipped = true
		result.SkipReason = "no tasks need rebasing"

		return result
	}

	// Prompt user for confirmation (Tier 3 policy compliance)
	if !c.askUserRebaseConfirmation(preview) {
		result.UserDeclined = true
		c.publishStackMessage("Auto-rebase skipped. Run 'mehr stack rebase' later if needed.")

		return result
	}

	// Execute rebase
	rebaseResult, err := rebaser.RebaseAll(ctx, taskStack.ID)
	if err != nil {
		c.logError(fmt.Errorf("execute rebase: %w", err))
		result.Result = rebaseResult

		return result
	}

	result.Executed = true
	result.Result = rebaseResult

	// NOW mark children state and save (only after successful rebase)
	taskStack.MarkChildrenNeedsRebase(taskID)
	if err := stackStorage.Save(); err != nil {
		c.logError(fmt.Errorf("save stack state after rebase: %w", err))
	}

	// Publish success event
	c.eventBus.PublishRaw(eventbus.Event{
		Type: events.TypeProgress,
		Data: map[string]any{
			"message": fmt.Sprintf("Auto-rebased %d task(s) successfully", len(rebaseResult.RebasedTasks)),
			"percent": 100,
		},
	})

	return result
}

// displayRebasePreview shows the rebase preview to the user.
func (c *Conductor) displayRebasePreview(preview *stack.RebasePreview) {
	if len(preview.Tasks) == 0 {
		return
	}

	var sb strings.Builder
	sb.WriteString("\n╭─ Stack Rebase Preview ─────────────────────────────╮\n")

	for _, task := range preview.Tasks {
		status := "✓ Safe"
		if task.Unavailable {
			status = "? Unknown"
		} else if task.WouldConflict {
			status = "✗ CONFLICT"
		}

		sb.WriteString(fmt.Sprintf("│ %-12s → %-12s  %s\n", task.Branch, task.OntoBase, status))

		// Show conflicting files if any
		if task.WouldConflict && len(task.ConflictingFiles) > 0 {
			for _, file := range task.ConflictingFiles {
				sb.WriteString(fmt.Sprintf("│   └─ %s\n", file))
			}
		}
	}

	sb.WriteString("╰────────────────────────────────────────────────────╯\n")

	if preview.HasConflicts {
		sb.WriteString(fmt.Sprintf("\n⚠ %d task(s) have conflicts. Manual resolution required.\n", preview.ConflictCount))
	} else if preview.Unavailable {
		sb.WriteString(fmt.Sprintf("\n⚠ Conflict detection unavailable: %s\n", preview.UnavailableReason))
	} else {
		sb.WriteString(fmt.Sprintf("\n✓ %d task(s) can be safely rebased.\n", preview.SafeCount))
	}

	// Write to stdout
	if c.opts.Stdout != nil {
		_, _ = fmt.Fprint(c.opts.Stdout, sb.String())
	}
}

// askUserRebaseConfirmation prompts the user to confirm auto-rebase.
// Returns true if user confirms, false otherwise.
func (c *Conductor) askUserRebaseConfirmation(preview *stack.RebasePreview) bool {
	if c.opts.Stdout != nil {
		_, _ = fmt.Fprintf(c.opts.Stdout, "\nRebase %d branch(es) onto their new bases? [y/N] ", preview.SafeCount)
	}

	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		// EOF or empty input - treat as "no"
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))

	return response == "y" || response == "yes"
}

// publishStackMessage publishes a message related to stack operations.
func (c *Conductor) publishStackMessage(message string) {
	c.eventBus.PublishRaw(eventbus.Event{
		Type: events.TypeProgress,
		Data: map[string]any{
			"message": message,
			"percent": 0,
		},
	})

	if c.opts.Stdout != nil {
		_, _ = fmt.Fprintln(c.opts.Stdout, message)
	}
}
