package conductor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/valksor/kvelmo/pkg/memory"
	"github.com/valksor/kvelmo/pkg/provider"
	"github.com/valksor/kvelmo/pkg/worker"
)

// Submit submits the task to the provider (creates PR, updates issue, etc).
// The lock is released during network operations to avoid blocking other callers.
// State transition happens AFTER successful operations to avoid terminal state on failure.
func (c *Conductor) Submit(ctx context.Context, deleteBranch bool) error {
	// Phase 1: Pre-flight checks and validate state transition is possible
	c.mu.Lock()
	if c.workUnit == nil {
		c.mu.Unlock()

		return errors.New("no task loaded")
	}

	// Check quality gate - use cached result from Review() if available,
	// otherwise run synchronously (when Review was skipped)
	if c.workUnit.QualityGatePassed != nil {
		// Use cached result from async quality gate
		if !*c.workUnit.QualityGatePassed {
			errMsg := c.workUnit.QualityGateError
			c.mu.Unlock()

			return fmt.Errorf("quality gate failed: %s", errMsg)
		}
		slog.Debug("quality gate passed (cached)")
	} else {
		// No cached result - run synchronously (Review was skipped or old state)
		if err := c.runQualityGate(ctx); err != nil {
			c.mu.Unlock()

			return fmt.Errorf("quality gate failed: %w", err)
		}
	}

	// Verify state transition is possible before starting network operations.
	// Don't dispatch yet - we dispatch after success to avoid terminal state on failure.
	if can, reason := c.machine.CanDispatch(ctx, EventSubmit); !can {
		c.mu.Unlock()

		return fmt.Errorf("cannot submit: %s", reason)
	}

	// Phase 2: Copy state needed for network operations
	branch := c.workUnit.Branch
	title := c.workUnit.Title
	externalID := c.workUnit.ExternalID
	worktreePath := c.workUnit.WorktreePath
	workUnitDescription := c.workUnit.Description
	specCount := len(c.workUnit.Specifications)
	checkpointCount := len(c.workUnit.Checkpoints)
	var sourceProvider, sourceURL string
	if c.workUnit.Source != nil {
		sourceProvider = c.workUnit.Source.Provider
		sourceURL = c.workUnit.Source.URL
	}
	git := c.git
	providers := c.providers
	memoryIndexer := c.memoryIndexer
	lifecycleCtx := c.lifecycleCtx
	shouldComment := c.shouldPostTicketComment()
	c.mu.Unlock()

	// Phase 3: Network operations (no lock held)
	var prURL string
	if git != nil && branch != "" {
		if err := git.Push(ctx, "origin", branch); err != nil {
			return fmt.Errorf("push branch %s: %w", branch, err)
		}

		// Create PR via provider if supported
		if sourceProvider != "" && providers != nil {
			if p, err := providers.Get(sourceProvider); err == nil {
				if sp, ok := p.(provider.SubmitProvider); ok {
					// Get base branch (configured or auto-detected)
					baseBranch, err := c.getBaseBranch(ctx)
					if err != nil {
						return fmt.Errorf("determine base branch for PR: %w", err)
					}

					prOpts := provider.PROptions{
						Title:   "[kvelmo] " + title,
						Body:    buildPRDescription(workUnitDescription, specCount, checkpointCount),
						Head:    branch,
						Base:    baseBranch,
						TaskID:  externalID,
						TaskURL: sourceURL,
					}
					if result, err := sp.CreatePR(ctx, prOpts); err == nil {
						prURL = result.URL
						c.logVerbosef("Created PR: %s", prURL)
						// Add comment linking to PR on original task (if enabled)
						if shouldComment {
							if err := sp.AddComment(ctx, externalID,
								"Pull request created: "+prURL); err != nil {
								c.logVerbosef("Warning: could not add comment: %v", err)
							}
						}
					} else {
						// PR creation failed - state remains in StateReviewing (not terminal)
						slog.Error("failed to create PR", "error", err, "branch", branch)
						c.emit(ConductorEvent{
							Type:    "pr_creation_failed",
							Error:   err.Error(),
							Message: "Failed to create pull request",
						})

						return fmt.Errorf("create PR: %w", err)
					}
				}
			}
		}

		// Delete local branch after successful submission if requested
		if deleteBranch {
			if err := git.DeleteBranch(ctx, branch); err != nil {
				c.logVerbosef("Warning: could not delete branch: %v", err)
			}
		}
	}

	// Phase 4: State transition - only after all critical operations succeed.
	// This ensures we don't end up in terminal StateSubmitted on failure.
	if err := c.machine.Dispatch(ctx, EventSubmit); err != nil {
		// This shouldn't fail since we checked CanDispatch earlier, but handle it
		return fmt.Errorf("state transition failed: %w", err)
	}

	// Remove git worktree if isolation was used (branch has the changes now)
	if worktreePath != "" && git != nil {
		if err := git.RemoveWorktree(ctx, worktreePath, false); err != nil {
			c.logVerbosef("Warning: could not remove worktree %s: %v", worktreePath, err)
		}
	}

	// Phase 5: Re-acquire lock to persist state
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear worktree path so we don't try again
	if c.workUnit != nil && c.workUnit.WorktreePath != "" {
		c.workUnit.WorktreePath = ""
	}

	// Build event data
	eventData, err := json.Marshal(map[string]any{
		"pr_url": prURL,
	})
	if err != nil {
		slog.Warn("marshal event data failed", "error", err)
	}

	c.persistState()

	c.emit(ConductorEvent{
		Type:    "task_submitted",
		State:   c.machine.State(),
		Message: "Task submitted",
		Data:    eventData,
	})

	// Trigger async memory indexing for submitted task.
	// Use lifecycle context (not request ctx which may be cancelled when handler returns).
	// Get base branch BEFORE goroutine (ctx may be cancelled when handler returns).
	if memoryIndexer != nil && c.workUnit != nil {
		baseBranch, err := c.getBaseBranch(ctx)
		if err != nil {
			c.logVerbosef("Warning: cannot index task - %v", err)
		} else {
			//nolint:contextcheck // intentionally uses lifecycle context for background indexing
			go func(wu *WorkUnit, idx *memory.Indexer, lctx context.Context, base string) {
				if err := idx.IndexTask(lctx, wu.ID, wu.Title, wu.Description, wu.Branch, base); err != nil {
					slog.Warn("memory indexing failed after submit", "task_id", wu.ID, "error", err)
				}
			}(c.workUnit, memoryIndexer, lifecycleCtx, baseBranch)
		}
	}

	return nil
}

