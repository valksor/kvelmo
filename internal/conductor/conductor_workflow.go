package conductor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/eventbus"
)

// Plan enters the planning phase to create specifications.
func (c *Conductor) Plan(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return errors.New("no active task")
	}

	// Sync notes into WorkUnit.Description for guard evaluation
	wu := c.machine.WorkUnit()
	if wu != nil && wu.Description == "" {
		if notes, err := c.workspace.ReadNotes(c.activeTask.ID); err == nil && notes != "" {
			wu.Description = notes
		}
	}

	// Dispatch planning event first to validate the transition before persisting.
	if err := c.machine.Dispatch(ctx, workflow.EventPlan); err != nil {
		if strings.Contains(err.Error(), "guards not satisfied") {
			return errors.New("cannot plan: task has no description\n\n" +
				"Use 'mehr note' to add a task description first:\n" +
				"  mehr note \"Implement feature X with REST API\"\n\n" +
				"Then run 'mehr plan' again")
		}

		return fmt.Errorf("enter planning: %w", err)
	}

	// Machine transitioned successfully — now persist state to match.
	c.activeTask.State = string(c.machine.State())
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		return fmt.Errorf("save active task: %w", err)
	}

	return nil
}

// Implement enters the implementation phase.
func (c *Conductor) Implement(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return errors.New("no active task")
	}

	// Check for specifications
	specifications, err := c.workspace.ListSpecifications(c.activeTask.ID)
	if err != nil {
		return fmt.Errorf("list specifications: %w", err)
	}
	if len(specifications) == 0 {
		return errors.New("no specifications found - run 'task plan' first")
	}

	// Filter by component if OnlyComponent is set
	if c.opts.OnlyComponent != "" {
		filtered := c.filterSpecificationsByComponent(c.activeTask.ID, specifications, c.opts.OnlyComponent)
		if len(filtered) == 0 {
			return fmt.Errorf("no specifications found for component: %s", c.opts.OnlyComponent)
		}
		c.logVerbosef("Filtered to %d specification(s) for component: %s", len(filtered), c.opts.OnlyComponent)
		specifications = filtered
	}

	// Update machine with specifications
	wu := c.machine.WorkUnit()
	if wu != nil {
		wu.Specifications = make([]string, len(specifications))
		for i, num := range specifications {
			wu.Specifications[i] = fmt.Sprintf("specification-%d.md", num)
		}
	}

	// Dispatch implement event first to validate the transition before persisting.
	if err := c.machine.Dispatch(ctx, workflow.EventImplement); err != nil {
		return fmt.Errorf("enter implementation: %w", err)
	}

	// Machine transitioned successfully — now persist state to match.
	c.activeTask.State = string(c.machine.State())
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		return fmt.Errorf("save active task: %w", err)
	}

	return nil
}

// Review enters the review phase.
func (c *Conductor) Review(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return errors.New("no active task")
	}

	// Dispatch review event first to validate the transition before persisting.
	if err := c.machine.Dispatch(ctx, workflow.EventReview); err != nil {
		return fmt.Errorf("enter review: %w", err)
	}

	// Machine transitioned successfully — now persist state to match.
	c.activeTask.State = string(c.machine.State())
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		return fmt.Errorf("save active task: %w", err)
	}

	return nil
}

// ImplementReview enters the implementation phase to fix issues from a review.
// Unlike Implement(), this doesn't require specifications - it uses review feedback.
func (c *Conductor) ImplementReview(ctx context.Context, reviewNumber int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return errors.New("no active task")
	}

	// Verify the review exists
	reviews, err := c.workspace.ListReviews(c.activeTask.ID)
	if err != nil {
		return fmt.Errorf("list reviews: %w", err)
	}

	found := false
	for _, r := range reviews {
		if r == reviewNumber {
			found = true

			break
		}
	}
	if !found {
		return fmt.Errorf("review %d not found", reviewNumber)
	}

	// Verify the review is loadable before transitioning state
	// (prevents stuck state if file is corrupted or unreadable)
	if _, err := c.workspace.LoadReview(c.activeTask.ID, reviewNumber); err != nil {
		return fmt.Errorf("load review %d: %w", reviewNumber, err)
	}

	// Dispatch implement event to enter the implementing state.
	if err := c.machine.Dispatch(ctx, workflow.EventImplement); err != nil {
		return fmt.Errorf("enter implementation: %w", err)
	}

	// Machine transitioned successfully — now persist state to match.
	c.activeTask.State = string(c.machine.State())
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		return fmt.Errorf("save active task: %w", err)
	}

	return nil
}

