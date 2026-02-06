package conductor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// ContinueWithExisting reuses an existing work directory for an updated task.
// This is used when a user wants to continue work on a previously finished task
// with new/updated content from the provider.
func (c *Conductor) ContinueWithExisting(ctx context.Context, reference string, existingTaskID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check for existing active task
	if c.activeTask != nil {
		return fmt.Errorf("task already active: %s (use 'task status' to check)", c.activeTask.ID)
	}

	// Fetch updated work unit from provider
	p, workUnit, err := c.fetchWorkUnit(ctx, reference)
	if err != nil {
		return fmt.Errorf("fetch updated work unit: %w", err)
	}

	// Merge local metadata if a matching queue task exists
	var localQueueTask *storage.QueuedTask
	if workUnit.ExternalID != "" && c.workspace != nil {
		var queueErr error
		localQueueTask, queueErr = c.workspace.FindQueueTaskByExternalID(workUnit.ExternalID)
		if queueErr != nil {
			slog.Warn("search local queues for metadata enrichment", "external_id", workUnit.ExternalID, "error", queueErr)
		}
		if localQueueTask != nil {
			c.mergeLocalMetadata(workUnit, localQueueTask)
		}
	}

	// Capture task agent config from workUnit
	c.taskAgentConfig = workUnit.AgentConfig

	// Load existing work
	work, err := c.workspace.LoadWork(existingTaskID)
	if err != nil {
		return fmt.Errorf("load existing work: %w", err)
	}

	// Snapshot the updated source
	snapshot := c.snapshotSource(ctx, p, reference, workUnit)

	// Merge local source files into snapshot if available
	if localQueueTask != nil && localQueueTask.SourcePath != "" {
		c.mergeLocalSourceIntoSnapshot(snapshot, localQueueTask.SourcePath)
	}

	// Write updated source files to existing directory
	if err := c.writeSourceFiles(existingTaskID, snapshot); err != nil {
		return fmt.Errorf("write updated source files: %w", err)
	}

	// Update source info in work
	sourceInfo := c.buildSourceInfo(snapshot)
	work.Source = sourceInfo

	// Update metadata from work unit
	work.Metadata.Title = workUnit.Title
	if workUnit.ExternalKey != "" {
		work.Metadata.ExternalKey = workUnit.ExternalKey
	}

	// Resolve and update agent for this task
	agentInst, agentSource, err := c.resolveAgentForTask()
	if err != nil {
		return fmt.Errorf("resolve agent: %w", err)
	}
	c.activeAgent = agentInst
	work.Agent = storage.AgentInfo{
		Name:   agentInst.Name(),
		Source: agentSource,
	}
	if c.taskAgentConfig != nil && len(c.taskAgentConfig.Env) > 0 {
		work.Agent.InlineEnv = c.taskAgentConfig.Env
	}

	// Save updated work
	if err := c.workspace.SaveWork(work); err != nil {
		return fmt.Errorf("save updated work: %w", err)
	}

	// Create active task reference with state="idle"
	cfg, _ := c.workspace.LoadConfig()
	if cfg == nil {
		cfg = storage.NewDefaultWorkspaceConfig()
	}
	active := storage.NewActiveTask(existingTaskID, reference, c.workspace.EffectiveWorkDir(existingTaskID, cfg))

	// Preserve git info if it exists
	if c.git != nil {
		active.UseGit = true
		if work.Git.Branch != "" {
			active.Branch = work.Git.Branch
		}
		if work.Git.WorktreePath != "" {
			active.WorktreePath = work.Git.WorktreePath
		}
	}

	// Save active task
	if err := c.workspace.SaveActiveTask(active); err != nil {
		return fmt.Errorf("save active task: %w", err)
	}

	c.activeTask = active
	c.taskWork = work

	// Set up state machine with idle state
	c.machine.SetWorkUnit(c.buildWorkUnit())

	c.publishProgress("Resumed with existing work directory", 100)

	return nil
}

// Resume loads an existing active task.
func (c *Conductor) Resume(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.workspace.HasActiveTask() {
		return errors.New("no active task")
	}

	active, err := c.workspace.LoadActiveTask()
	if err != nil {
		return fmt.Errorf("load active task: %w", err)
	}

	work, err := c.workspace.LoadWork(active.ID)
	if err != nil {
		return fmt.Errorf("load work: %w", err)
	}

	c.activeTask = active
	c.taskWork = work
	c.machine.SetWorkUnit(c.buildWorkUnit())

	return nil
}

