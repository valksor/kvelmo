package conductor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/valksor/kvelmo/pkg/git"
	"github.com/valksor/kvelmo/pkg/memory"
	"github.com/valksor/kvelmo/pkg/storage"
	"github.com/valksor/kvelmo/pkg/worker"
)

func (c *Conductor) watchJob(ctx context.Context, jobID string, completionEvent Event) {
	if c.pool == nil {
		return
	}

	stream := c.pool.Stream(jobID)
	if stream == nil {
		return
	}

	// Create pre-job safety checkpoint so user can undo if the job fails or crashes
	c.mu.Lock()
	if c.workUnit != nil {
		workDir := c.getWorkDir()
		if repo, err := git.Open(workDir); err == nil {
			if err := repo.StageAll(ctx); err == nil {
				if hasChanges, _ := repo.HasUncommittedChanges(ctx); hasChanges {
					if sha, commitErr := repo.Commit(ctx, fmt.Sprintf("kvelmo: pre-%s checkpoint", completionEvent)); commitErr == nil {
						c.workUnit.Checkpoints = append(c.workUnit.Checkpoints, sha)
						slog.Info("pre-job checkpoint created", "sha", sha, "event", completionEvent)
					}
				}
			}
		}
	}
	c.mu.Unlock()

	for event := range stream {
		// Forward streaming events
		c.emit(ConductorEvent{
			Type:    "job_output",
			JobID:   jobID,
			Message: event.Content,
		})

		if event.Type == "job_completed" {
			c.mu.Lock()
			c.activeJobID = "" // Clear active job on completion
			var (
				wuSnapshot *WorkUnit
				indexer    *memory.Indexer
			)
			if c.workUnit != nil {
				// For planning jobs, detect newly written specification files
				if completionEvent == EventPlanDone {
					c.detectSpecificationFiles()
				}

				// Create checkpoint after job completion
				// Use work directory (isolated worktree if active, main worktree otherwise)
				workDir := c.getWorkDir()
				if repo, err := git.Open(workDir); err != nil {
					slog.Debug("checkpoint: git open failed", "error", err, "workDir", workDir)
				} else {
					// Stage all changes first
					if stageErr := repo.StageAll(ctx); stageErr != nil {
						slog.Warn("checkpoint: stage failed", "error", stageErr, "workDir", workDir)
					} else if hasChanges, _ := repo.HasUncommittedChanges(ctx); hasChanges {
						sha, commitErr := repo.Commit(ctx, fmt.Sprintf("kvelmo: %s complete", completionEvent))
						if commitErr == nil {
							c.workUnit.Checkpoints = append(c.workUnit.Checkpoints, sha)
							slog.Info("checkpoint created", "sha", sha, "event", completionEvent)
						} else {
							slog.Warn("checkpoint: commit failed", "error", commitErr, "workDir", workDir)
						}
					} else {
						// No uncommitted changes - but Claude may have committed during the job.
						// Capture the current HEAD if it's not already in checkpoints.
						if headSHA, headErr := repo.CurrentCommit(ctx); headErr == nil && headSHA != "" {
							isNew := true
							for _, cp := range c.workUnit.Checkpoints {
								if cp == headSHA {
									isNew = false

									break
								}
							}
							if isNew {
								c.workUnit.Checkpoints = append(c.workUnit.Checkpoints, headSHA)
								slog.Info("checkpoint captured (agent commit)", "sha", headSHA, "event", completionEvent)
							} else {
								slog.Debug("checkpoint: no new commits", "event", completionEvent)
							}
						}
					}
				}

				// Dispatch completion event
				_ = c.machine.Dispatch(ctx, completionEvent)

				// Persist updated state (new checkpoint + new state)
				c.persistState()

				// Capture snapshot for async memory indexing (only for major phases)
				if completionEvent == EventPlanDone || completionEvent == EventImplementDone {
					wuSnapshot = c.workUnit
					indexer = c.memoryIndexer
				}
			}

			// Detect base branch while we still have context.
			baseBranch, baseBranchErr := c.getBaseBranch(ctx)
			c.mu.Unlock()

			// Trigger async memory indexing so it does not block the workflow.
			// Use detached context so indexing continues even if parent ctx is cancelled.
			// Skip indexing if base branch detection failed (non-critical).
			if baseBranchErr != nil {
				slog.Debug("skipping memory indexing - cannot detect base branch", "error", baseBranchErr)
			} else if indexer != nil && wuSnapshot != nil {
				//nolint:contextcheck // Intentionally uses detached context for background indexing
				go func(wu *WorkUnit, idx *memory.Indexer, event Event, base string) {
					asyncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					if err := idx.IndexTask(asyncCtx, wu.ID, wu.Title, wu.Description, wu.Branch, base); err != nil {
						slog.Warn("memory indexing failed", "task_id", wu.ID, "event", event, "error", err)
					}
				}(wuSnapshot, indexer, completionEvent, baseBranch)
			}

			return
		}

		if event.Type == "job_failed" {
			c.mu.Lock()
			c.activeJobID = "" // Clear active job on failure
			// Capture any partial work the agent completed before crashing
			if c.workUnit != nil {
				workDir := c.getWorkDir()
				if repo, err := git.Open(workDir); err == nil {
					if stageErr := repo.StageAll(ctx); stageErr == nil {
						if hasChanges, _ := repo.HasUncommittedChanges(ctx); hasChanges {
							if sha, commitErr := repo.Commit(ctx, fmt.Sprintf("kvelmo: partial work before %s failure", completionEvent)); commitErr == nil {
								c.workUnit.Checkpoints = append(c.workUnit.Checkpoints, sha)
								slog.Info("partial work checkpoint saved", "sha", sha, "event", completionEvent)
							}
						}
					}
				}
			}
			_ = c.machine.Dispatch(ctx, EventError)
			c.persistState()
			c.mu.Unlock()

			c.emit(ConductorEvent{
				Type:    "job_failed",
				JobID:   jobID,
				Error:   event.Content,
				Message: "Job failed",
			})

			// Also emit enriched error for user-facing context
			if event.Content != "" {
				c.emitEnrichedError(fmt.Errorf("%s", event.Content), string(completionEvent))
			}

			return
		}
	}
}