// ResumePaused resumes a task that was paused due to budget limits.
func (c *Conductor) ResumePaused(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return errors.New("no active task")
	}
	if c.activeTask.State != string(workflow.StatePaused) {
		return fmt.Errorf("task is not paused (current state: %s)", c.activeTask.State)
	}

	// Dispatch first to validate the transition before persisting.
	if err := c.machine.Dispatch(ctx, workflow.EventResume); err != nil {
		return fmt.Errorf("resume workflow: %w", err)
	}

	// Machine transitioned successfully — now persist state to match.
	c.activeTask.State = string(c.machine.State())
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		return fmt.Errorf("save active task: %w", err)
	}

	return nil
}

// AnswerQuestion records an answer to a pending question and transitions from waiting state.
func (c *Conductor) AnswerQuestion(ctx context.Context, answer string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return errors.New("no active task")
	}

	taskID := c.activeTask.ID

	// Check if there's a pending question
	if !c.workspace.HasPendingQuestion(taskID) {
		return errors.New("no pending question")
	}

	// Load the pending question to format the answer
	q, err := c.workspace.LoadPendingQuestion(taskID)
	if err != nil {
		return fmt.Errorf("load pending question: %w", err)
	}

	// Save as Q&A note
	note := "**Q:** " + q.Question + "\n\n**A:** " + answer
	if err := c.workspace.AppendNote(taskID, note, "answer"); err != nil {
		return fmt.Errorf("save answer: %w", err)
	}

	// Clear pending question
	if err := c.workspace.ClearPendingQuestion(taskID); err != nil {
		return fmt.Errorf("clear question: %w", err)
	}

	// Dispatch EventAnswer to transition state machine from waiting to idle
	if err := c.machine.Dispatch(ctx, workflow.EventAnswer); err != nil {
		return fmt.Errorf("dispatch answer event: %w", err)
	}

	// Update active task state
	c.activeTask.State = string(c.machine.State())
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		c.logError(fmt.Errorf("save active task after answer: %w", err))
	}

	return nil
}

// ResetState resets the workflow state to idle without losing work.
// Use this to recover from hung agent sessions where the process was killed
// but the state remains stuck in planning/implementing/reviewing.
func (c *Conductor) ResetState(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return errors.New("no active task")
	}

	// Reset to idle
	c.activeTask.State = "idle"
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		return fmt.Errorf("save active task: %w", err)
	}

	// Also reset state machine to idle
	c.machine.Reset()

	return nil
}

// Undo reverts to the previous checkpoint.
func (c *Conductor) Undo(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return errors.New("no active task")
	}

	if c.git == nil {
		return errors.New("git not available")
	}

	taskID := c.activeTask.ID

	// Check if undo is possible
	can, err := c.git.CanUndo(ctx, taskID)
	if err != nil {
		return err
	}
	if !can {
		return errors.New("nothing to undo")
	}

	// Dispatch undo event
	if err := c.machine.Dispatch(ctx, workflow.EventUndo); err != nil {
		return fmt.Errorf("undo workflow: %w", err)
	}

	// Perform git undo
	checkpoint, err := c.git.Undo(ctx, taskID)
	if err != nil {
		return fmt.Errorf("git undo: %w", err)
	}

	// Publish event
	c.eventBus.PublishRaw(eventbus.Event{
		Type: events.TypeCheckpoint,
		Data: map[string]any{
			"action":     "undo",
			"checkpoint": checkpoint.Number,
			"commit":     checkpoint.ID,
		},
	})

	// Complete undo transition.
	if err := c.machine.Dispatch(ctx, workflow.EventUndoDone); err != nil {
		return fmt.Errorf("complete undo transition: %w", err)
	}

	return nil
}