// Abandon stops any running jobs, optionally deletes the branch, and resets state.
func (c *Conductor) Abandon(ctx context.Context, keepBranch bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Stop tracking jobs for this worktree (jobs may continue running in pool)
	//nolint:godox // Known limitation - tracked for future enhancement
	// TODO: Add pool.CancelJob() for true cancellation
	if c.pool != nil && c.workUnit != nil {
		for _, jobID := range c.workUnit.Jobs {
			job := c.pool.GetJob(jobID)
			if job != nil && (job.Status == worker.JobStatusQueued || job.Status == worker.JobStatusInProgress) {
				c.logVerbosef("Orphaning job (still running in pool): %s", jobID)
			}
		}
	}

	// Delete branch unless keep_branch is set
	if !keepBranch && c.git != nil && c.workUnit != nil && c.workUnit.Branch != "" {
		if err := c.git.DeleteBranch(ctx, c.workUnit.Branch); err != nil {
			c.logVerbosef("Warning: could not delete branch %s: %v", c.workUnit.Branch, err)
		}
	}

	// Remove git worktree if isolation was used
	if c.workUnit != nil && c.workUnit.WorktreePath != "" && c.git != nil {
		if err := c.git.RemoveWorktree(ctx, c.workUnit.WorktreePath, true); err != nil {
			c.logVerbosef("Warning: could not remove worktree %s: %v", c.workUnit.WorktreePath, err)
		}
	}

	// Delete persisted task state so it is not restored on next socket start.
	if c.store != nil && c.workUnit != nil {
		if err := c.store.DeleteTaskState(c.workUnit.ID); err != nil {
			slog.Warn("delete task state failed", "task_id", c.workUnit.ID, "error", err)
		}
	}

	// Reset state and clear work unit
	c.machine.Reset()
	c.workUnit = nil

	c.emit(ConductorEvent{
		Type:    "task_abandoned",
		State:   StateNone,
		Message: "Task abandoned",
	})

	return nil
}

// Delete clears the work unit when in a terminal or none state.
func (c *Conductor) Delete(ctx context.Context, deleteBranch bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.machine.State()
	if state != StateSubmitted && state != StateFailed && state != StateNone {
		return fmt.Errorf("delete only allowed in terminal states (submitted, failed, none); current state: %s", state)
	}

	// Optionally delete the branch
	if deleteBranch && c.git != nil && c.workUnit != nil && c.workUnit.Branch != "" {
		if err := c.git.DeleteBranch(ctx, c.workUnit.Branch); err != nil {
			c.logVerbosef("Warning: could not delete branch %s: %v", c.workUnit.Branch, err)
		}
	}

	// Remove git worktree if still present (cleanup safety net)
	if c.workUnit != nil && c.workUnit.WorktreePath != "" && c.git != nil {
		if err := c.git.RemoveWorktree(ctx, c.workUnit.WorktreePath, true); err != nil {
			c.logVerbosef("Warning: could not remove worktree %s: %v", c.workUnit.WorktreePath, err)
		}
	}

	// Delete persisted task state so it is not restored on next socket start.
	if c.store != nil && c.workUnit != nil {
		if err := c.store.DeleteTaskState(c.workUnit.ID); err != nil {
			slog.Warn("delete task state failed", "task_id", c.workUnit.ID, "error", err)
		}
	}

	// Reset state and clear work unit
	c.machine.Reset()
	c.workUnit = nil

	c.emit(ConductorEvent{
		Type:    "task_deleted",
		State:   StateNone,
		Message: "Task deleted",
	})

	return nil
}

// buildPRDescription constructs the PR body from task metadata.
// Takes explicit parameters so it can be called outside the lock with copied values.
func buildPRDescription(description string, specCount, checkpointCount int) string {
	desc := fmt.Sprintf("## Summary\n\n%s\n", description)

	if specCount > 0 {
		desc += "\n## Implementation\n\nImplemented according to kvelmo specifications.\n"
	}

	if checkpointCount > 0 {
		desc += fmt.Sprintf("\n## Checkpoints\n\n%d checkpoint(s) created during development.\n", checkpointCount)
	}

	desc += "\n---\n*Generated by [kvelmo](https://github.com/valksor/kvelmo)*"

	return desc
}