// saveJobSession persists a session entry for the given job so that it can be
// resumed later.  This is a best-effort operation; errors are logged and ignored.
func (c *Conductor) saveJobSession(jobID, phase, agentType string) {
	if c.store == nil || c.workUnit == nil {
		return
	}
	sessStore := storage.NewSessionStore(c.store)
	entry := storage.SessionEntry{
		SessionID: jobID,
		AgentType: agentType,
		TaskID:    c.workUnit.ID,
		Phase:     phase,
	}
	if err := sessStore.SaveSession(entry); err != nil {
		slog.Warn("persist session failed", "task_id", c.workUnit.ID, "phase", phase, "error", err)
	}
}

func (c *Conductor) generateBranchName(wu *WorkUnit) string {
	effectiveSettings := c.getEffectiveSettings()
	pattern := effectiveSettings.Git.BranchPattern
	if pattern == "" {
		pattern = "feature/{key}--{slug}"
	}

	// Determine key
	key := wu.ID
	if wu.ExternalID != "" {
		key = wu.ExternalID
	}

	// Generate slug from title
	slug := slugify(wu.Title)

	// Determine type (provider name or "local")
	taskType := "local"
	if wu.Source != nil {
		taskType = wu.Source.Provider
	}

	// Interpolate variables
	result := pattern
	if key == "" {
		// Remove {key}-- or {key}- patterns when key is empty to avoid leading dashes
		result = strings.ReplaceAll(result, "{key}--", "")
		result = strings.ReplaceAll(result, "{key}-", "")
	}
	result = strings.ReplaceAll(result, "{key}", key)
	result = strings.ReplaceAll(result, "{slug}", slug)
	result = strings.ReplaceAll(result, "{type}", taskType)

	// Clean up: collapse multiple dashes and remove trailing dashes
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	result = strings.TrimRight(result, "-")

	return result
}

// slugify converts a string to a URL-safe slug.
func slugify(s string) string {
	// Lowercase
	s = strings.ToLower(s)
	// Replace spaces and underscores with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	// Collapse multiple hyphens
	slug := result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	// Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")
	// Limit length (use runes for UTF-8 safety)
	runes := []rune(slug)
	if len(runes) > 50 {
		slug = string(runes[:50])
		// Don't end with hyphen
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}

// shouldPostTicketComment checks if ticket comments are enabled for the current provider.
func (c *Conductor) shouldPostTicketComment() bool {
	if c.workUnit == nil || c.workUnit.Source == nil {
		return false
	}

	effectiveSettings := c.getEffectiveSettings()

	switch c.workUnit.Source.Provider {
	case "github":
		return effectiveSettings.Providers.GitHub.AllowTicketComment
	case "gitlab":
		return effectiveSettings.Providers.GitLab.AllowTicketComment
	case "wrike":
		return effectiveSettings.Providers.Wrike.AllowTicketComment
	case "linear":
		return effectiveSettings.Providers.Linear.AllowTicketComment
	default:
		return false
	}
}

// buildJobOptions creates JobOptions with execution context for multi-project support.
// This ensures jobs carry full context (WorkDir, metadata) so any worker can execute them.
func (c *Conductor) buildJobOptions() *worker.JobOptions {
	opts := &worker.JobOptions{
		WorkDir:  c.getWorkDir(), // Use isolated worktree if available
		Metadata: make(map[string]any),
	}

	// Add task metadata
	if c.workUnit != nil {
		opts.Metadata["task_id"] = c.workUnit.ID
		opts.Metadata["task_title"] = c.workUnit.Title
		if c.workUnit.ExternalID != "" {
			opts.Metadata["external_id"] = c.workUnit.ExternalID
		}
		if c.workUnit.Source != nil {
			opts.Metadata["provider"] = c.workUnit.Source.Provider
			opts.Metadata["reference"] = c.workUnit.Source.Reference
		}
	}

	return opts
}

// detectSpecificationFiles scans for specification files and adds any new ones
// to the work unit's Specifications list. Uses the storage layer path which
// respects the saveInProject config setting.
// Must be called with c.mu held.
func (c *Conductor) detectSpecificationFiles() {
	if c.workUnit == nil || c.store == nil {
		return
	}

	// Build set of known specs for quick lookup (normalized for deduplication)
	known := make(map[string]bool)
	for _, sp := range c.workUnit.Specifications {
		known[filepath.Clean(sp)] = true
	}

	specDir := c.store.SpecificationsDir(c.workUnit.ID)
	entries, err := os.ReadDir(specDir)
	if err != nil {
		// Directory may not exist yet
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "specification-") || !strings.HasSuffix(name, ".md") {
			continue
		}
		fullPath := filepath.Join(specDir, name)
		if !known[filepath.Clean(fullPath)] {
			c.workUnit.Specifications = append(c.workUnit.Specifications, fullPath)
			slog.Info("detected new specification file", "path", fullPath)
		}
	}
}