// Redo moves forward to the next checkpoint.
func (c *Conductor) Redo(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return errors.New("no active task")
	}

	if c.git == nil {
		return errors.New("git not available")
	}

	taskID := c.activeTask.ID

	// Check if redo is possible
	can, err := c.git.CanRedo(ctx, taskID)
	if err != nil {
		return err
	}
	if !can {
		return errors.New("nothing to redo")
	}

	// Dispatch redo event
	if err := c.machine.Dispatch(ctx, workflow.EventRedo); err != nil {
		return fmt.Errorf("redo workflow: %w", err)
	}

	// Perform git redo
	checkpoint, err := c.git.Redo(ctx, taskID)
	if err != nil {
		return fmt.Errorf("git redo: %w", err)
	}

	// Publish event
	c.eventBus.PublishRaw(eventbus.Event{
		Type: events.TypeCheckpoint,
		Data: map[string]any{
			"action":     "redo",
			"checkpoint": checkpoint.Number,
			"commit":     checkpoint.ID,
		},
	})

	// Complete redo transition.
	if err := c.machine.Dispatch(ctx, workflow.EventRedoDone); err != nil {
		return fmt.Errorf("complete redo transition: %w", err)
	}

	return nil
}

// Simplify refines content based on the current workflow state.
// It automatically determines what to simplify based on task state:
// - No specs: Simplify input files (task description, source content)
// - Has specs, no implemented files: Simplify specifications
// - Has implemented files: Simplify code (even if review exists).
func (c *Conductor) Simplify(ctx context.Context, targetStep string, createCheckpoint bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return errors.New("no active task")
	}

	taskID := c.activeTask.ID

	// Create checkpoint before modifying files
	if createCheckpoint && c.git != nil {
		c.publishProgress("Creating checkpoint...", 0)
		if event := c.createCheckpointIfNeeded(ctx, taskID, "Simplify"); event != nil {
			c.eventBus.PublishRaw(*event)
		}
	}

	// Auto-detect based on current state
	specs, _ := c.workspace.ListSpecifications(taskID)
	hasSpecs := len(specs) > 0

	hasImplementedFiles := false
	if hasSpecs {
		for _, specNum := range specs {
			spec, err := c.workspace.ParseSpecification(taskID, specNum)
			if err == nil && len(spec.ImplementedFiles) > 0 {
				hasImplementedFiles = true

				break
			}
		}
	}

	// Determine what to simplify based on state
	if !hasSpecs {
		// Pre-plan: simplify input files
		return c.simplifyInput(ctx, taskID)
	} else if hasSpecs && !hasImplementedFiles {
		// After planning: simplify specifications
		return c.simplifyPlanning(ctx, taskID)
	} else if hasImplementedFiles {
		// After implementation (with or without review): simplify code
		return c.simplifyImplementing(ctx, taskID)
	}

	return errors.New("unable to determine what to simplify")
}