// Delete abandons the current task without merging.
func (c *Conductor) Delete(ctx context.Context, opts DeleteOptions) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return errors.New("no active task")
	}

	taskID := c.activeTask.ID

	// Handle git operations if applicable
	if c.git != nil && c.activeTask.UseGit && c.activeTask.Branch != "" && !opts.KeepBranch {
		currentBranch, _ := c.git.CurrentBranch(ctx)
		taskBranch := c.activeTask.Branch
		worktreePath := c.activeTask.WorktreePath

		// NOTE: Cleanup errors below are logged but not returned intentionally.
		// Delete operation should succeed even if cleanup partially fails.
		// This is best-effort cleanup that should not block task deletion.

		// If using worktree, remove it first
		if worktreePath != "" {
			if err := c.git.RemoveWorktree(ctx, worktreePath, true); err != nil {
				c.logError(fmt.Errorf("remove worktree: %w", err))
			}
		} else if currentBranch == taskBranch {
			// If we're on the task branch (not worktree), switch to base branch first
			var baseBranch string
			if c.taskWork != nil && c.taskWork.Git.BaseBranch != "" {
				baseBranch = c.taskWork.Git.BaseBranch
			} else {
				var err error
				baseBranch, err = c.git.GetBaseBranch(ctx)
				if err != nil {
					return fmt.Errorf("get base branch: %w", err)
				}
			}

			if err := c.git.Checkout(ctx, baseBranch); err != nil {
				return fmt.Errorf("checkout base branch: %w", err)
			}
		}

		// Checkpoint deletion is best-effort; ignore errors
		_ = c.git.DeleteAllCheckpoints(ctx, taskID)

		// Delete the branch
		if err := c.git.DeleteBranch(ctx, taskBranch, true); err != nil {
			c.logError(fmt.Errorf("delete branch: %w", err))
		}
	}

	// Delete work directory based on: CLI flag > config > default (delete)
	var shouldDelete bool
	if opts.DeleteWork != nil {
		shouldDelete = *opts.DeleteWork // CLI explicitly set
	} else {
		cfg, _ := c.workspace.LoadConfig()              // ignore error, use defaults
		shouldDelete = cfg.Workflow.DeleteWorkOnAbandon // default: true
	}
	if shouldDelete {
		if err := c.workspace.DeleteWork(taskID); err != nil {
			c.logError(fmt.Errorf("delete work directory: %w", err))
		}
	}

	// Clear active task
	if err := c.workspace.ClearActiveTask(); err != nil {
		c.logError(fmt.Errorf("clear active task: %w", err))
	}

	c.activeTask = nil
	c.taskWork = nil

	c.publishProgress("Task deleted", 100)

	return nil
}

// Status returns the current task status.
func (c *Conductor) Status(ctx context.Context) (*TaskStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.activeTask == nil {
		return nil, errors.New("no active task")
	}

	// Count specifications - errors ignored; empty list is acceptable for status display
	specifications, _ := c.workspace.ListSpecifications(c.activeTask.ID)

	status := &TaskStatus{
		TaskID:         c.activeTask.ID,
		State:          c.activeTask.State,
		Ref:            c.activeTask.Ref,
		Branch:         c.activeTask.Branch,
		WorktreePath:   c.activeTask.WorktreePath,
		Specifications: len(specifications),
		Checkpoints:    c.countCheckpoints(ctx),
		Started:        c.activeTask.Started,
	}

	// Add work metadata if available (may be nil if work directory is missing)
	if c.taskWork != nil {
		status.Title = c.taskWork.Metadata.Title
		status.ExternalKey = c.taskWork.Metadata.ExternalKey
	}

	// Add agent info
	if c.taskWork != nil && c.taskWork.Agent.Name != "" {
		status.Agent = c.taskWork.Agent.Name
		status.AgentSource = c.taskWork.Agent.Source
	} else if c.activeAgent != nil {
		status.Agent = c.activeAgent.Name()
		status.AgentSource = "auto"
	}

	return status, nil
}

// ListExistingWorkDirs returns all task IDs with existing work directories.
func (c *Conductor) ListExistingWorkDirs() ([]string, error) {
	return c.workspace.ListWorks()
}

// ArchiveWorkDir archives a specific work directory.
func (c *Conductor) ArchiveWorkDir(taskID string) error {
	return c.workspace.ArchiveWorkDir(taskID)
}

// TaskStatus represents the current task state.
type TaskStatus struct {
	TaskID         string
	Title          string
	ExternalKey    string // User-facing key (e.g., "FEATURE-123")
	State          string
	Ref            string
	Branch         string
	WorktreePath   string
	Specifications int
	Checkpoints    int
	Started        time.Time
	Agent          string // Agent name being used
	AgentSource    string // Where agent was configured from: "cli", "task", "workspace", "auto"
}