// AskQuestion sends a user question to the agent during planning, implementation, or review.
// Does NOT change the workflow state - the agent responds and the current state continues.
// Returns a callback function that the caller should use to stream events.
func (c *Conductor) AskQuestion(ctx context.Context, question string) error {
	c.mu.Lock()

	if c.activeTask == nil {
		c.mu.Unlock()

		return errors.New("no active task")
	}

	taskID := c.activeTask.ID
	currentState := workflow.State(c.activeTask.State)

	// Get title for prompt (with nil check)
	title := ""
	if c.taskWork != nil {
		title = c.taskWork.Metadata.Title
	}

	// Determine which step's agent to use
	var step workflow.Step
	switch currentState {
	case workflow.StatePlanning:
		step = workflow.StepPlanning
	case workflow.StateImplementing:
		step = workflow.StepImplementing
	case workflow.StateReviewing:
		step = workflow.StepReviewing
	case workflow.StateIdle, workflow.StateDone, workflow.StateFailed, workflow.StateWaiting, workflow.StatePaused, workflow.StateCheckpointing, workflow.StateReverting, workflow.StateRestoring:
		c.mu.Unlock()

		return fmt.Errorf("cannot ask questions in state '%s'; use during planning, implementing, or reviewing", currentState)
	}

	// Get session history before releasing lock
	var sessionHistory string
	if c.currentSession != nil && len(c.currentSession.Exchanges) > 0 {
		// Get last 10 exchanges for context (most recent first, limit to avoid token bloat)
		historyStart := len(c.currentSession.Exchanges) - 10
		if historyStart < 0 {
			historyStart = 0
		}
		recentExchanges := c.currentSession.Exchanges[historyStart:]

		var historyBuilder strings.Builder
		for _, ex := range recentExchanges {
			historyBuilder.WriteString(fmt.Sprintf("**%s:** %s\n", ex.Role, ex.Content))
		}
		sessionHistory = historyBuilder.String()
	}

	// Release lock before calling GetAgentForStep (which also acquires the lock)
	c.mu.Unlock()

	// Get agent for this step (this will acquire its own lock)
	questionAgent, err := c.GetAgentForStep(ctx, step)
	if err != nil {
		return fmt.Errorf("get agent for question: %w", err)
	}

	// Get latest specification content for context
	specificationContent, _, _ := c.workspace.GetLatestSpecificationContent(taskID)

	// Build prompt
	prompt := buildQuestionPrompt(title, question, specificationContent, sessionHistory)

	// Record user question in session (re-acquire lock).
	c.mu.Lock()
	defer c.mu.Unlock()

	// Re-validate: activeTask may have been cleared while lock was released.
	if c.activeTask == nil {
		return errors.New("task was cleared while resolving agent")
	}
	if c.activeTask.ID != taskID {
		return errors.New("active task changed while resolving agent")
	}

	if c.currentSession != nil {
		c.currentSession.Exchanges = append(c.currentSession.Exchanges, storage.Exchange{
			Role:      "user",
			Content:   "QUESTION: " + question,
			Timestamp: time.Now(),
		})
	}

	// Run agent with streaming callback
	c.logVerbosef("Asking agent question...")
	response, err := questionAgent.RunWithCallback(ctx, prompt, func(event agent.Event) error {
		// Publish to event bus for Web UI consumption
		c.eventBus.PublishRaw(eventbus.Event{
			Type: events.TypeAgentMessage,
			Data: map[string]any{"event": event},
		})

		return nil
	})
	if err != nil {
		return fmt.Errorf("agent question: %w", err)
	}

	// Record agent response in session
	if c.currentSession != nil && response != nil {
		responseContent := response.Summary
		if responseContent == "" && len(response.Messages) > 0 {
			responseContent = strings.Join(response.Messages, "\n\n")
		}
		c.currentSession.Exchanges = append(c.currentSession.Exchanges, storage.Exchange{
			Role:      "assistant",
			Content:   responseContent,
			Timestamp: time.Now(),
		})
	}

	// Handle if agent asked a back-question
	if response != nil && response.Question != nil {
		// Convert agent.QuestionOption to storage.QuestionOption
		var options []storage.QuestionOption
		for _, opt := range response.Question.Options {
			options = append(options, storage.QuestionOption{
				Label:       opt.Label,
				Description: opt.Description,
			})
		}
		pendingQuestion := &storage.PendingQuestion{
			Question:    response.Question.Text,
			Options:     options,
			FullContext: prompt,
		}
		if err := c.workspace.SavePendingQuestion(taskID, pendingQuestion); err != nil {
			c.logError(fmt.Errorf("save pending question: %w", err))
		}
		// Transition to waiting state
		if err := c.dispatchWithRetry(ctx, workflow.EventWait); err != nil {
			return err
		}

		return ErrPendingQuestion
	}

	return nil
}

// Finish completes the task.
func (c *Conductor) Finish(ctx context.Context, opts FinishOptions) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return errors.New("no active task")
	}

	// Determine action based on flags and provider support
	if opts.ForceMerge {
		// User explicitly requested local merge
		if err := c.finishWithMerge(ctx, opts); err != nil {
			return err
		}
	} else if c.providerSupportsPR(ctx) {
		// Provider supports PR, create one by default
		prResult, err := c.finishWithPR(ctx, opts)
		if err != nil {
			return err
		}
		// Store PR info for later reference.
		if prResult != nil {
			c.lastPRResult = prResult
			c.logVerbosef("Created PR #%d: %s", prResult.Number, prResult.URL)

			// Try auto-rebase for stacked features (after PR creation)
			// This is non-blocking: failures are logged but don't fail the finish operation
			taskID := c.activeTask.ID
			rebaseResult := c.tryAutoRebase(ctx, taskID, opts)
			if rebaseResult != nil && rebaseResult.Executed {
				c.logVerbosef("Auto-rebased %d task(s)", len(rebaseResult.Result.RebasedTasks))
			}
		}
	} else if c.git != nil && c.activeTask.UseGit && c.activeTask.Branch != "" {
		// Provider doesn't support PR, ask user what to do
		action := c.askUserFinishAction()
		switch action {
		case "merge":
			if err := c.finishWithMerge(ctx, opts); err != nil {
				return err
			}
		case "done":
			// Just mark as done, no merge
			c.logVerbosef("Marking task as done without merging")
		case "cancel":
			return errors.New("cancelled by user")
		}
	} else {
		// No git, just mark as done
		c.logVerbosef("No git branch associated, marking task as done")
	}

	// Dispatch finish event first to validate the transition before persisting.
	if err := c.machine.Dispatch(ctx, workflow.EventFinish); err != nil {
		return fmt.Errorf("finish workflow: %w", err)
	}

	// Machine transitioned successfully — now persist state to match.
	c.activeTask.State = string(c.machine.State())
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		return fmt.Errorf("save active task: %w", err)
	}

	// Index completed task into memory (if enabled)
	if err := c.IndexCompletedTask(ctx); err != nil {
		// Memory indexing is non-critical, log error but don't fail
		c.logError(fmt.Errorf("index completed task (non-fatal): %w", err))
	}

	// Clear active task
	if err := c.workspace.ClearActiveTask(); err != nil {
		c.logError(fmt.Errorf("clear active task: %w", err))
	}

	// Delete work directory based on: CLI flag > config > default (keep)
	var shouldDelete bool
	if opts.DeleteWork != nil {
		shouldDelete = *opts.DeleteWork // CLI explicitly set
	} else {
		cfg, _ := c.workspace.LoadConfig() // ignore error, use defaults
		shouldDelete = cfg.Workflow.DeleteWorkOnFinish
	}
	if shouldDelete {
		taskID := c.activeTask.ID
		if err := c.workspace.DeleteWork(taskID); err != nil {
			c.logError(fmt.Errorf("delete work directory: %w", err))
		}
	}

	c.activeTask = nil
	c.taskWork = nil

	return nil
}

// buildWorkUnit creates a workflow.WorkUnit from current state.
func (c *Conductor) buildWorkUnit() *workflow.WorkUnit {
	if c.taskWork == nil {
		return nil
	}

	wu := &workflow.WorkUnit{
		ID:         c.taskWork.Metadata.ID,
		ExternalID: c.taskWork.Source.Ref,
		Title:      c.taskWork.Metadata.Title,
		Source: &workflow.Source{
			Reference: c.taskWork.Source.Ref,
			Content:   c.taskWork.Source.Content,
		},
	}

	// Add specifications if any - errors ignored; empty list is acceptable for WorkUnit
	specifications, _ := c.workspace.ListSpecifications(c.taskWork.Metadata.ID)
	for _, num := range specifications {
		wu.Specifications = append(wu.Specifications, fmt.Sprintf("specification-%d.md", num))
	}

	return wu
}

// onStateChanged handles state change events.
func (c *Conductor) onStateChanged(e eventbus.Event) {
	if c.opts.OnStateChange == nil {
		return
	}

	from, ok := e.Data["from"].(string)
	if !ok {
		from = ""
	}
	to, ok := e.Data["to"].(string)
	if !ok {
		to = ""
	}
	c.opts.OnStateChange(from, to)
}

// countCheckpoints returns the number of checkpoints for current task.
func (c *Conductor) countCheckpoints(ctx context.Context) int {
	if c.activeTask == nil || c.git == nil {
		return 0
	}
	checkpoints, err := c.git.ListCheckpoints(ctx, c.activeTask.ID)
	if err != nil {
		return 0
	}

	return len(checkpoints)
}

// publishProgress publishes a progress event.
func (c *Conductor) publishProgress(message string, percent int) {
	c.eventBus.PublishRaw(eventbus.Event{
		Type: events.TypeProgress,
		Data: map[string]any{
			"message": message,
			"percent": percent,
		},
	})

	if c.opts.OnProgress != nil {
		c.opts.OnProgress(message, percent)
	}
}

// finishWithMerge performs a local merge operation.
func (c *Conductor) finishWithMerge(ctx context.Context, opts FinishOptions) error {
	// Handle git merge operations if applicable
	if c.git == nil || !c.activeTask.UseGit || c.activeTask.Branch == "" {
		return errors.New("git not available or no branch associated with task")
	}

	if err := c.performMerge(ctx, opts); err != nil {
		return err
	}

	// Push if requested
	if opts.PushAfter {
		remote, err := c.git.GetDefaultRemote(ctx)
		if err != nil {
			return fmt.Errorf("get default remote: %w", err)
		}
		targetBranch := c.resolveTargetBranch(ctx, opts.TargetBranch)
		if err := c.git.PushBranch(ctx, targetBranch, remote, false); err != nil {
			return fmt.Errorf("push: %w", err)
		}
	}

	// Cleanup branch and worktree if requested
	c.cleanupAfterMerge(ctx, opts)

	return nil
}

// providerSupportsPR checks if the current task's provider supports PR creation.
func (c *Conductor) providerSupportsPR(ctx context.Context) bool {
	if c.activeTask == nil || c.activeTask.Ref == "" {
		return false
	}

	// Resolve provider from the stored reference
	resolveOpts := provider.ResolveOptions{
		DefaultProvider: c.opts.DefaultProvider,
	}

	// Load workspace config and build provider config
	workspaceCfg, _ := c.workspace.LoadConfig() // ignore error, use defaults
	scheme := parseScheme(c.activeTask.Ref)
	providerCfg := buildProviderConfig(workspaceCfg, scheme)

	p, _, err := c.providers.Resolve(ctx, c.activeTask.Ref, providerCfg, resolveOpts)
	if err != nil {
		return false
	}

	// Check if provider implements PRCreator interface
	_, ok := p.(provider.PRCreator)

	return ok
}

// askUserFinishAction prompts the user to choose an action when PR is not supported.
func (c *Conductor) askUserFinishAction() string {
	// For non-interactive use (auto mode), default to "done"
	if c.opts.AutoMode || c.opts.SkipAgentQuestions {
		return "done"
	}

	fmt.Println("\nThe provider for this task does not support pull requests.")
	fmt.Println("What would you like to do?")
	fmt.Println("  1. Merge changes to target branch locally")
	fmt.Println("  2. Mark task as done (no merge)")
	fmt.Println("  3. Cancel")

	for {
		var choice string
		fmt.Print("\nEnter choice (1-3): ")
		if _, err := fmt.Scanln(&choice); err != nil {
			// Handle EOF or empty input
			fmt.Println("\nCancelled")

			return "cancel"
		}

		switch choice {
		case "1", "merge":
			return "merge"
		case "2", "done":
			return "done"
		case "3", "cancel", "q":
			return "cancel"
		default:
			fmt.Println("Invalid choice. Please enter 1, 2, or 3.")
		}
	}
}

// filterSpecificationsByComponent filters specifications by component name.
// Matches if the spec's component field exactly matches or is contained in the component list.
func (c *Conductor) filterSpecificationsByComponent(taskID string, specNumbers []int, component string) []int {
	var filtered []int

	for _, num := range specNumbers {
		spec, err := c.workspace.ParseSpecification(taskID, num)
		if err != nil {
			c.logVerbosef("Failed to parse spec %d: %v", num, err)

			continue
		}

		// Check if component matches
		if spec.Component == component {
			filtered = append(filtered, num)
		}
	}

	return filtered
}
